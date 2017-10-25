package core

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"os/exec"
	"time"
)

func proxy_relay_workunit(work *Workunit, perfstat *WorkPerf) (err error) {
	//notify server the final process results
	err = NotifyWorkunitProcessed(work, perfstat)
	if err != nil {
		time.Sleep(3 * time.Second) //wait 3 seconds and try another time
		err = NotifyWorkunitProcessed(work, perfstat)
		if err != nil {
			fmt.Printf("!!!NotifyWorkunitDone returned error: %s\n", err.Error())
			logger.Error("err@NotifyWorkunitProcessed: workid=" + work.Id + ", err=" + err.Error())
			//mark this work in Current_work map as false, something needs to be done in the future
			//to clean this kind of work that has been proccessed but its result can't be sent to server!
			//xerr := Self.Current_work_false(work.Id)
			//if xerr != nil {
			//	err = xerr
			//	return
			//}
		}
	}

	//now final status report sent to server, update some local info
	if work.State == WORK_STAT_DONE {
		logger.Event(event.WORK_DONE, "workid="+work.Id)
		Self.Increment_total_completed()
	} else {
		logger.Event(event.WORK_RETURN, "workid="+work.Id)
		Self.Increment_total_failed(true)
	}
	err = Self.Current_work.Delete(work.Workunit_Unique_Identifier, true)
	if err != nil {
		return
	}
	//delete(Self.Current_work, work.Id)
	return
}

func notifySubClients(clientid string, count int) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	target_url := fmt.Sprintf("%s/client/%s?subclients=%d", conf.SERVER_URL, clientid, count)
	argv = append(argv, target_url)
	cmd := exec.Command("curl", argv...)
	err = cmd.Run()
	if err != nil {
		return
	}
	return
}
