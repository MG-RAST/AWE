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
	workChan     = make(chan *Workunit, 2)
	aweServerUrl = "http://localhost:8001"
	self         *Client
)

func workStealer(control chan int, self *Client) {
	fmt.Printf("workStealer lanched\n")
	defer fmt.Printf("workStealer exiting...\n")
	retry := 0
	for {
		wu, err := CheckoutWorkunitRemote(conf.SERVER_URL, self.Id)
		if err != nil {
			if err.Error() == e.WorkUnitQueueEmpty {
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
		workChan <- wu
	}
	control <- 0 //we are ending
}

func worker(control chan int) {
	fmt.Printf("worker lanched\n")
	defer fmt.Printf("worker exiting...\n")
	for {
		work := <-workChan
		if err := RunWorkunit(work); err != nil {
			fmt.Errorf("RunWorkunit() returned error: %s\n", err.Error())
			Log.Error("RunWorkunit() returned error: " + err.Error())
			continue
		}
		if err := NotifyWorkunitDone(conf.SERVER_URL, work.Id); err != nil {
			fmt.Errorf("worker: NotifyWorkunitDone returned error: %s\n", err.Error())
			Log.Error("NotifyWorkunitDone() returned error: " + err.Error())
		}
		time.Sleep(2 * time.Second) //have a rest, just for demo or testing
	}
	control <- 1 //we are ending
}

func main() {
	//launch client
	conf.PrintClientCfg()
	control := make(chan int)

	self, err := Register(conf.SERVER_URL)
	if err != nil {
		fmt.Printf("fail to register: %v\n", err)
		os.Exit(1)
	}

	Log = NewLogger("client-" + self.Id)

	fmt.Printf("Client registration done, client id=%s\n", self.Id)

	go workStealer(control, self)
	go worker(control)
	for {
		who := <-control //block till someone dies and then restart it
		if who == 0 {
			go workStealer(control, self)
		} else {
			go worker(control)
		}
	}
}
