package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	"time"
)

var (
	workChan     = make(chan *Workunit, conf.TOTAL_WORKER)
	aweServerUrl = "http://localhost:8001/work"
)

func workStealer(control chan int) {
	fmt.Printf("workStealer lanched\n")
	defer fmt.Printf("workStealer exiting...\n")
	for {
		wu, err := CheckoutWorkunitRemote(aweServerUrl)
		if err != nil {
			if err.Error() == "empty workunit queue" {
				fmt.Printf("queue empty, try again 5 seconds later\n")
				time.Sleep(5 * time.Second)
			} else {
				fmt.Printf("error in checkoutWorkunitRemote %v\n", err)
			}
			continue
		}
		fmt.Printf("checked out a workunit: id=%s\n", wu.Id)
		workChan <- wu
	}
	control <- 1 //we are ending
}

func workProcessor(control chan int, num int) {
	fmt.Printf("workProcessor %d lanched\n", num)
	defer fmt.Printf("workProcessor exiting...\n")
	for {
		work := <-workChan
		fmt.Printf("work=%v\n", *work)
		if err := RunWorkunit(work, num); err != nil {
			fmt.Printf("RunWorkunit returned error: %v\n", err)
			continue
		}
		if err := NotifyWorkunitDone(aweServerUrl, work.Id); err != nil {
			fmt.Printf("NotifyWorkunitDone returned erro: %v\n", err)
		}
	}
	control <- num //we are ending
}

func main() {
	//launch client
	conf.PrintClientCfg()
	fmt.Printf("total worker=%d\n", conf.TOTAL_WORKER)
	control := make(chan int)
	go workStealer(control)
	for i := 0; i < conf.TOTAL_WORKER; i++ {
		go workProcessor(control, i)
	}
	for {
		who := <-control //block till something dies and then restart it
		go workProcessor(control, who)
	}
}
