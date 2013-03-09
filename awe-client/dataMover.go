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
	"strings"
)

func dataMover(control chan int) {
	fmt.Printf("dataMover lanched, client=%s\n", self.Id)
	defer fmt.Printf("dataMover exiting...\n")
	for {
		work := <-chanRaw

		parsed := &parsedWork{
			workunit: work,
			args:     []string{},
			status:   "unknown",
		}

		//make a working directory for the workunit
		if err := work.Mkdir(); err != nil {
			Log.Error("err@dataMover_work.Mkdir, workid=" + work.Id + " error=" + err.Error())
			parsed.status = WORK_STAT_FAIL
		}

		//parse the args, including fetching input data from Shock and composing the local file path
		if arglist, err := ParseWorkunitArgs(work); err == nil {
			parsed.status = WORK_STAT_PREPARED
			parsed.args = arglist
		} else {
			Log.Error("err@dataMover_work.ParseWorkunitArgs, workid=" + work.Id + " error=" + err.Error())
			parsed.status = WORK_STAT_FAIL
		}

		chanParsed <- parsed
	}
	control <- ID_DATAMOVER //we are ending
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

				inputFilePath := fmt.Sprintf("%s/%s", work.Path(), inputname)

				fmt.Printf("worker: fetching input from url %s\n", dataUrl)
				Log.Event(EVENT_FILE_IN, "workid="+work.Id+" url="+dataUrl)
				if err := fetchFile(inputFilePath, dataUrl); err != nil { //get file from Shock
					return []string{}, err
				}
				Log.Event(EVENT_FILE_READY, "workid="+work.Id+" url="+dataUrl)

				parsedArg := fmt.Sprintf("%s%s", segs[0], inputFilePath)
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
