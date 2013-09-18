package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/worker"
	"os"
)

func main() {

	if !conf.INIT_SUCCESS {
		conf.PrintClientUsage()
		os.Exit(1)
	}

	if _, err := os.Stat(conf.WORK_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.WORK_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating work_path %s\n", err.Error())
			os.Exit(1)
		}
	}

	if _, err := os.Stat(conf.DATA_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.DATA_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating data_path %s\n", err.Error())
			os.Exit(1)
		}
	}

	if _, err := os.Stat(conf.LOGS_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.LOGS_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating log_path %s\n", err.Error())
			os.Exit(1)
		}
	}

	profile, err := worker.ComposeProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to compose profile: %s\n", err.Error())
		os.Exit(1)
	}

	self, err := worker.RegisterWithAuth(conf.SERVER_URL, profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to register: %s\n", err.Error())
		os.Exit(1)
	}
	core.InitClientProfile(self)

	var logdir string
	if self.Name != "" {
		logdir = self.Name
	} else {
		logdir = conf.CLIENT_NAME
	}

	logger.Initialize("client-" + logdir)

	fmt.Printf("Client registered, name=%s, id=%s\n", self.Name, self.Id)
	logger.Event(event.CLIENT_REGISTRATION, "clientid="+self.Id)

	if err := worker.InitWorkers(self); err == nil {
		worker.StartClientWorkers()
	} else {
		fmt.Printf("failed to initialize and start workers:" + err.Error())
	}
}
