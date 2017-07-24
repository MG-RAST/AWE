package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"os"
	"path"
	"time"
)

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

}
