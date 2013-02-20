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
				//normal, do nothing
			} else if err.Error() == e.ClientNotFound { //server may be restarted, re-register
				fmt.Printf("lost contact with server, try to re-register\n")
				if _, err := ReRegisterWithSelf(conf.SERVER_URL); err != nil {
					retry += 1
				}
			} else { //something is wrong, server may be down
				fmt.Printf("error in checking out workunits: %v\n", err)
				retry += 1
			}
			if retry == 5 {
				os.Exit(1)
			}
			time.Sleep(5 * time.Second)
			continue
		} else {
			retry = 0
		}
		fmt.Printf("workStealer: checked out a workunit: id=%s\n", wu.Id)
		//log event about work checktout (WC)
		Log.Event(EVENT_WORK_CHECKOUT, "workid="+wu.Id)
		self.Total_checkout += 1
		self.Current_work[wu.Id] = true
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

			//restart once and if it still fails
			if err := RunWorkunit(work); err != nil {
				fmt.Printf("!!!ReRunWorkunit() returned error: %s\n", err.Error())
				Log.Error("ReRunWorkunit(): workid=" + work.Id + ", " + err.Error())

				//send back the workunit to server
				if err := NotifyWorkunitProcessed(conf.SERVER_URL, work.Id, "fail"); err != nil {
					fmt.Printf("!!!NotifyWorkunitFail returned error: %s\n", err.Error())
					Log.Error("NotifyWorkunitFail: workid=" + work.Id + ", err=" + err.Error())
				}
				Log.Event(EVENT_WORK_RETURN, "workid="+work.Id)
				self.Total_failed += 1
				delete(self.Current_work, work.Id)
				continue
			}
		}
		if err := NotifyWorkunitProcessed(conf.SERVER_URL, work.Id, "done"); err != nil {
			fmt.Printf("!!!NotifyWorkunitDone returned error: %s\n", err.Error())
			Log.Error("NotifyWorkunitDone: workid=" + work.Id + ", err=" + err.Error())
		}
		Log.Event(EVENT_WORK_DONE, "workid="+work.Id)
		self.Total_completed += 1
		delete(self.Current_work, work.Id)
	}
	control <- 1 //we are ending
}

func heartBeater(control chan int) {
	for {
		time.Sleep(10 * time.Second)
		SendHeartBeat(conf.SERVER_URL, self.Id)
	}
	control <- 2 //we are ending
}

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
