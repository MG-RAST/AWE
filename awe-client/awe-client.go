package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/worker"
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
			logger.Error("ERROR: " + err_msg)
			os.Exit(1)
		}
		defer f.Close()
		pid := os.Getpid()
		fmt.Fprintln(f, pid)
		fmt.Println("##### pidfile #####")
		fmt.Printf("pid: %d saved to file: %s\n\n", pid, conf.PID_FILE_PATH)
	}

	logger.Initialize("client")

	logger.Debug(0, "PATH="+os.Getenv("PATH"))

	profile, err := worker.ComposeProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to compose profile: %s\n", err.Error())
		os.Exit(1)
	}

	if conf.SERVER_URL == "" {
		fmt.Fprintf(os.Stderr, "AWE server url not configured or is empty. Please check the [Client]serverurl field in the configuration file.\n")
		os.Exit(1)
	}
	if strings.HasPrefix(conf.SERVER_URL, "http") == false {
		fmt.Fprintf(os.Stderr, "serverurl not valid (require http://): %s \n", conf.SERVER_URL)
		os.Exit(1)
	}

	self, err := worker.RegisterWithAuth(conf.SERVER_URL, profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to register: %s\n", err.Error())
		logger.Error(fmt.Sprintf("fail to register: %s\n", err.Error()))
		os.Exit(1)
	}
	core.InitClientProfile(self)

	fmt.Printf("Client registered, name=%s, id=%s\n", self.Name, self.Id)
	logger.Event(event.CLIENT_REGISTRATION, "clientid="+self.Id)

	if err := worker.InitWorkers(self); err == nil {
		worker.StartClientWorkers()
	} else {
		fmt.Printf("failed to initialize and start workers:" + err.Error())
	}
}
