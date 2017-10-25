package worker

import (
	//"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/core"
	//"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
)

var (
	FromStealer   chan *core.Workunit // workStealer -> dataMover
	fromMover     chan *core.Workunit // dataMover -> processor
	fromProcessor chan *core.Workunit // processor -> deliverer
	chanPermit    chan bool
	chankill      chan bool //heartbeater -> worker
	workmap       *WorkMap
	//workmap       map[string]int //workunit map [work_id]stage_id}
	Client_mode string
)

type Mediumwork struct {
	Workunit *core.Workunit
	//Perfstat          *core.WorkPerf

}

const (
	ID_HEARTBEATER    = 0
	ID_WORKSTEALER    = 1
	ID_DATADOWNLOADER = 2
	ID_WORKER         = 3
	ID_DELIVERER      = 4
	ID_REDISTRIBUTOR  = 5
	ID_DISCARDED      = 6 // flag acts as a message
)

func InitWorkers() {
	fmt.Printf("InitWorkers()\n")

	FromStealer = make(chan *core.Workunit)   // workStealer -> dataMover
	fromMover = make(chan *core.Workunit)     // dataMover -> processor
	fromProcessor = make(chan *core.Workunit) // processor -> deliverer
	chankill = make(chan bool)                //heartbeater -> processor
	chanPermit = make(chan bool)
	//workmap = map[string]int{} //workunit map [work_id]stage_idgit
	workmap = NewWorkMap()
	return
}

func StartClientWorkers() {
	control := make(chan int)
	fmt.Printf("start ClientWorkers, client=%s\n", core.Self.Id)

	mode := Client_mode
	if mode == "online" {
		go heartBeater(control)
		go workStealer(control)
	}
	go dataDownloader(control)
	go processor(control)
	go deliverer(control)

	for {
		who := <-control //block till someone dies and then restart it

		if mode == "offline" {
			fmt.Println("Done.")
			return
		}

		switch who {
		case ID_HEARTBEATER:
			go heartBeater(control)
			logger.Error("heartBeater died and restarted")
		case ID_WORKSTEALER:
			go workStealer(control)
			logger.Error("workStealer died and restarted")
		case ID_DATADOWNLOADER:
			go dataDownloader(control)
			logger.Error("dataMover died and restarted")
		case ID_WORKER:
			go processor(control)
			logger.Error("worker died and restarted")
		case ID_DELIVERER:
			go deliverer(control)
			logger.Error("deliverer died and restarted")
		}
	}
}

func StartProxyWorkers() {
	control := make(chan int)
	go heartBeater(control)
	go workStealer(control)
	//go redistributor(control)
	for {
		who := <-control //block till someone dies and then restart it
		switch who {
		case ID_HEARTBEATER:
			go heartBeater(control)
			logger.Error("heartBeater died and restarted")
		case ID_WORKSTEALER:
			go workStealer(control)
			logger.Error("workStealer died and restarted")
		case ID_REDISTRIBUTOR:
			//go redistributor(control)
			logger.Error("deliverer died and restarted")
		}
	}
}
