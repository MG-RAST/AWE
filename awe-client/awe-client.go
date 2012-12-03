package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	"time"
)

var (
	workChan     = make(chan *Workunit, conf.TOTAL_WORKER)
	aweServerUrl = "http://localhost:8001"
)

func worker(control chan int, num int) {
	fmt.Printf("worker %d launched\n", num)
	defer fmt.Printf("worker exiting...\n")

	for {
		work, err := CheckoutWorkunitRemote(aweServerUrl)
		if err != nil {
			if err.Error() == "empty workunit queue" {
				//fmt.Printf("queue empty, try again 5 seconds later\n")
				time.Sleep(5 * time.Second)
			} else {
				fmt.Printf("error in checkoutWorkunitRemote %v\n", err)
			}
			continue
		}

		fmt.Printf("worker %d checked out a workunit: id=%s\n", num, work.Id)

		if err := RunWorkunit(work, num); err != nil {
			fmt.Printf("RunWorkunit returned error: %v\n", err)
			continue
		}
		if err := NotifyWorkunitDone(aweServerUrl, work.Id); err != nil {
			fmt.Printf("NotifyWorkunitDone returned error: %v\n", err)
		}
	}
}

func main() {
	//launch client
	conf.PrintClientCfg()
	fmt.Printf("total worker=%d\n", conf.TOTAL_WORKER)
	control := make(chan int)
	for i := 0; i < conf.TOTAL_WORKER; i++ {
		go worker(control, i)
	}
	for {
		who := <-control //block till some work dies and then restart it
		go worker(control, who)
	}
}
