package worker

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"time"
)

func deliverer(control chan int) {
	fmt.Printf("deliverer lanched, client=%s\n", core.Self.Id)
	defer fmt.Printf("deliverer exiting...\n")
	for {
		processed := <-fromProcessor
		work := processed.workunit
		workmap[work.Id] = ID_DELIVERER
		perfstat := processed.perfstat

		//post-process for works computed successfully: push output data to Shock
		move_start := time.Now().Unix()
		if work.State == core.WORK_STAT_COMPUTED {
			if err := core.PushOutputData(work); err != nil {
				work.State = core.WORK_STAT_FAIL
				logger.Error("err@pushOutputData: workid=" + work.Id + ", err=" + err.Error())
			} else {
				work.State = core.WORK_STAT_DONE
			}
		}
		move_end := time.Now().Unix()
		perfstat.DataOut = move_end - move_start
		perfstat.Deliver = move_end
		perfstat.ClientResp = perfstat.Deliver - perfstat.Checkout
		perfstat.ClientId = core.Self.Id

		//notify server the final process results
		if err := core.NotifyWorkunitProcessed(work, perfstat); err != nil {
			time.Sleep(3 * time.Second) //wait 3 seconds and try another time
			if err := core.NotifyWorkunitProcessed(work, perfstat); err != nil {
				fmt.Printf("!!!NotifyWorkunitDone returned error: %s\n", err.Error())
				logger.Error("err@NotifyWorkunitProcessed: workid=" + work.Id + ", err=" + err.Error())
				//mark this work in Current_work map as false, something needs to be done in the future
				//to clean this kind of work that has been proccessed but its result can't be sent to server!
				core.Self.Current_work[work.Id] = false //server doesn't know this yet
			}
		}
		//now final status report sent to server, update some local info
		if work.State == core.WORK_STAT_DONE {
			logger.Event(event.WORK_DONE, "workid="+work.Id)
			core.Self.Total_completed += 1

			if conf.AUTO_CLEAN_DIR {
				if err := work.RemoveDir(); err != nil {
					logger.Error("err@work.RemoveDir(): workid=" + work.Id + ", err=" + err.Error())
				}
			}
		} else {
			logger.Event(event.WORK_RETURN, "workid="+work.Id)
			core.Self.Total_failed += 1
		}
		delete(core.Self.Current_work, work.Id)
		delete(workmap, work.Id)

	}
	control <- ID_DELIVERER //we are ending
}
