package worker

import (
	"bufio"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"io"
	"os"
	"os/exec"
	"runtime"
	"time"
)

func processor(control chan int) {
	fmt.Printf("processor lanched, client=%s\n", core.Self.Id)
	defer fmt.Printf("processor exiting...\n")
	for {
		parsedwork := <-fromMover
		work := parsedwork.workunit
		workmap[work.Id] = ID_WORKER

		processed := &mediumwork{
			workunit: work,
			perfstat: parsedwork.perfstat,
		}

		//if the work is not succesfully parsed in last stage, pass it into the next one immediately
		if work.State == core.WORK_STAT_FAIL {
			processed.workunit.State = core.WORK_STAT_FAIL
			fromProcessor <- processed
			//release the permit lock, for work overlap inhibitted mode only
			if !conf.WORKER_OVERLAP && core.Service != "proxy" {
				<-chanPermit
			}
			continue
		}

		run_start := time.Now().Unix()
		pstat, err := RunWorkunit(work)
		if err != nil {
			fmt.Printf("!!!RunWorkunit() returned error: %s\n", err.Error())
			logger.Error("RunWorkunit(): workid=" + work.Id + ", " + err.Error())
			processed.workunit.State = core.WORK_STAT_FAIL
		} else {
			processed.workunit.State = core.WORK_STAT_COMPUTED
			processed.perfstat.MaxMemUsage = pstat.MaxMemUsage
		}
		run_end := time.Now().Unix()
		computetime := run_end - run_start
		processed.perfstat.Runtime = computetime
		processed.workunit.ComputeTime = int(computetime)

		fromProcessor <- processed

		//release the permit lock, for work overlap inhibitted mode only
		if !conf.WORKER_OVERLAP && core.Service != "proxy" {
			<-chanPermit
		}
	}
	control <- ID_WORKER //we are ending
}

func RunWorkunit(work *core.Workunit) (pstats *core.WorkPerf, err error) {
	pstats = new(core.WorkPerf)

	args := work.Cmd.ParsedArgs

	//change cwd to the workunit's working directory
	if err := work.CDworkpath(); err != nil {
		return nil, err
	}

	commandName := work.Cmd.Name
	cmd := exec.Command(commandName, args...)

	msg := fmt.Sprintf("worker: start cmd=%s, args=%v", commandName, args)
	fmt.Println(msg)
	logger.Debug(1, msg)
	logger.Event(event.WORK_START, "workid="+work.Id,
		"cmd="+commandName,
		fmt.Sprintf("args=%v", args))

	var stdout, stderr io.ReadCloser
	if conf.PRINT_APP_MSG {
		stdout, err = cmd.StdoutPipe()
		if err != nil {
			return nil, err
		}
		stderr, err = cmd.StderrPipe()
		if err != nil {
			return nil, err
		}
	}

	if err := cmd.Start(); err != nil {
		return nil, errors.New(fmt.Sprintf("start_cmd=%s, err=%s", commandName, err.Error()))
	}

	stdoutFilePath := fmt.Sprintf("%s/%s", work.Path(), conf.STDOUT_FILENAME)
	stderrFilePath := fmt.Sprintf("%s/%s", work.Path(), conf.STDERR_FILENAME)
	outfile, err := os.Create(stdoutFilePath)
	defer outfile.Close()
	errfile, err := os.Create(stderrFilePath)
	defer errfile.Close()
	out_writer := bufio.NewWriter(outfile)
	defer out_writer.Flush()
	err_writer := bufio.NewWriter(errfile)
	defer err_writer.Flush()

	if conf.PRINT_APP_MSG {
		go io.Copy(out_writer, stdout)
		go io.Copy(err_writer, stderr)
	}

	var MaxMem uint64 = 0
	done := make(chan error)
	go func() {
		done <- cmd.Wait()
	}()
	go func() {
		mstats := new(runtime.MemStats)
		runtime.ReadMemStats(mstats)
		MaxMem = mstats.Alloc
		time.Sleep(2 * time.Second)
		for {
			mstats := new(runtime.MemStats)
			runtime.ReadMemStats(mstats)
			if mstats.Alloc > MaxMem {
				MaxMem = mstats.Alloc
			}
			time.Sleep(conf.MEM_CHECK_INTERVAL)
		}
	}()

	select {
	case <-chankill:
		if err := cmd.Process.Kill(); err != nil {
			fmt.Println("failed to kill" + err.Error())
		}
		<-done // allow goroutine to exit
		fmt.Println("process killed")
		return nil, errors.New("process killed")
	case err := <-done:
		if err != nil {
			return nil, errors.New(fmt.Sprintf("wait_cmd=%s, err=%s", commandName, err.Error()))
		}
	}
	logger.Event(event.WORK_END, "workid="+work.Id)
	pstats.MaxMemUsage = MaxMem
	return
}
