package main

import (
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/MG-RAST/AWE/core"
	"io"
	"io/ioutil"
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

	res, err := http.Get(fmt.Sprintf("%s/work", serverhost))

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

func NotifyWorkunitDone(serverhost string, workid string) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	target_url := fmt.Sprintf("%s/work/%s?status=done", serverhost, workid)
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

	err = cmd.Run()
	if err != nil {
		return
	}

	for name, io := range work.Outputs {

		if _, err := os.Stat(name); err != nil {
			fmt.Printf("worker: error:output %s not generated for workunit %s\n", name, work.Id)
			return errors.New(fmt.Sprintf("error:output %s not generated for workunit %s", name, work.Id))
		}

		fmt.Printf("worker: push output to shock, filename=%s\n", name)
		if err := pushFileByCurl(name, io.Host, io.Node, work.Rank); err != nil {
			fmt.Printf("push file error")
			return err
		}
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

				if err := fetchFile(inputname, dataUrl); err != nil { //get file from Shock
					return []string{}, err
				}

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
	defer res.Body.Close()

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

	fmt.Printf("curl argv=%v\n", argv)

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
	//create a shock node for output
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
