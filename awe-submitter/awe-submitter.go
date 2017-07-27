package main

import (
	//"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v2"
	"net/url"
	"os"
	"path"
	"reflect"
	"time"
)

func uploadFile(file *cwl_types.File, inputfile_path string) (err error) {
	fmt.Printf("(uploadFile) start\n")
	defer fmt.Printf("(uploadFile) end\n")
	//if err := core.PutFileToShock(file_path, io.Host, io.Node, work.Rank, work.Info.DataToken, attrfile_path, io.Type, io.FormOptions, io.NodeAttr); err != nil {

	//	time.Sleep(3 * time.Second) //wait for 3 seconds and try again
	//	if err := core.PutFileToShock(file_path, io.Host, io.Node, work.Rank, work.Info.DataToken, attrfile_path, io.Type, io.FormOptions, io.NodeAttr); err != nil {
	//		fmt.Errorf("push file error\n")
	//		logger.Error("op=pushfile,err=" + err.Error())
	//		return size, err
	//	}
	//}

	scheme := ""
	if file.Location_url != nil {
		scheme = file.Location_url.Scheme
		host := file.Location_url.Host
		path := file.Location_url.Path
		fmt.Printf("scheme: %s", scheme)

		if scheme == "file" {
			if host == "" || host == "localhost" {
				file.Path = path
			}
		} else {
			return
		}

	}

	if file.Location_url == nil && file.Location != "" {
		err = fmt.Errorf("URL has not been parsed correctly")
		return
	}

	file_path := file.Path

	if file_path == "" {
		return
	}

	fmt.Printf("file.Path: %s\n", file_path)

	if !path.IsAbs(file_path) {
		file_path = path.Join(inputfile_path, file_path)
	}

	fmt.Printf("Using path %s\n", file_path)

	sc := shock.ShockClient{Host: conf.SHOCK_URL, Token: "", Debug: true} // "shock:7445"

	opts := shock.Opts{"upload_type": "basic", "file": file_path}
	node, err := sc.CreateOrUpdate(opts, "", nil)
	if err != nil {
		return
	}
	spew.Dump(node)

	file.Location_url, err = url.Parse(conf.SERVER_URL + "/node/" + node.Id + "?download")
	if err != nil {
		return
	}

	file.Location = file.Location_url.String()
	file.Path = ""

	return
}

func processInputData(native interface{}, inputfile_path string) (err error) {
	fmt.Printf("(processInputData) start\n")
	defer fmt.Printf("(processInputData) end\n")
	switch native.(type) {
	case *cwl.Job_document:

		job_doc_ptr := native.(*cwl.Job_document)

		job_doc := *job_doc_ptr

		for key, value := range job_doc {

			fmt.Printf("recurse into key: %s\n", key)
			err = processInputData(value, inputfile_path)
			if err != nil {
				return
			}

		}

		return

	case *cwl_types.String:
		fmt.Printf("found string\n")
		return
	case *cwl_types.File:

		fmt.Printf("found File\n")
		file, ok := native.(*cwl_types.File)
		if !ok {
			err = fmt.Errorf("could not cast to *cwl_types.File")
			return
		}
		err = uploadFile(file, inputfile_path)
		if err != nil {
			return
		}

		return
	default:
		spew.Dump(native)
		err = fmt.Errorf("(processInputData) No handler for type \"%s\"\n", reflect.TypeOf(native))
		return
	}

	return
}

func main() {

	conf.LOG_OUTPUT = "console"

	err := conf.Init_conf("submitter")

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: error reading conf file: "+err.Error())
		os.Exit(1)
	}

	logger.Initialize("client")

	if conf.CWL_JOB == "" {
		logger.Error("cwl job file missing")
		time.Sleep(time.Second)
		os.Exit(1)
	}

	inputfile_path := path.Dir(conf.CWL_JOB)
	fmt.Printf("job path: %s\n", inputfile_path) // needed to resolve relative paths

	job_doc, err := cwl.ParseJob(conf.CWL_JOB)
	if err != nil {
		logger.Error("error parsing cwl job: %v", err)
		time.Sleep(time.Second)
		os.Exit(1)
	}

	fmt.Println("Job input fter reading from file:")
	spew.Dump(*job_doc)

	data, err := yaml.Marshal(*job_doc)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Printf("json:\n%s\n", string(data[:]))

	// process input files

	err = processInputData(job_doc, inputfile_path)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}
	time.Sleep(2)

	fmt.Println("------------Job input after parsing:")
	data, err = yaml.Marshal(*job_doc)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Printf("json:\n%s\n", string(data[:]))

}
