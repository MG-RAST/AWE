package main

import (
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/worker"
)

func main() {

	// workstealer -> dataMover (download from Shock) -> processor -> deliverer (upload to Shock)

	err := conf.Init_conf("worker")
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: error reading conf file: "+err.Error())

		os.Exit(1)
	}

	worker.Client_mode = "online"
	if conf.CWL_TOOL != "" || conf.CWL_JOB != "" {
		worker.Client_mode = "offline"
		conf.LOG_OUTPUT = "console"
	}

	if _, err := os.Stat(conf.WORK_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.WORK_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating work_path \"%s\" : %s\n", conf.WORK_PATH, err.Error())
			os.Exit(1)
		}
	}

	if _, err := os.Stat(conf.DATA_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.DATA_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating data_path \"%s\" : %s\n", conf.DATA_PATH, err.Error())
			os.Exit(1)
		}
	}
	if conf.PREDATA_PATH != conf.DATA_PATH {
		if _, err := os.Stat(conf.PREDATA_PATH); err != nil && os.IsNotExist(err) {
			if err := os.MkdirAll(conf.PREDATA_PATH, 0777); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR in creating predata_path \"%s\" : %s\n", conf.PREDATA_PATH, err.Error())
				os.Exit(1)
			}
		}
	}

	if _, err := os.Stat(conf.LOGS_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.LOGS_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating log_path \"%s\" : %s\n", conf.LOGS_PATH, err.Error())
			os.Exit(1)
		}
	}

	if conf.PID_FILE_PATH != "" {
		f, err := os.Create(conf.PID_FILE_PATH)
		if err != nil {
			errMsg := "Could not create pid file: " + conf.PID_FILE_PATH + "\n"
			fmt.Fprintf(os.Stderr, errMsg)
			os.Exit(1)
		}
		defer f.Close()
		pid := os.Getpid()
		fmt.Fprintln(f, pid)
		fmt.Println("##### pidfile #####")
		fmt.Printf("pid: %d saved to file: %s\n\n", pid, conf.PID_FILE_PATH)
	}

	logger.Initialize("client")

	logger.Debug(1, "PATH="+os.Getenv("PATH"))
	logger.Debug(3, "worker.Client_mode="+worker.Client_mode)

	profile, err := worker.ComposeProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to compose profile: %s\n", err.Error())
		os.Exit(1)
	}

	core.SetClientProfile(profile)
	self := core.Self
	//var self *core.Client
	if worker.Client_mode == "online" {
		if conf.SERVER_URL == "" {
			fmt.Fprintf(os.Stderr, "(worker.main) AWE server url not configured or is empty. Please check the [Client]serverurl field in the configuration file.\n")
			os.Exit(1)
		}
		if strings.HasPrefix(conf.SERVER_URL, "http") == false {
			fmt.Fprintf(os.Stderr, "(worker.main) serverurl not valid (require http://): %s \n", conf.SERVER_URL)
			os.Exit(1)
		}

		count := 0

		retrySleep := 5

		for true {
			count++
			err = worker.RegisterWithAuth(conf.SERVER_URL, profile)
			if err != nil {

				if count >= 10 {
					message := fmt.Sprintf("(worker.main) failed to register: %s (SERVER_URL=%s) exiting!\n", err.Error(), conf.SERVER_URL)
					fmt.Fprintf(os.Stderr, message)
					logger.Error(message)

					os.Exit(1)
				}

				message := fmt.Sprintf("(worker.main) failed to register: %s (SERVER_URL=%s) (%d), next retry in %d seconds...\n", err.Error(), conf.SERVER_URL, count, retrySleep)
				fmt.Fprintf(os.Stderr, message)
				logger.Error(message)

				time.Sleep(time.Second * time.Duration(retrySleep))
			} else {
				break
			}
		}

	}

	if worker.Client_mode == "online" {
		fmt.Printf("Client registered, name=%s, id=%s\n", self.WorkerRuntime.Name, self.ID)
		logger.Event(event.CLIENT_REGISTRATION, "clientid="+self.ID)
	}

	worker.InitWorkers()

	if worker.Client_mode == "offline" {
		if conf.CWL_JOB == "" {
			logger.Error("cwl job file missing")
			time.Sleep(time.Second)
			os.Exit(1)
		}
		jobDoc, err := cwl.ParseJobFile(conf.CWL_JOB)
		if err != nil {
			logger.Error("error parsing cwl job: %v", err)
			time.Sleep(time.Second)
			os.Exit(1)
		}

		//fmt.Println("Job input:")
		//spew.Dump(*job_doc)

		os.Getwd() //https://golang.org/pkg/os/#Getwd

		workunit := &core.Workunit{ID: "00000000-0000-0000-0000-000000000000_0_0", CWLWorkunit: core.NewCWLWorkunit()}

		workunit.CWLWorkunit.JobInput = jobDoc
		workunit.CWLWorkunit.JobInputFilename = conf.CWL_JOB

		workunit.CWLWorkunit.ToolFilename = conf.CWL_TOOL
		workunit.CWLWorkunit.Tool = &cwl.CommandLineTool{} // TODO parsing and testing ?

		currentWorkingDirectory, err := os.Getwd()
		if err != nil {
			logger.Error("cannot get currentWorkingDirectory")
			time.Sleep(time.Second)
			os.Exit(1)
		}
		workunit.WorkPath = currentWorkingDirectory

		cmd := &core.Command{}
		cmd.Local = true // this makes sure the working directory is not deleted
		cmd.Name = "cwl-runner"

		//"--provenance", "cwl_tool_provenance",
		cmd.ArgsArray = []string{"--leave-outputs", "--leave-tmpdir", "--tmp-outdir-prefix", "./tmp/", "--tmpdir-prefix", "./tmp/", "--rm-container", "--on-error", "stop", workunit.CWLWorkunit.ToolFilename, workunit.CWLWorkunit.JobInputFilename}
		if conf.CWL_RUNNER_ARGS != "" {
			cwlRunnerArgsArray := strings.Split(conf.CWL_RUNNER_ARGS, " ")
			cmd.ArgsArray = append(cwlRunnerArgsArray, cmd.ArgsArray...)
		}

		workunit.Cmd = cmd

		workunit.WorkPerf = core.NewWorkPerf()
		workunit.WorkPerf.Checkout = time.Now().Unix()

		logger.Debug(1, "injecting cwl job into worker...")
		go func() {
			worker.FromStealer <- workunit
		}()

	}

	time.Sleep(time.Second)

	worker.StartClientWorkers()

}
