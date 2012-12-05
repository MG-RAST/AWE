package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	e "github.com/MG-RAST/AWE/errors"
	"os"
	"time"
)

var (
	workChan     = make(chan *Workunit, 2)
	aweServerUrl = "http://localhost:8001"
)

func workStealer(control chan int) {
	fmt.Printf("workStealer lanched\n")
	defer fmt.Printf("workStealer exiting...\n")
	retry := 0
	for {
		wu, err := CheckoutWorkunitRemote(conf.SERVER_URL)
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
			fmt.Printf("worker: RunWorkunit() %s returned error: %v\n", work.Id, err)
			continue
		}
		if err := NotifyWorkunitDone(conf.SERVER_URL, work.Id); err != nil {
			fmt.Printf("worker: NotifyWorkunitDone returned error: %v\n", err)
		}
		time.Sleep(2 * time.Second) //have a rest, just for demo or testing
	}
	control <- 1 //we are ending
}

func main() {
	//launch client
	conf.PrintClientCfg()
	control := make(chan int)
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
