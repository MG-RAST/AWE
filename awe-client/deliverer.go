package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/logger"
	"io/ioutil"
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
		perfstat := processed.perfstat

		//post-process for works computed successfully: push output data to Shock
		move_start := time.Now().Unix()
		if processed.status == WORK_STAT_COMPUTED {
			if err := pushOutputData(work); err != nil {
				processed.status = WORK_STAT_FAIL
				Log.Error("err@pushOutputData: workid=" + work.Id + ", err=" + err.Error())
			} else {
				processed.status = WORK_STAT_DONE
			}
		}
		move_end := time.Now().Unix()
		perfstat.DataOut = move_end - move_start
		perfstat.Deliver = move_end
		perfstat.ClientResp = perfstat.Deliver - perfstat.Checkout
		perfstat.ClientId = self.Id

		//notify server the final process results
		if err := notifyWorkunitProcessed(work, processed.status, perfstat); err != nil {
			time.Sleep(3 * time.Second) //wait 3 seconds and try another time
			if err := notifyWorkunitProcessed(work, processed.status, perfstat); err != nil {
				fmt.Printf("!!!NotifyWorkunitDone returned error: %s\n", err.Error())
				Log.Error("err@NotifyWorkunitProcessed: workid=" + work.Id + ", err=" + err.Error())
				//mark this work in Current_work map as false, something needs to be done in the future
				//to clean this kind of work that has been proccessed but its result can't be sent to server!
				self.Current_work[work.Id] = false //server doesn't know this yet
				continue
			}
		}
		//now final status report sent to server, update some local info
		if processed.status == WORK_STAT_DONE {
			Log.Event(EVENT_WORK_DONE, "workid="+work.Id)
			self.Total_completed += 1

			if conf.AUTO_CLEAN_DIR {
				if err := work.RemoveDir(); err != nil {
					Log.Error("err@work.RemoveDir(): workid=" + work.Id + ", err=" + err.Error())
				}
			}
		} else {
			Log.Event(EVENT_WORK_RETURN, "workid="+work.Id)
			self.Total_failed += 1
		}
		delete(self.Current_work, work.Id)

		//release the permit lock, for work overlap inhibitted mode only
		if !conf.WORKER_OVERLAP {
			<-chanPermit
		}
	}
	control <- ID_DELIVERER //we are ending
}

func pushOutputData(work *Workunit) (err error) {
	for name, io := range work.Outputs {
		if _, err := os.Stat(name); err != nil {
			return errors.New(fmt.Sprintf("output %s not generated for workunit %s", name, work.Id))
		}
		Log.Debug(2, "deliverer: push output to shock, filename="+name)
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

//notify AWE server a workunit is finished with status either "failed" or "done", and with perf statistics if "done"
func notifyWorkunitProcessed(work *Workunit, status string, perf *WorkPerf) (err error) {
	target_url := fmt.Sprintf("%s/work/%s?status=%s&client=%s", conf.SERVER_URL, work.Id, status, self.Id)

	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	if status == WORK_STAT_DONE {
		reportFile, err := getPerfFilePath(work, perf)
		if err == nil {
			argv = append(argv, "-F")
			argv = append(argv, fmt.Sprintf("perf=@%s", reportFile))
			target_url = target_url + "&report"
		}
	}
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
	Log.Debug(2, fmt.Sprintf("deliverer: curl argv=%#v", argv))
	cmd := exec.Command("curl", argv...)
	err = cmd.Run()
	if err != nil {
		return
	}
	return
}

func getPerfFilePath(work *Workunit, perfstat *WorkPerf) (reportPath string, err error) {
	perfJsonstream, err := json.Marshal(perfstat)
	if err != nil {
		return reportPath, err
	}
	reportFile := fmt.Sprintf("%s/%s.perf", work.Path(), work.Id)
	if err := ioutil.WriteFile(reportFile, []byte(perfJsonstream), 0644); err != nil {
		return reportPath, err
	}
	return reportFile, nil
}
