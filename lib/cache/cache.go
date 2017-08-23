package cache

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	//"github.com/MG-RAST/AWE/lib/core/cwl"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/MG-RAST/golib/httpclient"
	"github.com/davecgh/go-spew/spew"
	"io"
	"io/ioutil"
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
				fmt.Printf("input found in cache, making link: " + file_path + " -> " + linkname + "\n")
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

func CWL_File_2_AWE_IO(file *cwl_types.File) (io *core.IO, err error) {

	url_obj := file.Location_url

	// example: http://localhost:8001/node/429a47aa-85e4-4575-9347-a78cfacb6979?download

	io = core.NewIO()

	url_string := url_obj.String()

	io.Url = url_string
	err = io.Url2Shock() // populates Host and Node
	if err != nil {
		err = fmt.Errorf("(CWL_File2AWE_IO) %s", err.Error())
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

	if work.CWL != nil {

		job_input := work.CWL.Job_input
		spew.Dump(job_input)

		for input_name, input := range *job_input {
			fmt.Println(input_name)
			spew.Dump(input)
			switch input.(type) {
			case *cwl_types.File:
				file := input.(*cwl_types.File)
				spew.Dump(*file)
				fmt.Printf("file: %+v\n", *file)

				var io *core.IO
				io, err = CWL_File_2_AWE_IO(file)
				if err != nil {
					return
				}

				var io_size int64
				io_size, err = MoveInputIO(work, io, work_path)
				if err != nil {
					err = fmt.Errorf("(MoveInputData) MoveInputIO returns %s", err.Error())
					return
				}
				spew.Dump(io)
				size += io_size

				continue
			case *cwl_types.String:
				continue
			default:
				err = fmt.Errorf("(MoveInputData) type %s not supoorted yet", reflect.TypeOf(input))
				return
			}
		}

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

//fetch file by shock url TODO deprecated
func fetchFile_deprecated(filename string, url string, token string) (size int64, err error) {
	fmt.Printf("(fetchFile_deprecated) fetching file name=%s, url=%s\n", filename, url)
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

func UploadOutputData(work *core.Workunit) (size int64, err error) {

	logger.Info("Processing %d outputs for uploading", len(work.Outputs))
	work_path, err := work.Path()
	if err != nil {
		return
	}
	for _, io := range work.Outputs {
		name := io.FileName
		var local_filepath string //local file name generated by the cmd
		var file_path string      //file name to be uploaded to shock

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

		if (io.Type == "copy") || (io.Type == "update") || io.NoFile {
			file_path = ""
		} else if fi, xerr := os.Stat(file_path); err != nil {
			//skip this output if missing file and optional
			if io.Optional {
				continue
			} else {
				err = fmt.Errorf("output %s not generated for workunit %s %s()", name, work.Id, xerr.Error())
				return
			}
		} else {
			if io.Nonzero && fi.Size() == 0 {
				err = fmt.Errorf("workunit %s generated zero-sized output %s while non-zero-sized file required", work.Id, name)
				return
			}
			size += fi.Size()
		}

		logger.Debug(1, "deliverer: push output to shock, filename="+name)
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

		logger.Debug(1, "UploadOutputData, core.PutFileToShock: %s (%s)", file_path, io.Node)
		sc := shock.ShockClient{Host: io.Host, Token: work.Info.DataToken}
		if err := sc.PutFileToShock(file_path, io.Node, work.Rank, attrfile_path, io.Type, io.FormOptions, io.NodeAttr); err != nil {

			time.Sleep(3 * time.Second) //wait for 3 seconds and try again
			if err := sc.PutFileToShock(file_path, io.Node, work.Rank, attrfile_path, io.Type, io.FormOptions, io.NodeAttr); err != nil {
				fmt.Errorf("push file error\n")
				logger.Error("op=pushfile,err=" + err.Error())
				return size, err
			}
		}
		logger.Event(event.FILE_DONE,
			"workid="+work.Id,
			"filename="+name,
			fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))

		if io.ShockIndex != "" {
			sc := shock.ShockClient{Host: io.Host, Token: work.Info.DataToken}
			if err := sc.ShockPutIndex(io.Node, io.ShockIndex); err != nil {
				logger.Error("warning: fail to create index on shock for shock node: " + io.Node)
			}
		}

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
	}
	return
}
