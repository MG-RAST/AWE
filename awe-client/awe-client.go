package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/logger"
	"os"
)

var (
	chanRaw       = make(chan *rawWork)       // workStealer -> dataMover
	chanParsed    = make(chan *parsedWork)    // dataMover -> worker
	chanProcessed = make(chan *processedWork) //worker -> deliverer
	chanPerf      = make(chan *WorkPerf)      // -> perfmon
	chanPermit    = make(chan bool)
	self          = &Client{Id: "default-client"}
	chankill      = make(chan bool)            //heartbeater -> worker
)

type rawWork struct {
	workunit *Workunit
	perfstat *WorkPerf
}

type parsedWork struct {
	workunit *Workunit
	perfstat *WorkPerf
	args     []string
	status   string
}

type processedWork struct {
	workunit *Workunit
	perfstat *WorkPerf
	status   string
}

const (
	ID_HEARTBEATER = 0
	ID_WORKSTEALER = 1
	ID_DATAMOVER   = 2
	ID_WORKER      = 3
	ID_DELIVERER   = 4
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

	var err error
	var profile *Client
	profile, err = ComposeProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to compose profile: %s\n", err.Error())
		os.Exit(1)
	}
	self, err = RegisterWithProfile(conf.SERVER_URL, profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to register: %s\n", err.Error())
		os.Exit(1)
	}

	var logdir string
	if self.Name != "" {
		logdir = self.Name
	} else {
		logdir = conf.CLIENT_NAME
	}

	Log = NewLogger("client-" + logdir)
	go Log.Handle()

	fmt.Printf("Client registered, name=%s, id=%s\n", self.Name, self.Id)
	Log.Event(EVENT_CLIENT_REGISTRATION, "clientid="+self.Id)

	control := make(chan int)
	go heartBeater(control)
	go workStealer(control)
	go dataMover(control)
	go worker(control)
	go deliverer(control)
	for {
		who := <-control //block till someone dies and then restart it
		switch who {
		case ID_HEARTBEATER:
			go heartBeater(control)
			Log.Error("heartBeater died and restarted")
		case ID_WORKSTEALER:
			go workStealer(control)
			Log.Error("workStealer died and restarted")
		case ID_DATAMOVER:
			go dataMover(control)
			Log.Error("dataMover died and restarted")
		case ID_WORKER:
			go worker(control)
			Log.Error("worker died and restarted")
		case ID_DELIVERER:
			go deliverer(control)
			Log.Error("deliverer died and restarted")
		}
	}
}
