package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/logger"
	"os"
)

var (
	workChan = make(chan *Workunit)
	self     = &Client{Id: "default-client"}
)

func main() {

	if !conf.INIT_SUCCESS {
		conf.PrintClientUsage()
		os.Exit(1)
	}

	//launch client
	if _, err := os.Stat(conf.WORK_PATH); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(conf.WORK_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating work_path %v\n", err)
		}
		os.Exit(1)
	}

	var err error
	self, err = RegisterWithProfile(conf.SERVER_URL, conf.CLIENT_PROFILE)
	if err != nil {
		fmt.Printf("fail to register: %v\n", err)
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
	go worker(control)
	for {
		who := <-control //block till someone dies and then restart it
		if who == 0 {
			go workStealer(control)
			Log.Error("workStealer died and restarted")
		} else if who == 1 {
			go worker(control)
			Log.Error("worker died and restarted")
		} else if who == 2 {
			go heartBeater(control)
			Log.Error("heartbeater died and restarted")
		}
	}
}
