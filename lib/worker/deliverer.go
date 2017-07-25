package worker

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/cache"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"net/http"
	"os"
	"strings"
	"time"
)

func deliverer(control chan int) {
	fmt.Printf("deliverer launched, client=%s\n", core.Self.Id)
	defer fmt.Printf("deliverer exiting...\n")
	for {
		deliverer_run(control)
	}
	control <- ID_DELIVERER //we are ending
}

func deliverer_run(control chan int) {

	logger.Debug(3, "deliverer_run")

	// this makes sure new work is only requested when deliverer is done
	defer func() { <-chanPermit }()

	workunit := <-fromProcessor

	work_id := workunit.Id

	work_state, ok, err := workmap.Get(work_id)
	if err != nil {
		logger.Error("error: %s", err.Error())
		return
	}
	if !ok {
		logger.Error("(deliverer) work id %s not found", work_id)
		return
	}

	if work_state == ID_DISCARDED {
		workunit.SetState(core.WORK_STAT_DISCARDED)
		logger.Event(event.WORK_DISCARD, "workid="+work_id)
	}

	workmap.Set(work_id, ID_DELIVERER, "deliverer")

	perfstat := workunit.WorkPerf

	if Client_mode == "offline" {
		return
	}

	//post-process for works computed successfully: push output data to Shock
	move_start := time.Now().UnixNano()
	logger.Debug(3, "(deliverer_run) work.State: %s", workunit.State)
	if workunit.State == core.WORK_STAT_COMPUTED {
		data_moved, err := cache.UploadOutputData(workunit)
		if err != nil {
			workunit.SetState(core.WORK_STAT_ERROR)
			logger.Error("[deliverer#UploadOutputData]workid=" + work_id + ", err=" + err.Error())
			workunit.Notes = append(workunit.Notes, "[deliverer#UploadOutputData]"+err.Error())
		} else {
			workunit.SetState(core.WORK_STAT_DONE)
			perfstat.OutFileSize = data_moved
		}
	}
	move_end := time.Now().UnixNano()
	perfstat.DataOut = float64(move_end-move_start) / 1e9
	perfstat.Deliver = int64(move_end / 1e9)
	perfstat.ClientResp = perfstat.Deliver - perfstat.Checkout
	perfstat.ClientId = core.Self.Id

	//notify server the final process results; send perflog, stdout, and stderr if needed
	// detect e.ClientNotFound
	do_retry := true
	retry_count := 0
	for do_retry {
		response, err := core.NotifyWorkunitProcessedWithLogs(workunit, perfstat, conf.PRINT_APP_MSG)
		if err != nil {
			logger.Error("[deliverer: workid=" + work_id + ", err=" + err.Error())
			workunit.Notes = append(workunit.Notes, "[deliverer]"+err.Error())
			// keep retry
		} else {

			error_message := strings.Join(response.E, ",")
			if strings.Contains(error_message, e.ClientNotFound) { // TODO need better method than string search. Maybe a field awe_status.
				//mark this work in Current_work map as false, something needs to be done in the future
				//to clean this kind of work that has been proccessed but its result can't be sent to server!
				//core.Self.Current_work_false(work.Id) //server doesn't know this yet
				do_retry = false
			}

			if response.S == http.StatusOK {
				// success, work delivered
				logger.Debug(1, "work delivered successfully")
				do_retry = false
			} else {
				logger.Error("deliverer: workid=%s, err=%s", work_id, error_message)
			}
		}

		if do_retry {
			time.Sleep(time.Second * 60)
			retry_count += 1
		} else {
			if retry_count > 100 { // TODO 100 ?
				break
			}
			break
		}
	}

	//now final status report sent to server, update some local info
	if workunit.State == core.WORK_STAT_DONE {
		logger.Event(event.WORK_DONE, "workid="+work_id)
		core.Self.Increment_total_completed()
		if conf.AUTO_CLEAN_DIR && workunit.Cmd.Local == false {
			go removeDirLater(workunit.Path(), conf.CLIEN_DIR_DELAY_DONE)
		}
	} else {
		logger.Event(event.WORK_RETURN, "workid="+work_id)
		core.Self.Increment_total_failed(true)
		if conf.AUTO_CLEAN_DIR && workunit.Cmd.Local == false {
			go removeDirLater(workunit.Path(), conf.CLIEN_DIR_DELAY_FAIL)
		}
	}
	err = core.Self.Current_work_delete(work_id, true)
	if err != nil {
		logger.Error("Could not remove work_id %s", work_id)
	}
	//delete(core.Self.Current_work, work_id)
	//delete(workmap, work.Id)
	workmap.Delete(work_id)
}

func removeDirLater(path string, duration time.Duration) (err error) {
	time.Sleep(duration)
	return os.RemoveAll(path)
}
