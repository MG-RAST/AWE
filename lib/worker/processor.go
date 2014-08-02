package worker

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/httpclient"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	//"github.com/davecgh/go-spew/spew"
	"github.com/wgerlach/go-dockerclient"
	"io"
	"io/ioutil"
	//"log"
	"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type Shock_Dockerimage_attributes struct {
	Id          string `bson:"id" json:"id"`                       // this is docker image id, not Shock id
	Name        string `bson:"name" json:"name"`                   // docker image name
	Type        string `bson:"type" json:"type"`                   // should be "dockerimage"
	BaseImageId string `bson:"base_image_id" json:"base_image_id"` // could used to reference parent image
}

func processor(control chan int) {
	fmt.Printf("processor launched, client=%s\n", core.Self.Id)
	defer fmt.Printf("processor exiting...\n")
	for {
		parsedwork := <-fromMover
		work := parsedwork.workunit

		processed := &mediumwork{
			workunit: work,
			perfstat: parsedwork.perfstat,
		}

		//if the work is not succesfully parsed in last stage, pass it into the next one immediately
		if work.State == core.WORK_STAT_FAIL || workmap[work.Id] == ID_DISCARDED {
			if workmap[work.Id] == ID_DISCARDED {
				processed.workunit.State = core.WORK_STAT_DISCARDED
			} else {
				processed.workunit.State = core.WORK_STAT_FAIL
			}
			fromProcessor <- processed
			//release the permit lock, for work overlap inhibitted mode only
			if !conf.WORKER_OVERLAP && core.Service != "proxy" {
				<-chanPermit
			}
			continue
		}

		workmap[work.Id] = ID_WORKER

		envkeys, err := SetEnv(work)
		if err != nil {
			logger.Error("SetEnv(): workid=" + work.Id + ", " + err.Error())
			processed.workunit.Notes = processed.workunit.Notes + "###[precessor#SetEnv]" + err.Error()
			processed.workunit.State = core.WORK_STAT_FAIL
			//release the permit lock, for work overlap inhibitted mode only
			if !conf.WORKER_OVERLAP && core.Service != "proxy" {
				<-chanPermit
			}
			continue
		}

		run_start := time.Now().Unix()

		pstat, err := RunWorkunit(work)

		if err != nil {
			logger.Error("RunWorkunit(): returned error , workid=" + work.Id + ", " + err.Error())
			processed.workunit.Notes = processed.workunit.Notes + "###[precessor#RunWorkunit]" + err.Error()
			processed.workunit.State = core.WORK_STAT_FAIL
		} else {
			logger.Debug(1, "RunWorkunit() returned without error, workid="+work.Id)
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

func RemoveOldAWEContainers(client *docker.Client, container_name string) (err error) {

	containers, _ := client.ListContainers(docker.ListContainersOptions{All: true})

	old_containers_deleted := 0
	for _, cont := range containers {
		//spew.Dump(cont)
		delete_old_container := false

		logger.Debug(1, fmt.Sprintf("(RemoveOldAWEContainers) check container with ID: %s", cont.ID))
		for _, cname := range cont.Names {
			logger.Debug(1, fmt.Sprintf("(RemoveOldAWEContainers) container name: %s", cname))
			if cname == container_name {
				delete_old_container = true
			}
			if cname == "/"+container_name {
				delete_old_container = true
			}
		}

		if delete_old_container == true {
			logger.Debug(1, fmt.Sprintf("(RemoveOldAWEContainers) found old container %s and try to delete it...", container_name))
			container, err := client.InspectContainer(cont.ID)
			if err != nil {
				return errors.New(fmt.Sprintf("(RemoveOldAWEContainers) error inspecting old container id=%s, err=%s", cont.ID, err.Error()))
			}
			if container.State.Running == true {
				logger.Debug(1, fmt.Sprintf("(RemoveOldAWEContainers) try to kill old container %s...", container_name))
				err := client.KillContainer(docker.KillContainerOptions{ID: cont.ID})
				if err != nil {
					return errors.New(fmt.Sprintf("(RemoveOldAWEContainers) error killing old container id=%s, err=%s", cont.ID, err.Error()))
				}
			}
			container, err = client.InspectContainer(cont.ID)
			if err != nil {
				return errors.New(fmt.Sprintf("(RemoveOldAWEContainers) error inspecting old container id=%s, err=%s", cont.ID, err.Error()))
			}
			if container.State.Running == true {
				return errors.New(fmt.Sprintf("(RemoveOldAWEContainers) old container is still running"))
			}
			logger.Debug(1, fmt.Sprintf("(RemoveOldAWEContainers) try to remove old container %s...", container_name))
			c_remove_opts := docker.RemoveContainerOptions{ID: cont.ID}
			err = client.RemoveContainer(c_remove_opts)
			if err != nil {
				return errors.New(fmt.Sprintf("(RemoveOldAWEContainers) error removing old container id=%s, err=%s", cont.ID, err.Error()))
			}
			logger.Debug(1, fmt.Sprintf("(RemoveOldAWEContainers) old container %s should have been removed", container_name))
			old_containers_deleted += 1
		}

	}
	logger.Debug(1, fmt.Sprintf("old_containers_deleted: %d", old_containers_deleted))
	return
}

func RunWorkunitDocker(work *core.Workunit) (pstats *core.WorkPerf, err error) {
	pstats = new(core.WorkPerf)
	pstats.MaxMemUsage = -1
	pstats.MaxMemoryTotalRss = -1
	pstats.MaxMemoryTotalSwap = -1
	args := work.Cmd.ParsedArgs

	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return nil, err
	}

	commandName := work.Cmd.Name

	use_wrapper_script := false

	wrapper_script_filename := "awe_workunit_wrapper.sh"
	wrapper_script_filename_host := path.Join(work.Path(), wrapper_script_filename)
	wrapper_script_filename_docker := path.Join(conf.DOCKER_WORK_DIR, wrapper_script_filename)
	if strings.HasPrefix(commandName, "app:") {

		if len(work.Cmd.ParsedArgs) > 0 {
			use_wrapper_script = true

			// create wrapper script

			//conf.DOCKER_WORK_DIR
			var wrapper_content_string = "#!/bin/bash\n" + strings.Join(work.Cmd.Cmd_script, "\n") + "\n"

			logger.Debug(1, fmt.Sprintf("write wrapper script: %s\n%s", wrapper_script_filename_host, strings.Join(work.Cmd.Cmd_script, ", ")))

			var wrapper_content_bytes = []byte(wrapper_content_string)

			err = ioutil.WriteFile(wrapper_script_filename_host, wrapper_content_bytes, 0644)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("error writing wrapper script, err=%s", err.Error()))
			}

		}

	}

	//cmd := exec.Command(commandName, args...)

	container_name := "AWE_workunit"

	Dockerimage := work.Cmd.Dockerimage

	logger.Debug(1, fmt.Sprintf("Dockerimage: %s", Dockerimage))

	endpoint := "unix:///var/run/docker.sock"
	client, err := docker.NewClient(endpoint)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error creating docker client", err.Error()))
	}

	//imgs, _ := client.ListImages(false)
	//for _, img := range imgs {
	//	spew.Dump(img)
	//}

	// delete any old AWE_container
	err = RemoveOldAWEContainers(client, container_name)
	if err != nil {
		return nil, err
	}

	//var node *core.ShockNode = nil
	// find image in repo (e.g. extract docker image id)
	node, dockerimage_download_url, err := findDockerImageInShock(Dockerimage)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error getting docker url, err=%s", err.Error()))
	}

	node_attr_map, ok := node.Attributes.(map[string]interface{})
	if !ok {
		return nil, errors.New(fmt.Sprintf("(1) could not type assert Shock_Dockerimage_attributes, Dockerimage=%s", Dockerimage))
	}

	dockerimage_id, ok := node_attr_map["id"].(string)
	if !ok {
		return nil, errors.New(fmt.Sprintf("(2) could not type assert Shock_Dockerimage_attributes, Dockerimage=%s", Dockerimage))
	}

	if dockerimage_id == "" {
		return nil, errors.New(fmt.Sprintf("Id of Dockerimage=%s not found", Dockerimage))
	}
	logger.Debug(1, fmt.Sprintf("using dockerimage id %s instead of name %s ", dockerimage_id, Dockerimage))

	// *** find/inspect image
	image, err := client.InspectImage(dockerimage_id)

	if err != nil {

		logger.Debug(1, fmt.Sprintf("docker image %s is not yet in local repository", Dockerimage))

		image_retrieval := "load" // TODO only load is guaraneed to work
		switch {
		case image_retrieval == "load":
			{ // for images that have been saved
				err = dockerLoadImage(client, dockerimage_download_url)
			}
		case image_retrieval == "import":
			{ // for containers that have been exported
				err = dockerImportImage(client, Dockerimage)
			}
		case image_retrieval == "build":
			{ // to create image from Dockerfile
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

		if node != nil {

		}
		// last test
		if dockerimage_id != "" {
			image, err = client.InspectImage(dockerimage_id)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("(InspectImage) Docker image (%s , %s) was not correctly imported or built, err=%s", Dockerimage, dockerimage_id, err.Error()))
			}
		} else {
			image, err = client.InspectImage(Dockerimage)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("(InspectImage) Docker image (%s) was not correctly imported or built, err=%s", Dockerimage, err.Error()))
			}
		}

	} else {
		logger.Debug(1, fmt.Sprintf("docker image %s is already in local repository", Dockerimage))
	}

	if dockerimage_id != image.ID {
		return nil, errors.New(fmt.Sprintf("error: dockerimage_id != image.ID, %s != %s (%s)", dockerimage_id, image.ID, Dockerimage))
	}

	// tag image to make debugging easier
	if Dockerimage != "" {
		tag_opts := docker.TagImageOptions{Repo: Dockerimage}
		err = client.TagImage(dockerimage_id, tag_opts)
		if err != nil {
			logger.Error(fmt.Sprintf("warning: tagging of image %s with %s failed, err:", dockerimage_id, Dockerimage, err.Error()))
		}
	}

	pipe_output := fmt.Sprintf(" 2> %s 1> %s", conf.STDERR_FILENAME, conf.STDOUT_FILENAME)

	bash_command := ""
	if use_wrapper_script {
		bash_command = fmt.Sprint("/bin/bash", " ", wrapper_script_filename_docker, " ", pipe_output)
	} else {
		bash_command = fmt.Sprint(commandName, " ", strings.Join(args, " "), " ", pipe_output)

	}

	logger.Debug(1, fmt.Sprint("bash_command: ", bash_command))

	// example: "/bin/bash", "-c", "bowtie2 -h 2> awe_stderr.txt 1> awe_stdout.txt"

	container_cmd := []string{"/bin/bash", "-c", bash_command} // TODO remove bash if possible, but is needed for piping

	config := docker.Config{Image: dockerimage_id, WorkingDir: conf.DOCKER_WORK_DIR, AttachStdout: true, AttachStderr: true, AttachStdin: false, Cmd: container_cmd, Volumes: map[string]struct{}{conf.DOCKER_WORK_DIR: {}}}
	opts := docker.CreateContainerOptions{Name: container_name, Config: &config}

	// *** create container (or find container ?)
	logger.Debug(1, fmt.Sprintf("creating docker container from image %s (%s)", Dockerimage, dockerimage_id))
	container_incomplete, err := client.CreateContainer(opts)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error creating container, err=%s", err.Error()))
	}
	container_id := container_incomplete.ID
	logger.Debug(1, fmt.Sprintf("created docker container with ID: %s", container_id))

	// *** inspect the new container
	if false {
		container, err := client.InspectContainer(container_id)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error inspecting container, err=%s", err.Error()))
		}
		//spew.Dump(container)
		//spew.Dump(container.Config)
		fmt.Println("name: ", container.Name)
	}

	// *** start container

	logger.Debug(1, "starting docker container...")

	var bindarray = []string{}

	bindstr_workdir := work.Path() + "/:" + conf.DOCKER_WORK_DIR
	logger.Debug(1, "bindstr_workdir: "+bindstr_workdir)

	// only mount predata if it is actually used
	if len(work.Predata) > 0 {
		predata_directory := path.Join(conf.DATA_PATH, "predata")
		bindstr_predata := predata_directory + "/:" + "/db:ro" // TODO put in config
		logger.Debug(1, "bindstr_predata: "+bindstr_predata)
		bindarray = []string{bindstr_workdir, bindstr_predata}
	} else {
		bindarray = []string{bindstr_workdir}
	}

	err = client.StartContainer(container_id, &docker.HostConfig{Binds: bindarray})
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error starting container, id=%s, err=%s", container_id, err.Error()))
	}

	defer func(container_id string) {
		// *** clean up
		// ** kill container
		err = client.KillContainer(docker.KillContainerOptions{ID: container_id})
		if err != nil {
			logger.Error(fmt.Sprintf("error killing container id=%s, err=%s", container_id, err.Error()))
		}

		// *** remove Container
		opts_remove := docker.RemoveContainerOptions{ID: container_id}
		err = client.RemoveContainer(opts_remove)
		if err != nil {
			logger.Error(fmt.Sprintf("error removing container id=%s, err=%s", container_id, err.Error()))
		} else {
			logger.Debug(1, "removed docker container")
		}

		return
	}(container_id)

	var status int = 0

	// wait for container to finish
	done := make(chan error)
	go func() {
		status, err = client.WaitContainer(container_id)
		done <- err // inform main function
		done <- err // inform memory checker
	}()

	var MaxMem int64 = -1
	var max_memory_total_rss int64 = -1
	var max_memory_total_swap int64 = -1

	memory_stat_filename := path.Join(conf.CGROUP_MEMORY_DOCKER_DIR, container_id, "/memory.stat")

	go func() { // memory checker

		for {

			select {
			case err = <-done:
				if err != nil {
					logger.Error("channerl done returned error: " + err.Error())
				}
				return
			default:
			}

			//container, err := client.InspectContainer(container_id)

			//if err != nil {
			//	logger.Debug(1, fmt.Sprint("error inspecting container (for mem info) ", err.Error()))
			//} else {
			// /sys/fs/cgroup/memory/docker/<id>/memory.stat
			//memory := uint64(container.Config.Memory)
			var memory_total_rss int64 = -1
			var memory_total_swap int64 = -1
			memory_stat_file, err := os.Open(memory_stat_filename)
			if err != nil {
				logger.Error("warning: error opening memory_stat_file file:" + err.Error())
				err = nil
				time.Sleep(conf.MEM_CHECK_INTERVAL)
				continue
			}

			// Closes the file when we leave the scope of the current function,
			// this makes sure we never forget to close the file if the
			// function can exit in multiple places.
			defer memory_stat_file.Close()

			memory_stat_file_scanner := bufio.NewScanner(memory_stat_file)

			memory_total_rss_read := false
			memory_total_swap_read := false
			// scanner.Scan() advances to the next token returning false if an error was encountered
			for memory_stat_file_scanner.Scan() {
				line := memory_stat_file_scanner.Text()
				if strings.HasPrefix(line, "total_rss ") { // TODO what is total_rss_huge
					//logger.Debug(1, fmt.Sprint("inspecting container with memory line=", line))

					memory_total_rss, err = strconv.ParseInt(strings.TrimPrefix(line, "total_rss "), 10, 64)
					if err != nil {
						memory_total_rss = -1
					}
					memory_total_rss_read = true
				} else if strings.HasPrefix(line, "total_swap ") { // TODO what is total_rss_huge
					//logger.Debug(1, fmt.Sprint("inspecting container with memory line=", line))

					memory_total_swap, err = strconv.ParseInt(strings.TrimPrefix(line, "total_swap "), 10, 64)
					if err != nil {
						memory_total_swap = -1
					}
					memory_total_swap_read = true
				} else {
					continue
				}
				if memory_total_rss_read && memory_total_swap_read {
					break
				}

			}

			// When finished scanning if any error other than io.EOF occured
			// it will be returned by scanner.Err().
			if err = memory_stat_file_scanner.Err(); err != nil {
				logger.Error(fmt.Sprintf("warning: could no read memory usage from cgroups=", memory_stat_file_scanner.Err()))
				err = nil
			} else {

				// RSS maxium
				if memory_total_rss >= 0 && memory_total_rss > max_memory_total_rss {
					max_memory_total_rss = memory_total_rss
				}

				// SWAP maximum
				if memory_total_swap >= 0 && memory_total_swap > max_memory_total_swap {
					max_memory_total_swap = memory_total_swap
				}

				// RSS+SWAP maximum
				if memory_total_rss >= 0 && memory_total_swap >= 0 {

					memory_combined := memory_total_rss + memory_total_swap
					if memory_combined > MaxMem {
						MaxMem = memory_combined
					}

				}

				logger.Debug(1, fmt.Sprintf("memory: rss=%d, swap=%d, maximum: rss=%d swap=%d combined=%d", memory_total_rss, memory_total_swap))

			}

			//time.Sleep(5 * time.Second)
			time.Sleep(conf.MEM_CHECK_INTERVAL)
		}
	}()

	select {
	case <-chankill:
		logger.Debug(1, fmt.Sprint("chankill, try to kill conatiner %s... ", container_id))
		if err := client.KillContainer(docker.KillContainerOptions{ID: container_id}); err != nil {
			return nil, errors.New(fmt.Sprintf("error killing container id=%s, err=%s", container_id, err.Error()))
		}

		<-done // allow goroutine to exit

		return nil, errors.New("process killed as requested from chankill")
	case err = <-done:
		if err != nil {
			return nil, errors.New(fmt.Sprintf("wait_cmd=%s, err=%s", commandName, err.Error()))
		}
		logger.Debug(1, fmt.Sprint("docker command returned with status ", status))
	}

	if status != 0 {
		return nil, errors.New(fmt.Sprintf("error WaitContainer returned non-zero status=%d", status))
	}
	logger.Debug(1, fmt.Sprint("pstats.MaxMemUsage: ", pstats.MaxMemUsage))
	pstats.MaxMemUsage = MaxMem
	pstats.MaxMemoryTotalRss = max_memory_total_rss
	pstats.MaxMemoryTotalSwap = max_memory_total_swap
	logger.Debug(1, fmt.Sprint("pstats.MaxMemUsage: ", pstats.MaxMemUsage))

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
	pstats.MaxMemUsage = int64(MaxMem)
	return
}

func runPreWorkExecutionScript(work *core.Workunit) (err error) {
	// conf.PreWorkScript is a string
	// conf.PreWorkScriptArgs is a string array
	args := conf.PRE_WORK_SCRIPT_ARGS
	commandName := conf.PRE_WORK_SCRIPT

	if commandName == "" {
		return nil
	}

	cmd := exec.Command(commandName, args...)

	msg := fmt.Sprintf("worker: start pre-work cmd=%s, args=%v", commandName, args)
	fmt.Println(msg)
	logger.Debug(1, msg)
	logger.Event(event.PRE_WORK_START, "workid="+work.Id,
		"pre-work cmd="+commandName,
		fmt.Sprintf("args=%v", args))

	var stdout, stderr io.ReadCloser

	if conf.PRINT_APP_MSG {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return
		}
		stderr, err = cmd.StderrPipe()
		if err != nil {
			return
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
		msg := fmt.Sprintf(fmt.Sprintf("start pre-work cmd=%s, err=%s", commandName, err.Error()))
		fmt.Println(msg)
		logger.Debug(1, msg)
		return errors.New(msg)
	}

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()

	select {
	case <-chankill:
		if err := cmd.Process.Kill(); err != nil {
			fmt.Println("failed to kill" + err.Error())
		}
		<-done // allow goroutine to exit
		fmt.Println("process killed")
		return errors.New("process killed")
	case err := <-done:
		if err != nil {
			return errors.New(fmt.Sprintf("wait on pre-work cmd=%s, err=%s", commandName, err.Error()))
		}
	}
	logger.Event(event.PRE_WORK_END, "workid="+work.Id)
	return
}

func dockerBuildImage(client *docker.Client, Dockerimage string) (err error) {

	shock_docker_repo := core.ShockClient{conf.SHOCK_DOCKER_IMAGE_REPOSITORY, ""}

	logger.Debug(1, fmt.Sprint("try to build docker image from dockerfile, Dockerimage=", Dockerimage))

	query_response_p, err := shock_docker_repo.Query(url.Values{"dockerfile": {"1"}, "tag": {Dockerimage}})
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
		Name:           "testImage",
		Remote:         download_url,
		SuppressOutput: true,
		OutputStream:   &buf,
	}
	err = client.BuildImage(opts)
	if err != nil {
		return errors.New(fmt.Sprintf("Error importing docker image, err=%s", err.Error()))
	}

	return nil
}

// was getDockerImageUrl(Dockerimage string) (download_url string, err error)
func findDockerImageInShock(Dockerimage string) (node *core.ShockNode, download_url string, err error) {

	shock_docker_repo := core.ShockClient{conf.SHOCK_DOCKER_IMAGE_REPOSITORY, ""}

	logger.Debug(1, fmt.Sprint("try to import docker image, Dockerimage=", Dockerimage))
	//query url = type=dockerimage&name=wgerlach/bowtie2:2.2.0"

	query_response_p, err := shock_docker_repo.Query(url.Values{"type": {"dockerimage"}, "name": {Dockerimage}})
	if err != nil {
		return nil, "", errors.New(fmt.Sprintf("shock node not found for image=%s, err=%s", Dockerimage, err.Error()))
	}
	logger.Debug(1, fmt.Sprintf("query result: %v", query_response_p))

	datalen := len((*query_response_p).Data)

	if datalen == 0 {
		return nil, "", errors.New(fmt.Sprintf("image %s not found in shocks docker repo", Dockerimage))
	} else if datalen > 1 {
		return nil, "", errors.New(fmt.Sprintf("more than one image %s found in shocks docker repo", Dockerimage))
	}

	node = &(*query_response_p).Data[0]
	logger.Debug(1, fmt.Sprintf("found SHOCK node for docker image: %s", node.Id))

	download_url, err = shock_docker_repo.Get_node_download_url(*node)
	if err != nil {
		return nil, "", errors.New(fmt.Sprintf("Could not create download url, err=%s", err.Error()))
	}

	return
}

func dockerLoadImage(client *docker.Client, download_url string) (err error) {

	image_stream, err := fetchShockStream(download_url, "") // token empty here, assume that images are public
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting Shock stream, err=%s", err.Error()))
	}

	gr, err := gzip.NewReader(image_stream) //returns (*Reader, error) // TODO not sure if I have to close gr later ?

	logger.Debug(1, fmt.Sprintf("loading image..."))
	//go io.Copy(image_writer, image_stream)

	var buf bytes.Buffer

	err = client.LoadImage(gr, &buf) // in io.Reader, w io.Writer
	if err != nil {
		return errors.New(fmt.Sprintf("Error loading image, err=%s", err.Error()))
	}

	logger.Debug(1, fmt.Sprintf("load image returned: %v", &buf))
	return
}

func dockerImportImage(client *docker.Client, Dockerimage string) (err error) {

	_, download_url, err := findDockerImageInShock(Dockerimage) // TODO get node

	if err != nil {
		return err
	}

	logger.Debug(1, fmt.Sprintf("docker image url=%s", download_url))

	// TODO import base image if needed

	// *** import image
	Dockerimage_array := strings.Split(Dockerimage, ":")
	Dockerimage_repo, Dockerimage_tag := Dockerimage_array[0], Dockerimage_array[1]

	logger.Debug(1, fmt.Sprintf("importing image..."))
	var buf bytes.Buffer
	opts := docker.ImportImageOptions{
		Source:       download_url,
		Repository:   Dockerimage_repo,
		Tag:          Dockerimage_tag,
		OutputStream: &buf,
	}

	err = client.ImportImage(opts)
	if err != nil {
		return errors.New(fmt.Sprintf("Error importing docker image, err=%s", err.Error()))
	}

	return
}

func SetEnv(work *core.Workunit) (envkeys []string, err error) {
	for key, val := range work.Cmd.Environ.Public {
		if err := os.Setenv(key, val); err == nil {
			envkeys = append(envkeys, key)
		}
	}
	if work.Cmd.HasPrivateEnv {
		envs, err := FetchPrivateEnvByWorkId(work.Id)
		if err != nil {
			return envkeys, err
		}
		for key, val := range envs {
			if err := os.Setenv(key, val); err == nil {
				envkeys = append(envkeys, key)
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

func FetchPrivateEnvByWorkId(workid string) (envs map[string]string, err error) {
	targeturl := fmt.Sprintf("%s/work/%s?privateenv&client=%s", conf.SERVER_URL, workid, core.Self.Id)
	var headers httpclient.Header
	if conf.CLIENT_GROUP_TOKEN != "" {
		headers = httpclient.Header{
			"Authorization": []string{"CG_TOKEN " + conf.CLIENT_GROUP_TOKEN},
		}
	}
	res, err := httpclient.Get(targeturl, headers, nil, nil)
	if err != nil {
		return envs, err
	}
	defer res.Body.Close()
	var jsonstream string
	if res.Header != nil {
		if _, ok := res.Header["Privateenv"]; ok {
			jsonstream = res.Header["Privateenv"][0]
		}
	}
	tmp_map := new(map[string]string)

	if err := json.Unmarshal([]byte(jsonstream), tmp_map); err != nil {
		return nil, err
	}

	envs = make(map[string]string)

	for key, val := range *tmp_map {
		envs[key] = val
	}
	return
}
