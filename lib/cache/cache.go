package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/davecgh/go-spew/spew"
	//"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	shock "github.com/MG-RAST/go-shock-client"
	//"github.com/davecgh/go-spew/spew"
	"io/ioutil"
	"net/url"
	"os"
	"path"
	"reflect"
	"strings"
	"time"
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
	fmt.Printf("(uploadFile) start\n")
	defer fmt.Printf("(uploadFile) end\n")

	scheme := ""
	if file.Location_url != nil {
		scheme = file.Location_url.Scheme
		host := file.Location_url.Host
		path := file.Location_url.Path
		//fmt.Printf("Location: '%s' '%s' '%s'\n", scheme, host, path)

		if scheme == "" {
			if host == "" {
				scheme = "file"
			} else {
				scheme = "http"
			}
			//fmt.Printf("Location (updated): '%s' '%s' '%s'\n", scheme, host, path)
		}

		if scheme == "file" {
			if host == "" || host == "localhost" {
				file.Path = path
			}
		} else {
			return
		}

	}

	if file.Location_url == nil && file.Location != "" {
		err = fmt.Errorf("(UploadFile) URL has not been parsed correctly")
		return
	}

	file_path := file.Path

	basename := path.Base(file_path)

	if file_path == "" {
		err = fmt.Errorf("(UploadFile) file.Path is empty")
		return
	}

	//fmt.Printf("file.Path: %s\n", file_path)

	if !path.IsAbs(file_path) {
		file_path = path.Join(inputfile_path, file_path)
	}

	nodeid, err := shock_client.PostFile(file_path, "")
	if err != nil {
		err = fmt.Errorf("(UploadFile) %s", err.Error())
		return
	}

	file.Location_url, err = url.Parse(shock_client.Host + "/node/" + nodeid + "?download")
	if err != nil {
		err = fmt.Errorf("(UploadFile) url.Parse returned: %s", err.Error())
		return
	}

	file.Location = file.Location_url.String()

	fmt.Printf("file.Path A: %s", file.Path)

	file.Path = strings.TrimPrefix(file.Path, inputfile_path)
	file.Path = strings.TrimPrefix(file.Path, "/")

	fmt.Printf("file.Path B: %s", file.Path)
	file.Basename = basename

	return
}

func DownloadFile(file *cwl.File, download_path string) (err error) {

	if file.Location == "" {
		err = fmt.Errorf("Location is empty")
		return
	}

	//file_path := file.Path

	basename := file.Basename

	if basename == "" {
		err = fmt.Errorf("Basename is empty") // TODO infer basename if not found
		return
	}

	//if file_path == "" {
	//	return
	//}

	//if !path.IsAbs(file_path) {
	//	file_path = path.Join(path, basename)
	//}
	file_path := path.Join(download_path, basename)
	logger.Debug(3, "file.Path, downloading to: %s\n", file_path)

	//fmt.Printf("Using path %s\n", file_path)

	_, _, err = shock.FetchFile(file_path, file.Location, "", "", false)
	if err != nil {
		return
	}
	file.Location = "file://" + file_path
	file.Path = file_path

	//fmt.Println("file:")
	//spew.Dump(file)

	return
}

func ProcessIOData(native interface{}, path string, io_type string, shock_client *shock.ShockClient) (count int, err error) {

	//fmt.Printf("(processIOData) start\n")
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

			sub_count, err = ProcessIOData(value, path, io_type, shock_client)
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
			sub_count, err = ProcessIOData(job_doc[i], path, io_type, shock_client)
			if err != nil {
				return
			}
			count += sub_count
		}

		return
	case cwl.NamedCWLType:
		named := native.(cwl.NamedCWLType)
		var sub_count int
		sub_count, err = ProcessIOData(named.Value, path, io_type, shock_client)
		if err != nil {
			return
		}
		count += sub_count

	case *cwl.String:
		//fmt.Printf("found string\n")
		return
	case *cwl.Double:
		//fmt.Printf("found double\n")
		return
	case *cwl.Boolean:
		return
	case *cwl.File:

		//fmt.Printf("found File\n")
		file, ok := native.(*cwl.File)
		if !ok {
			err = fmt.Errorf("could not cast to *cwl.File")
			return
		}

		if io_type == "upload" {
			err = UploadFile(file, path, shock_client)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) UploadFile returned: %s (file: %s)", err.Error(), file)
				return
			}
			count += 1
		} else {

			// download
			err = DownloadFile(file, path)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) DownloadFile returned: %s (file: %s)", err.Error(), file)
				return
			}
		}

		if file.SecondaryFiles != nil {
			for i, _ := range file.SecondaryFiles {
				value := file.SecondaryFiles[i]
				var sub_count int
				sub_count, err = ProcessIOData(value, path, io_type, shock_client)
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
			sub_count, err = ProcessIOData((*array)[i], path, io_type, shock_client)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) (for *cwl.Array) ProcessIOData returned: %s", err.Error())
				return
			}
			count += sub_count

		}
		return

	case *cwl.Directory:

		fmt.Printf("XXX *cwl.Directory\n")
		dir, ok := native.(*cwl.Directory)
		if !ok {
			err = fmt.Errorf("(ProcessIOData) could not cast to *cwl.Directory")
			return
		}

		if dir.Listing != nil {

			for k, _ := range dir.Listing {

				value := dir.Listing[k]
				fmt.Printf("XXX *cwl.Directory, Listing %d (%s)\n", k, reflect.TypeOf(value))
				var sub_count int
				sub_count, err = ProcessIOData(value, path, io_type, shock_client)
				if err != nil {
					err = fmt.Errorf("(ProcessIOData) ProcessIOData for Directory.Listing returned (value: %s): %s", value, err.Error())
					return
				}
				count += sub_count

			}

		}
		logger.Debug(3, "dir.Path: %s", dir.Path)
		if io_type == "upload" {
			dir.Path = strings.TrimPrefix(dir.Path, path)
			dir.Path = strings.TrimPrefix(dir.Path, "/")
		}
		logger.Debug(3, "dir.Path: %s", dir.Path)

		return
	case *cwl.Record:

		rec := native.(*cwl.Record)

		for _, value := range *rec {
			//value := rec.Fields[k]
			var sub_count int
			sub_count, err = ProcessIOData(value, path, io_type, shock_client)
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
			sub_count, err = ProcessIOData(value, path, io_type, shock_client)
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
	default:
		//spew.Dump(native)
		err = fmt.Errorf("(processIOData) No handler for type \"%s\"\n", reflect.TypeOf(native))
		return
	}

	return
}

func CWL_File_2_AWE_IO_deprecated(file *cwl.File) (io *core.IO, err error) { // TODO deprecate this !

	url_obj := file.Location_url

	if url_obj == nil {
		url_obj, err = url.Parse(file.Location)
		if err != nil {
			err = fmt.Errorf("(CWL_File2AWE_IO) url.Parse returned: %s", err.Error())
			return
		}
		file.Location_url = url_obj
	}
	// example: http://localhost:8001/node/429a47aa-85e4-4575-9347-a78cfacb6979?download

	io = core.NewIO()

	url_string := url_obj.String()

	io.Url = url_string
	err = io.Url2Shock() // populates Host and Node
	if err != nil {
		err = fmt.Errorf("(CWL_File2AWE_IO) io.Url2Shock returned: %s", err.Error())
		return
	}

	if file.Basename == "" {
		basename := path.Base(url_string)
		io.FileName = basename
	} else {
		io.FileName = file.Basename
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

		job_input := work.CWL_workunit.Job_input
		//fmt.Printf("job_input1:\n")
		spew.Dump(job_input)

		_, err = ProcessIOData(job_input, work_path, "download", nil)
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
			fmt.Println("Outputs 1")
			scs := spew.Config
			scs.DisableMethods = true

			scs.Dump(work.CWL_workunit.Outputs)
			var upload_count int
			upload_count, err = ProcessIOData(work.CWL_workunit.Outputs, "", "upload", shock_client)
			if err != nil {
				err = fmt.Errorf("(UploadOutputData) ProcessIOData returned: %s", err.Error())
			}
			logger.Debug(3, "(UploadOutputData) %d files uploaded to shock", upload_count)
			fmt.Println("Outputs 2")
			scs.Dump(work.CWL_workunit.Outputs)
		}

		// tool_result_map := work.CWL_workunit.Outputs.GetMap()

		// result_array := cwl.Job_document{}

		// for _, expected_output := range *work.CWL_workunit.OutputsExpected {
		// 	expected_full := expected_output.Id
		// 	logger.Debug(3, "(B) expected_full: %s", expected_full)
		// 	expected := path.Base(expected_full)

		// 	tool_result, ok := tool_result_map[expected] // cwl.File
		// 	if !ok {
		// 		resultlist := ""
		// 		for key, _ := range tool_result_map {
		// 			resultlist = resultlist + " " + key
		// 		}

		// 		// TODO check CommandLineOutput if output is optional
		// 		logger.Debug(3, "(UploadOutputData) Expected output %s is missing, might be optional (available: %s)", expected, resultlist)
		// 		continue
		// 	}

		// 	result_array = append(result_array, cwl.NewNamedCWLType(expected, tool_result))

		// 	output_class := tool_result.GetClass()
		// 	if output_class != string(cwl.CWL_File) {
		// 		continue
		// 	}

		// 	cwl_file, ok := tool_result.(*cwl.File)
		// 	if !ok {
		// 		err = fmt.Errorf("(UploadOutputData) Could not type-assert file (expected: %s)", expected)
		// 		return
		// 	}

		// 	if work.ShockHost == "" {
		// 		err = fmt.Errorf("No default Shock host defined !")
		// 		return
		// 	}

		// 	new_io := &core.IO{Name: expected}
		// 	new_io.FileName = cwl_file.Basename
		// 	new_io.Path = cwl_file.Path

		// 	new_io.Host = work.ShockHost

		// 	//outputs = append(outputs, new_io)

		// 	var io_size int64
		// 	var new_node_id string
		// 	io_size, new_node_id, err = UploadOutputIO(work, new_io)
		// 	if err != nil {
		// 		return
		// 	}
		// 	size += io_size

		// 	cwl_file.Location = work.ShockHost + "/node/" + new_node_id + "?download"
		// 	cwl_file.Path = ""

		// }
		// //spew.Dump(work.CWL_workunit.OutputsExpected)
		// //panic("uga")
		// work.CWL_workunit.Results = &result_array

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
