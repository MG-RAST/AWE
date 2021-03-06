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

	"github.com/davecgh/go-spew/spew"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	shock "github.com/MG-RAST/go-shock-client"
	"golang.org/x/text/collate"
	"golang.org/x/text/language"
)

func getCacheDir(id string) string {
	if len(id) < 7 {
		return conf.DATA_PATH
	}
	return fmt.Sprintf("%s/%s/%s/%s/%s", conf.DATA_PATH, id[0:2], id[2:4], id[4:6], id)
}

// getCacheFilePath _
func getCacheFilePath(id string) string {
	cacheDir := getCacheDir(id)
	return fmt.Sprintf("%s/%s.data", cacheDir, id)
}

// StatCacheFilePath _
func StatCacheFilePath(id string) (filePath string, err error) {
	filePath = getCacheFilePath(id)
	_, err = os.Stat(filePath)
	return filePath, err
}

// MoveInputIO _
func MoveInputIO(work *core.Workunit, io *core.IO, workPath string) (size int64, err error) {

	if !io.NoFile { // is file !
		var dataURL string
		dataURL, err = io.DataUrl()
		if err != nil {
			err = fmt.Errorf("(MoveInputIO) io.DataUrl returned: %s", err.Error())
			return
		}
		if io.FileName == "" {
			err = fmt.Errorf("(MoveInputIO) io.Filename is empty")
			return
		}

		inputFilePath := path.Join(workPath, io.FileName)

		// create symlink if file has been cached
		if work.Rank == 0 && conf.CACHE_ENABLED && io.Node != "" {
			var file_path string
			file_path, err = StatCacheFilePath(io.Node)
			if err == nil {
				//make a link in work dir from cached file
				linkname := fmt.Sprintf("%s/%s", workPath, io.FileName)
				//fmt.Printf("input found in cache, making link: " + file_path + " -> " + linkname + "\n")
				err = os.Symlink(file_path, linkname)
				if err != nil {
					return
				}
				logger.Event(event.FILE_READY, "workid="+work.ID+";url="+dataURL)
				return
			}

		}

		// only get file Part based on work.Partition
		if (work.Rank > 0) && (work.Partition != nil) && (work.Partition.Input == io.FileName) {
			dataURL = fmt.Sprintf("%s&index=%s&part=%s", dataURL, work.Partition.Index, work.Part())
		}
		logger.Debug(2, "mover: fetching input file from url:"+dataURL)
		logger.Event(event.FILE_IN, "workid="+work.ID+";url="+dataURL)

		// download file
		retry := 1
		for true {
			var datamoved int64
			datamoved, _, err = shock.FetchFile(inputFilePath, dataURL, work.Info.DataToken, io.Uncompress, false)
			if err != nil {
				if strings.Contains(err.Error(), "Node has no file") {
					//logger.Debug(3, "(MoveInputData) got: %s", err.Error())
					err = fmt.Errorf("(MoveInputData) Node has no file, do not retry: %s", err.Error())
					return
				}

				if retry >= 3 {
					err = fmt.Errorf("(MoveInputData) (retry: %d) shock.FetchFile returned: %s", retry, err.Error())
					return
				}

				logger.Warning("(MoveInputData) Will retry download, got this error: %s", err.Error())
				err = nil

				time.Sleep(time.Second * 20)
				retry++

				continue
			}

			size += datamoved
			break
		}
		logger.Event(event.FILE_READY, "workid="+work.ID+";url="+dataURL)
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
		logger.Event(event.ATTR_IN, "workid="+work.ID+";node="+node.Id)
		// print node attributes
		workPathNew, yerr := work.Path()
		if yerr != nil {

			return 0, yerr
		}
		attrFilePath := fmt.Sprintf("%s/%s", workPathNew, io.AttrFile)
		attrJSON, _ := json.Marshal(node.Attributes)
		err = ioutil.WriteFile(attrFilePath, attrJSON, 0644)
		if err != nil {
			return
		}
		logger.Event(event.ATTR_READY, "workid="+work.ID+";path="+attrFilePath)
	}
	return
}

// UploadFile _
func UploadFile(file *cwl.File, inputfilePath string, shockClient *shock.ShockClient, lazyUpload bool) (count int, err error) {

	if strings.HasPrefix(inputfilePath, "file:") {
		err = fmt.Errorf("(UploadFile) prefix file: not allowed in inputfile_path (%s)", inputfilePath)
		return
	}

	//if strings.HasPrefix(file.Path, "file:") {
	//	err = fmt.Errorf("(UploadFile) prefix file: not allowed in file.Path (%s)", file.Path)
	//	return
	//}

	filePath := file.Path
	var fileSize int64
	newFileName := ""

	if file.Contents == "" {

		filePath = strings.TrimPrefix(filePath, "file://")

		//fmt.Println("(UploadFile) here")
		//fmt.Printf("(UploadFile) file_path: %s\n", file_path)
		//fmt.Printf("(UploadFile) file.Location: %s\n", file.Location)
		if filePath == "" {

			pos := strings.Index(file.Location, "://")
			//fmt.Printf("(UploadFile) pos: %d\n", pos)
			if pos > 0 {
				scheme := file.Location[0:pos]
				//fmt.Printf("scheme: %s\n", scheme)

				switch scheme {

				case "file":
					filePath = strings.TrimPrefix(file.Location, "file://")

				case "http":
					// file already non-local
					return
				case "https":
					// file already non-local
					return
				case "ftp":
					// file already non-local
					return

				default:
					err = fmt.Errorf("(UploadFile) unkown scheme \"%s\"", scheme)
					return
				}

			} else {
				//file.Location has no scheme, must be local file
				filePath = file.Location
			}

			if filePath == "" {
				err = fmt.Errorf("(UploadFile) filePath is empty and could not be derived (Location: %s)", file.Location)
				return
			}

		}

		if !path.IsAbs(filePath) {
			filePath = path.Join(inputfilePath, filePath)
		}

		var fileInfo os.FileInfo
		fileInfo, err = os.Stat(filePath)
		if err != nil {

			var currentWorkingDir string
			currentWorkingDir, _ = os.Getwd()

			err = fmt.Errorf("(UploadFile) os.Stat returned: %s (inputfilePath: %s, file.Path: %s, file.Location: %s, currentWorkingDir: %s)", err.Error(), inputfilePath, file.Path, file.Location, currentWorkingDir)
			return
		}

		if fileInfo.IsDir() {

			err = fmt.Errorf("(UploadFile) file_path is a directory: %s", filePath)
			return
		}

		fileSize = fileInfo.Size()

		basename := path.Base(filePath)

		if filePath == "" {
			err = fmt.Errorf("(UploadFile) file.Path is empty")
			return
		}

		if file.Basename != "" {
			newFileName = file.Basename
		} else {
			newFileName = basename
		}

		file_size_int32 := int32(fileSize)
		file.Size = &file_size_int32

		var file_handle *os.File
		file_handle, err = os.Open(filePath)
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

	} else {
		fileSize = int64(len(file.Contents))

		if file.Basename == "" {
			file.Basename = file.GetID()
		}
		newFileName = file.Basename
		filePath = ""
	}
	//shockClient.Debug = true
	var nodeid string
	if lazyUpload {
		//example: "${SHOCK_SERVER}/node?querynode&file.checksum.md5=37cbb1f811a144158cec0cbd87d97941"
		//nodeid, err = shockClient.PostFileIf(file_path, new_file_name)

		nodeid, err = shockClient.PostFileLazy(filePath, newFileName, file.Contents, false)
		if err != nil {
			err = fmt.Errorf("(UploadFile) shockClient.PostFileLazy returned: %s", err.Error())
			return
		}
	} else {

		nodeid, err = shockClient.PostFile(filePath, file.Contents, newFileName)
		if err != nil {
			err = fmt.Errorf("(UploadFile) shockClient.PostFile returned: %s", err.Error())
			return
		}
	}
	file.Contents = ""

	var locationURL *url.URL
	locationURL, err = url.Parse(shockClient.Host) // Host includes a path to the API !
	if err != nil {
		err = fmt.Errorf("(UploadFile) url.Parse returned: %s", err.Error())
		return
	}
	locationURL.Path = path.Join(locationURL.Path, "node", nodeid)
	locationURL.RawQuery = strings.TrimPrefix(shock.DATA_SUFFIX, "?")
	file.Location = locationURL.String()

	//fmt.Printf("file.Path A: %s", file.Path)

	//file.Path = strings.TrimPrefix(file.Path, inputfile_path)
	//file.Path = strings.TrimPrefix(file.Path, "/")
	file.SetPath("")
	if file.Path != "" {
		err = fmt.Errorf("(UploadFile) A) file.Path is not empty !?")
		return
	}
	//fmt.Printf("file.Path B: %s\n", file.Path)
	file.Basename = newFileName

	count = 1
	if file.Path != "" {
		err = fmt.Errorf("(UploadFile) A) file.Path is not empty !?")
		return
	}
	return
}

// DownloadFile _
func DownloadFile(file *cwl.File, downloadPath string, basePath string, shockClient *shock.ShockClient) (err error) {

	if file.Contents != "" {
		err = fmt.Errorf("(DownloadFile) File is a literal")
		return
	}

	if file.Location == "" {
		err = fmt.Errorf("(DownloadFile) Location is empty (%s)", spew.Sdump(file))
		return
	}

	//file_path := file.Path

	basename := file.Basename

	if basename == "" {

		basenameArray := strings.Split(file.Location, "?")
		basenameArrayFirst := basenameArray[0] // remove query string

		new_basename := path.Base(basenameArrayFirst)

		basename = new_basename
		file.Basename = basename
	}

	if basename == "" {
		err = fmt.Errorf("(DownloadFile) Basename is empty (%s)", spew.Sdump(*file)) // TODO infer basename if not found
		return
	}

	//if file_path == "" {
	//	return
	//}

	//if !path.IsAbs(file_path) {
	//	file_path = path.Join(path, basename)
	//}
	filePath := path.Join(downloadPath, basename)

	os.Stat(filePath)
	_, err = os.Stat(filePath)
	if err == nil {
		// file exists !!
		// create subfolder

		var newdir string
		newdir, err = ioutil.TempDir(downloadPath, "input_")
		if err != nil {
			err = fmt.Errorf("(DownloadFile) ioutil.TempDir returned: %s", err.Error())
			return
		}
		filePath = path.Join(newdir, basename)

	} else {
		err = nil
	}
	logger.Debug(3, "(DownloadFile) file.Path, downloading to: %s", filePath)

	//fmt.Printf("Using path %s\n", filePath)

	token := ""

	if shockClient != nil {
		if strings.HasPrefix(file.Location, shockClient.Host) {
			token = shockClient.Token
		}
	}

	_, err = os.Stat(downloadPath)
	if err != nil {
		if os.IsNotExist(err) {
			err = os.MkdirAll(downloadPath, 0777)
			if err != nil {
				err = fmt.Errorf("(DownloadFile) os.MkdirAll returned: %s", err.Error())
				return
			}
		} else {
			err = fmt.Errorf("(DownloadFile) os.Stat returned: %s", err.Error())
			return
		}
	}

	_, _, err = shock.FetchFile(filePath, file.Location, token, "", false)
	if err != nil {
		err = fmt.Errorf("(DownloadFile) shock.FetchFile returned: %s (download_path: %s, basename: %s, file.Location: %s, TokenLength: %d)", err.Error(), downloadPath, basename, file.Location, len(token))
		return
	}

	basePath = path.Join(strings.TrimSuffix(basePath, "/"), "/")

	file.Location = strings.TrimPrefix(strings.TrimPrefix(filePath, basePath), "/") // "file://" +  ?
	file.Nameroot = ""
	//file.SetPath(file.Location)
	//fmt.Printf("file.Path: %s\n", file.Path)

	return
}

// UploadDirectory _
func UploadDirectory(dir *cwl.Directory, currentPath string, shockClient *shock.ShockClient, context *cwl.WorkflowContext, lazyUpload bool) (count int, err error) {

	//pwd, _ := os.Getwd()
	//fmt.Printf("current working directory: %s\n", pwd)

	dirPathFixed := path.Join(currentPath, dir.Path)
	//fmt.Printf("dir_path_fixed: %s\n", dir_path_fixed)

	currentPath = strings.TrimSuffix(currentPath, "/") + "/" // make sure it has esxactly one / at the end
	//fmt.Printf("currentPath: %s\n", currentPath)

	//currentPath_abs := path.Join(pwd, currentPath)
	//fmt.Printf("currentPath_abs: %s\n", currentPath_abs)

	var fi os.FileInfo

	fi, err = os.Stat(dirPathFixed)
	if err != nil {
		err = fmt.Errorf("(UploadDirectory) directory %s not found: %s", dirPathFixed, err.Error())
		return
	}

	if !fi.IsDir() {
		err = fmt.Errorf("(UploadDirectory) %s is not a directory", dirPathFixed)
		return
	}

	globPattern := path.Join(dirPathFixed, "*")

	var matches []string
	matches, err = filepath.Glob(globPattern)
	if err != nil {
		err = fmt.Errorf("(UploadDirectory) filepath.Glob returned: %s", err.Error())
		return
	}
	// fmt.Println("matches:")
	// spew.Dump(matches)

	// make sure that the result of glob is sorted
	c := collate.New(language.Und)
	c.SortStrings(matches)
	// fmt.Println("matches:")
	// spew.Dump(matches)

	//fmt.Printf("files matching: %d\n", len(matches))
	count = 0
	for _, match := range matches {
		// match is relative to current working directory
		//fmt.Printf("match : %s\n", match)

		if path.Ext(match) == ".md5" {
			continue
		}

		// matchRel is relative to directory dir
		matchRel := strings.TrimPrefix(match, currentPath)

		//fmt.Printf("matchRel : %s\n", matchRel)

		fi, err = os.Stat(match)
		if err != nil {
			err = fmt.Errorf("(UploadDirectory) os.Stat returned: %s", err.Error())
			return
		}

		if fi.IsDir() {
			subdir := cwl.NewDirectory()

			subdir.Path = matchRel
			subdir.Basename = path.Base(match)
			var subcount int
			subcount, err = UploadDirectory(subdir, currentPath, shockClient, context, lazyUpload)
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
		file.SetPath(matchRel)
		//file.Basename = path.Base(match)
		var uploadCount int
		uploadCount, err = ProcessIOData(file, currentPath, currentPath, "upload", shockClient, context, lazyUpload, false)
		if err != nil {
			err = fmt.Errorf("(UploadDirectory) ProcessIOData returned: %s", err.Error())
			return
		}
		// fix path
		file.SetPath(match)
		file.Path = ""
		dir.Listing = append(dir.Listing, file)

		count += uploadCount
	}
	// fmt.Println("Listing:")
	// for _, listing := range dir.Listing {
	// 	spew.Dump(listing.String())
	// }

	return
}

// ProcessIOData
// lazyUpload boolean: get Shock Copy node is data already exists in Shock
func ProcessIOData(native interface{}, currentPath string, basePath string, ioType string, shockClient *shock.ShockClient, context *cwl.WorkflowContext, lazyUpload bool, removeIDField bool) (count int, err error) {

	//fmt.Printf("(processIOData) start (type:  %s) \n", reflect.TypeOf(native))

	//defer fmt.Printf("(processIOData) end\n")
	switch native.(type) {

	case map[string]interface{}:
		nativeMap := native.(map[string]interface{})

		keys := make([]string, len(nativeMap))

		i := 0
		for key := range nativeMap {
			keys[i] = key
			i++
		}

		for _, key := range keys {
			value := nativeMap[key]
			var sub_count int

			//value_file, ok := value.(*cwl.File)
			//if ok {
			//	spew.Dump(*value_file)
			//	fmt.Printf("location: %s\n", value_file.Location)
			//}

			sub_count, err = ProcessIOData(value, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
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
			sub_count, err = ProcessIOData(job_doc[i], currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				return
			}
			count += sub_count
		}

		return
	case cwl.NamedCWLType:
		named := native.(cwl.NamedCWLType)
		var sub_count int
		sub_count, err = ProcessIOData(named.Value, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
		if err != nil {
			return
		}
		count += sub_count
		return
	case cwl.NamedCWLObject:
		named := native.(cwl.NamedCWLObject)
		var sub_count int
		sub_count, err = ProcessIOData(named.Value, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
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

		//spew.Dump(native)

		//fmt.Printf("found File\n")
		file, ok := native.(*cwl.File)
		if !ok {
			err = fmt.Errorf("could not cast to *cwl.File")
			return
		}

		if ioType == "upload" {
			if !conf.SUBMITTER_QUIET {
				logger.Debug(0, "(ProcessIOData) Uploading file %s %s", file.Path, file.Location)
			}

			//fmt.Println("XXXXX")
			//spew.Dump(*file)
			//fmt.Printf("file.Path: %s\n", file.Path)
			//fmt.Printf("file.Location: %s\n", file.Location)
			//filePath := file.Path
			var sub_count int // sub_count is 0 or 1

			sub_count, err = UploadFile(file, currentPath, shockClient, lazyUpload)
			if err != nil {

				err = fmt.Errorf("(ProcessIOData) *cwl.File currentPath:%s UploadFile returned: %s (file: %s)", currentPath, err.Error(), spew.Sdump(*file))
				return
			}

			count += sub_count
			//file.UpdateComponents(filePath)
			//fmt.Printf("file.Path: %s\n", file.Path)

			//fmt.Printf("file.Location: %s\n", file.Location)
			//count += 1
		} else {

			// download

			err = DownloadFile(file, currentPath, basePath, shockClient)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) DownloadFile returned: %s (file: %s)", err.Error(), file)
				return
			}
			count += 1
			//file.UpdateComponents(file.Path)
		}

		if file.SecondaryFiles != nil {
			for i, _ := range file.SecondaryFiles {
				value := file.SecondaryFiles[i]
				var sub_count int
				sub_count, err = ProcessIOData(value, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
				if err != nil {
					err = fmt.Errorf("(ProcessIOData) (for SecondaryFiles) ProcessIOData returned: %s", err.Error())
					return
				}
				count += sub_count
			}

		}
		//spew.Dump(native)
		return
	case *cwl.Array:

		array, ok := native.(*cwl.Array)
		if !ok {
			err = fmt.Errorf("could not cast to *cwl.Array")
			return
		}

		for i, _ := range *array {

			//id := value.GetID()
			//fmt.Printf("recurse into key: %s\n", id)
			var sub_count int
			sub_count, err = ProcessIOData((*array)[i], currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) (for *cwl.Array) currentPath:%s ProcessIOData returned: %s", currentPath, err.Error())
				return
			}
			count += sub_count

		}
		return
	case []cwl.NamedCWLObject:
		array := native.([]cwl.NamedCWLObject)
		for i, _ := range array {

			//id := value.GetID()

			var sub_count int
			sub_count, err = ProcessIOData((array)[i], currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				err = fmt.Errorf("(ProcessIOData) (for *cwl.Array) currentPath:%s ProcessIOData returned: %s", currentPath, err.Error())
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

		if dir.Listing == nil && dir.Location == "" && dir.Path == "" {
			err = fmt.Errorf("(ProcessIOData) cwl.Directory needs either Listing, Location or Path")
			return
		}

		path_to_download_to := currentPath

		if ioType == "download" {
			dir_basename := dir.Basename
			if dir_basename == "" {
				dir_basename = path.Base(dir.Path)

			}

			if dir_basename == "" {
				err = fmt.Errorf("(ProcessIOData) basename of subdir is empty")
				return
			}
			path_to_download_to = path.Join(currentPath, dir_basename)

			dir.Path = ""
			if dir.Location != "" {
				// in case of a Directory literal, do not set Location field
				dir.Location = dir_basename
				err = os.MkdirAll(path_to_download_to, 0777)
				if err != nil {
					err = fmt.Errorf("(ProcessIOData) MkdirAll returned: %s", err.Error())
					return
				}
			}

		}

		if dir.Listing != nil {
			for k := range dir.Listing {

				value := dir.Listing[k]
				//fmt.Printf("XXX *cwl.Directory, Listing %d (%s)\n", k, reflect.TypeOf(value))

				var sub_count int
				sub_count, err = ProcessIOData(value, path_to_download_to, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
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
		if ioType == "upload" {

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

			sub_count, err = UploadDirectory(dir, currentPath, shockClient, context, lazyUpload)
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
			sub_count, err = ProcessIOData(value, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				return
			}
			count += sub_count
		}
		if removeIDField {
			delete(*rec, "id")
		}
	case cwl.Record:

		rec := native.(cwl.Record)

		for _, value := range rec {
			//value := rec.Fields[k]
			var sub_count int
			sub_count, err = ProcessIOData(value, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				return
			}
			count += sub_count
		}

		if removeIDField {
			delete(rec, "id")
		}

	case string:
		//fmt.Printf("found Null\n")
		return
	case *cwl.Int:

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
			sub_count, err = ProcessIOData(file, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
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
			sub_count, err = ProcessIOData(dir, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
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
			//spew.Dump(input)
			if input.Default != nil {

				var sub_count int
				sub_count, err = ProcessIOData(&input, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
				if err != nil {
					return
				}
				count += sub_count

			}

		}
		//panic(count)
		for step_pos, _ := range workflow.Steps {
			step := &workflow.Steps[step_pos]

			var sub_count int
			sub_count, err = ProcessIOData(step, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				return
			}
			count += sub_count

		}

	case *cwl.WorkflowStep:
		//fmt.Println("(processIOData) *cwl.WorkflowStep")
		step := native.(*cwl.WorkflowStep)
		for pos, _ := range step.In {
			input := &step.In[pos]

			var sub_count int
			sub_count, err = ProcessIOData(input, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				return
			}
			count += sub_count

		}

		//spew.Dump(step)
		//panic("oh no!")
		var process interface{}
		process, _, err = step.GetProcess(context)
		if process != nil {
			var sub_count int
			sub_count, err = ProcessIOData(process, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
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
			sub_count, err = ProcessIOData(file, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				return
			}
			count += sub_count
		}
	case *cwl.CommandLineTool:
		//fmt.Println("(processIOData) *cwl.CommandLineTool")
		clt := native.(*cwl.CommandLineTool)

		for i := range clt.Inputs { // CommandInputParameter

			commandInputParameter := &clt.Inputs[i]

			var sub_count int
			sub_count, err = ProcessIOData(commandInputParameter, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				err = fmt.Errorf("(processIOData) CommandLineTool.Default ProcessIOData(for download) returned: %s", err.Error())
				return
			}
			count += sub_count

		}

		for i, _ := range clt.Requirements {
			requirement := &clt.Requirements[i]

			var sub_count int
			sub_count, err = ProcessIOData(requirement, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
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
				sub_count, err = ProcessIOData(input_parameter, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
				if err != nil {
					err = fmt.Errorf("(processIOData) ExpressionTool,InputParameter ProcessIOData(for download) returned: %s", err.Error())
					return
				}
				count += sub_count

			}

		}
	case *cwl.CommandInputParameter:
		// https://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputParameter
		//fmt.Println("(processIOData) *cwl.CommandInputParameter")
		cip := native.(*cwl.CommandInputParameter)

		if cip.Default != nil {

			var file *cwl.File
			var ok bool
			file, ok = cip.Default.(*cwl.File)
			if ok {
				if ioType == "upload" {
					var file_exists bool
					file_exists, err = file.Exists(currentPath)
					if err != nil {
						err = fmt.Errorf("(processIOData) cwl.CommandInputParameter file.Exists returned: %s", err.Error())
						return
					}
					if !file_exists {
						logger.Debug(3, "(processIOData) cwl.CommandInputParameter default does not exist")
						// Defaults are optional, file missing is no error
						return
					}
					logger.Debug(3, "(processIOData) cwl.CommandInputParameter default exists")
				}

				if ioType == "download" {
					if file.Location == "" {
						// Default file with no Location can be ignored
						return
					}
				}
			}

			var sub_count int
			sub_count, err = ProcessIOData(cip.Default, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				err = fmt.Errorf("(processIOData) CommandInputParameter ProcessIOData(for download) returned: %s", err.Error())
				return
			}
			count += sub_count
		}

		if cip.SecondaryFiles != nil {

			err = fmt.Errorf("(processIOData) CommandInputParameter.SecondaryFiles SecondaryFiles are not supported by AWE at this point")
			return

			cipType := cip.Type
			var cipTypeRepresentative cwl.CWLType_Type
			var ok bool

			switch cipType.(type) {
			case []interface{}:
				cipTypeArray := cipType.([]interface{})
				cipTypeRepresentativeIf := cipTypeArray[0]

				cipTypeRepresentative, ok = cipTypeRepresentativeIf.(cwl.CWLType_Type)
				if !ok {
					err = fmt.Errorf("(processIOData) could not convert into CWLType_Type")
					return
				}

			default:

				cipTypeRepresentative, ok = cipType.(cwl.CWLType_Type)
				if !ok {
					err = fmt.Errorf("(processIOData) could not convert into CWLType_Type")
					return
				}
			}

			if cipTypeRepresentative == cwl.CWLFile {
				// ok
			} else if cipTypeRepresentative == cwl.CWLArray {
				var arraySchema *cwl.ArraySchema
				var ok bool
				arraySchema, ok = cipTypeRepresentative.(*cwl.ArraySchema)
				if !ok {
					err = fmt.Errorf("(processIOData) CommandInputParameter.SecondaryFiles could not convert to ArraySchema")
					return
				}
				firstItem := arraySchema.Items[0]

				if firstItem != cwl.CWLFile {
					err = fmt.Errorf("(processIOData) CommandInputParameter.SecondaryFiles array items are expected to be File, but got %s", reflect.TypeOf(firstItem))
					return
				}

			} else {
				err = fmt.Errorf("(processIOData) CommandInputParameter.SecondaryFiles, type not supported, got %s", reflect.TypeOf(cipTypeRepresentative))
				return
			}

			secFiles := cip.SecondaryFiles
			switch secFiles.(type) {
			case string:
				secFilesStr := secFiles.(string)

				//exp := cwl.NewExpressionFromString(secFilesStr)

				// rules: https://www.commonwl.org/v1.0/CommandLineTool.html#CommandInputParameter
				for strings.HasPrefix(secFilesStr, "^") {
					extension := path.Ext(secFilesStr)

					secFilesStr = strings.TrimPrefix(secFilesStr, "^")
					if extension != "" {
						secFilesStr = strings.TrimSuffix(secFilesStr, extension)
					}
				}

			default:
				err = fmt.Errorf("(processIOData) CommandInputParameter got SecondaryFiles, but tyype not supported, type=%s", reflect.TypeOf(secFiles))
				return
			}

			return
		}

	case *cwl.InputParameter:

		ip := native.(*cwl.InputParameter)

		if ip.Default == nil {
			//fmt.Println("(processIOData) *cwl.CommandInputParameter return")
			return
		}

		var file *cwl.File
		var ok bool
		file, ok = ip.Default.(*cwl.File)
		if ok {
			if ioType == "upload" {
				var file_exists bool
				file_exists, err = file.Exists(currentPath)
				if err != nil {
					err = fmt.Errorf("(processIOData) cwl.InputParameter file.Exists returned: %s", err.Error())
					return
				}
				if !file_exists {
					// InputParameter, Default file not found (currentPath: %s)", currentPath)
					// Defaults are optional, file missing is no error
					return
				}
			}

			if ioType == "download" {
				if file.Location == "" {
					// Default file with no Location can be ignored
					return
				}
			}
		}

		var sub_count int
		sub_count, err = ProcessIOData(ip.Default, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
		if err != nil {
			err = fmt.Errorf("(processIOData) InputParameter ProcessIOData(for download) returned: %s", err.Error())
			return
		}
		count += sub_count

	case *core.CWLWorkunit:

		if ioType == "download" {
			work := native.(*core.CWLWorkunit)

			var sub_count int
			sub_count, err = ProcessIOData(work.JobInput, currentPath, basePath, "download", shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				err = fmt.Errorf("(processIOData) work.Job_input ProcessIOData(for download) returned: %s", err.Error())
				return
			}
			count += sub_count

			sub_count = 0
			sub_count, err = ProcessIOData(work.Tool, currentPath, basePath, "download", shockClient, context, lazyUpload, removeIDField)
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
				sub_count, err = ProcessIOData(entry_array[i], currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
				if err != nil {
					err = fmt.Errorf("(processIOData) work.Tool ProcessIOData(for download) returned: %s", err.Error())
					return
				}
				count += sub_count
			}

		case cwl.File:
			file := dirent.Entry.(cwl.File)
			sub_count := 0
			sub_count, err = ProcessIOData(file, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
			if err != nil {
				err = fmt.Errorf("(processIOData) work.Tool ProcessIOData(for download) returned: %s", err.Error())
				return
			}
			count += sub_count
		case *cwl.File:
			file := dirent.Entry.(*cwl.File)
			sub_count := 0
			sub_count, err = ProcessIOData(file, currentPath, basePath, ioType, shockClient, context, lazyUpload, removeIDField)
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
				case []cwl.CWLObject:
					obj_array := iwdr.Listing.([]cwl.CWLObject)
					for i, _ := range obj_array {

						sub_count := 0
						sub_count, err = ProcessIOData(obj_array[i], currentPath, basePath, "download", shockClient, context, lazyUpload, removeIDField)
						if err != nil {
							err = fmt.Errorf("(processIOData) []cwl.CWLObject cwl.Requirement/Listing returned: %s", err.Error())
							return
						}
						count += sub_count
					}
				case cwl.File:
					sub_count := 0
					sub_count, err = ProcessIOData(iwdr.Listing, currentPath, basePath, "download", shockClient, context, lazyUpload, removeIDField)
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
		nativeArray := native.([]interface{})

		if ioType == "upload" {

			for i, _ := range nativeArray {
				switch nativeArray[i].(type) {
				case cwl.File:
					// continue
				case string:

					schema_str := nativeArray[i].(string)

					this_file := cwl.NewFile()
					this_file.Path = schema_str
					//this_file.UpdateComponents(schema_str)
					nativeArray[i] = this_file
					sub_count := 0
					sub_count, err = ProcessIOData(this_file, currentPath, basePath, "download", shockClient, context, lazyUpload, removeIDField)
					if err != nil {
						err = fmt.Errorf("(processIOData) []cwl.CWLObject cwl.Requirement/Listing returned: %s", err.Error())
						return
					}
					count += sub_count

				default:
					err = fmt.Errorf("(processIOData) schemata , unkown type: (%s)", reflect.TypeOf(nativeArray[i]))
					return

				}
			}

		}
		if ioType == "download" {
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

	if work.CWLWorkunit != nil {

		shockClient := shock.NewShockClient(work.ShockHost, work.Info.DataToken, false)

		context := cwl.NewWorkflowContext()
		//context.Init()

		var count int
		count, err = ProcessIOData(work.CWLWorkunit, work_path, work_path, "download", shockClient, context, false, false)
		if err != nil {
			err = fmt.Errorf("(MoveInputData) ProcessIOData(for download) returned: %s", err.Error())
			return
		}

		logger.Debug(1, "(MoveInputData) %d files downloaded", count)

		return
	}

	for _, io := range work.Inputs {
		// skip if NoFile == true
		var io_size int64
		io_size, err = MoveInputIO(work, io, work_path)
		if err != nil {
			err = fmt.Errorf("(MoveInputData) MoveInputIO returns: %s", err.Error())
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
				err = fmt.Errorf("(UploadOutputIO) output %s not generated for workunit %s err=%s", name, work.ID, err.Error())
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
			err = fmt.Errorf("(UploadOutputIO) workunit %s generated zero-sized output %s while non-zero-sized file required", work.ID, name)
			return
		}
		size += fi.Size()

	}
	logger.Debug(1, "(UploadOutputIO) deliverer: push output to shock, filename="+name)
	logger.Event(event.FILE_OUT,
		"workid="+work.ID,
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
		"workid="+work.ID,
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

// UploadOutputData _
func UploadOutputData(work *core.Workunit, shockClient *shock.ShockClient, context *cwl.WorkflowContext) (size int64, err error) {

	if work.CWLWorkunit != nil {

		if work.CWLWorkunit.Outputs != nil {
			//fmt.Println("Outputs 1")
			//scs := spew.Config
			//scs.DisableMethods = true

			//scs.Dump(work.CWL_workunit.Outputs)
			var upload_count int
			upload_count, err = ProcessIOData(work.CWLWorkunit.Outputs, "", "", "upload", shockClient, context, false, false)
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
