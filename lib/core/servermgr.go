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
	//"strconv"
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
	queueLock      sync.Mutex //only update one at a time
	lastUpdate     time.Time
	lastUpdateLock sync.RWMutex
	TaskMap        TaskMap
	taskIn         chan *Task //channel for receiving Task (JobController -> qmgr.Handler)
	ajLock         sync.RWMutex
	sjLock         sync.RWMutex
	actJobs        map[string]*JobPerf
	susJobs        map[string]bool
}

func NewServerMgr() *ServerMgr {
	return &ServerMgr{
		CQMgr: CQMgr{
			clientMap:    *NewClientMap(),
			workQueue:    NewWorkQueue(),
			suspendQueue: false,

			coReq:        make(chan CoReq, conf.COREQ_LENGTH), // number of clients that wait in queue to get a workunit. If queue is full, other client will be rejected and have to come back later again
			feedback:     make(chan Notice),
			coSem:        make(chan int, 1), //non-blocking buffered channel
			
		},
		lastUpdate: time.Now().Add(time.Second * -30),
		TaskMap:    *NewTaskMap(),
		taskIn:     make(chan *Task, 1024),
		actJobs:    map[string]*JobPerf{},
		susJobs:    map[string]bool{},
	}
}

//--------mgr methods-------

func (qm *ServerMgr) Lock()    {}
func (qm *ServerMgr) Unlock()  {}
func (qm *ServerMgr) RLock()   {}
func (qm *ServerMgr) RUnlock() {}

func (qm *ServerMgr) TaskHandle() {
	logger.Info("TaskHandle is starting")
	for {
		task := <-qm.taskIn

		task_id, err := task.GetId()
		if err != nil {
			logger.Error("(TaskHandle) %s", err.Error())
			task_id = "unknown"
		}

		logger.Debug(2, "ServerMgr/TaskHandle received task from channel taskIn, id=%s", task_id)
		qm.addTask(task)
	}
}

func (qm *ServerMgr) UpdateQueueLoop() {
	// TODO this may not be dynamic enough for small amounts of workunits, as they always have to wait
	for {
		qm.updateQueue()
		time.Sleep(30 * time.Second)
	}
}

func (qm *ServerMgr) ClientHandle() {
	logger.Info("(ServerMgr ClientHandle) starting")
	count := 0

	time.Sleep(3 * time.Second)

	for {
		//select {
		//case coReq := <-qm.coReq
		//logger.Debug(3, "(ServerMgr ClientHandle) try to pull work request")
		//coReq, err := qm.requestQueue.Pull()
		//for err != nil {
		//	time.Sleep(50 * time.Millisecond) // give clients time to put in requests or get a response
		//	time.Sleep(3 * time.Second)
		//	coReq, err = qm.requestQueue.Pull()
		//	logger.Debug(3, "(ServerMgr ClientHandle) waiting")
		//}
		//logger.Debug(3, "(ServerMgr ClientHandle) got work request")

		coReq := <-qm.coReq //written to in cqmgr.go
		count += 1
		request_start_time := time.Now()
		logger.Debug(3, "(ServerMgr ClientHandle) workunit checkout request received from client %s, Req=%v", coReq.fromclient, coReq)

		ok, err := qm.CQMgr.clientMap.Has(coReq.fromclient, true)
		if err != nil {
			logger.Warning("(ServerMgr ClientHandle) Could not get lock for client %s (%s)", coReq.fromclient, err.Error())
			continue
		}
		if !ok {
			logger.Error("(ServerMgr ClientHandle) Client %s not found. (It probably left in the mean-time)", coReq.fromclient)
			continue
		}

		var ack CoAck
		if qm.suspendQueue {
			// queue is suspended, return suspend error
			ack = CoAck{workunits: nil, err: errors.New(e.QueueSuspend)}
			logger.Debug(3, "(ServerMgr ClientHandle %s) nowworkunit: e.QueueSuspend", coReq.fromclient)
		} else {
			logger.Debug(3, "(ServerMgr ClientHandle %s) popWorks", coReq.fromclient)

			works, err := qm.popWorks(coReq)
			if err != nil {
				logger.Debug(3, "(ServerMgr ClientHandle) popWorks returned error: %s", err.Error())
			}
			logger.Debug(3, "(ServerMgr ClientHandle %s) popWorks done", coReq.fromclient)
			if err == nil {
				logger.Debug(3, "(ServerMgr ClientHandle %s) UpdateJobTaskToInProgress", coReq.fromclient)

				qm.UpdateJobTaskToInProgress(works)

				logger.Debug(3, "(ServerMgr ClientHandle %s) UpdateJobTaskToInProgress done", coReq.fromclient)
			}
			ack = CoAck{workunits: works, err: err}

			if len(works) > 0 {
				wu := works[0]

				logger.Debug(3, "(ServerMgr ClientHandle %s) workunit: %s", coReq.fromclient, wu.Id)
			} else {
				logger.Debug(3, "(ServerMgr ClientHandle %s) works is empty", coReq.fromclient)
			}
		}
		logger.Debug(3, "(ServerMgr ClientHandle %s) send response now", coReq.fromclient)

		start_time := time.Now()

		timer := time.NewTimer(20 * time.Second)

		select {
		case coReq.response <- ack:
			logger.Debug(3, "(ServerMgr ClientHandle %s) send workunit to client via response channel", coReq.fromclient)
		case <-timer.C:
			elapsed_time := time.Since(start_time)
			logger.Error("(ServerMgr ClientHandle %s) timed out after %s ", coReq.fromclient, elapsed_time)
			continue
		}
		logger.Debug(3, "(ServerMgr ClientHandle %s) done", coReq.fromclient)

		if count%10 == 0 { // use modulo to reduce number of log messages
			request_time_elapsed := time.Since(request_start_time)

			logger.Info("(ServerMgr ClientHandle) Responding to work request took %s", request_time_elapsed)
		}
	}
}

func (qm *ServerMgr) NoticeHandle() {
	logger.Info("(ServerMgr NoticeHandle) starting")
	for {
		notice := <-qm.feedback
		logger.Debug(3, "(ServerMgr NoticeHandle) got notice: workid=%s, status=%s, clientid=%s", notice.WorkId, notice.Status, notice.ClientId)
		if err := qm.handleWorkStatusChange(notice); err != nil {
			logger.Error("(NoticeHandle): " + err.Error())
		}
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
		//return qm.TaskMap.Map
		tasks, err := qm.TaskMap.GetTasks()
		if err != nil {
			return err
		}
		return tasks
	}
	if name == "work" {
		qm.ShowWorkQueue() // only if debug level is set
		return qm.workQueue.all.Map

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

	logger.Debug(3, "(updateQueue) wait for lock")
	qm.queueLock.Lock()
	defer qm.queueLock.Unlock()

	logger.Debug(3, "(updateQueue) starting")
	tasks, err := qm.TaskMap.GetTasks()
	if err != nil {
		return
	}
	logger.Debug(3, "(updateQueue) range tasks (%d)", len(tasks))
	for _, task := range tasks {
		task_id, err := task.GetId()
		if err != nil {
			return err
		}

		logger.Debug(3, "(updateQueue) task: %s", task_id)
		task_ready, err := qm.isTaskReady(task)
		if err != nil {
			logger.Error("(updateQueue) isTaskReady=%s error: %s", task_id, err.Error())
			continue
		}

		if task_ready {
			logger.Debug(3, "(updateQueue) task ready: %s", task_id)
			err = qm.taskEnQueue(task)
			if err != nil {
				_ = task.SetState(TASK_STAT_SUSPEND)
				job_id, _ := GetJobIdByTaskId(task_id)
				jerror := &JobError{
					TaskFailed:  task_id,
					ServerNotes: "failed enqueuing task, err=" + err.Error(),
					Status:      JOB_STAT_SUSPEND,
				}
				if err = qm.SuspendJob(job_id, jerror); err != nil {
					logger.Error("(updateQueue:SuspendJob) job_id=%s; err=%s", job_id, err.Error())
				}
				continue
			}
			logger.Debug(3, "(updateQueue) task enqueued: %s", task_id)
		} else {
			logger.Debug(3, "(updateQueue) task not ready: %s", task_id)
		}
	}

	logger.Debug(3, "(updateQueue) range qm.workQueue.Clean()")
	for _, id := range qm.workQueue.Clean() {
		job_id, err := GetJobIdByWorkId(id)
		if err != nil {
			logger.Error("(updateQueue) workunit %s is nil, cannot get job id", id)
			continue
		}
		task_id, err := GetTaskIdByWorkId(id)
		if err != nil {
			logger.Error("(updateQueue) workunit %s is nil, cannot get task id", id)
			continue
		}
		jerror := &JobError{
			WorkFailed:  id,
			TaskFailed:  task_id,
			ServerNotes: "workunit is nil",
			Status:      JOB_STAT_SUSPEND,
		}
		if err = qm.SuspendJob(job_id, jerror); err != nil {
			logger.Error("(updateQueue:SuspendJob) job_id=%s; err=%s", job_id, err.Error())
		}
		logger.Error("(updateQueue) workunit %s is nil, suspending job %s", id, job_id)
	}

	logger.Debug(3, "(updateQueue) ending")

	return
}

func RemoveWorkFromClient(client *Client, clientid string, workid string) (err error) {
	err = client.Current_work_delete(workid, true)
	if err != nil {
		return
	}

	work_length, err := client.Current_work_length(true)
	if err != nil {
		return
	}

	if work_length > 0 {
		logger.Error("(RemoveWorkFromClient) Client %s still has %d workunits, after delivering one workunit", clientid, work_length)

		current_work_ids, err := client.Get_current_work(true)
		if err != nil {
			return err
		}
		for _, work_id := range current_work_ids {
			_ = client.Current_work_delete(work_id, true)
		}

		work_length, err = client.Current_work_length(true)
		if err != nil {
			return err
		}
		if work_length > 0 {
			logger.Error("(RemoveWorkFromClient) Client still has work, even after everything should have been deleted.")
			return fmt.Errorf("(RemoveWorkFromClient) Client %s still has %d workunits", clientid, work_length)
		}
	}
	return
}

func (qm *ServerMgr) handleWorkStatDone(client *Client, clientid string, task *Task, task_id string, workid string, computetime int) (err error) {
	//log event about work done (WD)
	logger.Event(event.WORK_DONE, "workid="+workid+";clientid="+clientid)
	//update client status

	defer func() {
		//done, remove from the workQueue
		err = qm.workQueue.Delete(workid)
		if err != nil {
			return
		}
	}()

	client.Increment_total_completed()

	remain_work, xerr := task.IncrementRemainWork(-1, true)
	if xerr != nil {
		err = fmt.Errorf("(RemoveWorkFromClient:IncrementRemainWork) client=%s work=%s %s", clientid, workid, xerr.Error())
		return
	}

	err = task.IncrementComputeTime(computetime)
	if xerr != nil {
		err = fmt.Errorf("(RemoveWorkFromClient:IncrementComputeTime) client=%s work=%s %s", clientid, workid, xerr.Error())
		return
	}

	logger.Debug(3, "(RemoveWorkFromClient) remain_work: %d (%s)", remain_work, workid)

	if remain_work > 0 {
		return
	}

	// ******* LAST WORKUNIT ******

	// check file sizes of all outputs
	outputs_modified := false
	outputs := task.Outputs
	for _, io := range outputs {
		size, modified, xerr := io.GetFileSize()
		if xerr != nil {
			err = xerr
			logger.Error("task %s, err: %s", task_id, err.Error())
			yerr := task.SetState(TASK_STAT_SUSPEND)
			if yerr != nil {
				err = yerr
				return
			}
			return
		}

		if !modified {
			continue
		}
		outputs_modified = true
		logger.Debug(3, "New output file %s has size %d", io.FileName, size)
	}

	if outputs_modified {
		err = task.UpdateOutputs()
		if err != nil {
			return
		}
	}

	err = task.SetState(TASK_STAT_COMPLETED)
	if err != nil {
		return
	}

	outputs, xerr = task.GetOutputs()
	if xerr != nil {
		err = xerr
		return
	}

	for _, output := range outputs {
		if _, err = output.DataUrl(); err != nil {
			return
		}
		hasFile := output.HasFile()
		if !hasFile {
			err = fmt.Errorf("(RemoveWorkFromClient) task %s, output %s missing shock file", task_id, output.FileName)
			return
		}
	}

	//log event about task done (TD)
	qm.FinalizeTaskPerf(task)
	logger.Event(event.TASK_DONE, "task_id="+task_id)
	//update the info of the job which the task is belong to, could result in deletion of the
	//task in the task map when the task is the final task of the job to be done.
	err = qm.updateJobTask(task) //task state QUEUED -> COMPLETED

	return
}

//handle feedback from a client about the execution of a workunit
func (qm *ServerMgr) handleWorkStatusChange(notice Notice) (err error) {
	workid := notice.WorkId
	status := notice.Status
	clientid := notice.ClientId
	computetime := notice.ComputeTime
	notes := notice.Notes

	logger.Debug(3, "(handleWorkStatusChange) workid: %s status: %s client: %s", workid, status, clientid)

	// we should not get here, but if we do than end
	if status == WORK_STAT_DISCARDED {
		logger.Error("(handleWorkStatusChange) [warning] skip status change: workid=%s status=%s", workid, status)
		return
	}

	parts := strings.Split(workid, "_")
	task_id := fmt.Sprintf("%s_%s", parts[0], parts[1])
	job_id := parts[0]

	// *** Get Client
	client, ok, err := qm.GetClient(clientid, true)
	if err != nil {
		return
	}
	if !ok {
		return fmt.Errorf("(handleWorkStatusChange) client not found")
	}
	defer RemoveWorkFromClient(client, clientid, workid)

	// *** Get Task
	task, tok, err := qm.TaskMap.Get(task_id, true)
	if err != nil {
		return err
	}
	if !tok {
		//task not existed, possible when job is deleted before the workunit done
		logger.Error("Task %s for workunit %s not found", task_id, workid)
		qm.workQueue.Delete(workid)
		return fmt.Errorf("(handleWorkStatusChange) task %s for workunit %s not found", task_id, workid)
	}

	// *** Get workunit
	work, wok, err := qm.workQueue.Get(workid)
	if err != nil {
		return err
	}
	if !wok {
		return fmt.Errorf("(handleWorkStatusChange) workunit %s not found in workQueue", workid)
	}
	if work.State != WORK_STAT_CHECKOUT && work.State != WORK_STAT_RESERVED {
		return fmt.Errorf("(handleWorkStatusChange) workunit %s did not have state WORK_STAT_CHECKOUT or WORK_STAT_RESERVED (state is %s)", workid, work.State)
	}

	// *** update state of workunit
	if err = qm.workQueue.StatusChange("", work, status); err != nil {
		return err
	}

	if err = task.LockNamed("handleWorkStatusChange/noretry"); err != nil {
		return err
	}
	noretry := task.Info.NoRetry
	task.Unlock()

	var MAX_FAILURE int
	if noretry == true {
		MAX_FAILURE = 1
	} else {
		MAX_FAILURE = conf.MAX_WORK_FAILURE
	}

	task_state, err := task.GetState()
	if err != nil {
		return err
	}

	if task_state == TASK_STAT_FAIL_SKIP {
		// A work unit for this task failed before this one arrived.
		// User set Skip=2 so the task was just skipped. Any subsiquent
		// workunits are just deleted...
		qm.workQueue.Delete(workid)
		return fmt.Errorf("(handleWorkStatusChange) workunit %s failed due to skip", workid)
	}

	logger.Debug(3, "(handleWorkStatusChange) handling status %s", status)
	if status == WORK_STAT_DONE {
		if err = qm.handleWorkStatDone(client, clientid, task, task_id, workid, computetime); err != nil {
			return err
		}
	} else if status == WORK_STAT_FAILED_PERMANENT { // (special case !) failed and cannot be recovered
		logger.Event(event.WORK_FAILED, "workid="+workid+";clientid="+clientid)
		logger.Debug(3, "(handleWorkStatusChange) work failed (status=%s) workid=%s clientid=%s", status, workid, clientid)
		work.Failed += 1

		qm.workQueue.StatusChange(workid, work, WORK_STAT_FAILED_PERMANENT)

		if err = task.SetState(TASK_STAT_FAILED_PERMANENT); err != nil {
			return err
		}

		jerror := &JobError{
			ClientFailed: clientid,
			WorkFailed:   workid,
			TaskFailed:   task_id,
			ServerNotes:  "exit code 42 encountered",
			WorkNotes:    notes,
			AppError:     notice.Stderr,
			Status:       JOB_STAT_FAILED_PERMANENT,
		}
		if err = qm.SuspendJob(job_id, jerror); err != nil {
			logger.Error("(handleWorkStatusChange:SuspendJob) job_id=%s; err=%s", job_id, err.Error())
		}
	} else if status == WORK_STAT_ERROR { //workunit failed, requeue or put it to suspend list
		logger.Event(event.WORK_FAIL, "workid="+workid+";clientid="+clientid)
		logger.Debug(3, "(handleWorkStatusChange) work failed (status=%s) workid=%s clientid=%s", status, workid, clientid)
		work.Failed += 1

		if work.Failed < MAX_FAILURE {
			qm.workQueue.StatusChange(workid, work, WORK_STAT_QUEUED)
			logger.Event(event.WORK_REQUEUE, "workid="+workid)
		} else {
			//failure time exceeds limit, suspend workunit, task, job
			qm.workQueue.StatusChange(workid, work, WORK_STAT_SUSPEND)
			logger.Event(event.WORK_SUSPEND, "workid="+workid)

			if err = task.SetState(TASK_STAT_SUSPEND); err != nil {
				return err
			}

			jerror := &JobError{
				ClientFailed: clientid,
				WorkFailed:   workid,
				TaskFailed:   task_id,
				ServerNotes:  fmt.Sprintf("workunit failed %d time(s)", MAX_FAILURE),
				WorkNotes:    notes,
				AppError:     notice.Stderr,
				Status:       JOB_STAT_SUSPEND,
			}
			if err = qm.SuspendJob(job_id, jerror); err != nil {
				logger.Error("(handleWorkStatusChange:SuspendJob) job_id=%s; err=%s", job_id, err.Error())
			}
		}

		// Suspend client if needed
		client, ok, err := qm.GetClient(clientid, true)
		if err != nil {
			return err
		}
		if !ok {
			return fmt.Errorf(e.ClientNotFound)
		}
		if err = client.Append_Skip_work(workid, true); err != nil {
			return err
		}
		if err = client.Increment_total_failed(true); err != nil {
			return err
		}
		last_failed, err := client.Increment_last_failed(true)
		if err != nil {
			return err
		}
		if last_failed >= conf.MAX_CLIENT_FAILURE {
			qm.SuspendClient(clientid, client, true)
		}
	} else {
		return fmt.Errorf("No handler for workunit status '%s' implemented (allowd: %s, %s, %s)", status, WORK_STAT_DONE, WORK_STAT_FAILED_PERMANENT, WORK_STAT_ERROR)
	}
	return
}

func (qm *ServerMgr) GetJsonStatus() (status map[string]map[string]int, err error) {
	start := time.Now()
	queuing_work, err := qm.workQueue.Queue.Len()
	if err != nil {
		return
	}
	out_work, err := qm.workQueue.Checkout.Len()
	if err != nil {
		return
	}
	suspend_work, err := qm.workQueue.Suspend.Len()
	if err != nil {
		return
	}
	total_active_work, err := qm.workQueue.Len()
	if err != nil {
		return
	}
	elapsed := time.Since(start)
	logger.Debug(3, "time GetJsonStatus/Len: %s", elapsed)

	total_task := 0
	queuing_task := 0
	started_task := 0
	pending_task := 0
	completed_task := 0
	suspended_task := 0
	skipped_task := 0
	fail_skip_task := 0

	start = time.Now()
	task_list, err := qm.TaskMap.GetTasks()
	if err != nil {
		return
	}
	elapsed = time.Since(start)
	logger.Debug(3, "time GetJsonStatus/GetTasks: %s", elapsed)

	start = time.Now()
	for _, task := range task_list {
		total_task += 1
		task_state, xerr := task.GetState()
		if xerr != nil {
			err = xerr
			return
		}
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
	elapsed = time.Since(start)
	logger.Debug(3, "time GetJsonStatus/task_list: %s", elapsed)

	total_task -= skipped_task // user doesn't see skipped tasks
	active_jobs := qm.lenActJobs()
	suspend_job := qm.lenSusJobs()
	total_job := active_jobs + suspend_job
	total_client := 0
	busy_client := 0
	idle_client := 0
	suspend_client := 0

	start = time.Now()
	client_list, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}
	total_client = len(client_list)
	elapsed = time.Since(start)
	logger.Debug(3, "time GetJsonStatus/GetClients: %s", elapsed)

	start = time.Now()

	for _, client := range client_list {
		rlock, err := client.RLockNamed("GetJsonStatus")
		if err != nil {
			continue
		}

		busy, _ := client.IsBusy(false)
		status := client.Status

		client.RUnlockNamed(rlock)

		if status == CLIENT_STAT_SUSPEND {
			suspend_client += 1
		} else if busy {
			busy_client += 1
		} else {
			idle_client += 1
		}

	}
	elapsed = time.Since(start)
	logger.Debug(3, "time GetJsonStatus/client_list: %s", elapsed)

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
	status, _ := qm.GetJsonStatus() // TODO handle error
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
	client, ok, err := qm.GetClient(clientid, true)
	if err != nil {
		return
	}
	if !ok {
		return "", errors.New(e.ClientNotFound)
	}
	client_status, err := client.Get_Status(true)
	if err != nil {
		return
	}
	if client_status == CLIENT_STAT_SUSPEND {
		return "", errors.New(e.ClientSuspended)
	}
	jobid, err := GetJobIdByWorkId(workid)
	if err != nil {
		return "", err
	}
	job, err := GetJob(jobid)
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
	client, ok, err := qm.GetClient(clientid, true)
	if err != nil {
		return
	}
	if !ok {
		return nil, errors.New(e.ClientNotFound)
	}
	client_status, err := client.Get_Status(true)
	if err != nil {
		return
	}
	if client_status == CLIENT_STAT_SUSPEND {
		return nil, errors.New(e.ClientSuspended)
	}
	jobid, err := GetJobIdByWorkId(workid)
	if err != nil {
		return nil, err
	}

	job, err := GetJob(jobid)
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

func (qm *ServerMgr) EnqueueTasksByJobId(jobid string) (err error) {

	job, err := GetJob(jobid)
	if err != nil {
		return
	}

	tasks, err := job.GetTasks()
	if err != nil {
		return
	}

	for _, task := range tasks {
		qm.taskIn <- task
	}
	qm.CreateJobPerf(jobid)
	return
}

//---end of task methods

func (qm *ServerMgr) addTask(task *Task) (err error) {
	task_state, err := task.GetState()
	if err != nil {
		return
	}

	if (task_state == TASK_STAT_COMPLETED) || (task_state == TASK_STAT_PASSED) {
		qm.TaskMap.Add(task)
		return
	}

	defer qm.TaskMap.Add(task)
	err = task.SetState(TASK_STAT_PENDING)
	if err != nil {
		return
	}

	task_ready, err := qm.isTaskReady(task)
	if err != nil {
		return
	}
	if task_ready {
		err = qm.taskEnQueue(task)
		if err != nil {
			_ = task.SetState(TASK_STAT_SUSPEND)
			task_id, _ := task.GetId()
			job_id, _ := GetJobIdByTaskId(task_id)
			jerror := &JobError{
				TaskFailed:  task_id,
				ServerNotes: "failed in enqueuing task, err=" + err.Error(),
				Status:      JOB_STAT_SUSPEND,
			}
			if serr := qm.SuspendJob(job_id, jerror); serr != nil {
				logger.Error("(updateQueue:SuspendJob) job_id=%s; err=%s", job_id, serr.Error())
			}
			return err
		}
	}
	err = qm.updateJobTask(task) //task state INIT->PENDING
	return
}

//check whether a pending task is ready to enqueue (dependent tasks are all done)
// task is not locked
func (qm *ServerMgr) isTaskReady(task *Task) (ready bool, err error) {
	ready = false

	logger.Debug(3, "(isTaskReady) starting")

	task_state, err := task.GetStateNamed("isTaskReady")
	if err != nil {
		return
	}

	logger.Debug(3, "(isTaskReady) task state is %s", task_state)

	if task_state != TASK_STAT_PENDING {
		return
	}

	task_id, err := task.GetId()
	if err != nil {
		return
	}
	logger.Debug(3, "(isTaskReady) task_id: %s", task_id)

	//skip if the belonging job is suspended
	jobid, err := task.GetJobId()
	if err != nil {
		return
	}

	if qm.isSusJob(jobid) {
		return
	}

	logger.Debug(3, "(isTaskReady) GetDependsOn %s", task_id)
	deps, xerr := task.GetDependsOn()
	if xerr != nil {
		err = xerr
		return
	}

	logger.Debug(3, "(isTaskReady) range deps %s (%d)", task_id, len(deps))
	for _, predecessor := range deps {
		pretask, ok, yerr := qm.TaskMap.Get(predecessor, true)
		if yerr != nil {
			err = yerr

			return
		}

		if !ok {
			logger.Error("predecessor %s of task %s is unknown", predecessor, task_id)
			ready = false
			return
		}

		pretask_state, zerr := pretask.GetState()
		if zerr != nil {
			err = zerr

			return
		}

		if pretask_state != TASK_STAT_COMPLETED {
			return
		}

	}
	logger.Debug(3, "(isTaskReady) task %s is ready", task_id)

	modified := false
	for _, io := range task.Inputs {
		filename := io.FileName

		if io.Origin == "" {
			continue
		}

		preId := fmt.Sprintf("%s_%s", jobid, io.Origin)
		preTask, ok, xerr := qm.TaskMap.Get(preId, true)
		if xerr != nil {
			err = xerr
			return
		}
		if !ok {
			err = fmt.Errorf("Task %s not found", preId)
			return
		}

		pretask_state, zerr := preTask.GetState()
		if zerr != nil {
			err = zerr

			return
		}

		if pretask_state != TASK_STAT_COMPLETED {
			err = fmt.Errorf("pretask_state != TASK_STAT_COMPLETED  state: %s preId: %s", pretask_state, preId)
			return
		}

		// find matching output
		pretask_output, xerr := preTask.GetOutput(filename)
		if xerr != nil {
			err = xerr
			return
		}

		logger.Debug(3, "pretask_output size = %d, state = %s", pretask_output.Size, preTask.State)

		if io.Size != pretask_output.Size {
			io.Size = pretask_output.Size
			modified = true
		}

	}

	if modified {
		err = task.UpdateInputs()
		if err != nil {
			return
		}
	}

	ready = true

	return
}

// happens when task is ready
func (qm *ServerMgr) taskEnQueue(task *Task) (err error) {

	task_id, err := task.GetId()
	if err != nil {
		return
	}

	job_id, err := task.GetJobId()
	if err != nil {
		return
	}

	logger.Debug(2, "qmgr.taskEnQueue trying to enqueue task %s", task_id)

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
	if err := qm.CreateAndEnqueueWorkunits(task); err != nil {
		logger.Error("qmgr.taskEnQueue CreateAndEnqueueWorkunits:" + err.Error())
		return err
	}
	err = task.SetState(TASK_STAT_QUEUED)
	if err != nil {
		return
	}
	task.SetCreatedDate(time.Now())
	task.SetStartedDate(time.Now()) //TODO: will be changed to the time when the first workunit is checked out
	qm.updateJobTask(task)          //task status PENDING->QUEUED

	//log event about task enqueue (TQ)
	logger.Event(event.TASK_ENQUEUE, fmt.Sprintf("taskid=%s;totalwork=%d", task_id, task.TotalWork))
	qm.CreateTaskPerf(task_id)

	if IsFirstTask(task_id) {
		UpdateJobState(job_id, JOB_STAT_QUEUED, []string{JOB_STAT_INIT, JOB_STAT_SUSPEND})
	}

	return
}

func (qm *ServerMgr) locateInputs(task *Task) (err error) {
	task_id, err := task.GetId()
	if err != nil {
		return
	}
	logger.Debug(2, "trying to locate Inputs of task "+task_id)

	jobid, err := task.GetJobId()
	if err != nil {
		return
	}

	inputs_modified := false
	for _, io := range task.Inputs {
		filename := io.FileName
		if io.Url == "" {
			preId := fmt.Sprintf("%s_%s", jobid, io.Origin)
			preTask, ok, xerr := qm.TaskMap.Get(preId, true)
			if xerr != nil {
				err = xerr
				return
			}

			if ok {
				if preTask.State == TASK_STAT_SKIPPED ||
					preTask.State == TASK_STAT_FAIL_SKIP {
					// For now we know that skipped tasks have
					// just one input and one output. So we know
					// that we just need to change one file (this
					// may change in the future)
					//locateSkippedInput(qm, preTask, io)
				} else {

					output, xerr := preTask.GetOutput(filename)
					if xerr != nil {
						err = xerr
						return
					}

					if io.Node != output.Node {
						io.Node = output.Node
						inputs_modified = true
					}

				}
			}
		}
		logger.Debug(2, "(locateInputs) processing input %s, %s", filename, io.Node)
		if io.Node == "-" {
			err = fmt.Errorf("(locateInputs) error in locate input for task, no node id found. task_id: %s, input name: %s", task_id, filename)
			return
		}
		//need time out!
		_, modified, xerr := io.GetFileSize()
		if xerr != nil {
			err = fmt.Errorf("(locateInputs) task %s: input file %s GetFileSize returns: %s (DataToken len: %d)", task_id, filename, xerr.Error(), len(io.DataToken))
			return
		}
		if modified {
			inputs_modified = true
		}
		logger.Debug(3, "(locateInputs) (task=%s) input %s located, node=%s size=%d", task_id, filename, io.Node, io.Size)

	}
	if inputs_modified {
		err = task.UpdateInputs()
		if err != nil {
			return
		}
	}

	predata_modified := false
	// locate predata
	for _, io := range task.Predata {
		name := io.FileName
		logger.Debug(2, "processing predata %s, %s", name, io.Node)
		// only verify predata that is a shock node
		if (io.Node != "") && (io.Node != "-") {
			_, modified, xerr := io.GetFileSize()
			if xerr != nil {
				err = fmt.Errorf("task %s: input file %s GetFileSize returns: %s", task_id, name, xerr.Error())
				return
			}
			if modified {
				predata_modified = true
			}
			logger.Debug(2, "predata located %s, %s", name, io.Node)
		}
	}

	if predata_modified {
		err = task.UpdatePredata()
		if err != nil {
			return
		}
	}

	return
}

func (qm *ServerMgr) CreateAndEnqueueWorkunits(task *Task) (err error) {
	workunits, err := task.CreateWorkunits()
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

	modified := false

	outputs := task.Outputs
	for _, io := range outputs {
		name := io.FileName
		if io.Type == "update" {
			// this an update output, it will update an existing shock node and not create a new one (it will update metadata of the shock node)
			if (io.Node == "") || (io.Node == "-") {
				if io.Origin == "" {
					err = fmt.Errorf("(createOutputNode) update output %s in task %s is missing required origin", name, task.Id)
					return
				}
				var nodeid string
				nodeid, err = qm.locateUpdate(task, name, io.Origin) // TODO host missing ?
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

			sc := shock.ShockClient{Host: io.Host, Token: task.Info.DataToken}
			nodeid, err = sc.PostNodeWithToken(io.FileName, task.TotalWork)
			if err != nil {
				err = fmt.Errorf("PostNodeWithToken in createOutputNode failed: %v", err)
				return
			}
			io.Node = nodeid
			modified = true
			logger.Debug(2, "task %s: output Shock node created, node=%s", task.Id, nodeid)
		}
	}

	if modified {
		err = task.UpdateOutputs()
	}

	return
}

func (qm *ServerMgr) locateUpdate(task *Task, name string, origin string) (nodeid string, err error) {
	//jobid, _ := GetJobIdByTaskId(taskid)
	task_id := task.Id
	job_id := task.JobId
	preId := fmt.Sprintf("%s_%s", job_id, origin)
	logger.Debug(2, "task %s: trying to locate Node of update %s from task %s", task_id, name, preId)
	// scan outputs in origin task
	preTask, ok, err := qm.TaskMap.Get(preId, true)
	if err != nil {
		return
	}
	if !ok {
		err = fmt.Errorf("failed to locate Node for task %s / update %s from task %s", task_id, name, preId)
		return
	}
	outputs := preTask.Outputs
	for _, outio := range outputs {
		if outio.FileName == name {
			nodeid = outio.Node
			return
		}
	}

	return
}

// show functions used in debug
func (qm *ServerMgr) ShowTasks() {
	length, _ := qm.TaskMap.Len()

	logger.Debug(1, "current active tasks (%d)", length)
	tasks, err := qm.TaskMap.GetTasks()
	if err != nil {
		logger.Error("error: %s", err.Error())
	}
	for _, task := range tasks {
		state, err := task.GetState()
		if err != nil {
			state = "unknown"
		}
		logger.Debug(1, "taskid=%s;status=%s", task.Id, state)
	}
}

//---end of task methods---

//update job info when a task in that job changed to a new state
func (qm *ServerMgr) updateJobTask(task *Task) (err error) {
	//parts := strings.Split(task.Id, "_")
	//jobid := parts[0]
	jobid, err := task.GetJobId()
	if err != nil {
		return
	}
	job, err := GetJob(jobid)

	remainTasks, err := job.GetRemainTasks()
	if err != nil {
		return
	}
	logger.Debug(2, "remaining tasks for job %s: %d", task.Id, remainTasks)

	if remainTasks > 0 {
		return
	}

	job_state, err := job.GetState(true)
	if err != nil {
		return
	}
	if job_state == JOB_STAT_COMPLETED {
		err = fmt.Errorf("job state is already JOB_STAT_COMPLETED")
		return
	}

	err = job.SetState(JOB_STAT_COMPLETED)
	if err != nil {
		return
	}

	qm.FinalizeJobPerf(jobid)
	qm.LogJobPerf(jobid)
	qm.removeActJob(jobid)
	//delete tasks in task map
	//delete from shock output flagged for deletion

	modified := 0
	for _, task := range job.TaskList() {
		// delete nodes that have been flagged to be deleted
		modified += task.DeleteOutput()
		modified += task.DeleteInput()
		qm.TaskMap.Delete(task.Id)
	}

	if modified > 0 {
		// save only is something has changed
		job.Save() // TODO avoid this, try partial updates
	}

	//set expiration from conf if not set
	nullTime := time.Time{}

	job_expiration, xerr := dbGetJobFieldTime(jobid, "expiration")
	if xerr != nil {
		err = xerr
		return
	}

	if job_expiration == nullTime {
		expire := conf.GLOBAL_EXPIRE

		job_info_pipeline, xerr := dbGetJobFieldString(jobid, "info.pipeline")
		if xerr != nil {
			err = xerr
			return
		}

		if val, ok := conf.PIPELINE_EXPIRE_MAP[job_info_pipeline]; ok {
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

	return
}

//update job/task states from "queued" to "in-progress" once the first workunit is checked out
func (qm *ServerMgr) UpdateJobTaskToInProgress(works []*Workunit) {
	for _, work := range works {
		//job_was_inprogress := false
		//task_was_inprogress := false
		taskid, _ := GetTaskIdByWorkId(work.Id)
		jobid, _ := GetJobIdByWorkId(work.Id)

		// get job state
		job_state, err := dbGetJobFieldString(jobid, "state")

		//update job status
		if job_state != JOB_STAT_INPROGRESS {

			dbUpdateJobState(jobid, JOB_STAT_INPROGRESS, "")

			DbUpdateJobField(jobid, "info.startedtime", time.Now())

			qm.UpdateJobPerfStartTime(jobid)
		}

		task, ok, err := qm.TaskMap.Get(taskid, true)
		if err != nil {
			logger.Error("(UpdateJobTaskToInProgress) %s", err.Error())
			continue
		}
		if !ok {
			logger.Error("(UpdateJobTaskToInProgress) task %s not found", taskid)
			continue
		}

		task_state, err := task.GetState()
		if err != nil {
			logger.Error("(UpdateJobTaskToInProgress) dbGetJobTaskField: %s", err.Error())
			continue
		}

		if task_state != TASK_STAT_INPROGRESS {
			err := task.SetState(TASK_STAT_INPROGRESS)
			if err != nil {
				logger.Error("(UpdateJobTaskToInProgress) could not update task %s", taskid)
				continue
			}
			qm.UpdateTaskPerfStartTime(taskid)
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

// use for JOB_STAT_SUSPEND and JOB_STAT_FAILED_PERMANENT
func (qm *ServerMgr) SuspendJob(jobid string, jerror *JobError) (err error) {
	job, err := GetJob(jobid)
	if err != nil {
		return
	}

	err = job.SetState(jerror.Status)
	if err != nil {
		return
	}
	if jerror.Status == JOB_STAT_SUSPEND {
		qm.putSusJob(jobid)
	}

	// set error struct
	err = job.SetError(jerror)
	if err != nil {
		return
	}

	//suspend queueing workunits
	workunit_list, err := qm.workQueue.GetAll()
	if err != nil {
		return err
	}

	new_work_state := WORK_STAT_SUSPEND
	new_task_state := TASK_STAT_SUSPEND
	this_event := event.JOB_SUSPEND
	if jerror.Status == JOB_STAT_FAILED_PERMANENT {
		new_work_state = WORK_STAT_FAILED_PERMANENT
		new_task_state = TASK_STAT_FAILED_PERMANENT
		this_event = event.JOB_FAILED_PERMANENT
	}

	// update all workunits
	for _, workunit := range workunit_list {
		workid := workunit.Id
		parentid, _ := GetJobIdByWorkId(workid)
		if jobid == parentid {
			qm.workQueue.StatusChange(workid, nil, new_work_state)
		}
	}

	//suspend parsed tasks
	for _, task := range job.Tasks {
		task_state, err := task.GetState()
		if err != nil {
			continue
		}
		if task_state == TASK_STAT_QUEUED || task_state == TASK_STAT_INIT || task_state == TASK_STAT_INPROGRESS {
			err = task.SetState(new_task_state)
			if err != nil {
				logger.Error("(SuspendJob) : %s", err.Error())
				continue
			}
		}
	}
	qm.LogJobPerf(jobid)
	qm.removeActJob(jobid)

	// log event and reason
	var reason string
	if jerror.ServerNotes != "" {
		reason = jerror.ServerNotes
	} else if jerror.WorkNotes != "" {
		reason = jerror.WorkNotes
	}
	logger.Event(this_event, "jobid="+jobid+";reason="+reason)
	return
}

func (qm *ServerMgr) DeleteJobByUser(jobid string, u *user.User, full bool) (err error) {
	job, err := GetJob(jobid)
	if err != nil {
		return
	}
	// User must have delete permissions on job or be job owner or be an admin
	rights := job.Acl.Check(u.Uuid)
	if job.Acl.Owner != u.Uuid && rights["delete"] == false && u.Admin == false {
		return errors.New(e.UnAuth)
	}
	if err := job.SetState(JOB_STAT_DELETED); err != nil {
		return err
	}
	//delete queueing workunits
	workunit_list, err := qm.workQueue.GetAll()
	if err != nil {
		return
	}
	for _, workunit := range workunit_list {
		workid := workunit.Id
		parentid, _ := GetJobIdByWorkId(workid)
		if jobid == parentid {
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
	dbjob, err := GetJob(id)
	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}

	job_state, err := dbjob.GetState(true)
	if err != nil {
		return
	}

	// User must have write permissions on job or be job owner or be an admin
	rights := dbjob.Acl.Check(u.Uuid)
	if dbjob.Acl.Owner != u.Uuid && rights["write"] == false && u.Admin == false {
		return errors.New(e.UnAuth)
	}

	if job_state != JOB_STAT_SUSPEND {
		return errors.New("job " + id + " is not in 'suspend' status")
	}

	remain_tasks, err := dbjob.GetRemainTasks()
	if err != nil {
		return
	}

	if remain_tasks < len(dbjob.Tasks) {
		dbjob.SetState(JOB_STAT_INPROGRESS)
	} else {
		dbjob.SetState(JOB_STAT_QUEUED)
	}

	err = dbjob.IncrementResumed(1)
	if err != nil {
		return
	}

	qm.removeSusJob(id)
	qm.EnqueueTasksByJobId(dbjob.Id)
	return
}

//recover a job in db that is missing from queue (caused by server restarting)
func (qm *ServerMgr) RecoverJob(id string) (err error) {
	//Load job by id
	if qm.isActJob(id) {
		return errors.New("job " + id + " is already active")
	}
	dbjob, err := GetJob(id)

	job_state, err := dbjob.GetState(true)
	if err != nil {
		return
	}

	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}
	if job_state == JOB_STAT_SUSPEND {
		qm.putSusJob(id)
	} else {
		if job_state == JOB_STAT_COMPLETED || job_state == JOB_STAT_DELETED {
			return errors.New("job is in " + job_state + " state thus cannot be recovered")
		}
		tasks, xerr := dbjob.GetTasks()
		if xerr != nil {
			err = xerr
			return
		}
		for _, task := range tasks {
			task.Info = dbjob.Info // in-memory only
		}
		qm.EnqueueTasksByJobId(id)
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
	if conf.RECOVER_MAX > 0 {
		logger.Info("Recover %d jobs...", conf.RECOVER_MAX)
		if _, err := dbjobs.GetPaginated(q, conf.RECOVER_MAX, 0, "info.priority", "desc"); err != nil {
			logger.Error("RecoverJobs()->GetPaginated():" + err.Error())
			return err
		}
	} else {
		logger.Info("Recover all jobs")
		if err := dbjobs.GetAll(q, "info.submittime", "asc"); err != nil {
			logger.Error("RecoverJobs()->GetAll():" + err.Error())
			return err
		}
	}
	//Locate the job script and parse tasks for each job
	jobct := 0
	for _, dbjob := range *dbjobs {
		logger.Debug(2, "recovering %d: job=%s, state=%s", jobct, dbjob.Id, dbjob.State)

		job_state, err := dbjob.GetState(true)
		if err != nil {
			logger.Error(err.Error())
			continue
		}

		// Directly after AWE server restart no job can be in progress. (Unless we add this as a feature))
		if job_state == JOB_STAT_INPROGRESS {
			err = DbUpdateJobField(dbjob.Id, "state", JOB_STAT_QUEUED)
			if err != nil {
				logger.Error("error while recover: " + err.Error())
				continue
			}
			err = dbjob.SetState(JOB_STAT_QUEUED)
			if err != nil {
				logger.Error(err.Error())
				continue
			}
		}

		if job_state == JOB_STAT_SUSPEND {
			qm.putSusJob(dbjob.Id)
		} else {
			qm.EnqueueTasksByJobId(dbjob.Id)
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
	dbjob, err := GetJob(jobid)
	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}

	job_state, err := dbjob.GetState(true)
	if err != nil {
		return
	}

	if job_state != JOB_STAT_COMPLETED && job_state != JOB_STAT_SUSPEND {
		return errors.New("job " + jobid + " is not in 'completed' or 'suspend' status")
	}

	was_suspend := false
	if job_state == JOB_STAT_SUSPEND {
		was_suspend = true
	}

	from_task_id := fmt.Sprintf("%s_%s", jobid, stage)
	remaintasks := 0
	found := false

	tasks, err := dbjob.GetTasks()
	if err != nil {
		return
	}

	for _, task := range tasks {
		task_id, xerr := task.GetId()
		if xerr != nil {
			return xerr
		}

		if task_id == from_task_id {
			resetTask(task, dbjob.Info)
			remaintasks += 1
			found = true
		}
	}
	if !found {
		return errors.New("task not found:" + from_task_id)
	}
	for _, task := range tasks {
		task_id, xerr := task.GetId()
		if xerr != nil {
			return xerr
		}
		if isAncestor(dbjob, task_id, from_task_id) {
			resetTask(task, dbjob.Info)
			remaintasks += 1
		}
	}

	err = dbjob.IncrementResumed(1)
	if err != nil {
		return
	}
	err = dbjob.SetRemainTasks(remaintasks)
	if err != nil {
		return
	}

	var new_state = ""
	if remaintasks < len(tasks) {
		new_state = JOB_STAT_INPROGRESS
	} else {
		new_state = JOB_STAT_QUEUED
	}
	dbjob.SetState(new_state)

	if was_suspend {
		qm.removeSusJob(jobid)
	}
	qm.EnqueueTasksByJobId(jobid)

	logger.Debug(2, "Recomputed job %s from task %d", jobid, stage)
	return
}

//recompute job from beginning
func (qm *ServerMgr) ResubmitJob(jobid string) (err error) {
	if qm.isActJob(jobid) {
		return errors.New("job " + jobid + " is already active")
	}
	//Load job by id
	job, err := GetJob(jobid)
	if err != nil {
		return errors.New("failed to load job " + err.Error())
	}

	job_state, err := job.GetState(true)
	if err != nil {
		return
	}

	if job_state != JOB_STAT_COMPLETED && job_state != JOB_STAT_SUSPEND {
		return errors.New("job " + jobid + " is not in 'completed' or 'suspend' status")
	}

	was_suspend := false
	if job_state == JOB_STAT_SUSPEND {
		was_suspend = true
	}

	err = job.IncrementResumed(1)
	if err != nil {
		return
	}
	err = job.SetState(JOB_STAT_QUEUED)
	if err != nil {
		return
	}

	if was_suspend {
		qm.removeSusJob(jobid)
	}

	err = qm.EnqueueTasksByJobId(jobid)
	if err != nil {
		return
	}

	logger.Debug(2, "Restarted job %s from beginning", jobid)
	return
}

// TODO Lock !!!!
func resetTask(task *Task, info *Info) {
	task.Info = info
	_ = task.SetState(TASK_STAT_PENDING)
	_ = task.SetRemainWork(task.TotalWork, false) // TODO err

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

//update tokens for in-memory data structures
func (qm *ServerMgr) UpdateQueueToken(job *Job) (err error) {
	for _, task := range job.Tasks {
		mtask, ok, err := qm.TaskMap.Get(task.Id, true)
		if err != nil {
			return err
		}
		if ok {
			mtask.setTokenForIO(true)
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
	jobid, _ := GetJobIdByTaskId(taskid)
	if perf, ok := qm.getActJob(jobid); ok {
		perf.Ptasks[taskid] = NewTaskPerf(taskid)
		qm.putActJob(perf)
	}
}

func (qm *ServerMgr) UpdateTaskPerfStartTime(taskid string) {
	jobid, _ := GetJobIdByTaskId(taskid)
	if jobperf, ok := qm.getActJob(jobid); ok {
		if taskperf, ok := jobperf.Ptasks[taskid]; ok {
			now := time.Now().Unix()
			taskperf.Start = now
			qm.putActJob(jobperf)
		}
	}
}

func (qm *ServerMgr) FinalizeTaskPerf(task *Task) {
	jobid, _ := GetJobIdByTaskId(task.Id)
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
	jobid, _ := GetJobIdByWorkId(workid)
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
	jobid, _ := GetJobIdByWorkId(workid)
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
	client, ok, err := qm.GetClient(clientid, true)
	if err != nil {
		return
	}
	if !ok {
		return env, errors.New(e.ClientNotFound)
	}
	client_status, err := client.Get_Status(true)
	if err != nil {
		return
	}
	if client_status == CLIENT_STAT_SUSPEND {
		return env, errors.New(e.ClientSuspended)
	}
	jobid, err := GetJobIdByWorkId(workid)
	if err != nil {
		return env, err
	}
	//job, err := GetJob(jobid)
	//if err != nil {
	//	return env, err
	//}
	taskid, err := GetTaskIdByWorkId(workid)
	//env = job.GetPrivateEnv(taskid)

	env, err = dbGetPrivateEnv(jobid, taskid)
	if err != nil {
		return
	}

	return env, nil
}
