package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/worker"
	"github.com/davecgh/go-spew/spew"
	"os"
	"strings"
	"time"
)

func main() {

	// workstealer -> dataMover (download from Shock) -> processor -> deliverer (upload to Shock)

	err := conf.Init_conf("client")

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: error reading conf file: "+err.Error())
		os.Exit(1)
	}

	//if !conf.INIT_SUCCESS {
	//	conf.PrintClientUsage()
	//	os.Exit(1)
	//}

	worker.Client_mode = "online"
	if conf.CWL_TOOL != "" || conf.CWL_JOB != "" {
		worker.Client_mode = "offline"
		conf.LOG_OUTPUT = "console"
	}

	if _, err = os.Stat(conf.WORK_PATH); err != nil && os.IsNotExist(err) {
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

	if _, err := os.Stat(conf.LOGS_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.LOGS_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating log_path \"%s\" : %s\n", conf.LOGS_PATH, err.Error())
			os.Exit(1)
		}
	}

	if conf.PID_FILE_PATH != "" {
		f, err := os.Create(conf.PID_FILE_PATH)
		if err != nil {
			err_msg := "Could not create pid file: " + conf.PID_FILE_PATH + "\n"
			fmt.Fprintf(os.Stderr, err_msg)
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

	var self *core.Client
	if worker.Client_mode == "online" {
		if conf.SERVER_URL == "" {
			fmt.Fprintf(os.Stderr, "AWE server url not configured or is empty. Please check the [Client]serverurl field in the configuration file.\n")
			os.Exit(1)
		}
		if strings.HasPrefix(conf.SERVER_URL, "http") == false {
			fmt.Fprintf(os.Stderr, "serverurl not valid (require http://): %s \n", conf.SERVER_URL)
			os.Exit(1)
		}

		self, err = worker.RegisterWithAuth(conf.SERVER_URL, profile)
		if err != nil {
			fmt.Fprintf(os.Stderr, "fail to register: %s\n", err.Error())
			logger.Error("fail to register: %s\n", err.Error())
			os.Exit(1)
		}

	} else {
		self = core.NewClient()
	}

	core.SetClientProfile(self)
	if worker.Client_mode == "online" {
		fmt.Printf("Client registered, name=%s, id=%s\n", self.Name, self.Id)
		logger.Event(event.CLIENT_REGISTRATION, "clientid="+self.Id)
	}

	worker.InitWorkers()

	if worker.Client_mode == "offline" {
		if conf.CWL_JOB == "" {
			logger.Error("cwl job file missing")
			time.Sleep(time.Second)
			os.Exit(1)
		}
		job_doc, err := cwl.ParseJob(conf.CWL_JOB)
		if err != nil {
			logger.Error("error parsing cwl job: %v", err)
			time.Sleep(time.Second)
			os.Exit(1)
		}

		spew.Dump(*job_doc)

		for key, value := range *job_doc {
			fmt.Println(key)

			switch value.(type) {
			case *cwl.File:
				file, ok := value.(*cwl.File)
				if !ok {
					panic("not file")
				}
				fmt.Printf("%+v\n", *file)

			default:
				spew.Dump(value)
			}
		}
		mediumwork := worker.Mediumwork{}
		mediumwork.CWL_job = job_doc

		mediumwork.CWL_tool_filename = conf.CWL_TOOL
		mediumwork.CWL_tool = &cwl.CommandLineTool{} // TODO parsing and testing ?

		mediumwork.Workunit = &core.Workunit{Id: "00000000-0000-0000-0000-000000000000_0_0"}
		logger.Debug(1, "injecting cwl job into worker...")
		go func() {
			worker.FromStealer <- &mediumwork
		}()

	}

	time.Sleep(time.Second)

	worker.StartClientWorkers()

}
