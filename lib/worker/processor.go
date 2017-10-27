package worker

import (
	"bufio"
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/golib/httpclient"
	//"github.com/davecgh/go-spew/spew"
	"github.com/davecgh/go-spew/spew"
	"github.com/fsouza/go-dockerclient"
	"io"
	"io/ioutil"
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

type WaitContainerResult struct {
	Error  error
	Status int
}

func processor_run(control chan int) (err error) {

	workunit := <-fromMover
	//work := parsedwork.Workunit

	//processed := &Mediumwork{
	//	Workunit: work,
	//	Perfstat: parsedwork.Perfstat,
	//}

	//if the work is not succesfully parsed in last stage, pass it into the next one immediately

	work_id := workunit.Workunit_Unique_Identifier

	var work_str string
	work_str, err = work_id.String()
	if err != nil {
		err = fmt.Errorf("() workid.String() returned: %s", err.Error())
		return
	}

	work_state, ok, err := workmap.Get(work_id)
	if err != nil {
		err = fmt.Errorf("(processor_run) workmap.Get returned: %s", err.Error())
		return
	}
	if !ok {
		logger.Error("(processor) workunit.id %s not found", work_str)
		return
	}
	if workunit.State == core.WORK_STAT_ERROR || work_state == ID_DISCARDED {

		if work_state == ID_DISCARDED {
			workunit.SetState(core.WORK_STAT_DISCARDED, "workmap indicates that workunit has been discarded")
		} else {
			workunit.SetState(core.WORK_STAT_ERROR, "workmap indicates WORK_STAT_ERROR")
		}
		fromProcessor <- workunit
		//release the permit lock, for work overlap inhibitted mode only
		//if !conf.WORKER_OVERLAP && core.Service != "proxy" {
		//	<-chanPermit
		//}
		return
	}

	workmap.Set(work_id, ID_WORKER, "processor")

	var envkeys []string
	_ = envkeys

	wants_docker := false
	if workunit.Cmd.Dockerimage != "" || workunit.Cmd.DockerPull != "" {
		wants_docker = true
	}

	if !wants_docker {
		envkeys, err = SetEnv(workunit)
		if err != nil {
			logger.Error("(processor) SetEnv(): workid=" + work_str + ", " + err.Error())
			workunit.Notes = append(workunit.Notes, "[processor#SetEnv]"+err.Error())
			workunit.SetState(core.WORK_STAT_ERROR, "see notes")
			//release the permit lock, for work overlap inhibitted mode only
			//if !conf.WORKER_OVERLAP && core.Service != "proxy" {
			//	<-chanPermit
			//}
			return
		}
	}
	run_start := time.Now().Unix()

	var pstat *core.WorkPerf
	pstat, err = RunWorkunit(workunit)
	exit_status := workunit.ExitStatus
	logger.Debug(1, "(processor) ExitStatus of process: %d", exit_status)
	if err != nil {
		logger.Error("(processor) RunWorkunit returned error , workid=%s, %s", work_str, err.Error())
		workunit.Notes = append(workunit.Notes, "[processor#RunWorkunit]"+err.Error())

		if exit_status == 42 {
			workunit.SetState(core.WORK_STAT_FAILED_PERMANENT, "exit_status == 42") // process told us that is an error where resubmission does not make sense.
		} else {
			workunit.SetState(core.WORK_STAT_ERROR, "RunWorkunit failed")
		}
		err = nil
	} else {
		logger.Debug(1, "(processor) RunWorkunit() returned without error, workid=%s", work_str)
		workunit.SetState(core.WORK_STAT_COMPUTED, "")
		workunit.WorkPerf.MaxMemUsage = pstat.MaxMemUsage
		workunit.WorkPerf.MaxMemoryTotalRss = pstat.MaxMemoryTotalRss
		workunit.WorkPerf.MaxMemoryTotalSwap = pstat.MaxMemoryTotalSwap

		workunit.WorkPerf.DockerPrep = pstat.DockerPrep
	}
	run_end := time.Now().Unix()
	computetime := run_end - run_start
	workunit.WorkPerf.Runtime = computetime
	workunit.ComputeTime = int(computetime)

	if !wants_docker {
		if len(envkeys) > 0 {
			UnSetEnv(envkeys)
		}
	}
	logger.Debug(1, "(processor) sending work to datamover")
	fromProcessor <- workunit

	//release the permit lock, for work overlap inhibitted mode only
	//if !conf.WORKER_OVERLAP && core.Service != "proxy" {
	//	<-chanPermit
	//}
	return
}

func processor(control chan int) {
	fmt.Printf("(processor) launched, client=%s\n", core.Self.Id)
	defer fmt.Printf("(processor)  exiting...\n")
	count := 0
	for {
		count += 1
		logger.Debug(1, "(processor) for loop count=%d", count)

		if Client_mode == "offline" && count > 1 {
			logger.Debug(1, "(processor) leaving")
			time.Sleep(time.Second)
			control <- ID_WORKER
			return
		}
		err := processor_run(control)
		if err != nil {
			logger.Error("(processor) processor_run returns: %s", err.Error())
		}
	}
	control <- ID_WORKER //we are ending
}

func RunWorkunit(workunit *core.Workunit) (pstats *core.WorkPerf, err error) {

	if workunit.Cmd.Dockerimage != "" || workunit.Cmd.DockerPull != "" {
		pstats, err = RunWorkunitDocker(workunit)
		if err != nil {
			err = fmt.Errorf("(RunWorkunit) RunWorkunitDocker returned: %s", err.Error())
			return
		}
	} else {
		pstats, err = RunWorkunitDirect(workunit)
		if err != nil {
			err = fmt.Errorf("(RunWorkunit) RunWorkunitDirect returned: %s", err.Error())
			return
		}
	}

	if workunit.CWL_workunit != nil {
		work_path, xerr := workunit.Path()
		if xerr != nil {
			err = xerr
			return
		}

		stdout_file := work_path + "/" + conf.STDOUT_FILENAME

		// wait for awe_stdout to be available
		//for true {

		//	f, file_err := os.OpenFile(stdout_file, syscall.O_RDWR, 0660) // For write access.
		//	if file_err != nil {
		//		f.Close()
		//		break
		//	}
		//	time.Sleep(time.Second)

		//	fmt.Printf("waiting for %s ...\n", stdout_file)

		//}

		file, e := ioutil.ReadFile(stdout_file)
		if e != nil {
			err = fmt.Errorf("(RunWorkunit) Could read output of cwl-runner: %s", e.Error())
			return
		}

		var tool_results interface{}

		err = json.Unmarshal(file, &tool_results)
		if err != nil {
			err = fmt.Errorf("(RunWorkunit) Could not parse json output of cwl-runner: %s", err.Error())
			fmt.Println(string(file[:]))
			return
		}

		result_doc, xerr := cwl.NewJob_document(tool_results)
		if xerr != nil {
			err = xerr
			return
		}
		fmt.Println("CWL-runner receipt:")
		spew.Dump(result_doc)
		workunit.CWL_workunit.Outputs = result_doc

	}

	return
}

func RunWorkunitDocker(workunit *core.Workunit) (pstats *core.WorkPerf, err error) {
	pstats = new(core.WorkPerf)
	pstats.MaxMemUsage = -1
	pstats.MaxMemoryTotalRss = -1
	pstats.MaxMemoryTotalSwap = -1
	args := workunit.Cmd.ParsedArgs

	//change cwd to the workunit's working directory
	if err := workunit.CDworkpath(); err != nil {
		err = fmt.Errorf("(RunWorkunitDocker) CDworkpath returned: %s", err.Error())
		return nil, err
	}

	docker_preparation_start := time.Now().Unix()

	commandName := workunit.Cmd.Name

	use_wrapper_script := false

	work_path, err := workunit.Path()
	if err != nil {
		err = fmt.Errorf("(RunWorkunitDocker) workunit.Path() returned: %s", err.Error())
		return
	}

	wrapper_script_filename := "awe_workunit_wrapper.sh"
	wrapper_script_filename_host := path.Join(work_path, wrapper_script_filename)
	wrapper_script_filename_docker := path.Join(conf.DOCKER_WORK_DIR, wrapper_script_filename)

	if len(workunit.Cmd.Cmd_script) > 0 {
		use_wrapper_script = true

		// create wrapper script

		//conf.DOCKER_WORK_DIR
		var wrapper_content_string = "#!/bin/bash\n" + strings.Join(workunit.Cmd.Cmd_script, "\n") + "\n"

		logger.Debug(1, "write wrapper script: %s\n%s", wrapper_script_filename_host, strings.Join(workunit.Cmd.Cmd_script, ", "))

		var wrapper_content_bytes = []byte(wrapper_content_string)

		err = ioutil.WriteFile(wrapper_script_filename_host, wrapper_content_bytes, 0755) // not executable: 0644
		if err != nil {
			err = fmt.Errorf("(RunWorkunitDocker) error writing wrapper script, err=%s", err.Error())
			return
		}

	}

	//cmd := exec.Command(commandName, args...)

	container_name := "AWE_workunit_" + DockerizeName(conf.CLIENT_NAME)

	if workunit.Cmd.Dockerimage != "" {
		logger.Debug(1, "using Dockerimage: %s", workunit.Cmd.Dockerimage)
	} else if workunit.Cmd.DockerPull != "" {
		logger.Debug(1, "using DockerPull: %s", workunit.Cmd.DockerPull)
	} else {
		err = fmt.Errorf("(RunWorkunitDocker) Error Dockerimage/DockerPull string empty")
		return
	}

	dockerimage_repo := ""
	dockerimage_tag := ""

	if workunit.Cmd.Dockerimage != "" {

		dockerimage_repo, dockerimage_tag, err = SplitDockerimageName(workunit.Cmd.Dockerimage)
		if err != nil {
			err = fmt.Errorf("(RunWorkunitDocker) A SplitDockerimageName returned: %s", err.Error())
			return
		}
	}
	if workunit.Cmd.DockerPull != "" {

		dockerimage_repo, dockerimage_tag, err = SplitDockerimageName(workunit.Cmd.DockerPull)
		if err != nil {
			err = fmt.Errorf("(RunWorkunitDocker) B SplitDockerimageName returned: %s", err.Error())
			return
		}
	}

	Dockerimage_normalized := dockerimage_repo + ":" + dockerimage_tag
	logger.Debug(3, "Dockerimage_normalized: %s", Dockerimage_normalized)

	use_docker_api := true
	if conf.DOCKER_BINARY != "API" {
		use_docker_api = false
	}

	var client *docker.Client = nil

	if use_docker_api {
		logger.Debug(1, "Using docker API...")
		client, err = docker.NewClient(conf.DOCKER_SOCKET)
		if err != nil {
			err = fmt.Errorf("(RunWorkunitDocker) error creating docker client: %s", err.Error())
			return
		}
	} else {
		logger.Debug(1, "Using docker docker binary...")
	}

	//imgs, _ := client.ListImages(false)
	//for _, img := range imgs {
	//	spew.Dump(img)
	//}

	// delete any old AWE_container
	err = RemoveOldAWEContainers(client, container_name)
	if err != nil {
		err = fmt.Errorf("(RunWorkunitDocker) RemoveOldAWEContainers returned: %s", err.Error())
		return nil, err
	}

	//var node *core.ShockNode = nil
	// find image in repo (e.g. extract docker image id)

	dockerimage_id := ""

	if workunit.Cmd.Dockerimage != "" {

		node, dockerimage_download_url, xerr := findDockerImageInShock(Dockerimage_normalized, workunit.Info.DataToken)
		if xerr != nil {
			err = fmt.Errorf("Error getting docker url, err=%s", xerr.Error())
			return
		}

		// TODO attr_json, _ := json.Marshal(node.Attributes) might be better
		node_attr_map, ok := node.Attributes.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(1) could not type assert Shock_Dockerimage_attributes, Dockerimage=%s", Dockerimage_normalized)
			return
		}

		dockerimage_id, ok = node_attr_map["id"].(string)
		if !ok {
			err = fmt.Errorf("(2) could not type assert Shock_Dockerimage_attributes, Dockerimage=%s", Dockerimage_normalized)
			return
		}
		logger.Debug(3, "dockerimage_id: %s", dockerimage_id)

		if dockerimage_id == "" {
			err = fmt.Errorf("Id of Dockerimage=%s not found", Dockerimage_normalized)
			return
		}
		logger.Debug(1, "using dockerimage id %s instead of name %s ", dockerimage_id, Dockerimage_normalized)

		// *** find/inspect image
		var image *docker.Image
		image, err = InspectImage(client, dockerimage_id)

		if err != nil {

			logger.Debug(1, "docker image %s is not yet in local repository", Dockerimage_normalized)
			err = nil

			image_retrieval := "load" // TODO only load is guaraneed to workunit
			//image_retrieval := "pull"
			switch {
			case image_retrieval == "load":
				{ // for images that have been saved
					logger.Debug(1, "Loading image %s", dockerimage_download_url)
					err = dockerLoadImage(client, dockerimage_download_url, workunit.Info.DataToken)
					if err != nil {
						err = fmt.Errorf("Docker image was not correctly loaded, err=%s", err.Error())
						return
					}
				}
			case image_retrieval == "import":
				{ // for containers that have been exported
					logger.Debug(1, "Importing image %s", Dockerimage_normalized)
					xerr := dockerImportImage(client, Dockerimage_normalized, workunit.Info.DataToken)
					if xerr != nil {
						err = fmt.Errorf("Docker image was not correctly imported, err=%s", xerr.Error())
						return
					}
				}
			case image_retrieval == "build":
				{ // to create image from Dockerfile
					logger.Debug(1, "Building image %s", Dockerimage_normalized)
					xerr := dockerBuildImage(client, Dockerimage_normalized)
					if xerr != nil {
						err = fmt.Errorf("Docker image was not correctly built, err=%s", xerr.Error())
						return
					}
				}
			case image_retrieval == "pull":
				{ // pull from docker hub
					logger.Debug(1, "Pulling image %s", Dockerimage_normalized)
					var buf bytes.Buffer
					pio := docker.PullImageOptions{Repository: Dockerimage_normalized, OutputStream: &buf}
					xerr := client.PullImage(pio, docker.AuthConfiguration{})
					logger.Debug(3, "docker pull response: ", buf.String())
					if xerr != nil {
						err = fmt.Errorf("Docker image was not correctly pulled, err=%s", xerr.Error())
						return
					}
				}
			}

			// example urls
			// find image : http://shock.metagenomics.anl.gov/node/?query&docker=1&tag=wgerlach/bowtie2:2.2.0
			// view node: http://shock.metagenomics.anl.gov/node/ed0a6b20-c535-40d7-92e8-754bb8b6b48f
			// download http://shock.metagenomics.anl.gov/node/ed0a6b20-c535-40d7-92e8-754bb8b6b48f?download

			// last test
			if dockerimage_id != "" {
				image, err = InspectImage(client, dockerimage_id)
				if err != nil {
					err = fmt.Errorf("(InspectImage) Docker image (%s , %s) was not correctly imported or built, err=%s", Dockerimage_normalized, dockerimage_id, err.Error())
					return
				}
			} else {
				image, err = InspectImage(client, Dockerimage_normalized)
				if xerr != nil {
					err = fmt.Errorf("(InspectImage) Docker image (%s) was not correctly imported or built, err=%s", Dockerimage_normalized, err.Error())
					return
				}
			}

		} else {
			logger.Debug(1, "docker image %s is already in local repository", Dockerimage_normalized)
		}

		if dockerimage_id != image.ID {
			err = fmt.Errorf("error: dockerimage_id != image.ID, %s != %s (%s)", dockerimage_id, image.ID, Dockerimage_normalized)
			return
		}

		// tag image to make debugging easier
		tag_opts := docker.TagImageOptions{Repo: dockerimage_repo, Tag: dockerimage_tag}

		err = TagImage(client, dockerimage_id, tag_opts)
		if err != nil {
			logger.Error("warning: tagging of image %s with %s failed, err:", dockerimage_id, Dockerimage_normalized, err.Error())
		}

	} else if workunit.Cmd.DockerPull != "" {

		if false {
			logger.Debug(1, "Pulling image %s from Docker Hub", Dockerimage_normalized)
			var buf bytes.Buffer
			pio := docker.PullImageOptions{Repository: Dockerimage_normalized, OutputStream: &buf}
			err = client.PullImage(pio, docker.AuthConfiguration{})
			logger.Debug(3, "docker pull response: ", buf.String())
			if err != nil {
				err = fmt.Errorf("Docker image was not correctly pulled, err=%s", err.Error())
				return
			}
		}
		dockerimage_id = workunit.Cmd.DockerPull
	}

	if dockerimage_id == "" {
		err = fmt.Errorf("dockerimage_id empty")
		return
	}

	// collect environment
	var docker_environment []string
	docker_environment_string := "" // this is only for the debug output
	for key, val := range workunit.Cmd.Environ.Public {
		env_pair := key + "=" + val
		docker_environment = append(docker_environment, env_pair)
		docker_environment_string += " --env=" + env_pair
	}
	if workunit.Cmd.HasPrivateEnv {
		logger.Debug(3, "HasPrivateEnv true")
		private_envs, err := FetchPrivateEnvByWorkId(workunit.Id)
		if err != nil {
			return nil, err
		}
		for key, val := range private_envs {
			env_pair := key + "=" + val
			docker_environment = append(docker_environment, env_pair)
			docker_environment_string += " -e " + env_pair

		}
	} else {
		logger.Debug(3, "HasPrivateEnv false")
	}

	stdout_file := path.Join(conf.DOCKER_WORK_DIR, conf.STDOUT_FILENAME)
	stderr_file := path.Join(conf.DOCKER_WORK_DIR, conf.STDERR_FILENAME)

	pipe_output := fmt.Sprintf(" 2> %s 1> %s", stderr_file, stdout_file)

	bash_command := ""
	if use_wrapper_script {
		//bash_command = fmt.Sprint("/bin/bash", " ", wrapper_script_filename_docker, " ", pipe_output) // bash for wrapper script
		bash_command = fmt.Sprint(wrapper_script_filename_docker, " ", pipe_output)
	} else {

		bash_command = fmt.Sprintf("%s %s %s", commandName, strings.Join(args, " "), pipe_output)
		//bash_command = fmt.Sprintf("uname -a %s", pipe_output)

		var wrapper_content_string = "#!/bin/bash\n" + bash_command + "\n"

		logger.Debug(1, "write wrapper script: %s\n%s", wrapper_script_filename_host, bash_command)

		var wrapper_content_bytes = []byte(wrapper_content_string)

		err = ioutil.WriteFile(wrapper_script_filename_host, wrapper_content_bytes, 0755) // not executable: 0644
		if err != nil {
			err = fmt.Errorf("error writing wrapper script, err=%s", err.Error())
			return
		}
		//time.Sleep(time.Second * 3000)
		bash_command = wrapper_script_filename_docker
	}
	//bash_command = fmt.Sprintf("sleep 3 ; uname -a ; ./awe_qc.pl -format my_file_format.075 -out_prefix my_job_id.075 -assembled 1 -filter_options my_filter_options -proc 8 -input Ebin3.fa; uname -a 2> %s 1> %s", conf.STDERR_FILENAME, conf.STDOUT_FILENAME)
	logger.Debug(1, fmt.Sprint("bash_command: ", bash_command))

	// example: "/bin/bash", "-c", "bowtie2 -h 2> awe_stderr.txt 1> awe_stdout.txt"

	//container_cmd := []string{"/bin/bash", "-c", bash_command} // TODO remove bash if possible, but is needed for piping
	container_cmd := []string{bash_command}

	//var empty_struct struct{}
	bindstr_workdir := work_path + "/:" + conf.DOCKER_WORK_DIR
	logger.Debug(1, "bindstr_workdir: "+bindstr_workdir)

	var bindarray = []string{}

	// only mount predata if it is actually used
	//fake_predata := ""
	bindstr_predata := ""
	volume_str := ""
	if len(workunit.Predata) > 0 {
		predata_directory := path.Join(conf.DATA_PATH, "predata")
		bindstr_predata = predata_directory + "/:" + conf.DOCKER_WORKUNIT_PREDATA_DIR + ":ro"

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
		Volumes:      map[string]struct{}{conf.DOCKER_WORK_DIR: struct{}{}}, // old version
		//Volumes: map[string]struct{}{bindstr_workdir: struct{}{}},

		Env: docker_environment,
	}

	if len(workunit.Predata) > 0 {
		config.Volumes[bindstr_predata] = struct{}{}
	}

	docker_commandline_create = append(docker_commandline_create, dockerimage_id)   //
	docker_commandline_create = append(docker_commandline_create, container_cmd...) // argument to the "docker create" command

	opts := docker.CreateContainerOptions{Name: container_name, Config: &config, HostConfig: &docker.HostConfig{Binds: bindarray}}

	// note: docker binary mounts on creation, while docker API mounts on start of container

	container_id := ""

	// *** create container
	logger.Debug(1, "creating docker container %s from image %s (%s)", container_name, Dockerimage_normalized, dockerimage_id)

	if client != nil {

		container_obj, xerr := client.CreateContainer(opts)
		if xerr == nil {
			container_id = container_obj.ID
		} else {
			err = fmt.Errorf("error creating container, err=%s", xerr.Error())
			return
		}
	} else {
		container_id, err = CreateContainer(docker_commandline_create)
		if err != nil {
			err = fmt.Errorf("error creating container, err=%s", err.Error())
			return
		}
	}
	logger.Debug(3, "Container created.")

	if container_id == "" {
		err = fmt.Errorf("error creating container, container_id is empty")
		return
	}

	logger.Debug(1, "created docker container with ID: %s", container_id)

	// *** start container

	fake_docker_cmd := "sudo docker run -t -i --name test " + volume_str + " " + docker_environment_string + " --workdir=" + conf.DOCKER_WORK_DIR + " " + dockerimage_id + " " + strings.Join(container_cmd, " ")
	logger.Debug(1, "fake_docker_cmd ("+Dockerimage_normalized+"): "+fake_docker_cmd)
	logger.Debug(1, "starting docker container...")

	docker_preparation_end := time.Now().Unix()
	pstats.DockerPrep = docker_preparation_end - docker_preparation_start
	logger.Debug(1, "DockerPrep time in seconds: %d", pstats.DockerPrep)

	if client != nil {
		//err = client.StartContainer(container_id, &docker.HostConfig{Binds: bindarray}) // weired, seems to be needed
		err = client.StartContainer(container_id, nil)
		//err = client.StartContainer(container_id, &docker.HostConfig{})

	} else {
		err = StartContainer(container_id, volume_str)
	}

	if err != nil {
		err = fmt.Errorf("error starting container, id=%s, err=%s", container_id, err.Error())
		return
	}
	logger.Debug(3, "Container started.")

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
			logger.Debug(3, "(deferred func) (clean-up after running container) could not kill container id=%s, err=%s", container_id, err_kill.Error())
		}

	}(container_id)

	if client != nil {
		cont, err := client.InspectContainer(container_id)
		if err != nil {
			logger.Error("error inspecting container=%s, err=%s", container_id, err.Error())
		}

		logger.Debug(3, "Container status: %s", cont.State.Status)

		inspect_filename := path.Join(work_path, "container_inspect.json")

		b_inspect, _ := json.MarshalIndent(cont, "", "    ")

		err = ioutil.WriteFile(inspect_filename, b_inspect, 0666)
		if err != nil {
			fmt.Errorf("error writing inspect file for container=%s, err=%s", container_id, err.Error())
		} else {
			logger.Debug(1, "wrote %s for container %s", inspect_filename, container_id)
		}
	}

	// wait for container to finish
	done := make(chan WaitContainerResult)
	go func() {

		//time.Sleep(300 * time.Second)

		var errwait error
		var status int = -1
		if client != nil {
			status, errwait = client.WaitContainer(container_id)
		} else {
			status, errwait = WaitContainer(container_id)
		}

		cresult := WaitContainerResult{errwait, status}

		done <- cresult // inform main function
		if conf.MEM_CHECK_INTERVAL != 0 {
			done <- cresult // inform memory checker
		}
	}()

	var MaxMem int64 = -1
	var max_memory_total_rss int64 = -1
	var max_memory_total_swap int64 = -1

	memory_stat_filename := ""
	if conf.MEM_CHECK_INTERVAL != 0 {

		// documentation: https://docs.docker.com/articles/runmetrics/
		// e.g. ubuntu: /sys/fs/cgroup/memory/docker/[ID]/memory.stat
		//      coreos: /sys/fs/cgroup/memory/system.slice/docker-[ID].scope/memory.stat

		memory_stat_filename = strings.Replace(conf.CGROUP_MEMORY_DOCKER_DIR, "[ID]", container_id, -1)

		if _, err := os.Stat(memory_stat_filename); os.IsNotExist(err) {
			logger.Error("warning: memory measurement requested, but no memory.stat found: %s" + memory_stat_filename)
			memory_stat_filename = ""
		}

	}

	if conf.MEM_CHECK_INTERVAL != 0 && memory_stat_filename != "" {
		go func() { // memory checker

			for {

				select {
				case cresult := <-done:
					workunit.ExitStatus = cresult.Status
					if cresult.Error != nil {
						logger.Error("channel done returned error: " + cresult.Error.Error())
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
					logger.Error("warning: could no read memory usage from cgroups=%s", memory_stat_file_scanner.Err())
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

	cresult := WaitContainerResult{nil, -1}

	select {
	case <-chankill:
		logger.Debug(1, "chankill, try to kill container %s... ", container_id)

		if client != nil {
			err = client.KillContainer(docker.KillContainerOptions{ID: container_id})
		} else {
			err = KillContainer(container_id)
		}

		if err != nil {
			return nil, errors.New(fmt.Sprintf("(chankill) error killing container id=%s, err=%s", container_id, err.Error()))
		}

		<-done // allow goroutine to exit

		return nil, errors.New("process killed as requested from chankill")
	case cresult = <-done:
		workunit.ExitStatus = cresult.Status
		logger.Debug(3, "(1)docker wait returned with status %d", cresult.Status)
		if cresult.Error != nil {
			return nil, fmt.Errorf("dockerWait=%s, status=%d, err=%s", commandName, cresult.Status, cresult.Error.Error())
		}
		if cresult.Status != 0 {
			logger.Debug(3, "WaitContainer returned non-zero status=%d", cresult.Status)
			return nil, fmt.Errorf("error WaitContainer returned non-zero status=%d", cresult.Status)
		}
	}

	logger.Debug(1, fmt.Sprint("pstats.MaxMemUsage: ", pstats.MaxMemUsage))
	pstats.MaxMemUsage = MaxMem
	pstats.MaxMemoryTotalRss = max_memory_total_rss
	pstats.MaxMemoryTotalSwap = max_memory_total_swap
	logger.Debug(1, fmt.Sprint("pstats.MaxMemUsage: ", pstats.MaxMemUsage))

	return
}

func RunWorkunitDirect(workunit *core.Workunit) (pstats *core.WorkPerf, err error) {

	var args []string

	if len(workunit.Cmd.ArgsArray) > 0 {
		args = workunit.Cmd.ArgsArray
	} else {
		args = workunit.Cmd.ParsedArgs
	}

	//change cwd to the workunit's working directory
	if err := workunit.CDworkpath(); err != nil {
		return nil, err
	}

	commandName := workunit.Cmd.Name

	if commandName == "" {
		err = fmt.Errorf("error: command name is empty")
		return
	}

	cmd := exec.Command(commandName, args...)

	msg := fmt.Sprintf("(RunWorkunitDirect) worker: start cmd=%s, args=%v", commandName, args)
	//fmt.Println(msg)
	logger.Debug(1, msg)
	logger.Event(event.WORK_START, "workid="+workunit.Id,
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

	work_path, xerr := workunit.Path()
	if xerr != nil {
		err = xerr
		return
	}
	logger.Debug(3, "(RunWorkunitDirect) Using workpath: %s", work_path)

	stdoutFilePath := fmt.Sprintf("%s/%s", work_path, conf.STDOUT_FILENAME)
	stderrFilePath := fmt.Sprintf("%s/%s", work_path, conf.STDERR_FILENAME)
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

	if err = cmd.Start(); err != nil {
		err = fmt.Errorf("start_cmd=%s, err=%s", commandName, err.Error())
		return
	}

	var MaxMem uint64 = 0
	MaxMemChan := make(chan uint64)
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
		_MaxMem := mstats.Alloc
		MaxMemChan <- mstats.Alloc
		time.Sleep(2 * time.Second)
		for {
			select {
			default:
				mstats := new(runtime.MemStats)
				runtime.ReadMemStats(mstats)
				if mstats.Alloc > _MaxMem {
					MaxMemChan <- mstats.Alloc
				}
				time.Sleep(mem_check_interval_here)
			case <-memcheck_done:
				return
			}
		}
	}()

	do_loop := true
	for do_loop {
		logger.Debug(3, "(RunWorkunitDirect) for-loop")
		select {
		case MaxMem_value := <-MaxMemChan:
			logger.Debug(3, "(RunWorkunitDirect) received MaxMem_value %d", MaxMem_value)
			MaxMem = MaxMem_value
		case <-chankill:
			if err := cmd.Process.Kill(); err != nil {
				fmt.Println("(RunWorkunitDirect) failed to kill" + err.Error())
			}
			<-done // allow goroutine to exit
			logger.Info("(RunWorkunitDirect) worker process was killed")
			return nil, errors.New("(RunWorkunitDirect) process killed")
		case err = <-done:
			logger.Debug(3, "(RunWorkunitDirect) received done")
			if err != nil {
				workunit.ExitStatus = 1 // just in case we cannot figure out the error code

				if exiterr, ok := err.(*exec.ExitError); ok {
					// The program has exited with an exit code != 0

					// This works on both Unix and Windows. Although package
					// syscall is generally platform dependent, WaitStatus is
					// defined for both Unix and Windows and in both cases has
					// an ExitStatus() method with the same signature.
					status, ok := exiterr.Sys().(syscall.WaitStatus)
					if ok {
						workunit.ExitStatus = status.ExitStatus()
					}
				}

				err = fmt.Errorf("(RunWorkunitDirect) wait_cmd=%s, err=%s", commandName, err.Error())
				return
			} else {
				// I guess all is ok
				workunit.ExitStatus = 0

			}
			logger.Debug(3, "(RunWorkunitDirect) workunit.ExitStatus: %d", workunit.ExitStatus)
			do_loop = false
		}
	}

	logger.Event(event.WORK_END, "workid="+workunit.Id)

	pstats = new(core.WorkPerf)
	pstats.MaxMemUsage = int64(MaxMem)
	return
}

func runPreWorkExecutionScript(workunit *core.Workunit) (err error) {
	// conf.PreWorkScript is a string
	// conf.PreWorkScriptArgs is a string array
	args := conf.PRE_WORK_SCRIPT_ARGS
	commandName := conf.PRE_WORK_SCRIPT

	if commandName == "" {
		return nil
	}

	cmd := exec.Command(commandName, args...)

	msg := fmt.Sprintf("worker: start pre-work cmd=%s, args=%v", commandName, args)

	logger.Debug(1, msg)
	logger.Event(event.PRE_WORK_START, "workid="+workunit.Id,
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

	work_path, xerr := workunit.Path()
	if xerr != nil {
		err = xerr
		return
	}
	stdoutFilePath := fmt.Sprintf("%s/%s", work_path, conf.STDOUT_FILENAME)
	stderrFilePath := fmt.Sprintf("%s/%s", work_path, conf.STDERR_FILENAME)
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
	logger.Event(event.PRE_WORK_END, "workid="+workunit.Id)
	return
}

func SetEnv(workunit *core.Workunit) (envkeys []string, err error) {
	for key, val := range workunit.Cmd.Environ.Public {
		if err := os.Setenv(key, val); err == nil {
			envkeys = append(envkeys, key)
		}
	}
	if workunit.Cmd.HasPrivateEnv {
		envs, err := FetchPrivateEnvByWorkId(workunit.Id)
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
		err = fmt.Errorf("(FetchPrivateEnvByWorkId) json.Unmarshal returned: %s", err.Error())
		return nil, err
	}

	envs = make(map[string]string)

	for key, val := range *tmp_map {
		envs[key] = val
	}
	return
}
