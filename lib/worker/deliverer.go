package worker

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/cache"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"os"
	"time"
)

func deliverer(control chan int) {
	fmt.Printf("deliverer launched, client=%s\n", core.Self.Id)
	defer fmt.Printf("deliverer exiting...\n")
	for {
		processed := <-fromProcessor
		work := processed.workunit

		if workmap[work.Id] == ID_DISCARDED {
			work.State = core.WORK_STAT_DISCARDED
			logger.Event(event.WORK_DISCARD, "workid="+work.Id)
		}

		workmap[work.Id] = ID_DELIVERER
		perfstat := processed.perfstat

		//post-process for works computed successfully: push output data to Shock
		move_start := time.Now().UnixNano()
		if work.State == core.WORK_STAT_COMPUTED {
			if data_moved, err := cache.UploadOutputData(work); err != nil {
				work.State = core.WORK_STAT_FAIL
				logger.Error("[deliverer#UploadOutputData]workid=" + work.Id + ", err=" + err.Error())
				work.Notes = work.Notes + "###[deliverer#UploadOutputData]" + err.Error()
			} else {
				work.State = core.WORK_STAT_DONE
				perfstat.OutFileSize = data_moved
			}
		}
		move_end := time.Now().UnixNano()
		perfstat.DataOut = float64(move_end-move_start) / 1e9
		perfstat.Deliver = int64(move_end / 1e9)
		perfstat.ClientResp = perfstat.Deliver - perfstat.Checkout
		perfstat.ClientId = core.Self.Id

		//notify server the final process results; send perflog, stdout, and stderr if needed
		if err := core.NotifyWorkunitProcessedWithLogs(work, perfstat, conf.PRINT_APP_MSG); err != nil {
			fmt.Printf("!!!NotifyWorkunitDone returned error: %s\n", err.Error())
			logger.Error("[deliverer#NotifyWorkunitProcessedWithLogs]workid=" + work.Id + ", err=" + err.Error())
			work.Notes = work.Notes + "###[deliverer#NotifyWorkunitProcessedWithLogs]" + err.Error()
			//mark this work in Current_work map as false, something needs to be done in the future
			//to clean this kind of work that has been proccessed but its result can't be sent to server!
			core.Self.Current_work_false(work.Id) //server doesn't know this yet
		}

		//now final status report sent to server, update some local info
		if work.State == core.WORK_STAT_DONE {
			logger.Event(event.WORK_DONE, "workid="+work.Id)
			core.Self.Increment_total_completed()
			if conf.AUTO_CLEAN_DIR {
				go removeDirLater(work.Path(), conf.CLIEN_DIR_DELAY_DONE)
			}
		} else {
			logger.Event(event.WORK_RETURN, "workid="+work.Id)
			core.Self.Increment_total_failed(true)
			if conf.AUTO_CLEAN_DIR {
				go removeDirLater(work.Path(), conf.CLIEN_DIR_DELAY_FAIL)
			}
		}
		core.Self.Current_work_delete(work.Id, true)
		//delete(core.Self.Current_work, work.Id)
		delete(workmap, work.Id)
	}
	control <- ID_DELIVERER //we are ending
}

func removeDirLater(path string, duration time.Duration) (err error) {
	time.Sleep(duration)
	return os.RemoveAll(path)
}
