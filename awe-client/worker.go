package main

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/logger"
	"io"
	"os"
	"os/exec"
)

func worker(control chan int) {
	fmt.Printf("worker lanched, client=%s\n", self.Id)
	defer fmt.Printf("worker exiting...\n")
	for {
		work := <-workChan
		if err := RunWorkunit(work); err != nil {
			fmt.Printf("!!!RunWorkunit() returned error: %s\n", err.Error())
			Log.Error("RunWorkunit(): workid=" + work.Id + ", " + err.Error())

			//restart once and if it still fails
			if err := RunWorkunit(work); err != nil {
				fmt.Printf("!!!ReRunWorkunit() returned error: %s\n", err.Error())
				Log.Error("ReRunWorkunit(): workid=" + work.Id + ", " + err.Error())

				//send back the workunit to server
				if err := NotifyWorkunitProcessed(conf.SERVER_URL, work.Id, "fail"); err != nil {
					fmt.Printf("!!!NotifyWorkunitFail returned error: %s\n", err.Error())
					Log.Error("NotifyWorkunitFail: workid=" + work.Id + ", err=" + err.Error())
				}
				Log.Event(EVENT_WORK_RETURN, "workid="+work.Id)
				self.Total_failed += 1
				delete(self.Current_work, work.Id)
				continue
			}
		}
		if err := NotifyWorkunitProcessed(conf.SERVER_URL, work.Id, "done"); err != nil {
			fmt.Printf("!!!NotifyWorkunitDone returned error: %s\n", err.Error())
			Log.Error("NotifyWorkunitDone: workid=" + work.Id + ", err=" + err.Error())
		}
		Log.Event(EVENT_WORK_DONE, "workid="+work.Id)
		self.Total_completed += 1
		delete(self.Current_work, work.Id)
	}
	control <- 1 //we are ending
}

func RunWorkunit(work *Workunit) (err error) {

	fmt.Printf("++++++worker: started processing workunit id=%s++++++\n", work.Id)
	defer fmt.Printf("-------worker: finished processing workunit id=%s------\n\n", work.Id)

	//make a working directory for the workunit
	if err := work.Mkdir(); err != nil {
		return err
	}

	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return err
	}

	args, err := ParseWorkunitArgs(work)
	if err != nil {
		return
	}

	commandName := work.Cmd.Name
	cmd := exec.Command(commandName, args...)

	fmt.Printf("worker: start running cmd=%s, args=%v\n", commandName, args)
	Log.Event(EVENT_WORK_START, "workid="+work.Id,
		"cmd="+commandName,
		fmt.Sprintf("args=%v", args))

	var stdout, stderr io.ReadCloser
	if conf.PRINT_APP_MSG {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return err
		}
		stderr, err = cmd.StderrPipe()
		if err != nil {
			return err
		}
	}

	if err := cmd.Start(); err != nil {
		return errors.New(fmt.Sprintf("start_cmd=%s, err=%s", commandName, err.Error()))
	}

	if conf.PRINT_APP_MSG {
		go io.Copy(os.Stdout, stdout)
		go io.Copy(os.Stderr, stderr)
	}

	if err := cmd.Wait(); err != nil {
		return errors.New(fmt.Sprintf("wait_cmd=%s, err=%s", commandName, err.Error()))
	}

	Log.Event(EVENT_WORK_END, "workid="+work.Id)

	for name, io := range work.Outputs {

		if _, err := os.Stat(name); err != nil {
			return errors.New(fmt.Sprintf("output %s not generated for workunit %s", name, work.Id))
		}

		fmt.Printf("worker: push output to shock, filename=%s\n", name)
		Log.Event(EVENT_FILE_OUT,
			"workid="+work.Id,
			"filename="+name,
			fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))
		if err := pushFileByCurl(name, io.Host, io.Node, work.Rank); err != nil {
			fmt.Errorf("push file error\n")
			Log.Error("op=pushfile,err=" + err.Error())
			return err
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
