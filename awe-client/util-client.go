package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/logger"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

type WorkResponse struct {
	Code int      `bson:"S" json:"S"`
	Data Workunit `bson:"D" json:"D"`
	Errs []string `bson:"E" json:"E"`
}

type ClientResponse struct {
	Code int      `bson:"S" json:"S"`
	Data Client   `bson:"D" json:"D"`
	Errs []string `bson:"E" json:"E"`
}

func CheckoutWorkunitRemote(serverhost string) (workunit *Workunit, err error) {

	response := new(WorkResponse)

	res, err := http.Get(fmt.Sprintf("%s/work?client=%s", serverhost, self.Id))

	Log.Debug(3, fmt.Sprintf("client %s sent a checkout request to %s", self.Id, serverhost))

	if err != nil {
		return
	}

	defer res.Body.Close()

	jsonstream, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	if err = json.Unmarshal(jsonstream, response); err != nil {
		if len(response.Errs) > 0 {
			return nil, errors.New(strings.Join(response.Errs, ","))
		}
		return
	}

	if response.Code == 200 {
		workunit = &response.Data
		return workunit, nil
	}
	return
}

func NotifyWorkunitProcessed(serverhost string, workid string, status string) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	target_url := fmt.Sprintf("%s/work/%s?status=%s&client=%s", serverhost, workid, status, self.Id)
	argv = append(argv, target_url)

	cmd := exec.Command("curl", argv...)

	err = cmd.Run()

	if err != nil {
		return
	}
	return
}

func RunWorkunit(work *Workunit) (err error) {

	fmt.Printf("++++++worker: started processing workunit id=%s++++++\n", work.Id)
	defer fmt.Printf("-------worker: finished processing workunit id=%s------\n\n", work.Id)

	//make a working directory for the workunit
	if err := work.Mkdir(); err != nil {
		return err
	}

	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return err
	}

	args, err := ParseWorkunitArgs(work)
	if err != nil {
		return
	}

	commandName := work.Cmd.Name
	cmd := exec.Command(commandName, args...)

	fmt.Printf("worker: start running cmd=%s, args=%v\n", commandName, args)
	Log.Event(EVENT_WORK_START, "workid="+work.Id,
		"cmd="+commandName,
		fmt.Sprintf("args=%v", args))

	var stdout, stderr io.ReadCloser
	if conf.PRINT_APP_MSG {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err = cmd.StderrPipe()
		if err != nil {
			return err
		}
	}

	if err := cmd.Start(); err != nil {
		return errors.New(fmt.Sprintf("start_cmd=%s, err=%s", commandName, err.Error()))
	}

	if conf.PRINT_APP_MSG {
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
	}

	if err := cmd.Wait(); err != nil {
		return errors.New(fmt.Sprintf("wait_cmd=%s, err=%s", commandName, err.Error()))
	}

	Log.Event(EVENT_WORK_END, "workid="+work.Id)

	for name, io := range work.Outputs {

		if _, err := os.Stat(name); err != nil {
			return errors.New(fmt.Sprintf("output %s not generated for workunit %s", name, work.Id))
		}

		fmt.Printf("worker: push output to shock, filename=%s\n", name)
		Log.Event(EVENT_FILE_OUT,
			"workid="+work.Id,
			"filename="+name,
			fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))
		if err := pushFileByCurl(name, io.Host, io.Node, work.Rank); err != nil {
			fmt.Errorf("push file error\n")
			Log.Error("op=pushfile,err=" + err.Error())
			return err
		}
		Log.Event(EVENT_FILE_DONE,
			"workid="+work.Id,
			"filename="+name,
			fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))
	}
	return
}

//parse workunit, fetch input data, compose command arguments
func ParseWorkunitArgs(work *Workunit) (args []string, err error) {
	argstr := work.Cmd.Args
	if argstr == "" {
		return
	}

	argList := strings.Fields(argstr)
	inputsMap := work.Inputs

	for _, arg := range argList {
		if strings.Contains(arg, "@") { //parse input/output to accessible local file
			segs := strings.Split(arg, "@")
			if len(segs) > 2 {
				return []string{}, errors.New("invalid format in command args, multiple @ within one arg")
			}
			inputname := segs[1]

			if inputsMap.Has(inputname) {
				io := inputsMap[inputname]

				var dataUrl string
				if work.Rank == 0 {
					dataUrl = io.Url()
				} else {
					dataUrl = fmt.Sprintf("%s&index=record&part=%s", io.Url(), work.Part())
				}

				fmt.Printf("worker: fetching input from url %s\n", dataUrl)
				Log.Event(EVENT_FILE_IN, "url="+dataUrl)

				if err := fetchFile(inputname, dataUrl); err != nil { //get file from Shock
					return []string{}, err
				}

				Log.Event(EVENT_FILE_READY, "url="+dataUrl)

				filePath := fmt.Sprintf("%s/%s", work.Path(), inputname)

				parsedArg := fmt.Sprintf("%s%s", segs[0], filePath)
				args = append(args, parsedArg)
			}
		} else { //no @, has nothing to do with input/output, append directly
			args = append(args, arg)
		}
	}

	return args, nil
}

//fetch file by shock url
func fetchFile(filename string, url string) (err error) {

	localfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer localfile.Close()

	//download file from Shock
	res, err := http.Get(url)
	if err != nil {
		return err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 { //err in fetching data
		resbody, _ := ioutil.ReadAll(res.Body)
		msg := fmt.Sprintf("op=fetchFile, url=%s, res=%s", url, resbody)
		return errors.New(msg)
	}

	_, err = io.Copy(localfile, res.Body)
	if err != nil {
		return err
	}

	return
}

//push file to shock
func pushFileByCurl(filename string, host string, node string, rank int) (err error) {

	shockurl := fmt.Sprintf("%s/node/%s", host, node)

	if err := putFileByCurl(filename, shockurl, rank); err != nil {
		return err
	}
	//if err := makeIndexByCurl(shockurl, "record"); err != nil {
	//	return err
	//}
	return
}

func putFileByCurl(filename string, target_url string, rank int) (err error) {

	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	argv = append(argv, "-F")

	if rank == 0 {
		argv = append(argv, fmt.Sprintf("upload=@%s", filename))
	} else {
		argv = append(argv, fmt.Sprintf("%d=@%s", rank, filename))
	}

	argv = append(argv, target_url)

	fmt.Printf("curl argv=%#v\n", argv)

	cmd := exec.Command("curl", argv...)

	err = cmd.Run()

	if err != nil {
		return
	}
	return
}

func makeIndexByCurl(targetUrl string, indexType string) (err error) {

	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")

	indexUrl := fmt.Sprintf("%s?index=%s", targetUrl, indexType)
	argv = append(argv, indexUrl)

	cmd := exec.Command("curl", argv...)

	err = cmd.Run()

	if err != nil {
		return
	}
	return
}

func Register(host string) (client *Client, err error) {
	var res *http.Response
	serverUrl := fmt.Sprintf("%s/client", host)
	res, err = http.Post(serverUrl, "", strings.NewReader(""))
	if err != nil {
		return
	}

	jsonstream, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	//fmt.Printf("json=%s\n", jsonstream)

	response := new(ClientResponse)

	if err = json.Unmarshal(jsonstream, response); err != nil {
		if len(response.Errs) > 0 {
			//or if err.Error() == "json: cannot unmarshal null into Go value of type core.Client" 
			return nil, errors.New(strings.Join(response.Errs, ","))
		}
		return
	}

	client = &response.Data
	return
}

func RegisterWithProfile(host string, profile_path string) (client *Client, err error) {

	if _, err := os.Stat(profile_path); err != nil {
		return nil, errors.New("profile file not found: " + profile_path)
	}

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)

	fileWriter, err := bodyWriter.CreateFormFile("profile", profile_path)
	if err != nil {
		return nil, err
	}

	fh, err := os.Open(profile_path)
	if err != nil {
		return nil, err
	}

	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return nil, err
	}

	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()

	targetUrl := host + "/client"

	resp, err := http.Post(targetUrl, contentType, bodyBuf)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	jsonstream, err := ioutil.ReadAll(resp.Body)

	response := new(ClientResponse)

	if err = json.Unmarshal(jsonstream, response); err != nil {
		if len(response.Errs) > 0 {
			//or if err.Error() == "json: cannot unmarshal null into Go value of type core.Client" 
			return nil, errors.New(strings.Join(response.Errs, ","))
		}
		return
	}
	client = &response.Data
	return
}

func ReRegisterWithSelf(host string) (client *Client, err error) {
	self_jsonstream, err := json.Marshal(self)
	tmpProfile := conf.DATA_PATH + "/temp.profile"
	ioutil.WriteFile(tmpProfile, []byte(self_jsonstream), 0644)
	client, err = RegisterWithProfile(host, tmpProfile)
	if err != nil {
		Log.Error("Error: fail to re-register, clientid=" + self.Id)
		fmt.Printf("failed to re-register\n")
	} else {
		Log.Event(EVENT_CLIENT_AUTO_REREGI, "clientid="+self.Id)
		fmt.Printf("re-register successfully\n")
	}
	return
}

//client sends heartbeat to server to maintain active status
func SendHeartBeat(host string, clientid string) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	target_url := fmt.Sprintf("%s/client/%s", host, clientid)
	argv = append(argv, target_url)
	cmd := exec.Command("curl", argv...)
	err = cmd.Run()
	if err != nil {
		return
	}
	Log.Debug(3, fmt.Sprintf("client %s sent a heartbeat to %s", clientid, host))
	return
}
