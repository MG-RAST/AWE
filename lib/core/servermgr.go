package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/MG-RAST/AWE/lib/user"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"sync"
	"time"
)

type jQueueShow struct {
	Active  map[string]*JobPerf `bson:"active" json:"active"`
	Suspend map[string]bool     `bson:"suspend" json:"suspend"`
}

type ServerMgr struct {
	CQMgr
	queueLock sync.Mutex //only update one at a time
	//taskLock  sync.RWMutex
	TaskMap TaskMap
	ajLock  sync.RWMutex
	sjLock  sync.RWMutex
	//taskMap   map[string]*Task
	actJobs map[string]*JobPerf
	susJobs map[string]bool
	taskIn  chan *Task //channel for receiving Task (JobController -> qmgr.Handler)
	coSem   chan int   //semaphore for checkout (mutual exclusion between different clients)
}

func NewServerMgr() *ServerMgr {
	return &ServerMgr{
		CQMgr: CQMgr{
			clientMap:    *NewClientMap(),
			workQueue:    NewWorkQueue(),
			suspendQueue: true,
			coReq:        make(chan CoReq),
			//coAck:        make(chan CoAck),
			feedback: make(chan Notice),
			coSem:    make(chan int, 1), //non-blocking buffered channel
		},
		//TaskMap: map[string]*Task{},
		TaskMap: *NewTaskMap(),
		taskIn:  make(chan *Task, 1024),
		actJobs: map[string]*JobPerf{},
		susJobs: map[string]bool{},
	}
}

//--------mgr methods-------

func (qm *ServerMgr) Lock()    {}
func (qm *ServerMgr) Unlock()  {}
func (qm *ServerMgr) RLock()   {}
func (qm *ServerMgr) RUnlock() {}

func (qm *ServerMgr) TaskHandle() {
	for {
		task := <-qm.taskIn
		logger.Debug(2, "qmgr:task recived from chan taskIn, id=%s", task.Id)
		qm.addTask(task)
	}
}

func (qm *ServerMgr) ClientHandle() {
	logger.Info("(ServerMgr ClientHandle) starting")
	for {
		//select {
		//case coReq := <-qm.coReq
		coReq := <-qm.coReq
		logger.Debug(2, "qmgr: workunit checkout request received from client %s, Req=%v", coReq.fromclient, coReq)

		var ack CoAck
		if qm.suspendQueue {
			// queue is suspended, return suspend error
			ack = CoAck{workunits: nil, err: errors.New(e.QueueSuspend)}
		} else {
			logger.Debug(3, "(ServerMgr ClientHandle %s) updateQueue\n", coReq.fromclient)
			qm.updateQueue()
			logger.Debug(3, "(ServerMgr ClientHandle %s) popWorks\n", coReq.fromclient)
			works, err := qm.popWorks(coReq)
			logger.Debug(3, "(ServerMgr ClientHandle %s) popWorks done\n", coReq.fromclient)
			if err == nil {
				logger.Debug(3, "(ServerMgr ClientHandle %s) UpdateJobTaskToInProgress\n", coReq.fromclient)
				qm.UpdateJobTaskToInProgress(works)
				logger.Debug(3, "(ServerMgr ClientHandle %s) UpdateJobTaskToInProgress done\n", coReq.fromclient)
			}
			ack = CoAck{workunits: works, err: err}

		}
		logger.Debug(3, "(ServerMgr ClientHandle %s) send response now\n", coReq.fromclient)

		start_time := time.Now()
		timeout := make(chan bool, 1)
		go func() {
			time.Sleep(100 * time.Second)
			timeout <- true
		}()

		select {
		case coReq.response <- ack:
			logger.Debug(3, "(ServerMgr ClientHandle %s) send workunit to client via response channel", coReq.fromclient)
		case <-timeout:
			elapsed_time := time.Since(start_time)
			logger.Error("(ServerMgr ClientHandle %s) timed out after %s ", coReq.fromclient, elapsed_time)
			return
		}
		logger.Debug(3, "(ServerMgr ClientHandle %s) done\n", coReq.fromclient)
		// case notice := <-qm.feedback:
		// 			fmt.Println("(ServerMgr ClientHandle) got notice")
		// 			logger.Debug(2, fmt.Sprintf("qmgr: workunit feedback received, workid=%s, status=%s, clientid=%s", notice.WorkId, notice.Status, notice.ClientId))
		// 			if err := qm.handleWorkStatusChange(notice); err != nil {
		// 				fmt.Println("(ServerMgr ClientHandle) handleWorkStatusChange error")
		// 				logger.Error("handleWorkStatusChange(): " + err.Error())
		// 			}
		// 			fmt.Println("(ServerMgr ClientHandle) updateQueue")
		// 			qm.updateQueue()
		// 			fmt.Println("(ServerMgr ClientHandle) updateQueue done")
		//}
	}
}

func (qm *ServerMgr) NoticeHandle() {
	fmt.Println("(ServerMgr NoticeHandle) starting")
	for {
		notice := <-qm.feedback
		logger.Debug(3, "(ServerMgr NoticeHandle) got notice: workid=%s, status=%s, clientid=%s", notice.WorkId, notice.Status, notice.ClientId)
		if err := qm.handleWorkStatusChange(notice); err != nil {
			logger.Error("handleWorkStatusChange(): " + err.Error())
		}
		qm.updateQueue()
	}
}

//--------queue status methods-------

func (qm *ServerMgr) SuspendQueue() {
	qm.suspendQueue = true
}

func (qm *ServerMgr) ResumeQueue() {
	qm.suspendQueue = false
}

func (qm *ServerMgr) QueueStatus() string {
	if qm.suspendQueue {
		return "suspended"
	} else {
		return "running"
	}
}

func (qm *ServerMgr) GetQueue(name string) interface{} {
	if name == "job" {
		return jQueueShow{qm.actJobs, qm.susJobs}
	}
	if name == "task" {
		qm.ShowTasks() // only if debug level is set
		return qm.TaskMap.Map
	}
	if name == "work" {
		qm.ShowWorkQueue() // only if debug level is set
		return qm.workQueue.workMap.Map
	}
	if name == "client" {
		return qm.clientMap
	}
	return nil
}

//--------suspend job accessor methods-------

func (qm *ServerMgr) lenSusJobs() (l int) {
	qm.sjLock.RLock()
	l = len(qm.susJobs)
	qm.sjLock.RUnlock()
	return
}

func (qm *ServerMgr) putSusJob(id string) {
	qm.sjLock.Lock()
	qm.susJobs[id] = true
	qm.sjLock.Unlock()
}

func (qm *ServerMgr) GetSuspendJobs() (sjobs map[string]bool) {
	qm.sjLock.RLock()
	defer qm.sjLock.RUnlock()
	sjobs = make(map[string]bool)
	for id, _ := range qm.susJobs {
		sjobs[id] = true
	}
	return
}

func (qm *ServerMgr) removeSusJob(id string) {
	qm.sjLock.Lock()
	delete(qm.susJobs, id)
	qm.sjLock.Unlock()
}

func (qm *ServerMgr) isSusJob(id string) (has bool) {
	qm.sjLock.RLock()
	defer qm.sjLock.RUnlock()
	if _, ok := qm.susJobs[id]; ok {
		has = true
	} else {
		has = false
	}
	return
}

//--------active job accessor methods-------

func (qm *ServerMgr) copyJobPerf(a *JobPerf) (b *JobPerf) {
	b = new(JobPerf)
	*b = *a
	return
}

func (qm *ServerMgr) lenActJobs() (l int) {
	qm.ajLock.RLock()
	l = len(qm.actJobs)
	qm.ajLock.RUnlock()
	return
}

func (qm *ServerMgr) putActJob(jperf *JobPerf) {
	qm.ajLock.Lock()
	qm.actJobs[jperf.Id] = jperf
	qm.ajLock.Unlock()
}

func (qm *ServerMgr) getActJob(id string) (*JobPerf, bool) {
	qm.ajLock.RLock()
	defer qm.ajLock.RUnlock()
	if jobperf, ok := qm.actJobs[id]; ok {
		copy := qm.copyJobPerf(jobperf)
		return copy, true
	}
	return nil, false
}

func (qm *ServerMgr) GetActiveJobs() (ajobs map[string]bool) {
	qm.ajLock.RLock()
	defer qm.ajLock.RUnlock()
	ajobs = make(map[string]bool)
	for id, _ := range qm.actJobs {
		ajobs[id] = true
	}
	return
}

func (qm *ServerMgr) removeActJob(id string) {
	qm.ajLock.Lock()
	delete(qm.actJobs, id)
	qm.ajLock.Unlock()
}

func (qm *ServerMgr) isActJob(id string) (has bool) {
	qm.ajLock.RLock()
	defer qm.ajLock.RUnlock()
	if _, ok := qm.actJobs[id]; ok {
		has = true
	} else {
		has = false
	}
	return
}

//--------server methods-------

//poll ready tasks and push into workQueue
func (qm *ServerMgr) updateQueue() (err error) {
	qm.queueLock.Lock()
	defer qm.queueLock.Unlock()

	read_lock := qm.TaskMap.RLockNamed("updateQueue")
	for _, task := range qm.TaskMap.Map {
		if qm.isTaskReady(task) {
			if err := qm.taskEnQueue(task); err != nil {
				task.SetState(TASK_STAT_SUSPEND)
				jobid := getParentJobId(task.Id)
				qm.SuspendJob(jobid, fmt.Sprintf("failed enqueuing task %s, err=%s", task.Id, err.Error()), task.Id)
			}
		}
		//qm.updateTask(task)
	}
	qm.TaskMap.RUnlockNamed(read_lock)

	for _, id := range qm.workQueue.Clean() {
		jid, err := GetJobIdByWorkId(id)
		if err != nil {
			logger.Error("error: in updateQueue() workunit %s is nil, cannot get job id", id)
			continue
		}
		qm.SuspendJob(jid, fmt.Sprintf("workunit %s is nil", id), id)
		logger.Error("error: workunit %s is nil, suspending job %s", id, jid)
	}
	return
}

//handle feedback from a client about the execution of a workunit
func (qm *ServerMgr) handleWorkStatusChange(notice Notice) (err error) {
	workid := notice.WorkId
	status := notice.Status
	clientid := notice.ClientId
	computetime := notice.ComputeTime
	parts := strings.Split(workid, "_")
	taskid := fmt.Sprintf("%s_%s", parts[0], parts[1])
	jobid := parts[0]
	rank, err := strconv.Atoi(parts[2])
	if err != nil {
		err = fmt.Errorf("invalid workid %s", workid)
		return
	}

	client, ok := qm.GetClient(clientid, true)
	if ok {
		//delete(client.Current_work, workid)

		client.Current_work_delete(workid, true)

	}

	task, tok := qm.TaskMap.Get(taskid, true)
	if !tok {
		//task not existed, possible when job is deleted before the workunit done
		qm.workQueue.Delete(workid)
		return
	}
	work, wok := qm.workQueue.Get(workid)
	if (!wok) || (work.State != WORK_STAT_CHECKOUT) {
		return
	}

	var MAX_FAILURE int
	if task.Info.NoRetry == true {
		MAX_FAILURE = 1
	} else {
		MAX_FAILURE = conf.MAX_WORK_FAILURE
	}
	if task.GetState() == TASK_STAT_FAIL_SKIP {
		// A work unit for this task failed before this one arrived.
		// User set Skip=2 so the task was just skipped. Any subsiquent
		// workunits are just deleted...
		qm.workQueue.Delete(workid)
		return
	}

	// we want these to happen at end
	//defer qm.updateTask(task) // TODO use lock instead

	err = qm.updateTaskWorkStatus(task, rank, status)
	if err != nil {
		return
	}

	if status == WORK_STAT_DONE {
		//log event about work done (WD)
		logger.Event(event.WORK_DONE, "workid="+workid+";clientid="+clientid)
		//update client status

		client.Increment_total_completed()

		qm.AddClient(client, true)

		task.RemainWork -= 1
		task.ComputeTime += computetime
		if task.RemainWork == 0 {
			task.SetState(TASK_STAT_COMPLETED)
			task.CompletedDate = time.Now()
			for _, output := range task.Outputs {
				if _, err = output.DataUrl(); err != nil {
					return err
				}
				if hasFile := output.HasFile(); !hasFile {
					fmt.Errorf("Task %s, output %s missing shock file", taskid, output.FileName)
					return
				}
			}
			//log event about task done (TD)
			qm.FinalizeTaskPerf(task)
			logger.Event(event.TASK_DONE, "taskid="+taskid)
			//update the info of the job which the task is belong to, could result in deletion of the
			//task in the task map when the task is the final task of the job to be done.
			err = qm.updateJobTask(task) //task state QUEUED -> COMPELTED
		}
		//done, remove from the workQueue
		qm.workQueue.Delete(workid)
	} else if status == WORK_STAT_FAIL { //workunit failed, requeue or put it to suspend list
		logger.Event(event.WORK_FAIL, "workid="+workid+";clientid="+clientid)
		work.Failed += 1
		qm.workQueue.Put(work)
		if task.Skip == 2 && task.Skippable() {
			// user wants to skip task - not a real failure
			// don't mark client as failed
			task.RemainWork = 0 // not doing anything else...
			task.SetState(TASK_STAT_FAIL_SKIP)
			for _, output := range task.Outputs {
				output.GetFileSize()
				output.DataUrl()
			}
			qm.FinalizeTaskPerf(task)
			// log event about task skipped
			logger.Event(event.TASK_SKIPPED, "taskid="+taskid)
			//update the info of the job which the task is belong to, could result in deletion of the
			//task in the task map when the task is the final task of the job to be done.
			if err = qm.updateJobTask(task); err != nil { //task state QUEUED -> FAIL_SKIP
				return
			}
			// remove from the workQueue
			qm.workQueue.Delete(workid)
			return
		}
		if work.Failed < MAX_FAILURE {
			qm.workQueue.StatusChange(workid, WORK_STAT_QUEUED)
			logger.Event(event.WORK_REQUEUE, "workid="+workid)
		} else {
			//failure time exceeds limit, suspend workunit, task, job
			qm.workQueue.StatusChange(workid, WORK_STAT_SUSPEND)
			logger.Event(event.WORK_SUSPEND, "workid="+workid)
			err = qm.updateTaskWorkStatus(task, rank, WORK_STAT_SUSPEND)
			if err != nil {
				return
			}
			task.SetState(TASK_STAT_SUSPEND)

			reason := fmt.Sprintf("workunit %s failed %d time(s).", workid, MAX_FAILURE)
			if len(notice.Notes) > 0 {
				reason = reason + " msg from client:" + notice.Notes
			}
			if err := qm.SuspendJob(jobid, reason, workid); err != nil {
				logger.Error("error returned by SuspendJOb()" + err.Error())
			}
		}
		if client, ok := qm.GetClient(clientid, true); ok {
			client.LockNamed("ServerMgr/handleWorkStatusChange C")
			client.Append_Skip_work(workid, false)
			client.Increment_total_failed(false)
			client.Last_failed += 1 //last consecutive failures
			if client.Last_failed == conf.MAX_CLIENT_FAILURE {
				qm.SuspendClient(client.Id, client, false)
			}
			client.Unlock()
			qm.AddClient(client, true)
		}
	}
	return
}

func (qm *ServerMgr) GetJsonStatus() (status map[string]map[string]int) {
	queuing_work := qm.workQueue.Wait.Len()
	out_work := qm.workQueue.Checkout.Len()
	suspend_work := qm.workQueue.Suspend.Len()
	total_active_work := qm.workQueue.Len()
	total_task := 0
	queuing_task := 0
	started_task := 0
	pending_task := 0
	completed_task := 0
	suspended_task := 0
	skipped_task := 0
	fail_skip_task := 0
	read_lock := qm.TaskMap.RLockNamed("GetJsonStatus")
	for _, task := range qm.TaskMap.Map {
		total_task += 1
		task_state := task.GetState()
		switch task_state {
		case TASK_STAT_COMPLETED:
			completed_task += 1
		case TASK_STAT_PENDING:
			pending_task += 1
		case TASK_STAT_QUEUED:
			queuing_task += 1
		case TASK_STAT_INPROGRESS:
			started_task += 1
		case TASK_STAT_SUSPEND:
			suspended_task += 1
		case TASK_STAT_SKIPPED:
			skipped_task += 1
		case TASK_STAT_FAIL_SKIP:
			fail_skip_task += 1
		}
	}
	qm.TaskMap.RUnlockNamed(read_lock)

	total_task -= skipped_task // user doesn't see skipped tasks
	active_jobs := qm.lenActJobs()
	suspend_job := qm.lenSusJobs()
	total_job := active_jobs + suspend_job
	total_client := 0
	busy_client := 0
	idle_client := 0
	suspend_client := 0

	for _, client := range qm.clientMap.GetClients() {
		rlock := client.RLockNamed("GetJsonStatus")

		total_client += 1

		if client.Status == CLIENT_STAT_SUSPEND {
			suspend_client += 1
		} else if client.IsBusy(false) {
			busy_client += 1
		} else {
			idle_client += 1
		}
		client.RUnlockNamed(rlock)
	}

	jobs := map[string]int{
		"total":     total_job,
		"active":    active_jobs,
		"suspended": suspend_job,
	}
	tasks := map[string]int{
		"total":       total_task,
		"queuing":     queuing_task,
		"in-progress": started_task,
		"pending":     pending_task,
		"completed":   completed_task,
		"suspended":   suspended_task,
		"failed":      fail_skip_task,
	}
	workunits := map[string]int{
		"total":     total_active_work,
		"queuing":   queuing_work,
		"checkout":  out_work,
		"suspended": suspend_work,
	}
	clients := map[string]int{
		"total":     total_client,
		"busy":      busy_client,
		"idle":      idle_client,
		"suspended": suspend_client,
	}
	status = map[string]map[string]int{
		"jobs":      jobs,
		"tasks":     tasks,
		"workunits": workunits,
		"clients":   clients,
	}
	return
}

func (qm *ServerMgr) GetTextStatus() string {
	status := qm.GetJsonStatus()
	statMsg := "++++++++AWE server queue status++++++++\n" +
		fmt.Sprintf("total jobs ............... %d\n", status["jobs"]["total"]) +
		fmt.Sprintf("    active:           (%d)\n", status["jobs"]["active"]) +
		fmt.Sprintf("    suspended:        (%d)\n", status["jobs"]["suspended"]) +
		fmt.Sprintf("total tasks .............. %d\n", status["tasks"]["total"]) +
		fmt.Sprintf("    queuing:          (%d)\n", status["tasks"]["queuing"]) +
		fmt.Sprintf("    in-progress:      (%d)\n", status["tasks"]["in-progress"]) +
		fmt.Sprintf("    pending:          (%d)\n", status["tasks"]["pending"]) +
		fmt.Sprintf("    completed:        (%d)\n", status["tasks"]["completed"]) +
		fmt.Sprintf("    suspended:        (%d)\n", status["tasks"]["suspended"]) +
		fmt.Sprintf("    failed & skipped: (%d)\n", status["tasks"]["failed"]) +
		fmt.Sprintf("total workunits .......... %d\n", status["workunits"]["total"]) +
		fmt.Sprintf("    queuing:          (%d)\n", status["workunits"]["queuing"]) +
		fmt.Sprintf("    checkout:         (%d)\n", status["workunits"]["checkout"]) +
		fmt.Sprintf("    suspended:        (%d)\n", status["workunits"]["suspended"]) +
		fmt.Sprintf("total clients ............ %d\n", status["clients"]["total"]) +
		fmt.Sprintf("    busy:             (%d)\n", status["clients"]["busy"]) +
		fmt.Sprintf("    idle:             (%d)\n", status["clients"]["idle"]) +
		fmt.Sprintf("    suspend:          (%d)\n", status["clients"]["suspended"]) +
		fmt.Sprintf("---last update: %s\n\n", time.Now())
	return statMsg
}

//---end of mgr methods

//--workunit methds (servermgr implementation)
func (qm *ServerMgr) FetchDataToken(workid string, clientid string) (token string, err error) {
	//precheck if the client is registered
	client, ok := qm.GetClient(clientid, true)
	if !ok {
		return "", errors.New(e.ClientNotFound)
	}
	if client.Get_Status(true) == CLIENT_STAT_SUSPEND {
		return "", errors.New(e.ClientSuspended)
	}
	jobid, err := GetJobIdByWorkId(workid)
	if err != nil {
		return "", err
	}
	job, err := LoadJob(jobid)
	if err != nil {
		return "", err
	}
	token = job.GetDataToken()
	if token == "" {
		return token, errors.New("no data token set for workunit " + workid)
	}
	return token, nil
}

func (qm *ServerMgr) FetchPrivateEnvs(workid string, clientid string) (envs map[string]string, err error) {
	//precheck if the client is registered
	client, ok := qm.GetClient(clientid, true)
	if !ok {
		return nil, errors.New(e.ClientNotFound)
	}
	if client.Get_Status(true) == CLIENT_STAT_SUSPEND {
		return nil, errors.New(e.ClientSuspended)
	}
	jobid, err := GetJobIdByWorkId(workid)
	if err != nil {
		return nil, err
	}

	job, err := LoadJob(jobid)
	if err != nil {
		return nil, err
	}

	taskid, _ := GetTaskIdByWorkId(workid)

	idx := -1
	for i, t := range job.Tasks {
		if t.Id == taskid {
			idx = i
			break
		}
	}
	envs = job.Tasks[idx].Cmd.Environ.Private
	if envs == nil {
		return nil, errors.New("no private envs for workunit " + workid)
	}
	return envs, nil
}

func (qm *ServerMgr) SaveStdLog(workid string, logname string, tmppath string) (err error) {
	savedpath, err := getStdLogPathByWorkId(workid, logname)
	if err != nil {
		return err
	}
	os.Rename(tmppath, savedpath)
	return
}

func (qm *ServerMgr) GetReportMsg(workid string, logname string) (report string, err error) {
	logpath, err := getStdLogPathByWorkId(workid, logname)
	if err != nil {
		return "", err
	}
	if _, err := os.Stat(logpath); err != nil {
		return "", errors.New("log type '" + logname + "' not found")
	}

	content, err := ioutil.ReadFile(logpath)
	if err != nil {
		return "", err
	}
	return string(content), err
}

func deleteStdLogByTask(taskid string, logname string) (err error) {
	jobid, err := GetJobIdByTaskId(taskid)
	if err != nil {
		return err
	}
	var logdir string
	logdir, err = getPathByJobId(jobid)
	if err != nil {
		return
	}
	globpath := fmt.Sprintf("%s/%s_*.%s", logdir, taskid, logname)
	logfiles, err := filepath.Glob(globpath)
	if err != nil {
		return
	}
	for _, logfile := range logfiles {
		workid := strings.Split(filepath.Base(logfile), ".")[0]
		logger.Debug(2, "Deleted %s log for workunit %s", logname, workid)
		os.Remove(logfile)
	}
	return
}

func getStdLogPathByWorkId(workid string, logname string) (savedpath string, err error) {
	jobid, err := GetJobIdByWorkId(workid)
	if err != nil {
		return "", err
	}
	var logdir string
	logdir, err = getPathByJobId(jobid)
	if err != nil {
		return
	}
	savedpath = fmt.Sprintf("%s/%s.%s", logdir, workid, logname)
	return
}

//---task methods----

func (qm *ServerMgr) EnqueueTasksByJobId(jobid string, tasks []*Task) (err error) {
	for _, task := range tasks {
		qm.taskIn <- task
	}
	qm.CreateJobPerf(jobid)
	return
}

//---end of task methods

func (qm *ServerMgr) addTask(task *Task) (err error) {
	//for job recovery from db or for pseudo-task
	task_state := task.GetState()
	if (task_state == TASK_STAT_COMPLETED) || (task_state == TASK_STAT_PASSED) {
		qm.TaskMap.Add(task)
		return
	}
	if task.Skip == 1 && task.Skippable() {
		qm.skipTask(task)
		return
	}

	defer qm.TaskMap.Add(task)
	task.SetState(TASK_STAT_PENDING)

	if qm.isTaskReady(task) {
		if err = qm.taskEnQueue(task); err != nil {
			task.SetState(TASK_STAT_SUSPEND)
			jobid := getParentJobId(task.Id)
			qm.SuspendJob(jobid, fmt.Sprintf("failed in enqueuing task %s, err=%s", task.Id, err.Error()), task.Id)
			return err
		}
	}
	err = qm.updateJobTask(task) //task state INIT->PENDING
	return
}

func (qm *ServerMgr) skipTask(task *Task) (err error) {
	task.SetState(TASK_STAT_SKIPPED)
	task.RemainWork = 0
	//update job and queue info. Skipped task behaves as finished tasks
	if err = qm.updateJobTask(task); err != nil { //TASK state  -> SKIPPED
		return
	}
	logger.Event(event.TASK_SKIPPED, "taskid="+task.Id)
	return
}

//check whether a pending task is ready to enqueue (dependent tasks are all done)
func (qm *ServerMgr) isTaskReady(task *Task) (ready bool) {
	ready = false

	//skip if the belonging job is suspended
	jobid, _ := GetJobIdByTaskId(task.Id)
	if qm.isSusJob(jobid) {
		return false
	}

	task_state := task.GetState()
	if task_state == TASK_STAT_PENDING {
		ready = true
		for _, predecessor := range task.DependsOn {
			if pretask, ok := qm.TaskMap.Get(predecessor, true); ok {
				pretask_state := pretask.GetState()
				if pretask_state != TASK_STAT_COMPLETED &&
					pretask_state != TASK_STAT_PASSED &&
					pretask_state != TASK_STAT_SKIPPED &&
					pretask_state != TASK_STAT_FAIL_SKIP {
					ready = false
				}
			} else {
				logger.Error("warning: predecessor " + predecessor + " is unknown")
				ready = false
			}
		}
	}
	if task.Skip == 1 && task.Skippable() {
		qm.skipTask(task)
		ready = false
	}
	return
}

func (qm *ServerMgr) taskEnQueue(task *Task) (err error) {

	logger.Debug(2, "trying to enqueue task "+task.Id)

	if err := qm.locateInputs(task); err != nil {
		logger.Error("qmgr.taskEnQueue locateInputs:" + err.Error())
		return err
	}

	//create shock index on input nodes (if set in workflow document)
	if err := task.CreateIndex(); err != nil {
		logger.Error("qmgr.taskEnQueue CreateIndex:" + err.Error())
		return err
	}

	//init partition
	if err := task.InitPartIndex(); err != nil {
		logger.Error("qmgr.taskEnQueue InitPartitionIndex:" + err.Error())
		return err
	}

	if err := qm.createOutputNode(task); err != nil {
		logger.Error("qmgr.taskEnQueue createOutputNode:" + err.Error())
		return err
	}
	if err := qm.parseTask(task); err != nil {
		logger.Error("qmgr.taskEnQueue parseTask:" + err.Error())
		return err
	}
	task.SetState(TASK_STAT_QUEUED)
	task.CreatedDate = time.Now()
	task.StartedDate = time.Now() //to-do: will be changed to the time when the first workunit is checked out
	qm.updateJobTask(task)        //task status PENDING->QUEUED

	//log event about task enqueue (TQ)
	logger.Event(event.TASK_ENQUEUE, fmt.Sprintf("taskid=%s;totalwork=%d", task.Id, task.TotalWork))
	qm.CreateTaskPerf(task.Id)

	if IsFirstTask(task.Id) {
		jobid, _ := GetJobIdByTaskId(task.Id)
		UpdateJobState(jobid, JOB_STAT_QUEUED, []string{JOB_STAT_INIT, JOB_STAT_SUSPEND})
	}
	return
}

func (qm *ServerMgr) locateInputs(task *Task) (err error) {
	logger.Debug(2, "trying to locate Inputs of task "+task.Id)
	jobid, _ := GetJobIdByTaskId(task.Id)
	for _, io := range task.Inputs {
		name := io.FileName
		if io.Url == "" {
			preId := fmt.Sprintf("%s_%s", jobid, io.Origin)
			if preTask, ok := qm.TaskMap.Get(preId, true); ok {
				if preTask.State == TASK_STAT_SKIPPED ||
					preTask.State == TASK_STAT_FAIL_SKIP {
					// For now we know that skipped tasks have
					// just one input and one output. So we know
					// that we just need to change one file (this
					// may change in the future)
					//locateSkippedInput(qm, preTask, io)
				} else {
					outputs := preTask.Outputs
					for _, outio := range outputs {
						if outio.FileName == name {
							io.Node = outio.Node
						}
					}
				}
			}
		}
		logger.Debug(2, "processing input %s, %s", name, io.Node)
		if io.Node == "-" {
			err = fmt.Errorf("error in locate input for task %s, %s", task.Id, name)
			return
		}
		//need time out!
		if io.Node != "" && io.GetFileSize() < 0 {
			err = fmt.Errorf("task %s: input file %s not available", task.Id, name)
			return
		}
		logger.Debug(2, "inputs located %s, %s", name, io.Node)
	}
	// locate predata
	for _, io := range task.Predata {
		name := io.FileName
		logger.Debug(2, "processing predata %s, %s", name, io.Node)
		// only verify predata that is a shock node
		if (io.Node != "") && (io.Node != "-") && (io.GetFileSize() < 0) {
			// bad shock node
			if io.GetFileSize() < 0 {
				err = fmt.Errorf("task %s: predata file %s not available", task.Id, name)
				return
			}
			logger.Debug(2, "predata located %s, %s", name, io.Node)
		}
	}
	return
}

func (qm *ServerMgr) parseTask(task *Task) (err error) {
	workunits, err := task.ParseWorkunit()
	if err != nil {
		return err
	}
	for _, wu := range workunits {
		if err := qm.workQueue.Add(wu); err != nil {
			return err
		}
		qm.CreateWorkPerf(wu.Id)
	}
	return
}

func (qm *ServerMgr) createOutputNode(task *Task) (err error) {

	outputs := task.Outputs
	for _, io := range outputs {
		name := io.FileName
		if io.Type == "update" {
			// this an update output, it will update an existing shock node and not create a new one
			if (io.Node == "") || (io.Node == "-") {
				if io.Origin == "" {
					err = fmt.Errorf("(createOutputNode) update output %s in task %s is missing required origin", name, task.Id)
					return
				}
				var nodeid string
				nodeid, err = qm.locateUpdate(task.Id, name, io.Origin)
				if err != nil {
					err = fmt.Errorf("qm.locateUpdate in createOutputNode failed: %v", err)
					return
				}
				io.Node = nodeid
			}
			logger.Debug(2, "outout %s in task %s is an update of node %s", name, task.Id, io.Node)
		} else {
			// POST empty shock node for this output
			logger.Debug(2, "posting output Shock node for file %s in task %s", name, task.Id)
			var nodeid string
			nodeid, err = PostNodeWithToken(io, task.TotalWork, task.Info.DataToken)
			if err != nil {
				err = fmt.Errorf("PostNodeWithToken in createOutputNode failed: %v", err)
				return
			}
			io.Node = nodeid
			logger.Debug(2, "task %s: output Shock node created, node=%s", task.Id, nodeid)
		}
	}
	return
}

func (qm *ServerMgr) locateUpdate(taskid string, name string, origin string) (nodeid string, err error) {
	jobid, _ := GetJobIdByTaskId(taskid)
	preId := fmt.Sprintf("%s_%s", jobid, origin)
	logger.Debug(2, "task %s: trying to locate Node of update %s from task %s", taskid, name, preId)
	// scan outputs in origin task
	if preTask, ok := qm.TaskMap.Get(preId, true); ok {
		outputs := preTask.Outputs
		for _, outio := range outputs {
			if outio.FileName == name {
				return outio.Node, nil
			}
		}
	}
	err = fmt.Errorf("failed to locate Node for task %s / update %s from task %s", taskid, name, preId)
	return
}

func (qm *ServerMgr) updateTaskWorkStatus(task *Task, rank int, newstatus string) (err error) {

	rank_in_array := 0

	if rank > 0 {
		rank_in_array = rank - 1
	}

	if rank_in_array >= len(task.WorkStatus) {
		err = fmt.Errorf("index out of range, len(task.WorkStatus): %d rank_in_array: %d", len(task.WorkStatus), rank_in_array)
		return
	}

	task.WorkStatus[rank_in_array] = newstatus

	return
}

// show functions used in debug
func (qm *ServerMgr) ShowTasks() {
	logger.Debug(1, "current active tasks (%d)", qm.TaskMap.Len())
	for _, task := range qm.TaskMap.Map {
		logger.Debug(1, "taskid=%s;status=%s", task.Id, task.GetState())
	}
}

//---end of task methods---

//update job info when a task in that job changed to a new state
func (qm *ServerMgr) updateJobTask(task *Task) (err error) {
	parts := strings.Split(task.Id, "_")
	jobid := parts[0]
	job, err := LoadJob(jobid)
	if err != nil {
		return
	}
	remainTasks, err := job.UpdateTask(task) // sets job.State == completed if done
	if err != nil {
		return err
	}
	logger.Debug(2, "remaining tasks for task %s: %d", task.Id, remainTasks)

	if remainTasks == 0 { //job done
		qm.FinalizeJobPerf(jobid)
		qm.LogJobPerf(jobid)
		qm.removeActJob(jobid)
		//delete tasks in task map
		//delete from shock output flagged for deletion
		for _, task := range job.TaskList() {
			task.DeleteOutput()
			task.DeleteInput()
			qm.TaskMap.Delete(task.Id)
		}
		//set expiration from conf if not set
		nullTime := time.Time{}
		if job.Expiration == nullTime {
			expire := conf.GLOBAL_EXPIRE
			if val, ok := conf.PIPELINE_EXPIRE_MAP[job.Info.Pipeline]; ok {
				expire = val
			}
			if expire != "" {
				if err := job.SetExpiration(expire); err != nil {
					return err
				}
			}
		}
		//log event about job done (JD)
		logger.Event(event.JOB_DONE, "jobid="+job.Id+";name="+job.Info.Name+";project="+job.Info.Project+";user="+job.Info.User)
	}
	return
}

//update job/task states from "queued" to "in-progress" once the first workunit is checked out
func (qm *ServerMgr) UpdateJobTaskToInProgress(works []*Workunit) {
	for _, work := range works {
		job_was_inprogress := false
		task_was_inprogress := false
		taskid, _ := GetTaskIdByWorkId(work.Id)
		jobid, _ := GetJobIdByWorkId(work.Id)
		//Load job by id
		job, err := LoadJob(jobid)
		if err != nil {
			continue
		}
		//update job status
		if job.State == JOB_STAT_INPROGRESS {
			job_was_inprogress = true
		} else {
			job.State = JOB_STAT_INPROGRESS
			job.Info.StartedTme = time.Now()
			qm.UpdateJobPerfStartTime(jobid)
		}
		//update task status
		idx := -1
		for i, t := range job.Tasks {
			if t.Id == taskid {
				idx = i
				break
			}
		}
		if idx == -1 {
			continue
		}
		if job.Tasks[idx].State == TASK_STAT_INPROGRESS {
			task_was_inprogress = true
		} else {
			job.Tasks[idx].State = TASK_STAT_INPROGRESS
			qm.TaskMap.SetState(taskid, TASK_STAT_INPROGRESS)
			job.Tasks[idx].StartedDate = time.Now()
			qm.UpdateTaskPerfStartTime(taskid)
		}

		if !job_was_inprogress || !task_was_inprogress {
			job.Save()
		}
	}
}

func (qm *ServerMgr) IsJobRegistered(id string) bool {
	if qm.isActJob(id) {
		return true
	}
	if qm.isSusJob(id) {
		return true
	}
	return false
}

func (qm *ServerMgr) SuspendJob(jobid string, reason string, id string) (err error) {
	job, err := LoadJob(jobid)
	if err != nil {
		return
	}
	if id != "" {
		job.LastFailed = id
	}
	if err := job.UpdateState(JOB_STAT_SUSPEND, reason); err != nil {
		return err
	}
	qm.putSusJob(jobid)

	//suspend queueing workunits
	for _, workunit := range qm.workQueue.GetAll() {
		workid := workunit.Id
		if jobid == getParentJobId(workid) {
			qm.workQueue.StatusChange(workid, WORK_STAT_SUSPEND)
		}
	}

	//suspend parsed tasks
	for _, task := range job.Tasks {
		task_state := task.GetState()
		if task_state == TASK_STAT_QUEUED || task_state == TASK_STAT_INIT || task_state == TASK_STAT_INPROGRESS {
			qm.TaskMap.SetState(task.Id, TASK_STAT_SUSPEND)
			task.SetState(TASK_STAT_SUSPEND)
			job.UpdateTask(task)
		}
	}
	qm.LogJobPerf(jobid)
	qm.removeActJob(jobid)
	logger.Event(event.JOB_SUSPEND, "jobid="+jobid+";reason="+reason)
	return
}

func (qm *ServerMgr) DeleteJobByUser(jobid string, u *user.User, full bool) (err error) {
	job, err := LoadJob(jobid)
	if err != nil {
		return
	}
	// User must have delete permissions on job or be job owner or be an admin
	rights := job.Acl.Check(u.Uuid)
	if job.Acl.Owner != u.Uuid && rights["delete"] == false && u.Admin == false {
		return errors.New(e.UnAuth)
	}
	if err := job.UpdateState(JOB_STAT_DELETED, "deleted"); err != nil {
		return err
	}
	//delete queueing workunits
	for _, workunit := range qm.workQueue.GetAll() {
		workid := workunit.Id
		if jobid == getParentJobId(workid) {
			qm.workQueue.Delete(workid)
		}
	}
	//delete parsed tasks
	for i := 0; i < len(job.TaskList()); i++ {
		task_id := fmt.Sprintf("%s_%d", jobid, i)
		qm.TaskMap.Delete(task_id)
	}
	qm.removeActJob(jobid)
	qm.removeSusJob(jobid)

	// really delete it !
	if full {
		return job.Delete()
	} else {
		logger.Event(event.JOB_DELETED, "jobid="+jobid)
	}
	return
}

func (qm *ServerMgr) DeleteSuspendedJobsByUser(u *user.User, full bool) (num int) {
	for id := range qm.GetSuspendJobs() {
		if err := qm.DeleteJobByUser(id, u, full); err == nil {
			num += 1
		}
	}
	return
}

func (qm *ServerMgr) ResumeSuspendedJobsByUser(u *user.User) (num int) {
	for id := range qm.GetSuspendJobs() {
		if err := qm.ResumeSuspendedJobByUser(id, u); err == nil {
			num += 1
		}
	}
	return
}

//delete jobs in db with "queued" or "in-progress" state but not in the queue (zombie jobs) that user has access to
func (qm *ServerMgr) DeleteZombieJobsByUser(u *user.User, full bool) (num int) {
	dbjobs := new(Jobs)
	q := bson.M{}
	q["state"] = bson.M{"in": JOB_STATS_ACTIVE}
	if err := dbjobs.GetAll(q, "info.submittime", "asc"); err != nil {
		logger.Error("DeleteZombieJobs()->GetAllLimitOffset():" + err.Error())
		return
	}
	for _, dbjob := range *dbjobs {
		if !qm.isActJob(dbjob.Id) {
			if err := qm.DeleteJobByUser(dbjob.Id, u, full); err == nil {
				num += 1
			}
		}
	}
	return
}

//resubmit a suspended job if the user is authorized
func (qm *ServerMgr) ResumeSuspendedJobByUser(id string, u *user.User) (err error) {
	//Load job by id
	dbjob, err := LoadJob(id)
	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}

	// User must have write permissions on job or be job owner or be an admin
	rights := dbjob.Acl.Check(u.Uuid)
	if dbjob.Acl.Owner != u.Uuid && rights["write"] == false && u.Admin == false {
		return errors.New(e.UnAuth)
	}

	if dbjob.State != JOB_STAT_SUSPEND {
		return errors.New("job " + id + " is not in 'suspend' status")
	}

	if dbjob.RemainTasks < len(dbjob.Tasks) {
		dbjob.State = JOB_STAT_INPROGRESS
	} else {
		dbjob.State = JOB_STAT_QUEUED
	}
	dbjob.Resumed += 1
	dbjob.Save()

	qm.removeSusJob(id)
	qm.EnqueueTasksByJobId(dbjob.Id, dbjob.TaskList())
	return
}

//recover a job in db that is missing from queue (caused by server restarting)
func (qm *ServerMgr) RecoverJob(id string) (err error) {
	//Load job by id
	if qm.isActJob(id) {
		return errors.New("job " + id + " is already active")
	}
	dbjob, err := LoadJob(id)

	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}
	if dbjob.State == JOB_STAT_SUSPEND {
		qm.putSusJob(dbjob.Id)
	} else {
		if dbjob.State == JOB_STAT_COMPLETED || dbjob.State == JOB_STAT_DELETED {
			return errors.New("job is in " + dbjob.State + " state thus cannot be recovered")
		}
		for _, task := range dbjob.Tasks {
			task.Info = dbjob.Info
		}
		qm.EnqueueTasksByJobId(dbjob.Id, dbjob.TaskList())
	}

	logger.Debug(2, "Recovered job %s", id)
	return
}

//recover jobs not completed before awe-server restarts
func (qm *ServerMgr) RecoverJobs() (err error) {
	//Get jobs to be recovered from db whose states are "submitted"
	dbjobs := new(Jobs)
	q := bson.M{}
	q["state"] = bson.M{"$in": JOB_STATS_TO_RECOVER}
	if err := dbjobs.GetAll(q, "info.submittime", "asc"); err != nil {
		logger.Error("RecoverJobs()->GetAllLimitOffset():" + err.Error())
		return err
	}
	//Locate the job script and parse tasks for each job
	jobct := 0
	for _, dbjob := range *dbjobs {
		logger.Debug(2, "recovering %d: job=%s, state=%s", jobct, dbjob.Id, dbjob.State)
		if dbjob.State == JOB_STAT_SUSPEND {
			qm.putSusJob(dbjob.Id)
		} else {
			qm.EnqueueTasksByJobId(dbjob.Id, dbjob.TaskList())
		}
		jobct += 1
	}
	fmt.Printf("%d unfinished jobs recovered\n", jobct)
	return
}

//recompute job from specified task stage
func (qm *ServerMgr) RecomputeJob(jobid string, stage string) (err error) {
	if qm.isActJob(jobid) {
		return errors.New("job " + jobid + " is already active")
	}
	//Load job by id
	dbjob, err := LoadJob(jobid)
	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}
	if dbjob.State != JOB_STAT_COMPLETED && dbjob.State != JOB_STAT_SUSPEND {
		return errors.New("job " + jobid + " is not in 'completed' or 'suspend' status")
	}

	was_suspend := false
	if dbjob.State == JOB_STAT_SUSPEND {
		was_suspend = true
	}

	from_task_id := fmt.Sprintf("%s_%s", jobid, stage)
	remaintasks := 0
	found := false
	for _, task := range dbjob.Tasks {
		if task.Id == from_task_id {
			resetTask(task, dbjob.Info)
			remaintasks += 1
			found = true
		}
	}
	if !found {
		return errors.New("task not found:" + from_task_id)
	}
	for _, task := range dbjob.Tasks {
		if isAncestor(dbjob, task.Id, from_task_id) {
			resetTask(task, dbjob.Info)
			remaintasks += 1
		}
	}

	dbjob.Resumed += 1
	dbjob.RemainTasks = remaintasks
	if dbjob.RemainTasks < len(dbjob.Tasks) {
		dbjob.UpdateState(JOB_STAT_INPROGRESS, "recomputed from task "+from_task_id)
	} else {
		dbjob.UpdateState(JOB_STAT_QUEUED, "recomputed from task "+from_task_id)
	}

	if was_suspend {
		qm.removeSusJob(dbjob.Id)
	}
	qm.EnqueueTasksByJobId(dbjob.Id, dbjob.TaskList())

	logger.Debug(2, "Recomputed job %s from task %d", jobid, stage)
	return
}

//recompute job from beginning
func (qm *ServerMgr) ResubmitJob(jobid string) (err error) {
	if qm.isActJob(jobid) {
		return errors.New("job " + jobid + " is already active")
	}
	//Load job by id
	dbjob, err := LoadJob(jobid)
	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}
	if dbjob.State != JOB_STAT_COMPLETED && dbjob.State != JOB_STAT_SUSPEND {
		return errors.New("job " + jobid + " is not in 'completed' or 'suspend' status")
	}

	was_suspend := false
	if dbjob.State == JOB_STAT_SUSPEND {
		was_suspend = true
	}

	remaintasks := 0
	for _, task := range dbjob.Tasks {
		resetTask(task, dbjob.Info)
		remaintasks += 1
	}

	dbjob.Resumed += 1
	dbjob.RemainTasks = remaintasks
	dbjob.UpdateState(JOB_STAT_QUEUED, "restarted from the beginning")

	if was_suspend {
		qm.removeSusJob(dbjob.Id)
	}
	qm.EnqueueTasksByJobId(dbjob.Id, dbjob.TaskList())

	logger.Debug(2, "Restarted job %s from beginning", jobid)
	return
}

func resetTask(task *Task, info *Info) {
	task.Info = info
	task.SetState(TASK_STAT_PENDING)
	task.RemainWork = task.TotalWork
	task.ComputeTime = 0
	task.CompletedDate = time.Time{}
	// reset all inputs with an origin
	for _, input := range task.Inputs {
		if input.Origin != "" {
			input.Node = "-"
			input.Url = ""
			input.Size = 0
		}
	}
	// reset / delete all outputs
	for _, output := range task.Outputs {
		if dataUrl, _ := output.DataUrl(); dataUrl != "" {
			// delete dataUrl if is shock node
			if strings.HasSuffix(dataUrl, shock.DATA_SUFFIX) {
				if err := shock.ShockDelete(output.Host, output.Node, output.DataToken); err == nil {
					logger.Debug(2, "Deleted node %s from shock", output.Node)
				} else {
					logger.Error("resetTask: unable to deleted node %s from shock: %s", output.Node, err.Error())
				}
			}
		}
		output.Node = "-"
		output.Url = ""
		output.Size = 0
	}
	// delete all workunit logs
	for _, log := range conf.WORKUNIT_LOGS {
		deleteStdLogByTask(task.Id, log)
	}
}

func isAncestor(job *Job, taskId string, testId string) bool {
	if taskId == testId {
		return false
	}
	idx := -1
	for i, t := range job.Tasks {
		if t.Id == taskId {
			idx = i
			break
		}
	}
	if idx == -1 {
		return false
	}

	task := job.Tasks[idx]
	if len(task.DependsOn) == 0 {
		return false
	}
	if contains(task.DependsOn, testId) {
		return true
	} else {
		for _, t := range task.DependsOn {
			return isAncestor(job, t, testId)
		}
	}
	return false
}

//update job group
func (qm *ServerMgr) UpdateGroup(jobid string, newgroup string) (err error) {
	//update info in db
	dbjob, err := LoadJob(jobid)
	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}
	dbjob.Info.ClientGroups = newgroup
	for _, task := range dbjob.Tasks {
		task.Info.ClientGroups = newgroup
	}
	dbjob.Save()

	//update in-memory data structures
	for _, work := range qm.workQueue.GetForJob(jobid) {
		work.Info.ClientGroups = newgroup
		qm.workQueue.Put(work)
	}
	for _, task := range dbjob.Tasks {
		if mtask, ok := qm.TaskMap.Get(task.Id, true); ok {
			mtask.Info.ClientGroups = newgroup
			//qm.updateTask(mtask)
		}
	}
	return
}

func (qm *ServerMgr) UpdatePriority(jobid string, priority int) (err error) {
	//update info in db
	dbjob, err := LoadJob(jobid)
	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}
	dbjob.Info.Priority = priority
	for _, task := range dbjob.Tasks {
		task.Info.Priority = priority
	}
	dbjob.Save()

	//update in-memory data structures
	for _, work := range qm.workQueue.GetForJob(jobid) {
		work.Info.Priority = priority
		qm.workQueue.Put(work)
	}
	for _, task := range dbjob.Tasks {
		if mtask, ok := qm.TaskMap.Get(task.Id, true); ok {
			mtask.Info.Priority = priority
			//qm.updateTask(mtask)
		}
	}
	return
}

//---end of job methods

//---perf related methods
func (qm *ServerMgr) CreateJobPerf(jobid string) {
	if !qm.isActJob(jobid) {
		qm.putActJob(NewJobPerf(jobid))
	}
}

func (qm *ServerMgr) UpdateJobPerfStartTime(jobid string) {
	if perf, ok := qm.getActJob(jobid); ok {
		now := time.Now().Unix()
		perf.Start = now
		qm.putActJob(perf)
	}
	return
}

func (qm *ServerMgr) FinalizeJobPerf(jobid string) {
	if perf, ok := qm.getActJob(jobid); ok {
		now := time.Now().Unix()
		perf.End = now
		perf.Resp = now - perf.Queued
		qm.putActJob(perf)
	}
	return
}

func (qm *ServerMgr) CreateTaskPerf(taskid string) {
	jobid := getParentJobId(taskid)
	if perf, ok := qm.getActJob(jobid); ok {
		perf.Ptasks[taskid] = NewTaskPerf(taskid)
		qm.putActJob(perf)
	}
}

func (qm *ServerMgr) UpdateTaskPerfStartTime(taskid string) {
	jobid := getParentJobId(taskid)
	if jobperf, ok := qm.getActJob(jobid); ok {
		if taskperf, ok := jobperf.Ptasks[taskid]; ok {
			now := time.Now().Unix()
			taskperf.Start = now
			qm.putActJob(jobperf)
		}
	}
}

func (qm *ServerMgr) FinalizeTaskPerf(task *Task) {
	jobid := getParentJobId(task.Id)
	if jobperf, ok := qm.getActJob(jobid); ok {
		if taskperf, ok := jobperf.Ptasks[task.Id]; ok {
			now := time.Now().Unix()
			taskperf.End = now
			taskperf.Resp = now - taskperf.Queued

			for _, io := range task.Inputs {
				taskperf.InFileSizes = append(taskperf.InFileSizes, io.Size)
			}
			for _, io := range task.Outputs {
				taskperf.OutFileSizes = append(taskperf.OutFileSizes, io.Size)
			}
			qm.putActJob(jobperf)
			return
		}
	}
}

func (qm *ServerMgr) CreateWorkPerf(workid string) {
	if !conf.PERF_LOG_WORKUNIT {
		return
	}
	jobid := getParentJobId(workid)
	if jobperf, ok := qm.getActJob(jobid); ok {
		jobperf.Pworks[workid] = NewWorkPerf(workid)
		qm.putActJob(jobperf)
	}
}

func (qm *ServerMgr) FinalizeWorkPerf(workid string, reportfile string) (err error) {
	if !conf.PERF_LOG_WORKUNIT {
		return
	}
	workperf := new(WorkPerf)
	jsonstream, err := ioutil.ReadFile(reportfile)
	if err != nil {
		return err
	}
	if err := json.Unmarshal(jsonstream, workperf); err != nil {
		return err
	}
	jobid := getParentJobId(workid)
	jobperf, ok := qm.getActJob(jobid)
	if !ok {
		return errors.New("job perf not found:" + jobid)
	}
	if _, ok := jobperf.Pworks[workid]; !ok {
		return errors.New("work perf not found:" + workid)
	}

	workperf.Queued = jobperf.Pworks[workid].Queued
	workperf.Done = time.Now().Unix()
	workperf.Resp = workperf.Done - workperf.Queued
	jobperf.Pworks[workid] = workperf
	qm.putActJob(jobperf)
	os.Remove(reportfile)
	return
}

func (qm *ServerMgr) LogJobPerf(jobid string) {
	if perf, ok := qm.getActJob(jobid); ok {
		perfstr, _ := json.Marshal(perf)
		logger.Perf(string(perfstr)) //write into perf log
		dbUpsert(perf)               //write into mongodb
	}
}

//---end of perf related methods

func (qm *ServerMgr) FetchPrivateEnv(workid string, clientid string) (env map[string]string, err error) {
	//precheck if the client is registered
	client, ok := qm.GetClient(clientid, true)
	if !ok {
		return env, errors.New(e.ClientNotFound)
	}
	if client.Get_Status(true) == CLIENT_STAT_SUSPEND {
		return env, errors.New(e.ClientSuspended)
	}
	jobid, err := GetJobIdByWorkId(workid)
	if err != nil {
		return env, err
	}
	job, err := LoadJob(jobid)
	if err != nil {
		return env, err
	}
	taskid, err := GetTaskIdByWorkId(workid)
	env = job.GetPrivateEnv(taskid)
	return env, nil
}
