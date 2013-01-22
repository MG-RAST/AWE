package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	e "github.com/MG-RAST/AWE/errors"
	. "github.com/MG-RAST/AWE/logger"
	"os"
	"time"
)

var (
	workChan     = make(chan *Workunit, 1)
	aweServerUrl = "http://localhost:8001"
	self         = &Client{Id: "default-client"}
)

func workStealer(control chan int) {
	fmt.Printf("workStealer lanched, client=%s\n", self.Id)
	defer fmt.Printf("workStealer exiting...\n")
	retry := 0
	for {
		wu, err := CheckoutWorkunitRemote(conf.SERVER_URL)
		if err != nil {
			if err.Error() == e.QueueEmpty || err.Error() == e.NoEligibleWorkunitFound {
				time.Sleep(5 * time.Second)
			} else {
				fmt.Printf("error in checking out workunits: %v\n", err)
				retry += 1
				if retry == 3 {
					os.Exit(1)
				}
				time.Sleep(3 * time.Second)
			}
			continue
		}
		fmt.Printf("workStealer: checked out a workunit: id=%s\n", wu.Id)
		//log event about work checktout (WC)
		Log.Event(EVENT_WORK_CHECKOUT, "workid="+wu.Id)

		workChan <- wu
	}
	control <- 0 //we are ending
}

func worker(control chan int) {
	fmt.Printf("worker lanched, client=%s\n", self.Id)
	defer fmt.Printf("worker exiting...\n")
	for {
		work := <-workChan
		if err := RunWorkunit(work); err != nil {
			fmt.Printf("!!!RunWorkunit() returned error: %s\n", err.Error())
			Log.Error("RunWorkunit(): workid=" + work.Id + ", " + err.Error())

			//restart once
			if err := RunWorkunit(work); err != nil {
				fmt.Printf("!!!ReRunWorkunit() returned error: %s\n", err.Error())
				Log.Error("ReRunWorkunit(): workid=" + work.Id + ", " + err.Error())

				//send back the workunit to server
				if err := NotifyWorkunitProcessed(conf.SERVER_URL, work.Id, "fail"); err != nil {
					fmt.Printf("!!!NotifyWorkunitFail returned error: %s\n", err.Error())
					Log.Error("NotifyWorkunitFail: workid=" + work.Id + ", err=" + err.Error())
				}
				Log.Event(EVENT_WORK_RETURN, "workid="+work.Id)
				continue
			}
		}
		if err := NotifyWorkunitProcessed(conf.SERVER_URL, work.Id, "done"); err != nil {
			fmt.Printf("!!!NotifyWorkunitDone returned error: %s\n", err.Error())
			Log.Error("NotifyWorkunitDone: workid=" + work.Id + ", err=" + err.Error())
		}
		Log.Event(EVENT_WORK_DONE, "workid="+work.Id)
	}
	control <- 1 //we are ending
}

func heartBeater(control chan int) {
	for {
		time.Sleep(10 * time.Second)
		SendHeartBeat(conf.SERVER_URL, self.Id)
	}
}

func main() {

	conf.PrintClientCfg()

	//launch client
	if _, err := os.Stat(conf.WORK_PATH); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(conf.WORK_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating work_path %v\n", err)
		}
		os.Exit(1)
	}

	var err error
	self, err = RegisterWithProfile(conf.SERVER_URL)
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

	fmt.Printf("Client registered, client id=%s\n", self.Id)
	Log.Event(EVENT_CLIENT_REGISTRATION, "clientid="+self.Id)

	control := make(chan int)
	go heartBeater(control)
	go workStealer(control)
	go worker(control)
	for {
		who := <-control //block till someone dies and then restart it
		if who == 0 {
			go workStealer(control)
		} else {
			go worker(control)
		}
	}
}
