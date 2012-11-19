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
	"time"
)

type Response struct {
	Code int      `bson:"S" json:"S"`
	Data Workunit `bson:"D" json:"D"`
	Errs []string `bson:"E" json:"E"`
}

func CheckoutWorkunit() (workunit *Workunit, err error) {
	job := NewJob()
	task := NewTask(job, 0)
	workunit = NewWorkunit(task, 0)
	return workunit, nil
}

func CheckoutWorkunitRemote(url string) (workunit *Workunit, err error) {

	response := new(Response)

	res, err := http.Get(url)
	defer res.Body.Close()

	if err != nil {
		return
	}

	jsonstream, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return
	}

	json.Unmarshal(jsonstream, response)

	if response.Code == 200 {
		workunit = &response.Data
		return workunit, nil
	}
	return workunit, errors.New("empty workunit queue")
}

func RunWorkunit(work *Workunit, num int) (err error) {

	fmt.Printf("processor %d started run workunit id=%s\n", num, work.Id)
	defer fmt.Printf("processor %d finished run workunit id=%s\n", num, work.Id)

	//make a working directory for the workunit
	if err := work.Mkdir(); err != nil {
		return err
	}
	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return err
	}

	commandName := work.Cmd.Name

	fmt.Printf("commandName=%s\n", commandName)

	args, err := ParseWorkunitArgs(work)
	if err != nil {
		return
	}

	fmt.Printf("args=%v\n", args)

	cmd := exec.Command(commandName, args...)

	err = cmd.Start()
	if err != nil {
		return
	}

	err = cmd.Wait()
	if err != nil {
		return
	}

	fmt.Printf("work output=%v", work.Outputs)

	for name, io := range work.Outputs {
		fmt.Printf("name=%s, io=%v\n", name, io)
		if err := pushFileByCurl(name, io.Host, io.Node); err != nil {
			return err
		}
	}

	time.Sleep(5 * time.Second)

	return
}

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
				url := io.Url()
				if err := fetchFile(inputname, url); err != nil { //get file from Shock
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
	fmt.Printf("in fetchFile, filename=%s, url=%s\n", filename, url)
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
func pushFileByCurl(filename string, host string, node string) (err error) {
	shockurl := fmt.Sprintf("%s/node/%s", host, node)
	fmt.Printf("in pushFile, filename=%s, url=%s\n", filename, shockurl)

	err = postFileByCurl(filename, shockurl)
	if err != nil {
		return err
	}
	return
}

func postFileByCurl(filename string, target_url string) (err error) {

	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	argv = append(argv, "-F")
	argv = append(argv, fmt.Sprintf("upload=@%s", filename))
	argv = append(argv, target_url)

	cmd := exec.Command("curl", argv...)

	err = cmd.Run()

	if err != nil {
		return
	}
	return
}

/*
func pushFile(filename string, host string, node string) (err error) {
	shockurl := fmt.Sprintf("%s/node/%s", host, node)
	fmt.Printf("in pushFile, filename=%s, url=%s\n", filename, shockurl)

	res, err := postFile(filename, shockurl)
	defer res.Body.Close()
	if err != nil {
		return err
	}

	jsonstream, err := ioutil.ReadAll(res.Body)

	if err != nil {
		return
	}

	fmt.Printf("pushFile, json:=%s\n", jsonstream)

	return
}

func postFile(filename string, targetUrl string) (res *http.Response, err error) {
	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("upload", filename)
	if err != nil {
		fmt.Println("error writing to buffer")
		return
	}
	fh, err := os.Open(filename)
	if err != nil {
		fmt.Println("error opening file")
		return
	}
	io.Copy(fileWriter, fh)
	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	return http.Post(targetUrl, contentType, bodyBuf)
}
*/
