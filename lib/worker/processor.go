package worker

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	. "github.com/MG-RAST/AWE/lib/logger"
	"io"
	"os"
	"os/exec"
	"time"
)

func processor(control chan int) {
	fmt.Printf("processor lanched, client=%s\n", self.Id)
	defer fmt.Printf("processor exiting...\n")
	for {
		parsedwork := <-chanParsed
		work := parsedwork.workunit
		workmap[work.Id] = ID_WORKER

		processed := &mediumwork{
			workunit: work,
			perfstat: parsedwork.perfstat,
		}

		//if the work is not succesfully parsed in last stage, pass it into the next one immediately
		if work.State == core.WORK_STAT_FAIL {
			processed.workunit.State = core.WORK_STAT_FAIL
			chanProcessed <- processed
			continue
		}

		run_start := time.Now().Unix()
		if err := RunWorkunit(work); err != nil {
			fmt.Printf("!!!RunWorkunit() returned error: %s\n", err.Error())
			Log.Error("RunWorkunit(): workid=" + work.Id + ", " + err.Error())
			processed.workunit.State = core.WORK_STAT_FAIL
		} else {
			processed.workunit.State = core.WORK_STAT_COMPUTED
		}
		run_end := time.Now().Unix()
		processed.perfstat.Runtime = run_end - run_start

		chanProcessed <- processed
	}
	control <- ID_WORKER //we are ending
}

func RunWorkunit(work *core.Workunit) (err error) {

	args := work.Cmd.ParsedArgs

	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return err
	}

	commandName := work.Cmd.Name
	cmd := exec.Command(commandName, args...)

	msg := fmt.Sprintf("worker: start cmd=%s, args=%v", commandName, args)
	fmt.Println(msg)
	Log.Debug(1, msg)
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

	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()
	select {
	case <-chankill:
		if err := cmd.Process.Kill(); err != nil {
			fmt.Println("failed to kill" + err.Error())
		}
		<-done // allow goroutine to exit
		fmt.Println("process killed")
		return errors.New("process killed")
	case err := <-done:
		if err != nil {
			return errors.New(fmt.Sprintf("wait_cmd=%s, err=%s", commandName, err.Error()))
		}
	}
	Log.Event(EVENT_WORK_END, "workid="+work.Id)
	return
}
