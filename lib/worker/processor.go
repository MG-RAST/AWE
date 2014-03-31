package worker

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/wgerlach/go-dockerclient"
	"github.com/davecgh/go-spew/spew"
	"io"
	"os"
	"os/exec"
	"runtime"
	"net/url"
	"strings"
	"bytes"
	"time"
	"compress/gzip"
)

func processor(control chan int) {
	fmt.Printf("processor lanched, client=%s\n", core.Self.Id)
	defer fmt.Printf("processor exiting...\n")
	for {
		parsedwork := <-fromMover
		work := parsedwork.workunit
		workmap[work.Id] = ID_WORKER

		processed := &mediumwork{
			workunit: work,
			perfstat: parsedwork.perfstat,
		}

		//if the work is not succesfully parsed in last stage, pass it into the next one immediately
		if work.State == core.WORK_STAT_FAIL {
			processed.workunit.State = core.WORK_STAT_FAIL
			fromProcessor <- processed
			//release the permit lock, for work overlap inhibitted mode only
			if !conf.WORKER_OVERLAP && core.Service != "proxy" {
				<-chanPermit
			}
			continue
		}

		envkeys := SetEnv(work)

		run_start := time.Now().Unix()
		
		pstat, err := RunWorkunit(work)
		
		if err != nil {
			fmt.Printf("!!!RunWorkunit() returned error: %s\n", err.Error())
			logger.Error("RunWorkunit(): workid=" + work.Id + ", " + err.Error())
			processed.workunit.State = core.WORK_STAT_FAIL
		} else {
			processed.workunit.State = core.WORK_STAT_COMPUTED
			processed.perfstat.MaxMemUsage = pstat.MaxMemUsage
		}
		run_end := time.Now().Unix()
		computetime := run_end - run_start
		processed.perfstat.Runtime = computetime
		processed.workunit.ComputeTime = int(computetime)

		if len(envkeys) > 0 {
			UnSetEnv(envkeys)
		}

		fromProcessor <- processed

		//release the permit lock, for work overlap inhibitted mode only
		if !conf.WORKER_OVERLAP && core.Service != "proxy" {
			<-chanPermit
		}
	}
	control <- ID_WORKER //we are ending
}

func RunWorkunit(work *core.Workunit) (pstats *core.WorkPerf, err error) {
	
	if work.Cmd.Dockerimage == "" { 
		return RunWorkunitDirect(work)
	} else {
		return RunWorkunitDocker(work)
	}
	
	
}

func RunWorkunitDocker(work *core.Workunit) (pstats *core.WorkPerf, err error) {
	pstats = new(core.WorkPerf)

	args := work.Cmd.ParsedArgs

	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return nil, err
	}

	commandName := work.Cmd.Name
	//cmd := exec.Command(commandName, args...)

	
	container_name := "AWE_container"
		
	Dockerimage := work.Cmd.Dockerimage
	
	logger.Debug(1, fmt.Sprintf("Dockerimage: %s", Dockerimage))		
	

	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error creating docker client", err.Error()))
	}
	
	imgs, _ := client.ListImages(false)
	for _, img := range imgs {
		spew.Dump(img)
	}

	containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})
	
	// delete any AWE_container or container from same image
	for _, cont := range containers {
		spew.Dump(cont)
		delete_old_container := false
		if cont.Image == Dockerimage {
			delete_old_container = true
		} else {}
			for _, cname := range cont.Names {
				if cname == container_name {
					delete_old_container = true
				}
		}
		
		if delete_old_container == true {
			container, err := client.InspectContainer(cont.ID)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("error inspecting old container id=%s, err=%s", cont.ID, err.Error()))
			}
			if container.State.Running == true {
				err := client.KillContainer(cont.ID)
				if err != nil {
					return nil, errors.New(fmt.Sprintf("error killing old container id=%s, err=%s", cont.ID, err.Error()))
				}
			}
			container, err = client.InspectContainer(cont.ID)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("error inspecting old container id=%s, err=%s", cont.ID, err.Error()))
			}
			if container.State.Running == true {
				return nil, errors.New(fmt.Sprintf("old container is still running"))
			}
			
			c_remove_opts := docker.RemoveContainerOptions{ID: cont.ID}
			err = client.RemoveContainer(c_remove_opts)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("error removing old container id=%s, err=%s", cont.ID, err.Error()))
			}
			
		}
		
	}

	// *** find/inspect image
	image, err := client.InspectImage(Dockerimage)
	
	if err != nil {
		fmt.Println("image not found !? ")

		logger.Debug(1, fmt.Sprintf("docker image %s is not yet in local repository", Dockerimage))
		
		
		
		image_retrieval := "load"
		switch  {
			case image_retrieval == "load" : { // for images that have been saved
				err = dockerLoadImage(client, Dockerimage)
			}
			case image_retrieval == "import" : { // for containers that have been exported
				err = dockerImportImage(client, Dockerimage)
			}
			case image_retrieval == "build" : { // to create image from Dockerfile
				err = dockerBuildImage(client, Dockerimage)	
			}
		}
		
		if err != nil {
			return nil, errors.New(fmt.Sprintf("Docker image was not correctly imported or built, err=%s", err.Error()))
		}					
		// example urls							
		// find image : http://shock.metagenomics.anl.gov/node/?query&docker=1&tag=wgerlach/bowtie2:2.2.0
		// view node: http://shock.metagenomics.anl.gov/node/ed0a6b20-c535-40d7-92e8-754bb8b6b48f
		// download http://shock.metagenomics.anl.gov/node/ed0a6b20-c535-40d7-92e8-754bb8b6b48f?download
		
		// last test
		image, err = client.InspectImage(Dockerimage)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("(InspectImage) Docker image was not correctly imported or built, err=%s", err.Error()))
		}
	} else {
		logger.Debug(1, fmt.Sprintf("docker image %s is already in local repository", Dockerimage))
	}

	

	imageid := image.ID
	
	pipe_output := fmt.Sprintf(" 2> %s 1> %s", conf.STDERR_FILENAME ,conf.STDOUT_FILENAME)

	bash_command := fmt.Sprint(commandName, " ", strings.Join(args, " ") , " ", pipe_output)
	
	logger.Debug(1, fmt.Sprint("bash_command: ", bash_command))
	
	// example: "/bin/bash", "-c", "bowtie2 -h 2> awe_stderr.txt 1> awe_stdout.txt"
	container_cmd := []string{"/bin/bash", "-c", bash_command}
	
	
	
	config := docker.Config{Image: imageid, WorkingDir: conf.DOCKER_WORK_DIR, AttachStdout: true, AttachStderr: true, AttachStdin: false, Cmd: container_cmd, Volumes: map[string]struct{}{conf.DOCKER_WORK_DIR: {} }}
	opts := docker.CreateContainerOptions{Name: container_name, Config: &config}




	// *** create container (or find container ?)
	container_incomplete, err := client.CreateContainer(opts)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error creating container, err=%s",err.Error()))
	}
	container_id := container_incomplete.ID;
	logger.Debug(1,  fmt.Sprintf("got container ID: %s", container_id))
	
	
	
	// *** inspect the new container
	if false {
		container, err := client.InspectContainer(container_id)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error inspectinf container, err=%s",err.Error()))
		}
		spew.Dump(container)
		spew.Dump(container.Config)
		fmt.Println("name: ", container.Name)
	}
	
	
	
	
	
	// *** start container
	bindstr := fmt.Sprintf("%s/:%s", work.Path(), conf.DOCKER_WORK_DIR)
	info_binstr := fmt.Sprintf("docker container bindstr: (%s)", bindstr)
	logger.Debug(1,  fmt.Sprintf(info_binstr))
	
	err = client.StartContainer(container_id, &docker.HostConfig{Binds: []string{bindstr}})
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error starting container, id=%s, err=%s", container_id, err.Error()))
	}
	
	
	
	var status int = 0
	
	// wait for container to finish
	done := make(chan error)
	go func() {
		status, err = client.WaitContainer(container_id)
		done <- err // inform main function
		done <- err // inform memory checker
	}()
	
		
	var MaxMem uint64 = 0		
	go func() {
		
		for {
			
			select {
				case <-done:
				   return
				default:
			}
			
			container, err := client.InspectContainer(container_id)
			
			if err != nil {
				logger.Debug(1, fmt.Sprint("error inspecting container (for mem info) ", err.Error()))
			} else {
				memory := uint64(container.Config.Memory)
				//fmt.Println("memory: ", memory )
				if memory > MaxMem {
					MaxMem = memory
				}
			}
			
			time.Sleep(3 * time.Second)
			//time.Sleep(conf.MEM_CHECK_INTERVAL)
		}
	}()			
						
	
	select {
		case <-chankill:
		
			if err := client.KillContainer(container_id); err != nil {
				return nil, errors.New(fmt.Sprintf("error killing container id=%s, err=%s", container_id, err.Error()))
			}
			
			<-done // allow goroutine to exit
			
			return nil, errors.New("process killed as requested from chankill")
		case err := <-done:
			if err != nil {
				return nil, errors.New(fmt.Sprintf("wait_cmd=%s, err=%s", commandName, err.Error()))
			}
			logger.Debug(1, fmt.Sprint("docker command returned with status ", status) )
	}
	
	if status != 0 {
		return nil, errors.New(fmt.Sprintf("error status not zero"))
	}
	logger.Debug(1, fmt.Sprint("pstats.MaxMemUsage: ", pstats.MaxMemUsage))
	pstats.MaxMemUsage = MaxMem
	
	// *** clean up
	// ** kill container
	err = client.KillContainer(container_id)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error killing container id=%s, err=%s", container_id, err.Error()))
	}
	
	// *** remove Container
	opts_remove := docker.RemoveContainerOptions{ID: container_id}
	err = client.RemoveContainer(opts_remove)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error removing container id=%s, err=%s", container_id, err.Error()))
	}
	
	logger.Debug(1, "finished and removed container!")
	return pstats, err
}


func RunWorkunitDirect(work *core.Workunit) (pstats *core.WorkPerf, err error) {
	pstats = new(core.WorkPerf)

	args := work.Cmd.ParsedArgs

	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return nil, err
	}

	commandName := work.Cmd.Name
	cmd := exec.Command(commandName, args...)

	msg := fmt.Sprintf("worker: start cmd=%s, args=%v", commandName, args)
	fmt.Println(msg)
	logger.Debug(1, msg)
	logger.Event(event.WORK_START, "workid="+work.Id,
		"cmd="+commandName,
		fmt.Sprintf("args=%v", args))

	var stdout, stderr io.ReadCloser
	if conf.PRINT_APP_MSG {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		stderr, err = cmd.StderrPipe()
		if err != nil {
			return nil, err
		}
	}

	stdoutFilePath := fmt.Sprintf("%s/%s", work.Path(), conf.STDOUT_FILENAME)
	stderrFilePath := fmt.Sprintf("%s/%s", work.Path(), conf.STDERR_FILENAME)
	outfile, err := os.Create(stdoutFilePath)
	defer outfile.Close()
	errfile, err := os.Create(stderrFilePath)
	defer errfile.Close()
	out_writer := bufio.NewWriter(outfile)
	defer out_writer.Flush()
	err_writer := bufio.NewWriter(errfile)
	defer err_writer.Flush()

	if conf.PRINT_APP_MSG {
		go io.Copy(out_writer, stdout)
		go io.Copy(err_writer, stderr)
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.New(fmt.Sprintf("start_cmd=%s, err=%s", commandName, err.Error()))
	}

	var MaxMem uint64 = 0
	done := make(chan error)
	memcheck_done := make(chan bool)
	go func() {
		done <- cmd.Wait()
		memcheck_done <- true
	}()
	go func() {
		mstats := new(runtime.MemStats)
		runtime.ReadMemStats(mstats)
		MaxMem = mstats.Alloc
		time.Sleep(2 * time.Second)
		for {
			select {
			default:
				mstats := new(runtime.MemStats)
				runtime.ReadMemStats(mstats)
				if mstats.Alloc > MaxMem {
					MaxMem = mstats.Alloc
				}
				time.Sleep(conf.MEM_CHECK_INTERVAL)
			case <-memcheck_done:
				return
			}
		}
	}()

	select {
	case <-chankill:
		if err := cmd.Process.Kill(); err != nil {
			fmt.Println("failed to kill" + err.Error())
		}
		<-done // allow goroutine to exit
		fmt.Println("process killed")
		return nil, errors.New("process killed")
	case err := <-done:
		if err != nil {
			return nil, errors.New(fmt.Sprintf("wait_cmd=%s, err=%s", commandName, err.Error()))
		}
	}
	logger.Event(event.WORK_END, "workid="+work.Id)
	pstats.MaxMemUsage = MaxMem
	return
}


func dockerBuildImage(client *docker.Client, Dockerimage string) (err error) {
	
	shock_docker_repo := core.ShockClient{conf.SHOCK_DOCKER_IMAGE_REPOSITORY, ""}
	
	logger.Debug(1, fmt.Sprint("try to build docker image from dockerfile, Dockerimage=", Dockerimage))
	
	query_response_p, err := shock_docker_repo.Query(url.Values{"dockerfile": {"1"}, "tag" : {Dockerimage}});
	if err != nil {
		return errors.New(fmt.Sprintf("shock node not found for dockerfile=%s, err=%s", Dockerimage, err.Error()))
	}
	logger.Debug(1, fmt.Sprintf("query result: %v", query_response_p))
	
	
	
	datalen := len((*query_response_p).Data)
	
	if datalen == 0 {
		return errors.New(fmt.Sprintf("Dockerfile %s not found in shocks docker repo", Dockerimage))
	} else if datalen > 1 {
		return errors.New(fmt.Sprintf("more than one Dockerfile %s found in shocks docker repo", Dockerimage))
	}
	
	node := (*query_response_p).Data[0]
	logger.Debug(1, fmt.Sprintf("found SHOCK node for Dockerfile: %s", node.Id))
	
	download_url, err := shock_docker_repo.Get_node_download_url(node)
	if err != nil {
		return errors.New(fmt.Sprintf("Could not create download url, err=%s", err.Error()))
	}
	
	// get and build Dockerfile
	var buf bytes.Buffer
	opts := docker.BuildImageOptions{
		Name: "testImage",
		Remote: download_url,
		SuppressOutput: true,
		OutputStream: &buf,
	}
	err = client.BuildImage(opts)
	if err != nil {
		return errors.New(fmt.Sprintf("Error importing docker image, err=%s", err.Error()))
	}
	
	return nil
}


func getDockerImageUrl(Dockerimage string) (download_url string, err error) {
	
	shock_docker_repo := core.ShockClient{conf.SHOCK_DOCKER_IMAGE_REPOSITORY, ""}
	
	logger.Debug(1, fmt.Sprint("try to import docker image, Dockerimage=", Dockerimage))
	//query url = "docker=1&tag=wgerlach/bowtie2:2.2.0"
	
	
	query_response_p, err := shock_docker_repo.Query(url.Values{"docker": {"1"}, "tag" : {Dockerimage}});
	if err != nil {
		return "", errors.New(fmt.Sprintf("shock node not found for image=%s, err=%s", Dockerimage, err.Error()))
	}
	logger.Debug(1, fmt.Sprintf("query result: %v", query_response_p))
	
	datalen := len((*query_response_p).Data)
	
	if datalen == 0 {
		return "", errors.New(fmt.Sprintf("image %s not found in shocks docker repo", Dockerimage))
	} else if datalen > 1 {
		return "", errors.New(fmt.Sprintf("more than one image %s found in shocks docker repo", Dockerimage))
	}
	
	node := (*query_response_p).Data[0]
	logger.Debug(1, fmt.Sprintf("found SHOCK node for docker image: %s", node.Id))
	
	download_url, err = shock_docker_repo.Get_node_download_url(node)
	if err != nil {
		return "", errors.New(fmt.Sprintf("Could not create download url, err=%s", err.Error()))
	}
	
	return
}

func dockerLoadImage(client *docker.Client, Dockerimage string) (err error) {
	
	download_url, err := getDockerImageUrl(Dockerimage)
	if (err != nil) {
		return errors.New(fmt.Sprintf("Error getting docker url, err=%s", err.Error()))
	}
	
	image_stream, err := fetchShockStream(download_url, "") // token empty here, assume that images are public
	if (err != nil) {
		return errors.New(fmt.Sprintf("Error getting Shock stream, err=%s", err.Error()))
	}
	
	gr, err := gzip.NewReader(image_stream) //returns (*Reader, error) // TODO not sure if I have to close gr later ?
	
	logger.Debug(1, fmt.Sprintf("loading image..."))
	//go io.Copy(image_writer, image_stream)
	
	var buf bytes.Buffer
	
	err = client.LoadImage(gr, &buf) // in io.Reader, w io.Writer
	if (err != nil) {
		return errors.New(fmt.Sprintf("Error loading image, err=%s", err.Error()))
	}
	
	logger.Debug(1, fmt.Sprintf("load image returned: %v", &buf))
	return nil
}

func dockerImportImage(client *docker.Client, Dockerimage string) (err error) {
	
	
	download_url, err := getDockerImageUrl(Dockerimage)
	
	if (err != nil) {
		return err
	}
	
	logger.Debug(1, fmt.Sprintf("docker image url=%s", download_url))
	
	// TODO import base image if needed
	
	
	// *** import image
	Dockerimage_array := strings.Split(Dockerimage, ":")
	Dockerimage_repo, Dockerimage_tag := Dockerimage_array[0], Dockerimage_array[1]
	
	var buf bytes.Buffer
	opts := docker.ImportImageOptions{
		Source: download_url,
		Repository: Dockerimage_repo,
		Tag: Dockerimage_tag,
		OutputStream: &buf,
	}
	
	err = client.ImportImage(opts)
	if err != nil {
		return errors.New(fmt.Sprintf("Error importing docker image, err=%s", err.Error()))
	}
	
	return	
}

func SetEnv(work *core.Workunit) (envkeys []string) {
	for key, val := range work.Cmd.Environ {
		if key == conf.KB_AUTH_TOKEN {
			if work.Info.DataToken != "" {
				if err := os.Setenv(key, work.Info.DataToken); err == nil {
					envkeys = append(envkeys, key)
				}
			}
		} else {
			if oldval := os.Getenv(key); oldval == "" {
				if err := os.Setenv(key, val); err == nil {
					envkeys = append(envkeys, key)
				}
			}

		}
	}
	return
}

func UnSetEnv(envkeys []string) {
	for _, key := range envkeys {
		os.Setenv(key, "")
	}
}
