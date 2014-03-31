package worker

import (
	"bytes"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/cache"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/httpclient"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"io"
	"io/ioutil"
	"os"
	"regexp"
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
		predatamove_start := time.Now().UnixNano()
		if moved_data, err := movePreData(parsed.workunit); err != nil {
			logger.Error("err@dataMover_work.movePreData, workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.State = core.WORK_STAT_FAIL
			//hand the parsed workunit to next stage and continue to get new workunit to process
			fromMover <- parsed
			continue
		} else {
			if moved_data > 0 {
				parsed.perfstat.PreDataSize = moved_data
				predatamove_end := time.Now().UnixNano()
				parsed.perfstat.PreDataIn = float64(predatamove_end-predatamove_start) / 1e9
			}
		}

		//parse the args, replacing @input_name to local file path (file not downloaded yet)
		if arglist, err := ParseWorkunitArgs(parsed.workunit); err != nil {
			logger.Error("err@dataMover_work.ParseWorkunitArgs, workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.State = core.WORK_STAT_FAIL
			//hand the parsed workunit to next stage and continue to get new workunit to process
			fromMover <- parsed
			continue
		} else {
			parsed.workunit.Cmd.ParsedArgs = arglist
			parsed.workunit.State = core.WORK_STAT_PREPARED
		}

		//download input data
		datamove_start := time.Now().UnixNano()
		if moved_data, err := cache.MoveInputData(parsed.workunit); err != nil {
			logger.Error("err@dataMover_work.moveInputData, workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.State = core.WORK_STAT_FAIL
		} else {
			parsed.perfstat.InFileSize = moved_data
			datamove_end := time.Now().UnixNano()
			parsed.perfstat.DataIn = float64(datamove_end-datamove_start) / 1e9
		}

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

	for _, arg := range argList {
		match, err := regexp.Match(`\$\{\w+\}`, []byte(arg))
		if err == nil && match { //replace environment variable with its value
			reg := regexp.MustCompile(`\$\{\w+\}`)
			vabs := reg.FindAll([]byte(arg), -1)
			parsedArg := arg
			for _, vab := range vabs {
				vb := bytes.TrimPrefix(vab, []byte("${"))
				vb = bytes.TrimSuffix(vb, []byte("}"))
				envvalue := os.Getenv(string(vb))
				fmt.Printf("%s=%s\n", vb, envvalue)
				parsedArg = strings.Replace(parsedArg, string(vab), envvalue, 1)
			}
			args = append(args, parsedArg)
			continue
		}
		if strings.Contains(arg, "@") { //parse input/output to accessible local file
			segs := strings.Split(arg, "@")
			if len(segs) > 2 {
				return []string{}, errors.New("invalid format in command args, multiple @ within one arg")
			}
			inputname := segs[1]
			if work.Inputs.Has(inputname) {
				inputFilePath := fmt.Sprintf("%s/%s", work.Path(), inputname)
				parsedArg := fmt.Sprintf("%s%s", segs[0], inputFilePath)
				args = append(args, parsedArg)
			}
			continue
		}
		//no @ or $, append directly
		args = append(args, arg)
	}
	return args, nil
}

//fetch file by shock url
func fetchFile(filename string, url string, token string) (size int64, err error) {
	fmt.Printf("fetching file name=%s, url=%s\n", filename, url)
	localfile, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer localfile.Close()

	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}

	//download file from Shock
	res, err := httpclient.Get(url, httpclient.Header{}, nil, user)
	if err != nil {
		return 0, err
	}

	defer res.Body.Close()

	if res.StatusCode != 200 { //err in fetching data
		resbody, _ := ioutil.ReadAll(res.Body)
		msg := fmt.Sprintf("op=fetchFile, url=%s, res=%s", url, resbody)
		return 0, errors.New(msg)
	}

	size, err = io.Copy(localfile, res.Body)
	if err != nil {
		return 0, err
	}
	return
}

//fetch file by shock url
func fetchFile2(filename string, url string, token string) (size int64, err error) {
	fmt.Printf("fetching file name=%s, url=%s\n", filename, url)
	localfile, err := os.Create(filename)
	if err != nil {
		return 0, err
	}
	defer localfile.Close()

	body, err := fetchShockStream(url, token)

	defer body.Close()

	if err != nil {
		return 0, err
	}

	size, err = io.Copy(localfile, body)
	if err != nil {
		return 0, err
	}
	return
}

func fetchShockStream(url string, token string) (r io.ReadCloser, err error) {

	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}

	//download file from Shock
	res, err := httpclient.Get(url, httpclient.Header{}, nil, user)
	if err != nil {
		return nil, err
	}

	if res.StatusCode != 200 { //err in fetching data
		resbody, _ := ioutil.ReadAll(res.Body)
		return nil, errors.New(fmt.Sprintf("op=fetchFile, url=%s, res=%s", url, resbody))
	}

	return res.Body, err
}

//fetch prerequisite data (e.g. reference dbs)
func movePreData(workunit *core.Workunit) (size int64, err error) {
	for name, io := range workunit.Predata {
		file_path := fmt.Sprintf("%s/%s", conf.DATA_PATH, name)
		if !isFileExisting(file_path) {
			size, err = fetchFile(file_path, io.Url, "")
			if err != nil {
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

//fetch input data
func moveInputData(work *core.Workunit) (size int64, err error) {
	for inputname, io := range work.Inputs {
		var dataUrl string
		if work.Rank == 0 {
			dataUrl = io.DataUrl()
		} else {
			dataUrl = fmt.Sprintf("%s&index=%s&part=%s", io.DataUrl(), work.IndexType(), work.Part())
		}

		inputFilePath := fmt.Sprintf("%s/%s", work.Path(), inputname)

		logger.Debug(2, "mover: fetching input from url:"+dataUrl)
		logger.Event(event.FILE_IN, "workid="+work.Id+" url="+dataUrl)

		if datamoved, err := fetchFile(inputFilePath, dataUrl, work.Info.DataToken); err != nil {
			return size, err
		} else {
			size += datamoved
		}
		logger.Event(event.FILE_READY, "workid="+work.Id+";url="+dataUrl)
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
