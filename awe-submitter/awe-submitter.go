package main

import (
	//"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v2"
	"mime/multipart"
	//"net/http"
	"bytes"
	"io"
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

	file.Location_url, err = url.Parse(conf.SHOCK_URL + "/node/" + node.Id + "?download")
	if err != nil {
		return
	}

	file.Location = file.Location_url.String()
	file.Path = ""

	return
}

func processInputData(native interface{}, inputfile_path string) (count int, err error) {

	fmt.Printf("(processInputData) start\n")
	defer fmt.Printf("(processInputData) end\n")
	switch native.(type) {
	case *cwl.Job_document:
		fmt.Printf("found Job_document\n")
		job_doc_ptr := native.(*cwl.Job_document)

		job_doc := *job_doc_ptr

		for key, value := range job_doc {

			fmt.Printf("recurse into key: %s\n", key)
			var sub_count int
			sub_count, err = processInputData(value, inputfile_path)
			if err != nil {
				return
			}
			count += sub_count
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
		count += 1
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

	for _, value := range conf.ARGS {
		println(value)
	}
	panic("done")

	if conf.CWL_JOB == "" {
		logger.Error("cwl job file missing")
		time.Sleep(time.Second)
		os.Exit(1)
	}

	inputfile_path := path.Dir(conf.CWL_JOB)
	fmt.Printf("job path: %s\n", inputfile_path) // needed to resolve relative paths

	job_doc, err := cwl.ParseJobFile(conf.CWL_JOB)
	if err != nil {
		logger.Error("error parsing cwl job: %v", err)
		time.Sleep(time.Second)
		os.Exit(1)
	}

	fmt.Println("Job input after reading from file:")
	spew.Dump(*job_doc)

	data, err := yaml.Marshal(*job_doc)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}

	job_doc_string := string(data[:])
	fmt.Printf("job_doc_string: \"%s\"\n", job_doc_string)
	if job_doc_string == "" {
		fmt.Println("job_doc_string is empty")
		os.Exit(1)
	}

	fmt.Printf("yaml:\n%s\n", job_doc_string)

	// process input files

	upload_count, err := processInputData(job_doc, inputfile_path)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}
	fmt.Printf("%d files have been uploaded\n", upload_count)
	time.Sleep(2)

	fmt.Println("------------Job input after parsing:")
	data, err = yaml.Marshal(*job_doc)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Printf("yaml:\n%s\n", string(data[:]))

	// job submission example:
	// curl -X POST -F job=@test.yaml -F cwl=@/Users/wolfganggerlach/awe_data/pipeline/CWL/PackedWorkflow/preprocess-fasta.workflow.cwl http://localhost:8001/job

	var b bytes.Buffer
	w := multipart.NewWriter(&b)

	multipartWriter_AddFile(w, "cwl", "/Users/wolfganggerlach/awe_data/pipeline/CWL/PackedWorkflow/preprocess-fasta.workflow.cwl")

}

func multipartWriter_AddFile(w *multipart.Writer, fieldname string, filepath string) (err error) {

	f, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer f.Close()
	fw, err := w.CreateFormFile(fieldname, filepath)
	if err != nil {
		return
	}
	if _, err = io.Copy(fw, f); err != nil {
		return
	}
	return
}
