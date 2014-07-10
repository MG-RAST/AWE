package worker

import (
	"bytes"
	"encoding/json"
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
	"path"
	"regexp"
	"strings"
	"time"
)

func replace_filepath_with_full_filepath(inputs *core.IOmap, cmd_script []string) (err error) {

	for _, cmd_line := range cmd_script {
		logger.Debug(1, fmt.Sprintf("C cmd_line : %s", cmd_line))
	}

	// replace @files with abosulte file path
	match_at_file, err := regexp.Compile(`@[^\s]+`)
	if err != nil {

		err = errors.New(fmt.Sprintf("error: compiling regex (match_at_file), error=%s", err.Error()))
		return
	}

	replace_with_full_path := func(variable string) string {
		//cut name out of brackets....
		logger.Debug(1, fmt.Sprintf("variable: %s", variable))
		var inputname = variable[1:] // remove @ in front ; TODO filenames with spaces would need quotes
		logger.Debug(1, fmt.Sprintf("file_name: %s", inputname))

		if inputs.Has(inputname) {
			//inputFilePath := fmt.Sprintf("%s/%s", conf.DOCKER_WORK_DIR, inputname)
			inputFilePath := path.Join(conf.DOCKER_WORK_DIR, inputname)
			logger.Debug(1, fmt.Sprintf("return full file_name: %s", inputname))
			return inputFilePath
		}

		logger.Debug(1, fmt.Sprintf("warning: could not find input file for variable_name: %s", variable))
		return variable
	}

	//replace file names
	for cmd_line_index, _ := range cmd_script {
		cmd_script[cmd_line_index] = match_at_file.ReplaceAllStringFunc(cmd_script[cmd_line_index], replace_with_full_path)
	}

	for _, cmd_line := range cmd_script {
		logger.Debug(1, fmt.Sprintf("D cmd_line : %s", cmd_line))
	}

	return
}

func prepareAppTask(parsed *mediumwork, work *core.Workunit) (err error) {

	app_string := strings.TrimPrefix(parsed.workunit.Cmd.Name, "app:")

	app_array := strings.Split(app_string, ".")

	if len(app_array) != 3 {
		return errors.New("app could not be parsed, workid=" + work.Id + " app=" + app_string)
	}

	if core.MyAppRegistry == nil {
		core.MyAppRegistry, err = core.MakeAppRegistry() // TODO how do I read err ???
		if err != nil {
			return errors.New("error creating app registry, workid=" + work.Id + " error=" + err.Error())
		}
	}

	// get app definition
	app_cmd_mode_object, err := core.MyAppRegistry.Get_cmd_mode_object(app_array[0], app_array[1], app_array[2])

	if err != nil {
		return errors.New("error reading app registry, workid=" + work.Id + " error=" + err.Error())
	}

	parsed.workunit.Cmd.Dockerimage = app_cmd_mode_object.Dockerimage

	var cmd_interpreter = app_cmd_mode_object.Cmd_interpreter

	logger.Debug(1, fmt.Sprintf("success, cmd_interpreter: %s", cmd_interpreter))

	if len(app_cmd_mode_object.Cmd_script) > 0 {
		//logger.Debug(1, fmt.Sprintf("found cmd_script"))
		parsed.workunit.Cmd.ParsedArgs = app_cmd_mode_object.Cmd_script

		//logger.Debug(1, fmt.Sprintf("cmd_script: %s", strings.Join(cmd_script, ", ")))
	}
	//var cmd_script = parsed.workunit.Cmd.ParsedArgs[:]

	var cmd_script = parsed.workunit.Cmd.Cmd_script

	//if err != nil {
	//TODO error
	//	logger.Error(fmt.Sprintf("error: compiling regex, error=%s", err.Error()))
	//	continue
	//}

	// get arguments
	//args_array := strings.Split(args_string_space_removed, ",")
	//args_array := parsed.workunit.Cmd.App_args

	logger.Debug(1, "+++ replace_filepath_with_full_filepath")
	// expand filenames
	err = replace_filepath_with_full_filepath(&parsed.workunit.Inputs, cmd_script)
	if err != nil {
		return errors.New(fmt.Sprintf("error: replace_filepath_with_full_filepath, %s", err.Error()))
	}
	logger.Debug(1, fmt.Sprintf("cmd_script (expanded): %s", strings.Join(cmd_script, ", ")))
	return
}

func dataMover(control chan int) {
	var err error
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
			logger.Error("[dataMover#work.Mkdir], workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.Notes = parsed.workunit.Notes + "###[dataMover#work.Mkdir]" + err.Error()
			parsed.workunit.State = core.WORK_STAT_FAIL
		}

		//check the availability prerequisite data and download if needed
		predatamove_start := time.Now().UnixNano()
		if moved_data, err := movePreData(parsed.workunit); err != nil {
			logger.Error("[dataMover#movePreData], workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.State = core.WORK_STAT_FAIL
			parsed.workunit.Notes = parsed.workunit.Notes + "###[dataMover#movePreData]" + err.Error()
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

		if strings.HasPrefix(parsed.workunit.Cmd.Name, "app:") {
			logger.Debug(1, fmt.Sprintf("app mode, requested: %s", parsed.workunit.Cmd.Name))

			err = prepareAppTask(parsed, raw.workunit)
			if err != nil {
				logger.Error("err@dataMover_work.prepareAppTask, workid=" + work.Id + " error=" + err.Error())
				parsed.workunit.Notes = parsed.workunit.Notes + "###[dataMover#prepareAppTask]" + err.Error()
				parsed.workunit.State = core.WORK_STAT_FAIL
				fromMover <- parsed
				continue
			}

		} else {
			logger.Debug(1, fmt.Sprintf("normal mode"))
			//parse the args, replacing @input_name to local file path (file not downloaded yet)

			if err := ParseWorkunitArgs(parsed.workunit); err != nil {
				logger.Error("err@dataMover_work.ParseWorkunitArgs, workid=" + work.Id + " error=" + err.Error())
				parsed.workunit.Notes = parsed.workunit.Notes + "###[dataMover#ParseWorkunitArgs]" + err.Error()
				parsed.workunit.State = core.WORK_STAT_FAIL
				//hand the parsed workunit to next stage and continue to get new workunit to process
				fromMover <- parsed
				continue
			}
		}
		//download input data
		datamove_start := time.Now().UnixNano()
		if moved_data, err := cache.MoveInputData(parsed.workunit); err != nil {
			logger.Error("err@dataMover_work.moveInputData, workid=" + work.Id + " error=" + err.Error())
			parsed.workunit.Notes = parsed.workunit.Notes + "###[dataMover#MoveInputData]" + err.Error()
			parsed.workunit.State = core.WORK_STAT_FAIL
		} else {
			parsed.perfstat.InFileSize = moved_data
			datamove_end := time.Now().UnixNano()
			parsed.perfstat.DataIn = float64(datamove_end-datamove_start) / 1e9
		}

		//create userattr.json
		user_attr := getUserAttr(parsed.workunit)
		if len(user_attr) > 0 {
			attr_json, _ := json.Marshal(user_attr)
			if err := ioutil.WriteFile(fmt.Sprintf("%s/userattr.json", parsed.workunit.Path()), attr_json, 0644); err != nil {
				logger.Error("err@dataMover_work.getUserAttr, workid=" + work.Id + " error=" + err.Error())
				parsed.workunit.Notes = parsed.workunit.Notes + "###[dataMover#getUserAttr]" + err.Error()
				parsed.workunit.State = core.WORK_STAT_FAIL
			}
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
			parsed.workunit.Notes = parsed.workunit.Notes + "###[dataMover#proxyMovePreData]" + err.Error()
			parsed.workunit.State = core.WORK_STAT_FAIL
		}
		fromMover <- parsed
	}
	control <- ID_DATAMOVER
}

//parse workunit, fetch input data, compose command arguments
func ParseWorkunitArgs(work *core.Workunit) (err error) {

	args := []string{}

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
				return errors.New("invalid format in command args, multiple @ within one arg")
			}
			inputname := segs[1]
			if work.Inputs.Has(inputname) {
				inputFilePath := path.Join(work.Path(), inputname)
				parsedArg := fmt.Sprintf("%s%s", segs[0], inputFilePath)
				args = append(args, parsedArg)
			}
			continue
		}
		//no @ or $, append directly
		args = append(args, arg)
	}

	work.Cmd.ParsedArgs = args
	work.State = core.WORK_STAT_PREPARED
	return nil
}

//fetch file by shock url,  TODO remove
func fetchFile_old(filename string, url string, token string) (size int64, err error) {
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
func fetchFile(filename string, url string, token string) (size int64, err error) {
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
		predata_directory := path.Join(conf.DATA_PATH, "predata")
		err = os.MkdirAll(predata_directory, 755)
		if err != nil {
			return 0, errors.New("error creating predata_directory: " + err.Error())
		}

		file_path := path.Join(predata_directory, name)
		if !isFileExisting(file_path) {

			size, err = fetchFile(file_path, io.Url, workunit.Info.DataToken)
			if err != nil {
				return 0, errors.New("error in fetchFile:" + err.Error())
			}
		}
		//make a link in work dir to predata in conf.DATA_PATH // TODO not needed/usefull for docker containers
		linkname := path.Join(workunit.Path(), name)
		//fmt.Printf(linkname + " -> " + file_path + "\n")
		logger.Debug(1, "symlink:"+linkname+" -> "+file_path)
		err = os.Symlink(file_path, linkname)
		if err != nil {
			return 0, errors.New("error creating predata file symlink: " + err.Error())
		}
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

		inputFilePath := path.Join(work.Path(), inputname)

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

//fetch user attr - merged from job.info and task
func getUserAttr(work *core.Workunit) (userattr map[string]string) {
	userattr = make(map[string]string)
	if len(work.Info.UserAttr) > 0 {
		for k, v := range work.Info.UserAttr {
			userattr[k] = v
		}
	}
	if len(work.UserAttr) > 0 {
		for k, v := range work.UserAttr {
			userattr[k] = v
		}
	}
	return userattr
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
