package cache

import (
	"crypto/sha1"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"strings"
	"time"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	shock "github.com/MG-RAST/go-shock-client"
)

func getCacheDir(id string) string {
	if len(id) < 7 {
		return conf.DATA_PATH
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s", conf.DATA_PATH, id[0:2], id[2:4], id[4:6], id)
}

func getCacheFilePath(id string) string {
	cacheDir := getCacheDir(id)
	return fmt.Sprintf("%s/%s.data", cacheDir, id)
}

func StatCacheFilePath(id string) (file_path string, err error) {
	file_path = getCacheFilePath(id)
	_, err = os.Stat(file_path)
	return file_path, err
}

func MoveInputIO(work *core.Workunit, io *core.IO, work_path string) (size int64, err error) {

	if !io.NoFile { // is file !
		dataUrl, uerr := io.DataUrl()
		if uerr != nil {
			err = uerr
			return
		}
		if io.FileName == "" {
			err = fmt.Errorf("io.Filename is empty")
			return
		}

		inputFilePath := path.Join(work_path, io.FileName)

		// create symlink if file has been cached
		if work.Rank == 0 && conf.CACHE_ENABLED && io.Node != "" {
			var file_path string
			file_path, err = StatCacheFilePath(io.Node)
			if err == nil {
				//make a link in work dir from cached file
				linkname := fmt.Sprintf("%s/%s", work_path, io.FileName)
				//fmt.Printf("input found in cache, making link: " + file_path + " -> " + linkname + "\n")
				err = os.Symlink(file_path, linkname)
				if err != nil {
					return
				}
				logger.Event(event.FILE_READY, "workid="+work.Id+";url="+dataUrl)
				return
			}

		}

		// only get file Part based on work.Partition
		if (work.Rank > 0) && (work.Partition != nil) && (work.Partition.Input == io.FileName) {
			dataUrl = fmt.Sprintf("%s&index=%s&part=%s", dataUrl, work.Partition.Index, work.Part())
		}
		logger.Debug(2, "mover: fetching input file from url:"+dataUrl)
		logger.Event(event.FILE_IN, "workid="+work.Id+";url="+dataUrl)

		// download file
		retry := 1
		for true {
			datamoved, _, err := shock.FetchFile(inputFilePath, dataUrl, work.Info.DataToken, io.Uncompress, false)
			if err != nil {
				if !strings.Contains(err.Error(), "Node has no file") {
					logger.Debug(3, "(MoveInputData) got: %s", err.Error())
					return size, err
				}
				logger.Debug(3, "(MoveInputData) got: Node has no file")
				if retry >= 3 {
					return size, err
				}
				logger.Warning("(MoveInputData) Will retry download, got this error: %s", err.Error())
				time.Sleep(time.Second * 20)
				retry += 1
				continue
			}

			size += datamoved
			break
		}
		logger.Event(event.FILE_READY, "workid="+work.Id+";url="+dataUrl)
	}

	// download node attributes if requested
	if io.AttrFile != "" {
		// get node
		node, xerr := shock.ShockGet(io.Host, io.Node, work.Info.DataToken)
		if xerr != nil {
			//return size, err
			err = errors.New("shock.ShockGet (node attributes) returned: " + xerr.Error())
			return
		}
		logger.Debug(2, "mover: fetching input attributes from node:"+node.Id)
		logger.Event(event.ATTR_IN, "workid="+work.Id+";node="+node.Id)
		// print node attributes
		work_path, yerr := work.Path()
		if yerr != nil {

			return 0, yerr
		}
		attrFilePath := fmt.Sprintf("%s/%s", work_path, io.AttrFile)
		attr_json, _ := json.Marshal(node.Attributes)
		err = ioutil.WriteFile(attrFilePath, attr_json, 0644)
		if err != nil {
			return
		}
		logger.Event(event.ATTR_READY, "workid="+work.Id+";path="+attrFilePath)
	}
	return
}

func UploadFile(file *cwl.File, inputfile_path string, shock_client *shock.ShockClient) (err error) {

	if file.Contents != "" {
		return
	}

	if strings.HasPrefix(inputfile_path, "file:") {
		err = fmt.Errorf("(UploadFile) prefix file: not allowed in inputfile_path")
		return
	}

	if strings.HasPrefix(file.Path, "file:") {
		err = fmt.Errorf("(UploadFile) prefix file: not allowed in file.Path")
		return
	}

	file_path := file.Path
	//fmt.Println("(UploadFile) here")
	//fmt.Printf("(UploadFile) file_path: %s\n", file_path)
	//fmt.Printf("(UploadFile) file.Location: %s\n", file.Location)
	if file_path == "" {

		pos := strings.Index(file.Location, "://")
		//fmt.Printf("(UploadFile) pos: %d\n", pos)
		if pos > 0 {
			scheme := file.Location[0:pos]
			//fmt.Printf("scheme: %s\n", scheme)

			switch scheme {

			case "file":
				file_path = strings.TrimPrefix(file.Location, "file://")

			case "http":
				// file already non-local
			case "https":
				// file already non-local
			case "ftp":
				// file already non-local

			default:
				err = fmt.Errorf("(UploadFile) unkown scheme \"%s\"", scheme)
				return
			}

		} else {
			//file.Location has no scheme, must be local file
			file_path = file.Location
		}

	}

	if !path.IsAbs(file_path) {
		file_path = path.Join(inputfile_path, file_path)
	}
	//fmt.Printf("file_path: %s\n", file_path)
	var file_info os.FileInfo
	file_info, err = os.Stat(file_path)
	if err != nil {

		var current_working_dir string
		current_working_dir, _ = os.Getwd()

		err = fmt.Errorf("(UploadFile) os.Stat returned: %s (inputfile_path: %s, file.Path: %s, file.Location: %s, current_working_dir: %s)", err.Error(), inputfile_path, file.Path, file.Location, current_working_dir)
		return
	}
	file_size := file_info.Size()

	basename := path.Base(file_path)

	if file_path == "" {
		err = fmt.Errorf("(UploadFile) file.Path is empty")
		return
	}

	new_file_name := ""
	if file.Basename != "" {
		new_file_name = file.Basename
	} else {
		new_file_name = basename
	}

	nodeid, err := shock_client.PostFile(file_path, new_file_name)
	if err != nil {
		err = fmt.Errorf("(UploadFile) shock_client.PostFile returned: %s", err.Error())
		return
	}

	var location_url *url.URL
	location_url, err = url.Parse(shock_client.Host + "/node/" + nodeid + shock.DATA_SUFFIX)

	if err != nil {
		err = fmt.Errorf("(UploadFile) url.Parse returned: %s", err.Error())
		return
	}

	file.Location = location_url.String()

	//fmt.Printf("file.Path A: %s", file.Path)

	//file.Path = strings.TrimPrefix(file.Path, inputfile_path)
	//file.Path = strings.TrimPrefix(file.Path, "/")
	file.SetPath("")
	//fmt.Printf("file.Path B: %s", file.Path)
	file.Basename = new_file_name

	file_size_int32 := int32(file_size)
	file.Size = &file_size_int32

	file_handle, err := os.Open(file_path)
	if err != nil {
		err = fmt.Errorf("(UploadFile) os.Open returned: %s", err.Error())
		return
	}
	defer file_handle.Close()

	h := sha1.New()

	_, err = io.Copy(h, file_handle)
	if err != nil {
		err = fmt.Errorf("(UploadFile) io.Copy returned: %s", err.Error())
		return
	}

	//fmt.Printf("% x", h.Sum(nil))

	file.Checksum = "sha1$" + hex.EncodeToString(h.Sum(nil))
	//fmt.Println(file.Location)
	return
}

func DownloadFile(file *cwl.File, download_path string, base_path string) (err error) {

	if file.Contents != "" {
		err = fmt.Errorf("(DownloadFile) File is a literal")
		return
	}

	if file.Location == "" {
		err = fmt.Errorf("(DownloadFile) Location is empty")
		return
	}

	//file_path := file.Path

	basename := file.Basename

	if basename == "" {
		err = fmt.Errorf("(DownloadFile) Basename is empty") // TODO infer basename if not found
		return
	}

	//if file_path == "" {
	//	return
	//}

	//if !path.IsAbs(file_path) {
	//	file_path = path.Join(path, basename)
	//}
	file_path := path.Join(download_path, basename)

	os.Stat(file_path)
	_, err = os.Stat(file_path)
	if err == nil {
		// file exists !!
		// create subfolder

		var newdir string
		newdir, err = ioutil.TempDir(download_path, "input_")
		if err != nil {
			err = fmt.Errorf("(DownloadFile) ioutil.TempDir returned: %s", err.Error())
			return
		}
		file_path = path.Join(newdir, basename)

	} else {
		err = nil
	}
	logger.Debug(3, "(DownloadFile) file.Path, downloading to: %s\n", file_path)

	//fmt.Printf("Using path %s\n", file_path)

	_, _, err = shock.FetchFile(file_path, file.Location, "", "", false)
	if err != nil {
		err = fmt.Errorf("(DownloadFile) shock.FetchFile returned: %s (download_path: %s, basename: %s)", err.Error(), download_path, basename)
		return
	}

	base_path = path.Join(strings.TrimSuffix(base_path, "/"), "/")

	file.Location = strings.TrimPrefix(strings.TrimPrefix(file_path, base_path), "/") // "file://" +  ?

	file.SetPath(file.Location)
	//fmt.Println("file:")
	//spew.Dump(file)

	return
}

func UploadDirectory(dir *cwl.Directory, current_path string, shock_client *shock.ShockClient) (count int, err error) {

	//pwd, _ := os.Getwd()
	//fmt.Printf("current working directory: %s\n", pwd)

	dir_path_fixed := path.Join(current_path, dir.Path)
	//fmt.Printf("dir_path_fixed: %s\n", dir_path_fixed)

	current_path = strings.TrimSuffix(current_path, "/") + "/" // make sure it has esxactly one / at the end
	//fmt.Printf("current_path: %s\n", current_path)

	//current_path_abs := path.Join(pwd, current_path)
	//fmt.Printf("current_path_abs: %s\n", current_path_abs)

	var fi os.FileInfo

	fi, err = os.Stat(dir_path_fixed)
	if err != nil {
		err = fmt.Errorf("(UploadDirectory) directory %s not found: %s", dir_path_fixed, err.Error())
		return
	}

	if !fi.IsDir() {
		err = fmt.Errorf("(UploadDirectory) %s is not a directory", dir_path_fixed)
		return
	}

	glob_pattern := path.Join(dir_path_fixed, "*")

	var matches []string
	matches, err = filepath.Glob(glob_pattern)
	if err != nil {
		err = fmt.Errorf("(UploadDirectory) filepath.Glob returned: %s", err.Error())
		return
	}

	//fmt.Printf("files matching: %d\n", len(matches))
	count = 0
	for _, match := range matches {
		// match is relative to current working directory
		//fmt.Printf("match : %s\n", match)

		// match_rel is relative to directory dir
		match_rel := strings.TrimPrefix(match, current_path)

		//fmt.Printf("match_rel : %s\n", match_rel)

		fi, err = os.Stat(match)
		if err != nil {
			err = fmt.Errorf("(UploadDirectory) os.Stat returned: %s", err.Error())
			return
		}

		if fi.IsDir() {
			subdir := cwl.NewDirectory()

			subdir.Path = match_rel
			subdir.Basename = path.Base(match)
			var subcount int
			subcount, err = UploadDirectory(subdir, current_path, shock_client)
			if err != nil {
				err = fmt.Errorf("(UploadDirectory) UploadDirectory returned: %s", err.Error())
				return
			}
			subdir.Path = match

			dir.Listing = append(dir.Listing, subdir)

			count += subcount

			continue
		}

		file := cwl.NewFile()
		file.SetPath(match_rel)
		//file.Basename = path.Base(match)
		_, err = ProcessIOData(file, current_path, current_path, "upload", shock_client)
		if err != nil {
			err = fmt.Errorf("(UploadDirectory) ProcessIOData returned: %s", err.Error())
			return
		}
		// fix path
		file.SetPath(match)
		dir.Listing = append(dir.Listing, file)

		count += 1
	}
	//fmt.Println("Listing:")
	//for _, listing := range dir.Listing {
	//	spew.Dump(listing.String())
	//}

	return
}

func ProcessIOData(native interface{}, current_path string, base_path string, io_type string, shock_client *shock.ShockClient) (count int, err error) {

	//fmt.Printf("(processIOData) start (type:  %s) \n", reflect.TypeOf(native))
	//defer fmt.Printf("(processIOData) end\n")
	switch native.(type) {

	case map[string]interface{}:
		native_map := native.(map[string]interface{})

		keys := make([]string, len(native_map))

		i := 0
		for key := range native_map {
			keys[i] = key
			i++
		}

		for _, key := range keys {
			value := native_map[key]
			var sub_count int

			//value_file, ok := value.(*cwl.File)
			//if ok {
			//	spew.Dump(*value_file)
			//	fmt.Printf("location: %s\n", value_file.Location)
			//}

			sub_count, err = ProcessIOData(value, current_path, base_path, io_type, shock_client)
			if err != nil {
				return
			}
			//if ok {
			//	spew.Dump(*value_file)
			//	fmt.Printf("location: %s\n", value_file.Location)
			//}
			count += sub_count
		}

	case *cwl.Job_document:
		//fmt.Printf("found Job_document\n")
		job_doc_ptr := native.(*cwl.Job_document)

		job_doc := *job_doc_ptr

		for i, _ := range job_doc {

			//id := value.Id
			//fmt.Printf("recurse into key: %s\n", id)
			var sub_count int
			sub_count, err = ProcessIOData(job_doc[i], current_path, base_path, io_type, shock_client)
			if err != nil {
				return
			}
			count += sub_count
		}

		return
	case cwl.NamedCWLType:
		named := native.(cwl.NamedCWLType)
		var sub_count int
		sub_count, err = ProcessIOData(named.Value, current_path, base_path, io_type, shock_client)
		if err != nil {
			return
		}
		count += sub_count
		return
	case cwl.Named_CWL_object:
		named := native.(cwl.Named_CWL_object)
		var sub_count int
		sub_count, err = ProcessIOData(named.Value, current_path, base_path, io_type, shock_client)
		if err != nil {
			return
		}
		count += sub_count
		return
	case *cwl.String:
		//fmt.Printf("found string\n")
		return
	case *cwl.Double:
		//fmt.Printf("found double\n")
		return
	case *cwl.Boolean:
		return
	case *cwl.File:
		//fmt.Println("got *cwl.File")
		if !conf.SUBMITTER_QUIET {
			logger.Debug(0, "Uploading file")
		}

		//spew.Dump(native)

		//fmt.Printf("found File\n")
		file, ok := native.(*cwl.File)
		if !ok {
			err = fmt.Errorf("could not cast to *cwl.File")
			return
		}

		if file.Contents != "" {
			return
		}

		if io_type == "upload" {
			//fmt.Println("XXXXX")
			//spew.Dump(*file)
			//fmt.Println(file.Path)
			//fmt.Println(file.Location)

			err = UploadFile(file, current_path, shock_client)
			if err != nil {

				err = fmt.Errorf("(ProcessIOData) *cwl.File UploadFile returned: %s (file: %s)", err.Error(), file)
				return
			}
			count += 1
		} else {

			// download
			err = DownloadFile(file, current_path, base_path)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) DownloadFile returned: %s (file: %s)", err.Error(), file)
				return
			}
			count += 1
		}

		if file.SecondaryFiles != nil {
			for i, _ := range file.SecondaryFiles {
				value := file.SecondaryFiles[i]
				var sub_count int
				sub_count, err = ProcessIOData(value, current_path, base_path, io_type, shock_client)
				if err != nil {
					err = fmt.Errorf("(ProcessIOData) (for SecondaryFiles) ProcessIOData returned: %s", err.Error())
					return
				}
				count += sub_count
			}

		}

		return
	case *cwl.Array:

		array, ok := native.(*cwl.Array)
		if !ok {
			err = fmt.Errorf("could not cast to *cwl.Array")
			return
		}

		for i, _ := range *array {

			//id := value.GetId()
			//fmt.Printf("recurse into key: %s\n", id)
			var sub_count int
			sub_count, err = ProcessIOData((*array)[i], current_path, base_path, io_type, shock_client)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) (for *cwl.Array) ProcessIOData returned: %s", err.Error())
				return
			}
			count += sub_count

		}
		return
	case []cwl.Named_CWL_object:
		array := native.([]cwl.Named_CWL_object)
		for i, _ := range array {

			//id := value.GetId()

			var sub_count int
			sub_count, err = ProcessIOData((array)[i], current_path, base_path, io_type, shock_client)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) (for *cwl.Array) ProcessIOData returned: %s", err.Error())
				return
			}
			count += sub_count

		}
		return
	case *cwl.Directory:
		if !conf.SUBMITTER_QUIET {
			logger.Debug(0, "Uploading directory")
		}
		//fmt.Printf("XXX *cwl.Directory\n")
		dir, ok := native.(*cwl.Directory)
		if !ok {
			err = fmt.Errorf("(ProcessIOData) could not cast to *cwl.Directory")
			return
		}

		if dir.Listing == nil && dir.Location == "" {
			err = fmt.Errorf("(ProcessIOData) cwl.Directory needs either Listing or Location")
			return
		}

		path_to_download_to := current_path

		if io_type == "download" {
			dir_basename := dir.Basename
			if dir_basename == "" {
				dir_basename = path.Base(dir.Path)

			}

			if dir_basename == "" {
				err = fmt.Errorf("(ProcessIOData) basename of subdir is empty")
				return
			}
			path_to_download_to = path.Join(current_path, dir_basename)

			dir.Location = path_to_download_to
			dir.Path = ""
			err = os.MkdirAll(path_to_download_to, 0777)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) MkdirAll returned: %s", err.Error())
				return
			}
		}

		if dir.Listing != nil {
			for k, _ := range dir.Listing {

				value := dir.Listing[k]
				//fmt.Printf("XXX *cwl.Directory, Listing %d (%s)\n", k, reflect.TypeOf(value))

				var sub_count int
				sub_count, err = ProcessIOData(value, path_to_download_to, base_path, io_type, shock_client)
				if err != nil {
					err = fmt.Errorf("(ProcessIOData) ProcessIOData for Directory.Listing returned (value: %s): %s", value, err.Error())
					return
				}
				count += sub_count

			}
			return
		}

		// case: dir.Listing == nil

		logger.Debug(3, "dir.Path: %s", dir.Path)
		if io_type == "upload" {

			if dir.Path == "" {
				if dir.Location == "" {
					err = fmt.Errorf("(ProcessIOData) Directory does not have Path or Location")
					return
				}

				var location_url *url.URL
				location_url, err = url.Parse(dir.Location)
				if err != nil {
					err = fmt.Errorf("(ProcessIOData) url.Parse returned: %s", err.Error())
					return
				}
				//fmt.Println(location_url.Path)
				//panic(location_url.Path)

				if location_url.Scheme == "" {
					location_url.Scheme = "file"
				}
				if location_url.Scheme == "file" {
					dir.Path = location_url.Path
					//spew.Dump(location_url)
				}

			}

			//dir.Path = strings.TrimPrefix(dir.Path, path)
			//dir.Path = strings.TrimPrefix(dir.Path, "/")

			if dir.Path == "" {
				err = fmt.Errorf("(ProcessIOData) Path empty")
				return
			}

			//var matches []string
			//matches, err = filepath.Glob(dir.Path)
			//if err != nil {
			//	err = fmt.Errorf("(ProcessIOData) Glob returned: %s", err.Error())
			//	return
			//}
			var sub_count int

			sub_count, err = UploadDirectory(dir, current_path, shock_client)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) UploadDirectory returned: %s", err.Error())
				return
			}
			count += sub_count

			//TODO Use recursive function to convert Location-based Directory into Listing-based one.

			// fmt.Printf("dir_path_fixed: %s\n", dir_path_fixed)
			// err = filepath.Walk(dir_path_fixed, func(path string, info os.FileInfo, err error) error {
			// 	if err != nil {
			// 		fmt.Printf("prevent panic by handling failure accessing a path %q: %v\n", dir, err)
			// 		return err
			// 	}
			// 	if info.IsDir() && info.Name() == "skipping" {
			// 		fmt.Printf("skipping a dir without errors: %+v \n", info.Name())
			// 		return filepath.SkipDir
			// 	}
			// 	if info.IsDir() {
			// 		fmt.Printf("visited dir: %q\n", path)
			// 	} else {
			// 		fmt.Printf("visited file: %q\n", path)
			// 	}

			// 	return nil
			// })
			// if err != nil {
			// 	err = fmt.Errorf("(ProcessIOData) filepath.Walk returned: %s", err.Error())
			// 	return
			// }

		}
		logger.Debug(3, "dir.Path: %s", dir.Path)

		return
	case *cwl.Record:

		rec := native.(*cwl.Record)

		for _, value := range *rec {
			//value := rec.Fields[k]
			var sub_count int
			sub_count, err = ProcessIOData(value, current_path, base_path, io_type, shock_client)
			if err != nil {
				return
			}
			count += sub_count
		}

	case cwl.Record:

		rec := native.(cwl.Record)

		for _, value := range rec {
			//value := rec.Fields[k]
			var sub_count int
			sub_count, err = ProcessIOData(value, current_path, base_path, io_type, shock_client)
			if err != nil {
				return
			}
			count += sub_count
		}
	case string:
		//fmt.Printf("found Null\n")
		return

	case *cwl.Null:
		//fmt.Printf("found Null\n")
		return

	case *cwl.CWLType:

		var file *cwl.File
		var ok bool
		file, ok = native.(*cwl.File)
		if ok {

			var sub_count int
			sub_count, err = ProcessIOData(file, current_path, base_path, io_type, shock_client)
			if err != nil {
				return
			}
			count += sub_count
			return
		}

		var dir *cwl.Directory
		dir, ok = native.(*cwl.Directory)
		if ok {

			var sub_count int
			sub_count, err = ProcessIOData(dir, current_path, base_path, io_type, shock_client)
			if err != nil {
				return
			}
			count += sub_count
			return
		}

	case *cwl.Workflow:
		workflow := native.(*cwl.Workflow)

		for input_pos, _ := range workflow.Inputs {
			input := workflow.Inputs[input_pos]
			if input.Default != nil {

				var sub_count int
				sub_count, err = ProcessIOData(input.Default, current_path, base_path, io_type, shock_client)
				if err != nil {
					return
				}
				count += sub_count

			}

		}

		for step_pos, _ := range workflow.Steps {
			step := &workflow.Steps[step_pos]

			var sub_count int
			sub_count, err = ProcessIOData(step, current_path, base_path, io_type, shock_client)
			if err != nil {
				return
			}
			count += sub_count

		}

	case *cwl.WorkflowStep:
		step := native.(*cwl.WorkflowStep)
		for pos, _ := range step.In {
			input := &step.In[pos]

			var sub_count int
			sub_count, err = ProcessIOData(input, current_path, base_path, io_type, shock_client)
			if err != nil {
				return
			}
			count += sub_count

		}
	case *cwl.WorkflowStepInput:
		input := native.(*cwl.WorkflowStepInput)

		var file *cwl.File
		var ok bool
		file, ok = input.Default.(*cwl.File)
		if ok {
			var sub_count int
			sub_count, err = ProcessIOData(file, current_path, base_path, io_type, shock_client)
			if err != nil {
				return
			}
			count += sub_count
		}
	case *cwl.CommandLineTool:

		clt := native.(*cwl.CommandLineTool)

		for i, _ := range clt.Inputs { // CommandInputParameter

			command_input_parameter := &clt.Inputs[i]

			if command_input_parameter.Default != nil {

				var sub_count int
				sub_count, err = ProcessIOData(command_input_parameter, current_path, base_path, io_type, shock_client)
				if err != nil {
					err = fmt.Errorf("(processIOData) CommandLineTool.Default ProcessIOData(for download) returned: %s", err.Error())
					return
				}
				count += sub_count

			}

		}

		for i, _ := range clt.Requirements {
			requirement := &clt.Requirements[i]

			var sub_count int
			sub_count, err = ProcessIOData(requirement, current_path, base_path, io_type, shock_client)
			if err != nil {
				err = fmt.Errorf("(processIOData) CommandLineTool.Default ProcessIOData(for download) returned: %s", err.Error())
				return
			}
			count += sub_count

		}

	case *cwl.ExpressionTool:

		et := native.(*cwl.ExpressionTool)

		for i, _ := range et.Inputs { // InputParameter

			input_parameter := &et.Inputs[i]

			if input_parameter.Default != nil {

				var sub_count int
				sub_count, err = ProcessIOData(input_parameter, current_path, base_path, io_type, shock_client)
				if err != nil {
					err = fmt.Errorf("(processIOData) InputParameter ProcessIOData(for download) returned: %s", err.Error())
					return
				}
				count += sub_count

			}

		}
	case *cwl.CommandInputParameter:
		// https://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputParameter

		cip := native.(*cwl.CommandInputParameter)

		if io_type == "upload" {
			var file *cwl.File
			var ok bool
			file, ok = cip.Default.(*cwl.File)
			if ok {
				var file_exists bool
				file_exists, err = file.Exists(current_path)
				if err != nil {
					err = fmt.Errorf("(processIOData) cwl.CommandInputParameter file.Exists returned: %s", err.Error())
					return
				}
				if !file_exists {
					// Defaults are optional, file missing is no error
					return
				}
			}
		}
		var sub_count int
		sub_count, err = ProcessIOData(cip.Default, current_path, base_path, io_type, shock_client)
		if err != nil {
			err = fmt.Errorf("(processIOData) CommandInputParameter ProcessIOData(for download) returned: %s", err.Error())
			return
		}
		count += sub_count

		// secondaryFiles (on upload, evaluate Expression)
		// https://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputParameter
		// if cip.SecondaryFiles != nil {
		// 	sub_count = 0
		// 	switch cip.SecondaryFiles.(type) {
		// 	case []interface{}:
		// 		sec_files_array := cip.SecondaryFiles.([]interface{})
		// 		for i, _ := range sec_files_array {
		// 			_ = i
		// 			//do something: sec_files_array[i]

		// 			sub_count++
		// 		}

		// 	case cwl.Expression:

		// 		expr := cip.SecondaryFiles.(cwl.Expression)

		// 		var expr_result interface{}
		// 		expr_result, err = expr.EvaluateExpression(nil, nil) // TODO need inputs
		// 		if err != nil {
		// 			err = fmt.Errorf("(processIOData) EvaluateExpression returned: %s", err.Error())
		// 			return
		// 		}

		// 		// valid only : File or []File
		// 		switch expr_result.(type) {
		// 		case cwl.File:
		// 			file := expr_result.(cwl.File)
		// 			cip.SecondaryFiles = file
		// 		case []cwl.File:

		// 			files := expr_result.([]cwl.File)
		// 			cip.SecondaryFiles = files

		// 		default:
		// 			err = fmt.Errorf("(processIOData) (expr_result A) unsupported type: %s", reflect.TypeOf(expr_result))
		// 			return
		// 		}

		// 	case string:
		// 		// upload
		// 		sec_str := cip.SecondaryFiles.(string)
		// 		for strings.HasPrefix(sec_str, "^") {
		// 			ext := filepath.Ext(sec_str)
		// 			sec_str = strings.TrimSuffix(sec_str, ext)
		// 			sec_str = strings.TrimPrefix(sec_str, "^")
		// 		}

		// 	case cwl.String:

		// 	default:
		// 		err = fmt.Errorf("(processIOData) (expr_result B) unsupported type: %s", reflect.TypeOf(cip.SecondaryFiles))
		// 		return

		// 	}

		// 	count += sub_count
		//}

	case *cwl.InputParameter:

		ip := native.(*cwl.InputParameter)

		var file *cwl.File
		var ok bool
		file, ok = ip.Default.(*cwl.File)
		if ok {
			var file_exists bool
			file_exists, err = file.Exists(current_path)
			if err != nil {
				err = fmt.Errorf("(processIOData) InputParameter file.Exists returned: %s", err.Error())
				return
			}
			if !file_exists {
				// Defaults are optional, file missing is no error
				return
			}
		}

		var sub_count int
		sub_count, err = ProcessIOData(ip.Default, current_path, base_path, io_type, shock_client)
		if err != nil {
			err = fmt.Errorf("(processIOData) InputParameter ProcessIOData(for download) returned: %s", err.Error())
			return
		}
		count += sub_count

	case *core.CWL_workunit:

		if io_type == "download" {
			work := native.(*core.CWL_workunit)

			var sub_count int
			sub_count, err = ProcessIOData(work.Job_input, current_path, base_path, "download", shock_client)
			if err != nil {
				err = fmt.Errorf("(processIOData) work.Job_input ProcessIOData(for download) returned: %s", err.Error())
				return
			}
			count += sub_count

			sub_count = 0
			sub_count, err = ProcessIOData(work.Tool, current_path, base_path, "download", shock_client)
			if err != nil {
				err = fmt.Errorf("(processIOData) work.Tool ProcessIOData(for download) returned: %s", err.Error())
				return
			}
			count += sub_count
		}
	case *cwl.Dirent:
		dirent := native.(*cwl.Dirent)

		if dirent.Entry == nil {
			return
		}

		switch dirent.Entry.(type) {

		case []interface{}:
			entry_array := dirent.Entry.([]interface{})
			for i, _ := range entry_array {
				sub_count := 0
				sub_count, err = ProcessIOData(entry_array[i], current_path, base_path, io_type, shock_client)
				if err != nil {
					err = fmt.Errorf("(processIOData) work.Tool ProcessIOData(for download) returned: %s", err.Error())
					return
				}
				count += sub_count
			}

		case cwl.File:
			file := dirent.Entry.(cwl.File)
			sub_count := 0
			sub_count, err = ProcessIOData(file, current_path, base_path, io_type, shock_client)
			if err != nil {
				err = fmt.Errorf("(processIOData) work.Tool ProcessIOData(for download) returned: %s", err.Error())
				return
			}
			count += sub_count
		case *cwl.File:
			file := dirent.Entry.(*cwl.File)
			sub_count := 0
			sub_count, err = ProcessIOData(file, current_path, base_path, io_type, shock_client)
			if err != nil {
				err = fmt.Errorf("(processIOData) work.Tool ProcessIOData(for download) returned: %s", err.Error())
				return
			}
			count += sub_count
		default:

		}

	case *cwl.Requirement:
		requirement := native.(*cwl.Requirement)

		var requirement_if interface{}
		requirement_if = *requirement

		switch requirement_if.(type) {
		case *cwl.InitialWorkDirRequirement:

			iwdr := requirement_if.(*cwl.InitialWorkDirRequirement)

			if iwdr.Listing != nil {
				switch iwdr.Listing.(type) {
				//case []interface{}:
				case []cwl.CWL_object:
					obj_array := iwdr.Listing.([]cwl.CWL_object)
					for i, _ := range obj_array {

						sub_count := 0
						sub_count, err = ProcessIOData(obj_array[i], current_path, base_path, "download", shock_client)
						if err != nil {
							err = fmt.Errorf("(processIOData) []cwl.CWL_object cwl.Requirement/Listing returned: %s", err.Error())
							return
						}
						count += sub_count
					}
				case cwl.File:
					sub_count := 0
					sub_count, err = ProcessIOData(iwdr.Listing, current_path, base_path, "download", shock_client)
					if err != nil {
						err = fmt.Errorf("(processIOData) cwl.File cwl.Requirement/Listing  returned: %s", err.Error())
						return
					}
					count += sub_count
				case *cwl.String:
					// continue

				default:
					err = fmt.Errorf("(processIOData) cwl.Requirement/Listing , unkown type: (%s)", reflect.TypeOf(iwdr.Listing))
					return
				}
			}

			return

		}
		//class := (*requirement).GetClass()

		return

	case []interface{}:
		//fmt.Println("(processIOData) []interface{}")
		// that should trigger only for $schemas
		native_array := native.([]interface{})

		if io_type == "upload" {

			for i, _ := range native_array {
				switch native_array[i].(type) {
				case cwl.File:
					// continue
				case string:

					schema_str := native_array[i].(string)

					this_file := cwl.NewFile()
					this_file.Path = schema_str
					native_array[i] = this_file
					sub_count := 0
					sub_count, err = ProcessIOData(this_file, current_path, base_path, "download", shock_client)
					if err != nil {
						err = fmt.Errorf("(processIOData) []cwl.CWL_object cwl.Requirement/Listing returned: %s", err.Error())
						return
					}
					count += sub_count

				default:
					err = fmt.Errorf("(processIOData) schemata , unkown type: (%s)", reflect.TypeOf(native_array[i]))
					return

				}
			}

		}
		if io_type == "download" {
			panic("not implemented")
		}

	default:
		//spew.Dump(native)
		err = fmt.Errorf("(processIOData) No handler for type \"%s\"\n", reflect.TypeOf(native))
		return
	}

	return
}

//fetch input data
func MoveInputData(work *core.Workunit) (size int64, err error) {

	work_path, xerr := work.Path()
	if xerr != nil {
		err = xerr
		return
	}

	if work.CWL_workunit != nil {

		//job_input := work.CWL_workunit.Job_input
		//fmt.Printf("job_input1:\n")
		//spew.Dump(job_input)

		//_, err = ProcessIOData(job_input, work_path, work_path, "download", nil)
		//if err != nil {
		//	err = fmt.Errorf("(MoveInputData) ProcessIOData(for download) returned: %s", err.Error())
		//	return
		//}

		_, err = ProcessIOData(work.CWL_workunit, work_path, work_path, "download", nil)
		if err != nil {
			err = fmt.Errorf("(MoveInputData) ProcessIOData(for download) returned: %s", err.Error())
			return
		}

		//fmt.Printf("job_input2:\n")
		//spew.Dump(job_input)

		//for _, input := range *job_input {
		//fmt.Println(input_name)
		//	var io_size int64

		//io_size, err = MoveInputCWL(work, work_path, input.Value)
		//	if err != nil {
		//		return
		//	}
		//	size += io_size
		//}

		return
	}

	for _, io := range work.Inputs {
		// skip if NoFile == true
		var io_size int64
		io_size, err = MoveInputIO(work, io, work_path)
		if err != nil {
			err = fmt.Errorf("(MoveInputData) MoveInputIO returns %s", err.Error())
			return
		}

		size += io_size
	}
	return
}

func isFileExistingInCache(id string) bool {
	file_path := getCacheFilePath(id)
	if _, err := os.Stat(file_path); err == nil {
		return true
	}
	return false
}

func UploadOutputIO(work *core.Workunit, io *core.IO) (size int64, new_node_id string, err error) {
	name := io.FileName
	var local_filepath string //local file name generated by the cmd
	var file_path string      //file name to be uploaded to shock

	work_path, err := work.Path()
	if err != nil {
		return
	}

	if io.Path != "" {
		local_filepath = io.Path
		file_path = local_filepath
	} else {

		if io.Directory != "" {
			local_filepath = fmt.Sprintf("%s/%s/%s", work_path, io.Directory, name)
			//if specified, rename the local file name to the specified shock node file name
			//otherwise use the local name as shock file name
			file_path = local_filepath
			if io.ShockFilename != "" {
				file_path = fmt.Sprintf("%s/%s/%s", work_path, io.Directory, io.ShockFilename)
				os.Rename(local_filepath, file_path)
			}
		} else {
			local_filepath = fmt.Sprintf("%s/%s", work_path, name)
			file_path = local_filepath
			if io.ShockFilename != "" {
				file_path = fmt.Sprintf("%s/%s", work_path, io.ShockFilename)
				os.Rename(local_filepath, file_path)
			}
		}
	}
	if (io.Type == "copy") || (io.Type == "update") || io.NoFile {
		file_path = ""
	} else {

		var fi os.FileInfo
		fi, err = os.Stat(file_path)
		if err != nil {
			//skip this output if missing file and optional
			if !io.Optional {
				err = fmt.Errorf("(UploadOutputIO) output %s not generated for workunit %s err=%s", name, work.Id, err.Error())
				return
			}
			err = nil
			return
		}
		if fi == nil {
			err = fmt.Errorf("(UploadOutputIO) fi is nil !?")
			return
		}

		if io.Nonzero && fi.Size() == 0 {
			err = fmt.Errorf("(UploadOutputIO) workunit %s generated zero-sized output %s while non-zero-sized file required", work.Id, name)
			return
		}
		size += fi.Size()

	}
	logger.Debug(1, "(UploadOutputIO) deliverer: push output to shock, filename="+name)
	logger.Event(event.FILE_OUT,
		"workid="+work.Id,
		"filename="+name,
		fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))

	//upload attribute file to shock IF attribute file is specified in outputs AND it is found in local directory.
	var attrfile_path string = ""
	if io.AttrFile != "" {
		attrfile_path = fmt.Sprintf("%s/%s", work_path, io.AttrFile)
		if fi, err := os.Stat(attrfile_path); err != nil || fi.Size() == 0 {
			attrfile_path = ""
		}
	}

	//set io.FormOptions["parent_node"] if not present and io.FormOptions["parent_name"] exists
	if parent_name, ok := io.FormOptions["parent_name"]; ok {
		for _, in_io := range work.Inputs {
			if in_io.FileName == parent_name {
				io.FormOptions["parent_node"] = in_io.Node
			}
		}
	}

	sc := shock.ShockClient{Host: io.Host, Token: work.Info.DataToken}
	sc.Debug = true

	new_node_id, err = sc.PutOrPostFile(file_path, io.Node, work.Rank, attrfile_path, io.Type, io.FormOptions, io.NodeAttr)
	if err != nil {

		time.Sleep(3 * time.Second) //wait for 3 seconds and try again
		new_node_id, err = sc.PutOrPostFile(file_path, io.Node, work.Rank, attrfile_path, io.Type, io.FormOptions, io.NodeAttr)
		if err != nil {
			err = fmt.Errorf("push file error: %s", err.Error())
			logger.Error("op=pushfile,err=" + err.Error())
			return
		}
	}

	if new_node_id != "" {
		io.Node = new_node_id
	}

	// worker only index if not parts node, otherwise server is responsible
	if (io.ShockIndex != "") && (work.Rank == 0) {
		sc := shock.ShockClient{Host: io.Host, Token: work.Info.DataToken}
		if err := sc.PutIndex(io.Node, io.ShockIndex); err != nil {
			logger.Error("warning: fail to create index on shock for shock node: " + io.Node)
		}
	}

	logger.Event(event.FILE_DONE,
		"workid="+work.Id,
		"filename="+name,
		fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))

	if conf.CACHE_ENABLED {
		//move output files to cache
		cacheDir := getCacheDir(io.Node)
		if err := os.MkdirAll(cacheDir, 0777); err != nil {
			logger.Error("cache os.MkdirAll():" + err.Error())
		}
		cacheFilePath := getCacheFilePath(io.Node) //use the same naming mechanism used by shock server
		//fmt.Printf("moving file from %s to %s\n", file_path, cacheFilePath)
		if err := os.Rename(file_path, cacheFilePath); err != nil {
			logger.Error("cache os.Rename():" + err.Error())
		}
	}
	return
}

func UploadOutputData(work *core.Workunit, shock_client *shock.ShockClient) (size int64, err error) {

	if work.CWL_workunit != nil {

		if work.CWL_workunit.Outputs != nil {
			//fmt.Println("Outputs 1")
			//scs := spew.Config
			//scs.DisableMethods = true

			//scs.Dump(work.CWL_workunit.Outputs)
			var upload_count int
			upload_count, err = ProcessIOData(work.CWL_workunit.Outputs, "", "", "upload", shock_client)
			if err != nil {
				err = fmt.Errorf("(UploadOutputData) ProcessIOData returned: %s", err.Error())
			}
			logger.Debug(3, "(UploadOutputData) %d files uploaded to shock", upload_count)
			//fmt.Println("Outputs 2")
			//scs.Dump(work.CWL_workunit.Outputs)
		}

	} else {

		var outputs []*core.IO
		outputs = work.Outputs
		logger.Info("Processing %d outputs for uploading", len(outputs))

		for _, io := range outputs {
			var io_size int64
			io_size, _, err = UploadOutputIO(work, io)
			if err != nil {
				err = fmt.Errorf("(UploadOutputData) UploadOutputIO returned: %s", err.Error())
				break
			}
			size += io_size
		}
	}

	return
}
