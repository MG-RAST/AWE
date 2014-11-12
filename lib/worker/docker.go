package worker

import (
	"encoding/json"
	"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/MG-RAST/AWE/lib/logger/event"
	"bufio"
	"bytes"
	"compress/gzip"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/MG-RAST/go-dockerclient"
	"io"
	"io/ioutil"
	"net/url"
	"os/exec"
	"strconv"
	"strings"
)

func InspectImage(client *docker.Client, dockerimage_id string) (image *docker.Image, err error) {
	logger.Debug(1, fmt.Sprintf("(InspectImage) %s:", dockerimage_id))
	if client == nil {
		// if image does not exists, return status 1 and text on stderr

		cmd := exec.Command(conf.DOCKER_BINARY, "inspect", dockerimage_id)

		stdout, err := cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return nil, err
		}

		if err = cmd.Start(); err != nil {
			return nil, err
		}

		var image_array []docker.Image

		err_json := json.NewDecoder(stdout).Decode(&image_array)

		if err_json != nil {
			logger.Debug(1, fmt.Sprintf("(InspectImage) err_json: %s", err_json.Error()))
			image = nil
		}

		err = cmd.Wait() // wait just in case

		if err != nil {

			stderr_bytearray, err_read := ioutil.ReadAll(stderr)
			if err_read != nil {
				return nil, err_read
			}
			logger.Debug(1, fmt.Sprintf("(InspectImage) STDERR: %s", stderr_bytearray))
			return nil, err
		} else {
			err = err_json // in case that failed...
		}

		if len(image_array) == 1 {
			image = &image_array[0]
		} else {
			err = errors.New("error: inspect returned zero (or more than one) images")
		}

		return image, err
	} else {

		image, err = client.InspectImage(dockerimage_id)

	}
	return image, err
}

func RemoveOldAWEContainers(client *docker.Client, container_name string) (err error) {
	logger.Debug(1, fmt.Sprintf("(RemoveOldAWEContainers) %s", container_name))
	if client == nil {
		cmd := exec.Command(conf.DOCKER_BINARY, "rm", "-f", container_name)
		output, err := cmd.CombinedOutput()
		if err != nil {
			//fmt.Println(fmt.Sprint(err) + ": " + string(output))
			err = errors.New(fmt.Sprintf("(RemoveOldAWEContainers) error removing old container container_name=%s, err=%s, output=%s", container_name, err.Error(), output))
		} else {
			logger.Debug(1, fmt.Sprintf("(RemoveOldAWEContainers) deleteing %s:", container_name, output))
		}

		//if err = cmd.Start(); err != nil {
		//	return err
		//}

		err = cmd.Wait()

		return nil
	}

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

func TagImage(client *docker.Client, dockerimage_id string, tag_opts docker.TagImageOptions) (err error) {
	logger.Debug(1, fmt.Sprintf("(TagImage) %s:", dockerimage_id))
	if client == nil {
		// docker tag skyport/cap:20141020 skyport/cap:newname
		// returns error code 1 on failure

		//TagImageOptions{Repo: Dockerimage_array[0], Tag: Dockerimage_array[1]}

		tag_string := tag_opts.Repo + ":" + tag_opts.Tag

		cmd := exec.Command(conf.DOCKER_BINARY, "tag", dockerimage_id, tag_string)

		stderr, err := cmd.StderrPipe()
		if err != nil {
			return err
		}

		if err = cmd.Start(); err != nil {
			return err
		}

		err = cmd.Wait()

		if err != nil {

			stderr_bytearray, read_err := ioutil.ReadAll(stderr)
			if read_err == nil {
				logger.Debug(1, fmt.Sprintf("(InspectImage) STDERR: %s", stderr_bytearray))
			} else {
				logger.Debug(1, fmt.Sprintf("(InspectImage) could not read from STDERR."))
			}

		}

		return err

	} else {
		err = client.TagImage(dockerimage_id, tag_opts)
	}
	return err
}

func KillContainer(container_id string) (err error) {
	logger.Debug(1, fmt.Sprintf("(KillContainer) %s:", container_id))
	cmd := exec.Command(conf.DOCKER_BINARY, "kill", container_id)

	if err = cmd.Start(); err != nil {
		return err
	}

	err = cmd.Wait()
	return nil
}

func CreateContainer(create_args []string) (container_id string, err error) {

	//docker create [OPTIONS] IMAGE [COMMAND] [ARG...]
	// first line contains ID
	// must have "-t" to attach, this is not documented in docker.

	// prepend "create"
	create_args = append([]string{"create"}, create_args...)

	logger.Debug(1, fmt.Sprintf("(CreateContainer) cmd: %s %s", conf.DOCKER_BINARY, strings.Join(create_args, " ")))

	cmd := exec.Command(conf.DOCKER_BINARY, create_args...)

	logger.Debug(1, fmt.Sprintf("(CreateContainer) cmd struct: %#v", cmd))

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		logger.Debug(1, "(CreateContainer) error getting StdoutPipe")
		return "", err
	}

	//rd := bufio.NewReader(stdout)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		logger.Debug(1, "(CreateContainer) error getting StderrPipe")
		return "", err
	}

	//rd_err := bufio.NewReader(stderr)

	if err = cmd.Start(); err != nil {
		logger.Debug(1, "(CreateContainer) cmd.Start failed")
		return "", err
	}

	stdo, stdo_err := ioutil.ReadAll(stdout)
	stde, stde_err := ioutil.ReadAll(stderr)

	_ = stdo_err // because of stupid go warning
	_ = stde_err

	err = cmd.Wait()

	if err != nil {
		logger.Debug(1, fmt.Sprintf("(CreateContainer) cmd.Wait returned error: %s", err.Error()))
		if stde_err == nil {
			logger.Debug(1, fmt.Sprintf("(CreateContainer) error: %s", stde))
		}
		return "", err
	}

	endofline := bytes.IndexByte(stdo, '\n')

	stdout_line := ""
	if endofline >= 0 {
		stdout_line = string(stdo[0 : endofline-1])
	} else {
		err = errors.New("docker create returned empty string")
	}

	return stdout_line, err

}

// ** not tested **
func RunContainer(run_args []string) (container_id string, err error) {
	logger.Debug(1, fmt.Sprintf("(RunContainer) %s:", container_id))
	//docker run [OPTIONS] IMAGE [COMMAND] [ARG...]
	// first line contains ID
	// must have "-t" to attach, this is not documented in docker.

	// prepend "create"
	run_args = append([]string{"run"}, run_args...)

	logger.Debug(1, fmt.Sprintf("(RunContainer) cmd: %s %s", conf.DOCKER_BINARY, strings.Join(run_args, " ")))

	cmd := exec.Command(conf.DOCKER_BINARY, run_args...)

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return "", err
	}

	rd := bufio.NewReader(stdout)
	//stderr, err := cmd.StderrPipe()
	//if err != nil {
	//	return nil, err
	//}

	if err = cmd.Start(); err != nil {
		return "", err
	}

	err = cmd.Wait()
	var stdout_line string
	for {

		stdout_line, err = rd.ReadString('\n')

		if err == io.EOF {
			return "", err
		}
		break
	}

	if err != nil {
		return "", err
	}

	return stdout_line, err

}

func RemoveContainer(container_id string) (err error) {
	logger.Debug(1, fmt.Sprintf("(RemoveContainer) %s:", container_id))
	cmd := exec.Command(conf.DOCKER_BINARY, "rm", "-f", container_id)
	if err = cmd.Start(); err != nil {
		return err
	}
	err = cmd.Wait()
	return nil
}

func StartContainer(container_id string, args string) (err error) {
	// docker start CONTAINER [CONTAINER...]
	logger.Debug(1, fmt.Sprintf("(StartContainer) %s:", container_id))
	cmd := exec.Command(conf.DOCKER_BINARY, "start", container_id)

	if err = cmd.Start(); err != nil {
		return err
	}

	err = cmd.Wait()

	return nil

}

func WaitContainer(container_id string) (status int, err error) {
	logger.Debug(1, fmt.Sprintf("(WaitContainer) %s:", container_id))
	cmd := exec.Command(conf.DOCKER_BINARY, "wait", container_id)

	if err = cmd.Start(); err != nil {
		return 0, err
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return 0, err
	}

	rd := bufio.NewReader(stdout)

	var stdout_line string

	stdout_line, err = rd.ReadString('\n')

	if err == io.EOF {
		return 0, err
	}

	err = cmd.Wait()

	if err != nil {
		return 0, err
	}

	status, err = strconv.Atoi(stdout_line)
	if err != nil {
		logger.Debug(1, fmt.Sprintf("could not interpret status code: \"%s\"", stdout_line))
		// handle error
		return 0, err
	}

	return status, nil

}

func dockerBuildImage(client *docker.Client, Dockerimage string) (err error) {
	logger.Debug(1, fmt.Sprintf("(dockerBuildImage) %s:", Dockerimage))

	shock_docker_repo := shock.ShockClient{conf.SHOCK_DOCKER_IMAGE_REPOSITORY, ""}

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
func findDockerImageInShock(Dockerimage string, datatoken string) (node *shock.ShockNode, download_url string, err error) {

	logger.Debug(1, fmt.Sprint("datatoken for dockerimage: ", datatoken[0:15]))

	shock_docker_repo := shock.ShockClient{conf.SHOCK_DOCKER_IMAGE_REPOSITORY, datatoken}

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

func dockerLoadImage(client *docker.Client, download_url string, datatoken string) (err error) {

	image_stream, err := shock.FetchShockStream(download_url, datatoken) // token empty here, assume that images are public
	if err != nil {
		return errors.New(fmt.Sprintf("Error getting Shock stream, err=%s", err.Error()))
	}

	gr, err := gzip.NewReader(image_stream) //returns (*Reader, error) // TODO not sure if I have to close gr later ?

	logger.Debug(1, fmt.Sprintf("loading image..."))
	//go io.Copy(image_writer, image_stream)

	var buf bytes.Buffer

	if client != nil {

		err = client.LoadImage(gr, &buf) // in io.Reader, w io.Writer

	} else {

		//pipe stream into docker binary "docker load"
		cmd := exec.Command(conf.DOCKER_BINARY, "load")

		stdin, err := cmd.StdinPipe() // stdin is a io.WriteCloser

		//stderr, err := cmd.StderrPipe()
		//if err != nil {
		//	return err
		//}

		if err = cmd.Start(); err != nil {
			return err
		}

		go io.Copy(stdin, gr)

		err = cmd.Wait()

	}

	if err != nil {
		return errors.New(fmt.Sprintf("Error loading image, err=%s", err.Error()))
	}
	logger.Debug(1, fmt.Sprintf("load image returned: %v", &buf))

	return
}

func dockerImportImage(client *docker.Client, Dockerimage string, datatoken string) (err error) {

	_, download_url, err := findDockerImageInShock(Dockerimage, datatoken) // TODO get node

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
