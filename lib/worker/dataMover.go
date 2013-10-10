package worker

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/httpclient"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"io"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

func dataMover(control chan int) {
	fmt.Printf("dataMover lanched, client=%s\n", core.Self.Id)
	defer fmt.Printf("dataMover exiting...\n")
	for {
		raw := <-fromStealer
		parsed := &mediumwork{
			workunit: raw.workunit,
			perfstat: raw.perfstat,
		}
		work := raw.workunit
		workmap[work.Id] = ID_DATAMOVER
		//make a working directory for the workunit
		if err := work.Mkdir(); err != nil {
			logger.Error("err@dataMover_work.Mkdir, workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.State = core.WORK_STAT_FAIL
		}

		//check the availability prerequisite data and download if needed
		if err := movePreData(parsed.workunit); err != nil {
			logger.Error("err@dataMover_work.movePreData, workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.State = core.WORK_STAT_FAIL
		}

		//parse the args, including fetching input data from Shock and composing the local file path
		datamove_start := time.Now().Unix()
		if arglist, err := ParseWorkunitArgs(parsed.workunit); err == nil {
			parsed.workunit.State = core.WORK_STAT_PREPARED
			parsed.workunit.Cmd.ParsedArgs = arglist
		} else {
			logger.Error("err@dataMover_work.ParseWorkunitArgs, workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.State = core.WORK_STAT_FAIL
		}
		datamove_end := time.Now().Unix()
		parsed.perfstat.DataIn = datamove_end - datamove_start

		fromMover <- parsed
	}
	control <- ID_DATAMOVER //we are ending
}

func proxyDataMover(control chan int) {
	fmt.Printf("proxyDataMover lanched, client=%s\n", core.Self.Id)
	defer fmt.Printf("proxyDataMover exiting...\n")

	for {
		raw := <-fromStealer
		parsed := &mediumwork{
			workunit: raw.workunit,
			perfstat: raw.perfstat,
		}
		work := raw.workunit
		workmap[work.Id] = ID_DATAMOVER
		//check the availability prerequisite data and download if needed
		if err := proxyMovePreData(parsed.workunit); err != nil {
			logger.Error("err@dataMover_work.movePreData, workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.State = core.WORK_STAT_FAIL
		}
		fromMover <- parsed
	}
	control <- ID_DATAMOVER
}

//parse workunit, fetch input data, compose command arguments
func ParseWorkunitArgs(work *core.Workunit) (args []string, err error) {
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

				logger.Debug(2, "mover: fetching input from url:"+dataUrl)
				logger.Event(event.FILE_IN, "workid="+work.Id+" url="+dataUrl)

				if err := fetchFile(inputFilePath, dataUrl, work.Info.DataToken); err != nil { //get file from Shock
					return []string{}, err
				}
				logger.Event(event.FILE_READY, "workid="+work.Id+" url="+dataUrl)

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
func fetchFile(filename string, url string, token string) (err error) {
	fmt.Printf("fetching file name=%s, url=%s\n", filename, url)
	localfile, err := os.Create(filename)
	if err != nil {
		return err
	}
	defer localfile.Close()

	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}

	//download file from Shock
	res, err := httpclient.Get(url, httpclient.Header{}, nil, user)
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
func movePreData(workunit *core.Workunit) (err error) {
	for name, io := range workunit.Predata {
		file_path := fmt.Sprintf("%s/%s", conf.DATA_PATH, name)
		if !isFileExisting(file_path) {
			if err = fetchFile(file_path, io.Url, ""); err != nil {
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

func proxyMovePreData(workunit *core.Workunit) (err error) {
	//to be implemented
	return
}
