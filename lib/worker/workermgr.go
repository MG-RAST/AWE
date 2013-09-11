package worker

import (
	"errors"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
)

var (
	fromStealer   chan *mediumwork // workStealer -> dataMover
	fromMover     chan *mediumwork // dataMover -> processor
	fromProcessor chan *mediumwork // processor -> deliverer
	chanPermit    chan bool
	chankill      chan bool      //heartbeater -> worker
	workmap       map[string]int //workunit map [work_id]stage_id}
)

type mediumwork struct {
	workunit *core.Workunit
	perfstat *core.WorkPerf
}

const (
	ID_HEARTBEATER   = 0
	ID_WORKSTEALER   = 1
	ID_DATAMOVER     = 2
	ID_WORKER        = 3
	ID_DELIVERER     = 4
	ID_REDISTRIBUTOR = 5
)

func InitWorkers(client *core.Client) (err error) {
	if client == nil {
		return errors.New("InitClientWorkers(): empty client")
	}
	fromStealer = make(chan *mediumwork)   // workStealer -> dataMover
	fromMover = make(chan *mediumwork)     // dataMover -> processor
	fromProcessor = make(chan *mediumwork) // processor -> deliverer
	chankill = make(chan bool)             //heartbeater -> processor
	chanPermit = make(chan bool)
	workmap = map[string]int{} //workunit map [work_id]stage_idgit
	return
}

func StartClientWorkers() {
	control := make(chan int)
	go heartBeater(control)
	go workStealer(control)
	go dataMover(control)
	go processor(control)
	go deliverer(control)
	for {
		who := <-control //block till someone dies and then restart it
		switch who {
		case ID_HEARTBEATER:
			go heartBeater(control)
			logger.Error("heartBeater died and restarted")
		case ID_WORKSTEALER:
			go workStealer(control)
			logger.Error("workStealer died and restarted")
		case ID_DATAMOVER:
			go dataMover(control)
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
	go redistributor(control)
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
			go redistributor(control)
			logger.Error("deliverer died and restarted")
		}
	}
}
