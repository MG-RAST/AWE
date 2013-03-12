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
	"time"
)

func worker(control chan int) {
	fmt.Printf("worker lanched, client=%s\n", self.Id)
	defer fmt.Printf("worker exiting...\n")
	for {
		parsedwork := <-chanParsed
		work := parsedwork.workunit

		processed := &processedWork{
			workunit: work,
			perfstat: parsedwork.perfstat,
			status:   "unknown",
		}

		//if the work is not succesfully parsed in last stage, pass it into the next one immediately
		if parsedwork.status == WORK_STAT_FAIL {
			processed.status = WORK_STAT_FAIL
			chanProcessed <- processed
			continue
		}

		run_start := time.Now().Unix()
		if err := RunWorkunit(parsedwork); err != nil {
			fmt.Printf("!!!RunWorkunit() returned error: %s\n", err.Error())
			Log.Error("RunWorkunit(): workid=" + work.Id + ", " + err.Error())

			//restart once and if it still fails
			if err := RunWorkunit(parsedwork); err != nil {
				fmt.Printf("!!!RunWorkunit() returned error again: %s\n", err.Error())
				Log.Error("ReRunWorkunit(): workid=" + work.Id + ", " + err.Error())
				processed.status = WORK_STAT_FAIL
			}
		} else {
			processed.status = WORK_STAT_COMPUTED
		}
		run_end := time.Now().Unix()
		processed.perfstat.Runtime = run_end - run_start

		chanProcessed <- processed
	}
	control <- ID_WORKER //we are ending
}

func RunWorkunit(parsed *parsedWork) (err error) {

	work := parsed.workunit
	args := parsed.args

	fmt.Printf("++++++worker: started processing workunit id=%s++++++\n", work.Id)
	defer fmt.Printf("-------worker: finished processing workunit id=%s------\n\n", work.Id)

	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return err
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

	return
}
