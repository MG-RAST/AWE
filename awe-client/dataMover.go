package main

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/logger"
	"io"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

func dataMover(control chan int) {
	fmt.Printf("dataMover lanched, client=%s\n", self.Id)
	defer fmt.Printf("dataMover exiting...\n")
	for {
		raw := <-chanRaw
		parsed := &parsedWork{
			workunit: raw.workunit,
			perfstat: raw.perfstat,
			args:     []string{},
			status:   "unknown",
		}

		//make a working directory for the workunit
		work := raw.workunit
		if err := work.Mkdir(); err != nil {
			Log.Error("err@dataMover_work.Mkdir, workid=" + work.Id + " error=" + err.Error())
			parsed.status = WORK_STAT_FAIL
		}

		//check the availability prerequisite data and download if needed
		if err := movePreData(parsed.workunit); err != nil {
			Log.Error("err@dataMover_work.movePreData, workid=" + work.Id + " error=" + err.Error())
			parsed.status = WORK_STAT_FAIL
		}

		//parse the args, including fetching input data from Shock and composing the local file path
		datamove_start := time.Now().Unix()
		if arglist, err := ParseWorkunitArgs(parsed.workunit); err == nil {
			parsed.status = WORK_STAT_PREPARED
			parsed.args = arglist
		} else {
			Log.Error("err@dataMover_work.ParseWorkunitArgs, workid=" + work.Id + " error=" + err.Error())
			parsed.status = WORK_STAT_FAIL
		}
		datamove_end := time.Now().Unix()
		parsed.perfstat.DataIn = datamove_end - datamove_start

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
					dataUrl = io.DataUrl()
				} else {
					dataUrl = fmt.Sprintf("%s&index=%s&part=%s", io.DataUrl(), work.IndexType(), work.Part())
				}

				inputFilePath := fmt.Sprintf("%s/%s", work.Path(), inputname)

				Log.Debug(2, "mover: fetching input from url:"+dataUrl)
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
	fmt.Printf("fetching file name=%s, url=%s\n", filename, url)

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

//fetch prerequisite data (e.g. reference dbs)
func movePreData(workunit *Workunit) (err error) {
	for name, io := range workunit.Predata {
		file_path := fmt.Sprintf("%s/%s", conf.DATA_PATH, name)
		if !isFileExisting(file_path) {
			if err = fetchFile(file_path, io.Url); err != nil {
				return
			}
		}
		//make a link in work dir to predata in conf.DATA_PATH
		linkname := fmt.Sprintf("%s/%s", workunit.Path(), name)
		fmt.Printf(linkname + " -> " + file_path + "\n")
		os.Symlink(file_path, linkname)
	}
	return
}

func isFileExisting(path string) bool {
	if _, err := os.Stat(path); err == nil {
		return true
	}
	return false
}
