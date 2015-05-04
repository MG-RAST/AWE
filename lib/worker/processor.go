package worker

import (
	"bufio"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/go-dockerclient"
	"github.com/MG-RAST/golib/httpclient"
	"io"
	"io/ioutil"
	//"net/url"
	"os"
	"os/exec"
	"path"
	"runtime"
	"strconv"
	"strings"
	"syscall"
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

		var err error
		var envkeys []string
		_ = envkeys
		
		wants_docker := false
		if work.Cmd.Dockerimage != "" || work.App != nil {
			wants_docker = true
		}

		if ! wants_docker {
			envkeys, err = SetEnv(work)
			if err != nil {
				logger.Error("SetEnv(): workid=" + work.Id + ", " + err.Error())
				processed.workunit.Notes = processed.workunit.Notes + "###[processor#SetEnv]" + err.Error()
				processed.workunit.State = core.WORK_STAT_FAIL
				//release the permit lock, for work overlap inhibitted mode only
				if !conf.WORKER_OVERLAP && core.Service != "proxy" {
					<-chanPermit
				}
				continue
			}
		}
		run_start := time.Now().Unix()

		pstat, err := RunWorkunit(work)

		if err != nil {
			logger.Error("RunWorkunit(): returned error , workid=" + work.Id + ", " + err.Error())
			processed.workunit.Notes = processed.workunit.Notes + "###[processor#RunWorkunit]" + err.Error()
			processed.workunit.State = core.WORK_STAT_FAIL
		} else {
			logger.Debug(1, "RunWorkunit() returned without error, workid="+work.Id)
			processed.workunit.State = core.WORK_STAT_COMPUTED
			processed.perfstat.MaxMemUsage = pstat.MaxMemUsage
			processed.perfstat.MaxMemoryTotalRss = pstat.MaxMemoryTotalRss
			processed.perfstat.MaxMemoryTotalSwap = pstat.MaxMemoryTotalSwap

			processed.perfstat.DockerPrep = pstat.DockerPrep
		}
		run_end := time.Now().Unix()
		computetime := run_end - run_start
		processed.perfstat.Runtime = computetime
		processed.workunit.ComputeTime = int(computetime)

		if ! wants_docker {
			if len(envkeys) > 0 {
				UnSetEnv(envkeys)
			}
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

	if work.Cmd.Dockerimage != "" || work.App != nil {
		pstats, err = RunWorkunitDocker(work)
	} else {
		pstats, err = RunWorkunitDirect(work)
	}

	return
}

func getExitStatus(err_inspect error) (status_code int, err error) {
	exiterr, ok := err_inspect.(*exec.ExitError)

	if ok {
		// The program has exited with an exit code != 0

		// This works on both Unix and Windows. Although package
		// syscall is generally platform dependent, WaitStatus is
		// defined for both Unix and Windows and in both cases has
		// an ExitStatus() method with the same signature.
		if status, ok := exiterr.Sys().(syscall.WaitStatus); ok {
			return status.ExitStatus(), nil
		}
	} else {
		//log.Fatalf("cmd.Wait: %v", err)
	}

	return 0, nil
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

	docker_preparation_start := time.Now().Unix()

	commandName := work.Cmd.Name

	use_wrapper_script := false

	wrapper_script_filename := "awe_workunit_wrapper.sh"
	wrapper_script_filename_host := path.Join(work.Path(), wrapper_script_filename)
	wrapper_script_filename_docker := path.Join(conf.DOCKER_WORK_DIR, wrapper_script_filename)

	if len(work.Cmd.Cmd_script) > 0 {
		use_wrapper_script = true

		// create wrapper script

		//conf.DOCKER_WORK_DIR
		var wrapper_content_string = "#!/bin/bash\n" + strings.Join(work.Cmd.Cmd_script, "\n") + "\n"

		logger.Debug(1, fmt.Sprintf("write wrapper script: %s\n%s", wrapper_script_filename_host, strings.Join(work.Cmd.Cmd_script, ", ")))

		var wrapper_content_bytes = []byte(wrapper_content_string)

		err = ioutil.WriteFile(wrapper_script_filename_host, wrapper_content_bytes, 0755) // not executable: 0644
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error writing wrapper script, err=%s", err.Error()))
		}

	}

	//cmd := exec.Command(commandName, args...)

	container_name := "AWE_workunit"

	Dockerimage := work.Cmd.Dockerimage
	if work.App.Name != "" {
		Dockerimage = work.App.AppDef.Dockerimage
	}

	if Dockerimage == "" {
		return nil, errors.New(fmt.Sprintf("Error Dockerimage string empty"))
	}

	logger.Debug(1, fmt.Sprintf("Dockerimage: %s", Dockerimage))

	use_docker_api := true
	if conf.DOCKER_BINARY != "API" {
		use_docker_api = false
	}

	var client *docker.Client = nil

	if use_docker_api {
		logger.Debug(1, fmt.Sprintf("Using docker API..."))
		client, err = docker.NewClient(conf.DOCKER_SOCKET)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error creating docker client", err.Error()))
		}
	} else {
		logger.Debug(1, fmt.Sprintf("Using docker docker binary..."))
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
	node, dockerimage_download_url, err := findDockerImageInShock(Dockerimage, work.Info.DataToken)
	if err != nil {
		return nil, errors.New(fmt.Sprintf("Error getting docker url, err=%s", err.Error()))
	}

	// TODO attr_json, _ := json.Marshal(node.Attributes) might be better
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
	image, err := InspectImage(client, dockerimage_id)

	if err != nil {

		logger.Debug(1, fmt.Sprintf("docker image %s is not yet in local repository", Dockerimage))

		image_retrieval := "load" // TODO only load is guaraneed to work
		switch {
		case image_retrieval == "load":
			{ // for images that have been saved
				err = dockerLoadImage(client, dockerimage_download_url, work.Info.DataToken)
			}
		case image_retrieval == "import":
			{ // for containers that have been exported
				err = dockerImportImage(client, Dockerimage, work.Info.DataToken)
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
			image, err = InspectImage(client, dockerimage_id)
			if err != nil {
				return nil, errors.New(fmt.Sprintf("(InspectImage) Docker image (%s , %s) was not correctly imported or built, err=%s", Dockerimage, dockerimage_id, err.Error()))
			}
		} else {
			image, err = InspectImage(client, Dockerimage)
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
		Dockerimage_array := strings.Split(Dockerimage, ":") // TODO split by colon is risky
		tag_opts := docker.TagImageOptions{Repo: Dockerimage_array[0], Tag: Dockerimage_array[1]}

		err = TagImage(client, dockerimage_id, tag_opts)
		if err != nil {
			logger.Error(fmt.Sprintf("warning: tagging of image %s with %s failed, err:", dockerimage_id, Dockerimage, err.Error()))
		}
	}

	// collect environment
	var docker_environment []string
	docker_environment_string := "" // this is only for the debug output
	for key, val := range work.Cmd.Environ.Public {
		env_pair := key + "=" + val
		docker_environment = append(docker_environment, env_pair)
		docker_environment_string += " --env=" + env_pair
	}
	if work.Cmd.HasPrivateEnv {
		private_envs, err := FetchPrivateEnvByWorkId(work.Id)
		if err != nil {
			return nil, err
		}
		for key, val := range private_envs {
			env_pair := key + "=" + val
			docker_environment = append(docker_environment, env_pair)
			docker_environment_string += " -e " + env_pair

		}
	}

	pipe_output := fmt.Sprintf(" 2> %s 1> %s", conf.STDERR_FILENAME, conf.STDOUT_FILENAME)
	bash_command := ""
	if use_wrapper_script {
		//bash_command = fmt.Sprint("/bin/bash", " ", wrapper_script_filename_docker, " ", pipe_output) // bash for wrapper script
		bash_command = fmt.Sprint(wrapper_script_filename_docker, " ", pipe_output)
	} else {
		bash_command = fmt.Sprint(commandName, " ", strings.Join(args, " "), " ", pipe_output)

	}

	logger.Debug(1, fmt.Sprint("bash_command: ", bash_command))

	// example: "/bin/bash", "-c", "bowtie2 -h 2> awe_stderr.txt 1> awe_stdout.txt"

	container_cmd := []string{"/bin/bash", "-c", bash_command} // TODO remove bash if possible, but is needed for piping

	//var empty_struct struct{}
	bindstr_workdir := work.Path() + "/:" + conf.DOCKER_WORK_DIR
	logger.Debug(1, "bindstr_workdir: "+bindstr_workdir)

	var bindarray = []string{}

	// only mount predata if it is actually used
	//fake_predata := ""
	bindstr_predata := ""
	volume_str := ""
	if len(work.Predata) > 0 {
		predata_directory := path.Join(conf.DATA_PATH, "predata")
		bindstr_predata = predata_directory + "/:" + "/db:ro" // TODO put in config

		bindarray = []string{bindstr_workdir, bindstr_predata} //old version
		volume_str = "--volume=" + bindstr_workdir + " --volume=" + bindstr_predata
	} else {
		bindarray = []string{bindstr_workdir}
		volume_str = "--volume=" + bindstr_workdir
	}

	logger.Debug(1, "volume_str: "+volume_str)

	// version for docker command line
	docker_commandline_create := []string{
		// "-t" would be required if I want to attach to the container later, check again documentation if needed
		"--name=" + container_name,
		"--workdir=" + conf.DOCKER_WORK_DIR,
		volume_str, // for workdir and optionally predata
	}

	if docker_environment_string != "" {
		docker_commandline_create = append(docker_commandline_create, docker_environment_string)
	}

	// version for docker API
	config := docker.Config{Image: dockerimage_id,
		WorkingDir:   conf.DOCKER_WORK_DIR,
		AttachStdout: true,
		AttachStderr: true,
		AttachStdin:  false,
		Cmd:          container_cmd,
		//Volumes:      map[string]struct{}{conf.DOCKER_WORK_DIR: struct{}{}}, // old version
		Volumes: map[string]struct{}{bindstr_workdir: struct{}{}},
		Env:     docker_environment,
	}

	if len(work.Predata) > 0 {
		config.Volumes[bindstr_predata] = struct{}{}
	}

	docker_commandline_create = append(docker_commandline_create, dockerimage_id)   //
	docker_commandline_create = append(docker_commandline_create, container_cmd...) // argument to the "docker create" command

	opts := docker.CreateContainerOptions{Name: container_name, Config: &config}

	// note: docker binary mounts on creation, while docker API mounts on start of container

	container_id := ""

	// *** create container
	logger.Debug(1, fmt.Sprintf("creating docker container from image %s (%s)", Dockerimage, dockerimage_id))

	if client != nil {

		container_obj, err := client.CreateContainer(opts)
		if err == nil {
			container_id = container_obj.ID
		} else {
			return nil, errors.New(fmt.Sprintf("error creating container, err=%s", err.Error()))
		}
	} else {
		container_id, err = CreateContainer(docker_commandline_create)
		if err != nil {
			return nil, errors.New(fmt.Sprintf("error creating container, err=%s", err.Error()))
		}
	}

	if container_id == "" {
		return nil, errors.New(fmt.Sprintf("error creating container, container_id is empty"))
	}

	logger.Debug(1, fmt.Sprintf("created docker container with ID: %s", container_id))

	// *** start container

	fake_docker_cmd := "sudo docker run -t -i --name test " + volume_str + " " + docker_environment_string + " --workdir=" + conf.DOCKER_WORK_DIR + " " + dockerimage_id + " " + strings.Join(container_cmd, " ")
	logger.Debug(1, "fake_docker_cmd ("+Dockerimage+"): "+fake_docker_cmd)
	logger.Debug(1, "starting docker container...")

	docker_preparation_end := time.Now().Unix()
	pstats.DockerPrep = docker_preparation_end - docker_preparation_start
	logger.Debug(1, fmt.Sprintf("DockerPrep time in seconds: %d", pstats.DockerPrep))

	if client != nil {
		err = client.StartContainer(container_id, &docker.HostConfig{Binds: bindarray}) // weired, seems to be needed
		//err = client.StartContainer(container_id, &docker.HostConfig{})

	} else {
		err = StartContainer(container_id, volume_str)
	}
	if err != nil {
		return nil, errors.New(fmt.Sprintf("error starting container, id=%s, err=%s", container_id, err.Error()))
	}

	defer func(container_id string) {
		// *** clean up
		// ** kill container
		var err_kill error
		if client != nil {
			err_kill = client.KillContainer(docker.KillContainerOptions{ID: container_id})
		} else {
			err_kill = KillContainer(container_id)
		}
		if err_kill != nil {
			logger.Error(fmt.Sprintf("error killing container id=%s, err=%s", container_id, err_kill.Error()))
		}

		// *** remove Container

		var err error

		if client != nil {
			opts_remove := docker.RemoveContainerOptions{ID: container_id}
			err = client.RemoveContainer(opts_remove)
		} else {
			err = RemoveContainer(container_id)
		}
		if err != nil {
			logger.Error(fmt.Sprintf("error removing container id=%s, err=%s", container_id, err.Error()))
		} else {
			logger.Debug(1, "(deferred func) removed docker container")
		}

	}(container_id)

	if client != nil {
		cont, err := client.InspectContainer(container_id)
		if err != nil {
			logger.Error(fmt.Sprintf("error inspecting container=%s, err=%s", container_id, err.Error()))
		}

		inspect_filename := path.Join(work.Path(), "container_inspect.json")

		b_inspect, _ := json.MarshalIndent(cont, "", "    ")

		err = ioutil.WriteFile(inspect_filename, b_inspect, 0666)
		if err != nil {
			logger.Error(fmt.Sprintf("error writing inspect file for container=%s, err=%s", container_id, err.Error()))
		} else {
			logger.Debug(1, fmt.Sprintf("wrote %s for container %s", inspect_filename, container_id))
		}
	}

	var status int = 0

	// wait for container to finish
	done := make(chan error)
	go func() {
		var errwait error
		if client != nil {
			status, errwait = client.WaitContainer(container_id)
		} else {
			status, errwait = WaitContainer(container_id)
		}
		done <- errwait // inform main function
		if conf.MEM_CHECK_INTERVAL != 0 {
			done <- errwait // inform memory checker
		}
	}()

	var MaxMem int64 = -1
	var max_memory_total_rss int64 = -1
	var max_memory_total_swap int64 = -1

	// documentation: https://docs.docker.com/articles/runmetrics/
	// e.g. /sys/fs/cgroup/memory/docker/<id>/memory.stat
	memory_stat_filename := path.Join(conf.CGROUP_MEMORY_DOCKER_DIR, container_id, "/memory.stat")

	if conf.MEM_CHECK_INTERVAL != 0 {
		go func() { // memory checker

			for {

				select {
				case err_mem := <-done:
					if err_mem != nil {
						logger.Error("channel done returned error: " + err_mem.Error())
					}
					return
				default:
				}

				var memory_total_rss int64 = -1
				var memory_total_swap int64 = -1
				memory_stat_file, err_mem := os.Open(memory_stat_filename)
				if err_mem != nil {

					logger.Error("warning: error opening memory_stat_file file:" + err_mem.Error())
					time.Sleep(conf.MEM_CHECK_INTERVAL)
					continue
				}

				// Closes the file when we leave the scope of the current function,
				// this makes sure we never forget to close the file if the
				// function can exit in multiple places.

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
					if memory_total_rss_read && memory_total_swap_read { // we found all information we need, leave the loop
						break
					}

				}

				// When finished scanning if any error other than io.EOF occured
				// it will be returned by scanner.Err().
				if err := memory_stat_file_scanner.Err(); err != nil {
					logger.Error(fmt.Sprintf("warning: could no read memory usage from cgroups=%s", memory_stat_file_scanner.Err()))
					//err = nil
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

					logger.Debug(1, fmt.Sprintf("memory: rss=%d, swap=%d, max_rss=%d max_swap=%d max_combined=%d",
						memory_total_rss, memory_total_swap, max_memory_total_rss, max_memory_total_swap, MaxMem))

				}
				memory_stat_file.Close() // defer does not work in for loop !
				//time.Sleep(5 * time.Second)
				time.Sleep(conf.MEM_CHECK_INTERVAL)

			}
		}()
	} else {
		logger.Debug(1, "memory checking disabled")
	}

	select {
	case <-chankill:
		logger.Debug(1, fmt.Sprint("chankill, try to kill conatiner %s... ", container_id))

		if client != nil {
			err = client.KillContainer(docker.KillContainerOptions{ID: container_id})
		} else {
			err = KillContainer(container_id)
		}

		if err != nil {
			return nil, errors.New(fmt.Sprintf("error killing container id=%s, err=%s", container_id, err.Error()))
		}

		<-done // allow goroutine to exit

		return nil, errors.New("process killed as requested from chankill")
	case err = <-done:
		logger.Debug(1, fmt.Sprint("(1)docker wait returned with status ", status))
		if err != nil {
			return nil, errors.New(fmt.Sprintf("dockerWait=%s, err=%s", commandName, err.Error()))
		}
	}
	logger.Debug(1, fmt.Sprint("(2)docker wait returned with status ", status))
	if status != 0 {
		logger.Debug(1, fmt.Sprint("WaitContainer returned non-zero status ", status))
		return nil, errors.New(fmt.Sprintf("error WaitContainer returned non-zero status=%d", status))
	}
	logger.Debug(1, fmt.Sprint("pstats.MaxMemUsage: ", pstats.MaxMemUsage))
	pstats.MaxMemUsage = MaxMem
	pstats.MaxMemoryTotalRss = max_memory_total_rss
	pstats.MaxMemoryTotalSwap = max_memory_total_swap
	logger.Debug(1, fmt.Sprint("pstats.MaxMemUsage: ", pstats.MaxMemUsage))

	return
}

func RunWorkunitDirect(work *core.Workunit) (pstats *core.WorkPerf, err error) {
	pstats = new(core.WorkPerf)

	args := work.Cmd.ParsedArgs

	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return nil, err
	}

	commandName := work.Cmd.Name

	if commandName == "" {
		return nil, errors.New(fmt.Sprintf("error: command name is empty"))
	}

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

	mem_check_interval_here := conf.MEM_CHECK_INTERVAL
	if mem_check_interval_here == 0 {
		mem_check_interval_here = 10 * time.Second
	}

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
				time.Sleep(mem_check_interval_here)
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
