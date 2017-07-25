package main

import (
	//"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v2"
	"os"
	"path"
	"reflect"
	"time"
)

func processInputData(native interface{}) (err error) {
	fmt.Printf("processInputData\n")
	switch native.(type) {
	case *cwl.Job_document:

		job_doc_ptr := native.(*cwl.Job_document)

		job_doc := *job_doc_ptr

		for key, value := range job_doc {

			fmt.Printf("recurse into key: %s\n", key)
			err = processInputData(value)
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
		return
	default:
		err = fmt.Errorf("(processInputData) No handler for type \"%s\"\n", reflect.TypeOf(native))
		return
	}
	panic("argh")
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

	fmt.Println("Job input:")
	spew.Dump(*job_doc)

	data, err := yaml.Marshal(*job_doc)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Printf("json:\n%s\n", string(data[:]))

	// process input files

	err = processInputData(job_doc)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}

}
