package main

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/logger"
	"os"
	"os/exec"
	"time"
)

func deliverer(control chan int) {
	fmt.Printf("deliverer lanched, client=%s\n", self.Id)
	defer fmt.Printf("deliverer exiting...\n")
	for {
		processed := <-chanProcessed
		work := processed.workunit

		//post-process for works computed successfully: push output data to Shock
		if processed.status == WORK_STAT_COMPUTED {
			if err := PushOutputData(work); err != nil {
				processed.status = WORK_STAT_FAIL
			} else {
				processed.status = WORK_STAT_DONE
			}
		}

		//notify server the final process results
		if err := NotifyWorkunitProcessed(conf.SERVER_URL, work.Id, processed.status); err != nil {
			time.Sleep(3 * time.Second) //wait 3 seconds and try another time
			if err := NotifyWorkunitProcessed(conf.SERVER_URL, work.Id, processed.status); err != nil {
				fmt.Printf("!!!NotifyWorkunitDone returned error: %s\n", err.Error())
				Log.Error("err@NotifyWorkunitProcessed: workid=" + work.Id + ", err=" + err.Error())
				//mark this work in Current_work map as false, something needs to be done in the future
				//to clean this kind of work that has been proccessed but its result can't be sent to server!
				self.Current_work[work.Id] = false
				continue
			}
		}
		//now server got the status report, update some local info
		if processed.status == WORK_STAT_DONE {
			Log.Event(EVENT_WORK_DONE, "workid="+work.Id)
			self.Total_completed += 1
		} else {
			Log.Event(EVENT_WORK_RETURN, "workid="+work.Id)
			self.Total_failed += 1
		}
		delete(self.Current_work, work.Id)
	}
	control <- ID_DELIVERER //we are ending
}

func PushOutputData(work *Workunit) (err error) {
	for name, io := range work.Outputs {
		if _, err := os.Stat(name); err != nil {
			return errors.New(fmt.Sprintf("output %s not generated for workunit %s", name, work.Id))
		}
		fmt.Printf("worker: push output to shock, filename=%s\n", name)
		Log.Event(EVENT_FILE_OUT,
			"workid="+work.Id,
			"filename="+name,
			fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))

		file_path := fmt.Sprintf("%s/%s", work.Path(), name)
		if err := pushFileByCurl(file_path, io.Host, io.Node, work.Rank); err != nil {
			time.Sleep(3 * time.Second) //wait for 3 seconds and try again
			if err := pushFileByCurl(name, io.Host, io.Node, work.Rank); err != nil {
				fmt.Errorf("push file error\n")
				Log.Error("op=pushfile,err=" + err.Error())
				return err
			}
		}
		Log.Event(EVENT_FILE_DONE,
			"workid="+work.Id,
			"filename="+name,
			fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))
	}
	return
}

func NotifyWorkunitProcessed(serverhost string, workid string, status string) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	target_url := fmt.Sprintf("%s/work/%s?status=%s&client=%s", serverhost, workid, status, self.Id)
	argv = append(argv, target_url)

	cmd := exec.Command("curl", argv...)

	err = cmd.Run()

	if err != nil {
		return
	}
	return
}

//push file to shock
func pushFileByCurl(filename string, host string, node string, rank int) (err error) {

	shockurl := fmt.Sprintf("%s/node/%s", host, node)

	if err := putFileByCurl(filename, shockurl, rank); err != nil {
		return err
	}
	return
}

func putFileByCurl(filename string, target_url string, rank int) (err error) {

	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	argv = append(argv, "-F")

	if rank == 0 {
		argv = append(argv, fmt.Sprintf("upload=@%s", filename))
	} else {
		argv = append(argv, fmt.Sprintf("%d=@%s", rank, filename))
	}

	argv = append(argv, target_url)

	fmt.Printf("curl argv=%#v\n", argv)

	cmd := exec.Command("curl", argv...)

	err = cmd.Run()

	if err != nil {
		return
	}
	return
}
