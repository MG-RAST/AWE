package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/cache"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	//"github.com/MG-RAST/AWE/lib/core/cwl"
	//cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/MG-RAST/golib/httpclient"
	//"github.com/davecgh/go-spew/spew"
	"io"
	"io/ioutil"
	"os"
	"path"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"
)

// this functions replaces filename if they match regular expression and they match the filename reported in IOmap
func replace_filepath_with_full_filepath(inputs []*core.IO, workpath string, cmd_script []string) (err error) {

	// workpath may be conf.DOCKER_WORK_DIR

	for _, cmd_line := range cmd_script {
		logger.Debug(1, "C cmd_line : %s", cmd_line)
	}

	// replace @files with abosulte file path
	// was: match_at_file, err := regexp.Compile(`@[^\s]+`)
	match_at_file, err := regexp.Compile(`@[\w-\.]+`) // [0-9A-Za-z_] and "-" //TODO support for space using quotes
	if err != nil {
		err = fmt.Errorf("error: compiling regex (match_at_file), error=%s", err.Error())
		return
	}

	replace_with_full_path := func(variable string) string {
		//cut name out of brackets....
		logger.Debug(1, "variable: %s", variable)
		var inputname = variable[1:] // remove @ in front ; TODO filenames with spaces would need quotes
		logger.Debug(1, "file_name: %s", inputname)

		for _, io := range inputs {
			if io.FileName == inputname {
				//inputFilePath := fmt.Sprintf("%s/%s", conf.DOCKER_WORK_DIR, inputname)
				inputFilePath := path.Join(workpath, inputname)
				logger.Debug(1, "return full file_name: %s", inputname)
				return inputFilePath
			}
		}

		logger.Debug(1, "warning: could not find input file for variable_name: %s", variable)
		return variable
	}

	//replace file names
	for cmd_line_index, _ := range cmd_script {
		cmd_script[cmd_line_index] = match_at_file.ReplaceAllStringFunc(cmd_script[cmd_line_index], replace_with_full_path)
	}

	for _, cmd_line := range cmd_script {
		logger.Debug(1, "D cmd_line : %s", cmd_line)
	}

	return
}

func prepareAppTask(parsed *Mediumwork, work *core.Workunit) (err error) {

	// get app definition
	//app_cmd_mode_object, err := core.MyAppRegistry.Get_cmd_mode_object(app_array[0], app_array[1], app_array[2])

	//if work.App.AppDef == nil {
	//	return errors.New("error reading app defintion AppDef from workunit, workid=" + work.Id)
	//}
	//app_cmd_mode_object := work.App.AppDef
	//if err != nil {
	//	return errors.New("error reading app registry, workid=" + work.Id + " error=" + err.Error())
	//}

	// parsed.Workunit.Cmd.Dockerimage = app_cmd_mode_object.Dockerimage

	//var cmd_interpreter = app_cmd_mode_object.Cmd_interpreter

	//logger.Debug(1, fmt.Sprintf("cmd_interpreter: %s", cmd_interpreter))
	var cmd_script = parsed.Workunit.Cmd.Cmd_script

	//if len(app_cmd_mode_object.Cmd_script) > 0 {
	//	 parsed.Workunit.Cmd.Cmd_script = app_cmd_mode_object.Cmd_script
	//	logger.Debug(2, fmt.Sprintf("cmd_script: %s", strings.Join(cmd_script, ", ")))
	//}

	// expand variables on client-side

	numcpu := runtime.NumCPU() //TODO document reserved variable NumCPU // TODO read NumCPU from client profile info
	numcpu_str := strconv.Itoa(numcpu)
	logger.Debug(2, "NumCPU: %s", numcpu_str)

	//app_variables := make(core.AppVariables) // this does not reuse exiting app variables, this more like a constant
	//app_variables := work.AppVariables // workunit does not need it yet

	//app_variables["NumCPU"] = core.AppVariable{Key: "NumCPU", Var_type: core.Ait_string, Value: numcpu_str}

	//for _, io_obj := range work.Inputs {
	// name := io_obj.Name
	// 	if io_obj.Host != "" {
	// 		app_variables[name+".host"] = core.AppVariable{Key: name + ".host", Var_type: core.Ait_string, Value: io_obj.Host}
	// 	}
	// 	if io_obj.Node != "" {
	// 		app_variables[name+".node"] = core.AppVariable{Key: name + ".node", Var_type: core.Ait_string, Value: io_obj.Node}
	// 	}
	// 	if io_obj.Url != "" {
	// 		app_variables[name+".url"] = core.AppVariable{Key: name + ".url", Var_type: core.Ait_string, Value: io_obj.Url}
	// 	}
	//
	// }

	// sigil := "--" // TODO use config from app definition
	// 	arguments_string := ""
	// 	for _, app_arg := range work.App.App_args {
	// 		name := app_arg.Key
	// 		value, ok := app_variables[name]
	// 		if ok {
	//
	// 			if app_arg.Resource == "string" {
	// 				arguments_string += " " + sigil + name + "=" + value.Value
	// 			} else if app_arg.Resource == "bool" {
	// 				if value.Value == "true" {
	// 					arguments_string += " " + sigil + name
	// 				}
	// 			}
	// 		}
	// 	}
	//
	// 	app_variables["arguments"] = core.AppVariable{
	// 		Key:      "arguments",
	// 		Value:    arguments_string,
	// 		Var_type: core.Ait_string,
	// 	}
	//
	// 	app_variables["datatoken"] = core.AppVariable{Key: "datatoken", Var_type: core.Ait_string, Value: work.Info.DataToken}
	//
	// 	err = core.Expand_app_variables(app_variables, cmd_script)
	// 	if err != nil {
	// 		return errors.New(fmt.Sprintf("error: core.Expand_app_variables, %s", err.Error()))
	// 	}

	//for i, _ := range cmd_script {
	//	cmd_script[i] = strings.Replace(cmd_script[i], "${NumCPU}", numcpu_str, -1)
	//}

	// get arguments
	//args_array := strings.Split(args_string_space_removed, ",")
	//args_array :=  parsed.Workunit.Cmd.App_args

	logger.Debug(1, "+++ replace_filepath_with_full_filepath")
	logger.Debug(1, "conf.DOCKER_WORK_DIR: %s", conf.DOCKER_WORK_DIR)
	// expand filenames
	err = replace_filepath_with_full_filepath(parsed.Workunit.Inputs, conf.DOCKER_WORK_DIR, cmd_script)
	if err != nil {
		err = fmt.Errorf("error: replace_filepath_with_full_filepath, %s", err.Error())
		return
	}
	logger.Debug(1, "cmd_script (expanded): %s", strings.Join(cmd_script, ", "))
	return
}

func dataMover(control chan int) {
	//var err error
	fmt.Printf("dataMover launched, client=%s\n", core.Self.Id)
	logger.Debug(1, "dataMover launched, client=%s\n", core.Self.Id)

	defer fmt.Printf("dataMover exiting...\n")
	for {
		workunit := <-FromStealer

		logger.Debug(3, "(dataMover) received some work")
		//parsed := &Mediumwork{
		//	Workunit: raw.Workunit,
		//	Perfstat: raw.Perfstat,
		//	CWL_job:  raw.CWL_job,
		//	CWL_tool: raw.CWL_tool,
		//}
		//work := raw.Workunit

		work_id := workunit.Workunit_Unique_Identifier

		workmap.Set(work_id, ID_DATAMOVER, "dataMover")

		if conf.CWL_TOOL == "" {
			//make a working directory for the workunit (not for commandline execution !!!!!!)
			if err := workunit.Mkdir(); err != nil {
				logger.Error("[dataMover#workunit.Mkdir], workid=" + work_id.String() + " error=" + err.Error())
				workunit.Notes = append(workunit.Notes, "[dataMover#work.Mkdir]"+err.Error())
				workunit.SetState(core.WORK_STAT_ERROR, "see notes")
				//hand the parsed workunit to next stage and continue to get new workunit to process
				fromMover <- workunit
				continue
			}
		}
		if conf.CWL_TOOL == "" {
			//run the PreWorkExecutionScript
			if err := runPreWorkExecutionScript(workunit); err != nil {
				logger.Error("[dataMover#runPreWorkExecutionScript], workid=" + work_id.String() + " error=" + err.Error())
				workunit.Notes = append(workunit.Notes, "[dataMover#runPreWorkExecutionScript]"+err.Error())
				workunit.SetState(core.WORK_STAT_ERROR, "see notes")
				//hand the parsed workunit to next stage and continue to get new workunit to process
				fromMover <- workunit
				continue
			}
		}

		if conf.CWL_TOOL == "" {
			//check the availability prerequisite data and download if needed
			predatamove_start := time.Now().UnixNano()
			if moved_data, err := movePreData(workunit); err != nil {
				logger.Error("[dataMover#movePreData], workid=" + work_id.String() + " error=" + err.Error())
				workunit.Notes = append(workunit.Notes, "[dataMover#movePreData]"+err.Error())
				workunit.SetState(core.WORK_STAT_ERROR, "see notes")
				//hand the parsed workunit to next stage and continue to get new workunit to process
				fromMover <- workunit
				continue
			} else {
				if moved_data > 0 {
					workunit.WorkPerf.PreDataSize = moved_data
					predatamove_end := time.Now().UnixNano()
					workunit.WorkPerf.PreDataIn = float64(predatamove_end-predatamove_start) / 1e9
				}
			}
		}

		//parse the args, replacing @input_name to local file path (file not downloaded yet)

		if err := ParseWorkunitArgs(workunit); err != nil {
			logger.Error("err@dataMover.ParseWorkunitArgs, workid=" + work_id.String() + " error=" + err.Error())
			workunit.Notes = append(workunit.Notes, "[dataMover#ParseWorkunitArgs]"+err.Error())
			workunit.SetState(core.WORK_STAT_ERROR, "see notes")
			//hand the parsed workunit to next stage and continue to get new workunit to process
			fromMover <- workunit
			continue
		}

		//download input data
		if conf.CWL_TOOL == "" {

			datamove_start := time.Now().UnixNano()
			if moved_data, err := cache.MoveInputData(workunit); err != nil {
				logger.Error("err@dataMover.MoveInputData, workid=" + work_id.String() + " error=" + err.Error())
				workunit.Notes = append(workunit.Notes, "[dataMover#MoveInputData]"+err.Error())
				workunit.SetState(core.WORK_STAT_ERROR, "see notes")
				//hand the parsed workunit to next stage and continue to get new workunit to process
				fromMover <- workunit
				continue
			} else {
				workunit.WorkPerf.InFileSize = moved_data
				datamove_end := time.Now().UnixNano()
				workunit.WorkPerf.DataIn = float64(datamove_end-datamove_start) / 1e9
			}
		} else {
			// download required remote files (local files are ok, mostly for commandline execution)
			// job_doc := workunit.CWL.Job_input
			// 			for name, thing := range *job_doc {
			//
			// 				array, is_array := thing.(cwl_types.CWL_array_type)
			//
			// 				fmt.Println(name)
			// 				if is_array {
			//
			// 					for _, element := range *array.Get_Array() {
			// 						switch element.(type) {
			// 						case *cwl_types.File:
			// 							file, ok := element.(*cwl_types.File)
			// 							if !ok {
			// 								panic("not file")
			// 							}
			// 							fmt.Printf("%+v\n", *file)
			//
			// 							if file.Location != "" { // this is an IRI (URI)
			// 								// TODO: do something !!! download
			//
			// 							}
			//
			// 							if file.Path != "" {
			//
			// 								logger.Debug(1, "(dataMover) checking file %s=%s...", name, file.Path)
			//
			// 								if _, err := os.Stat(file.Path); os.IsNotExist(err) {
			// 									logger.Error("(dataMover) file %s=%s not found", name, file.Path)
			// 									workunit.SetState(core.WORK_STAT_ERROR)
			// 									//hand the parsed workunit to next stage and continue to get new workunit to process
			// 									fromMover <- workunit
			// 									continue
			// 								} else {
			// 									logger.Debug(1, "(dataMover) ok, found file %s=%s...", name, file.Path)
			// 								}
			// 							}
			// 						default:
			// 							spew.Dump(element)
			// 						}
			// 					}
			// 				}
			// 			}
		}

		if conf.CWL_TOOL == "" {

			//create userattr.json
			work_path, err := workunit.Path()
			if err != nil {
				return
			}
			user_attr := getUserAttr(workunit)
			if len(user_attr) > 0 {
				attr_json, _ := json.Marshal(user_attr)
				if err := ioutil.WriteFile(fmt.Sprintf("%s/userattr.json", work_path), attr_json, 0644); err != nil {
					logger.Error("err@dataMover_work.getUserAttr, workid=" + work_id.String() + " error=" + err.Error())
					workunit.Notes = append(workunit.Notes, "[dataMover#getUserAttr]"+err.Error())
					workunit.SetState(core.WORK_STAT_ERROR, "see notes")
					//hand the parsed workunit to next stage and continue to get new workunit to process
					fromMover <- workunit
					continue
				}
			}
		}

		fromMover <- workunit
	}
	control <- ID_DATAMOVER //we are ending
}

func proxyDataMover(control chan int) {
	fmt.Printf("proxyDataMover launched, client=%s\n", core.Self.Id)
	defer fmt.Printf("proxyDataMover exiting...\n")

	for {
		workunit := <-FromStealer
		work_id := workunit.Workunit_Unique_Identifier
		workmap.Set(work_id, ID_DATAMOVER, "proxyDataMover")
		//check the availability prerequisite data and download if needed
		if err := proxyMovePreData(workunit); err != nil {
			logger.Error("err@dataMover_work.movePreData, workid=" + work_id.String() + " error=" + err.Error())
			workunit.Notes = append(workunit.Notes, "[dataMover#proxyMovePreData]"+err.Error())
			workunit.SetState(core.WORK_STAT_ERROR, "see notes")
		}
		fromMover <- workunit
	}
	control <- ID_DATAMOVER
}

//parse workunit, fetch input data, compose command arguments
func ParseWorkunitArgs(work *core.Workunit) (err error) {

	workpath, err := work.Path()
	if err != nil {
		return
	}

	if work.Cmd.Dockerimage != "" || work.Cmd.DockerPull != "" {
		workpath = conf.DOCKER_WORK_DIR
	}

	args := []string{}
	var argList []string

	argstr := work.Cmd.Args
	if argstr != "" {
		logger.Debug(3, "argstr: %s", argstr)

		// use better file name replacement technique
		virtual_cmd_script := []string{argstr}
		replace_filepath_with_full_filepath(work.Inputs, workpath, virtual_cmd_script)
		argstr = virtual_cmd_script[0]

		argList = parse_arg_string(argstr)
	} else {

		if len(work.Cmd.ArgsArray) == 0 {
			return
		}

		logger.Debug(1, "work.Cmd.ArgsArray: %v (%d)", work.Cmd.ArgsArray, len(work.Cmd.ArgsArray))

		replace_filepath_with_full_filepath(work.Inputs, workpath, work.Cmd.ArgsArray)

		argList = work.Cmd.ArgsArray

	}

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

		// this might be deprecated by replace_filepath_with_full_filepath
		if strings.Contains(arg, "@") { //parse input/output to accessible local file
			segs := strings.Split(arg, "@")
			if len(segs) > 2 {
				return errors.New("invalid format in command args, multiple @ within one arg")
			}
			inputname := segs[1]
			for _, io := range work.Inputs {
				if io.FileName == inputname {
					inputFilePath := path.Join(workpath, inputname)
					parsedArg := fmt.Sprintf("%s%s", segs[0], inputFilePath)
					args = append(args, parsedArg)
				}
			}
			continue
		}
		//no @ or $, append directly
		args = append(args, arg)
	}

	work.Cmd.ParsedArgs = args
	logger.Debug(1, "work.Cmd.ParsedArgs: %v (%d)", work.Cmd.ParsedArgs, len(work.Cmd.ParsedArgs))
	work.SetState(core.WORK_STAT_PREPARED, "")
	return nil
}

//fetch file by shock url,  TODO remove
func fetchFile_old(filename string, url string, token string) (size int64, err error) {
	fmt.Printf("(fetchFile_old) fetching file name=%s, url=%s\n", filename, url)
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

//fetch prerequisite data (e.g. reference dbs)
func movePreData(workunit *core.Workunit) (size int64, err error) {
	for _, io := range workunit.Predata {
		name := io.FileName
		predata_directory := path.Join(conf.DATA_PATH, "predata")
		err = os.MkdirAll(predata_directory, 755)
		if err != nil {
			return 0, errors.New("error creating predata_directory: " + err.Error())
		}

		file_path := path.Join(predata_directory, name)
		dataUrl, uerr := io.DataUrl()
		if uerr != nil {
			return 0, uerr
		}

		// get shock and local md5sums
		isShockPredata := true
		node_md5 := ""
		if io.Node == "-" {
			isShockPredata = false
		} else {
			node, err := shock.ShockGet(io.Host, io.Node, workunit.Info.DataToken)
			if err != nil {
				return 0, errors.New("error in ShockGet: " + err.Error())
			}
			// rename file to be md5sum
			node_md5 = node.File.Checksum["md5"]
			file_path = path.Join(predata_directory, node_md5)
		}

		// file does not exist or its md5sum is wrong
		if !isFileExisting(file_path) {
			logger.Debug(2, "mover: fetching predata from url: "+dataUrl)
			logger.Event(event.PRE_IN, "workid="+workunit.Id+" url="+dataUrl)

			var md5sum string
			file_path_part := file_path + ".part" // temporary name
			// this gets file from any downloadable url, not just shock
			size, md5sum, err = shock.FetchFile(file_path_part, dataUrl, workunit.Info.DataToken, io.Uncompress, isShockPredata)
			if err != nil {
				return 0, errors.New("error in fetchFile: " + err.Error())
			}
			os.Rename(file_path_part, file_path)
			if err != nil {
				return 0, errors.New("error renaming after download of preData: " + err.Error())
			}
			if isShockPredata {
				if node_md5 != md5sum {
					return 0, errors.New("error downloaded file md5 does not mach shock md5, node: " + io.Node)
				} else {
					logger.Debug(2, "mover: predata "+name+" has md5sum "+md5sum)
				}
			}
		} else {
			logger.Debug(2, "mover: predata already exists: "+name)
		}

		// timstamp for last access - future caching
		accessfile, err := os.Create(file_path + ".access")
		if err != nil {
			return 0, errors.New("error creating predata access file: " + err.Error())
		}
		defer accessfile.Close()
		accessfile.WriteString(time.Now().String())

		// determine if running with docker
		wants_docker := false
		if workunit.Cmd.Dockerimage != "" {
			wants_docker = true
		}
		if wants_docker && conf.USE_DOCKER == "no" {
			return 0, errors.New("error: use of docker images has been disabled by administrator")
		}
		if wants_docker == false && conf.USE_DOCKER == "only" {
			return 0, errors.New("error: use of docker images is enforced by administrator")
		}

		// copy or create symlink in work dir
		work_path, xerr := workunit.Path()
		if xerr != nil {

			return 0, xerr
		}

		linkname := path.Join(work_path, name)
		if conf.NO_SYMLINK {
			// some programs do not accept symlinks (e.g. emirge), need to copy the file into the work directory
			logger.Debug(1, "copy predata: "+file_path+" -> "+linkname)
			_, err := shock.CopyFile(file_path, linkname)
			if err != nil {
				return 0, fmt.Errorf("error copying file from %s to % s: ", file_path, linkname, err.Error())
			}
		} else {
			if wants_docker {
				// new filepath for predata dir in container
				var docker_file_path string
				if isShockPredata {
					docker_file_path = path.Join(conf.DOCKER_WORKUNIT_PREDATA_DIR, node_md5)
				} else {
					docker_file_path = path.Join(conf.DOCKER_WORKUNIT_PREDATA_DIR, name)
				}
				logger.Debug(1, "creating dangling symlink: "+linkname+" -> "+docker_file_path)
				// dangling link will give error, we ignore that here
				_ = os.Symlink(docker_file_path, linkname)
			} else {
				logger.Debug(1, "symlink:"+linkname+" -> "+file_path)
				err = os.Symlink(file_path, linkname)
				if err != nil {
					return 0, errors.New("error creating predata file symlink: " + err.Error())
				}
			}
		}

		logger.Event(event.PRE_READY, "workid="+workunit.Id+";url="+dataUrl)
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

// split arg string into arg array, deliminated by space, ' '
func parse_arg_string(argstr string) (argarr []string) {
	var buf []byte
	argarr = []string{}
	startquote := false
	for j, c := range argstr {
		if c == '\'' {
			if startquote == true { // meet closing quote
				argarr = append(argarr, string(buf[:]))
				buf = nil
				startquote = false
				continue
			} else { //meet starting quote
				startquote = true
				continue
			}
		}
		//skip space if no quote encountered yet
		if startquote == false && c == ' ' {
			if len(buf) > 0 {
				argarr = append(argarr, string(buf[:]))
				buf = nil
			}
			continue
		}
		buf = append(buf, byte(c))
		if j == len(argstr)-1 {
			argarr = append(argarr, string(buf[:]))
		}
	}
	return
}
