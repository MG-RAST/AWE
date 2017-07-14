package worker

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
)

func redistributor(control chan int) {
	fmt.Printf("redistributor launched, client=%s\n", core.Self.Id)
	defer fmt.Printf("redistributor exiting...\n")
	for {
		parsed := <-FromStealer
		work := parsed.Workunit

		workmap.Set(work.Id, ID_REDISTRIBUTOR, "redistributor")

		queued := &Mediumwork{
			Workunit: parsed.Workunit,
			perfstat: parsed.perfstat,
		}

		if err := SubmitWorkProxy(work); err != nil {
			fmt.Printf("SubmitWorkProxy() returned error: %s\n", err.Error())
			logger.Error("SubmitWorkProxy(): workid=" + work.Id + ", " + err.Error())
			queued.Workunit.SetState(core.WORK_STAT_ERROR)
		} else {
			queued.Workunit.SetState(core.WORK_STAT_PROXYQUEUED)
			logger.Event(event.WORK_QUEUED, "workid="+work.Id)
		}
	}
	control <- ID_REDISTRIBUTOR //we are ending
}

func SubmitWorkProxy(work *core.Workunit) (err error) {
	err = core.QMgr.EnqueueWorkunit(work)
	return
}
