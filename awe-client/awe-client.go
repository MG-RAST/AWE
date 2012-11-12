package main

import (
	"fmt"
	. "github.com/MG-RAST/AWE/core"
	"runtime"
	"time"
)

var (
	TotalWorker = runtime.NumCPU()
	workChan    = make(chan *Workunit, TotalWorker)
)

func workStealer(control chan int) {
	fmt.Printf("workStealer lanched\n")
	defer fmt.Printf("workStealer exiting...\n")
	for {
		if wu, err := CheckoutWorkunit(); err == nil {
			fmt.Printf("checked out a workunit: id=%s\n", wu.Id)
			workChan <- wu
		}
	}
	control <- 1 //we are ending
}

func workProcessor(control chan int, num int) {
	fmt.Printf("workProcessor %d lanched\n", num)
	defer fmt.Printf("workProcessor exiting...\n")
	for {
		work := <-workChan
		run_work(work, num)
	}

	control <- 1 //we are ending
}

func run_work(work *Workunit, num int) {
	fmt.Printf("processor %d started run workunit id=%s\n", num, work.Id)
	defer fmt.Printf("processor %d finished run workunit id=%s\n", num, work.Id)

	time.Sleep(time.Duration(num+5) * time.Second)
}

func main() {
	//launch client
	fmt.Printf("total worker=%d\n", TotalWorker)
	control := make(chan int)
	go workStealer(control)
	for i := 0; i < TotalWorker; i++ {
		go workProcessor(control, i)
	}
	<-control //block till something dies
}
