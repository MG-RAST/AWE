package main

import (
	"errors"
	"fmt"
	. "github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/logger"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strings"
)

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
				Log.Event(EVENT_FILE_IN, "workid="+work.Id+" url="+dataUrl)

				if err := fetchFile(inputname, dataUrl); err != nil { //get file from Shock
					return []string{}, err
				}

				Log.Event(EVENT_FILE_READY, "workid="+work.Id+" url="+dataUrl)

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
