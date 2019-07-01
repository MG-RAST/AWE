package core

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"
	shock "github.com/MG-RAST/go-shock-client"
	"github.com/davecgh/go-spew/spew"
	"github.com/robertkrimen/otto"
	"gopkg.in/mgo.v2/bson"
)

type jQueueShow struct {
	Active  map[string]*JobPerf `bson:"active" json:"active"`
	Suspend map[string]bool     `bson:"suspend" json:"suspend"`
}

// ServerMgr _
type ServerMgr struct {
	CQMgr
	queueLock      sync.Mutex //only update one at a time
	lastUpdate     time.Time
	lastUpdateLock sync.RWMutex
	TaskMap        TaskMap
	ajLock         sync.RWMutex
	actJobs        map[string]*JobPerf
}

// NewServerMgr _
func NewServerMgr() *ServerMgr {
	return &ServerMgr{
		CQMgr: CQMgr{
			clientMap:    *NewClientMap(),
			workQueue:    NewWorkQueue(),
			suspendQueue: false,

			coReq:    make(chan CheckoutRequest, conf.COREQ_LENGTH), // number of clients that wait in queue to get a workunit. If queue is full, other client will be rejected and have to come back later again
			feedback: make(chan Notice),
			coSem:    make(chan int, 1), //non-blocking buffered channel

		},
		lastUpdate: time.Now().Add(time.Second * -30),
		TaskMap:    *NewTaskMap(),
		actJobs:    map[string]*JobPerf{},
	}
}

//--------mgr methods-------

// Lock _
func (qm *ServerMgr) Lock() {}

// Unlock _
func (qm *ServerMgr) Unlock() {}

// RLock _
func (qm *ServerMgr) RLock() {}

// RUnlock _
func (qm *ServerMgr) RUnlock() {}

// UpdateQueueLoop _
func (qm *ServerMgr) UpdateQueueLoop() {
	// TODO this may not be dynamic enough for small amounts of workunits, as they always have to wait

	var err error
	for {

		err = qm.updateWorkflowInstancesMap()
		if err != nil {
			logger.Error("(UpdateQueueLoop) updateWorkflowInstancesMap() returned: %s", err.Error())
			err = nil
		}

		// *** task ***

		logTimes := false
		start := time.Now()
		err = qm.updateQueue(logTimes)
		if err != nil {
			logger.Error("(UpdateQueueLoop) updateQueue returned: %s", err.Error())
			err = nil
		}
		elapsed := time.Since(start)        // type Duration
		elapsedSeconds := elapsed.Seconds() // type float64

		var sleeptime time.Duration
		if elapsedSeconds <= 1 {
			sleeptime = 1 * time.Second // wait at least 1 second
		} else if elapsedSeconds > 1 && elapsedSeconds < 30 {
			sleeptime = elapsed
		} else {
			sleeptime = 30 * time.Second // wait at most 30 seconds
		}

		logger.Debug(3, "(UpdateQueueLoop) elapsed: %s (sleeping for %s)", elapsed, sleeptime)

		time.Sleep(sleeptime)

	}
}

func (qm *ServerMgr) Is_WI_ready(job *Job, wi *WorkflowInstance) (ready bool, reason string, err error) {
	logger.Debug(3, "(Is_WI_ready) start")
	if wi.Inputs != nil {
		ready = true
		return
	}

	cwl_workflow := wi.Workflow
	if cwl_workflow == nil {
		err = fmt.Errorf("Is_WI_ready wi.Workflow==nil")
		return
	}

	//fmt.Println("cwl_workflow.Inputs:")
	//spew.Dump(cwl_workflow.Inputs)

	//fmt.Println("wi.LocalID: " + wi.LocalID)
	parentWorkflowInstanceName := path.Dir(wi.LocalID)

	if parentWorkflowInstanceName == "." {
		return
	}
	logger.Debug(3, "(Is_WI_ready) non-main")

	//fmt.Println("parentWorkflowInstanceName: " + parentWorkflowInstanceName)

	var parentWorkflowInstance *WorkflowInstance
	var ok bool
	parentWorkflowInstance, ok, err = job.GetWorkflowInstance(parentWorkflowInstanceName, true)
	if err != nil {
		err = fmt.Errorf("(Is_WI_ready) job.GetWorkflowInstance returned: %s", err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(Is_WI_ready) parent workflow_instance %s not found", parentWorkflowInstanceName)
		return
	}

	_ = parentWorkflowInstance
	_ = ok

	parent_step := wi.ParentStep

	if parent_step == nil {
		err = fmt.Errorf("(Is_WI_ready)  wi.ParentStep==nil")
		return
	}

	parent_workflow_input_map := parentWorkflowInstance.Inputs.GetMap()

	context := job.WorkflowContext

	ready, reason, err = qm.areSourceGeneratorsReady(parent_step, job, parentWorkflowInstance)
	if err != nil {
		err = fmt.Errorf("(Is_WI_ready) areSourceGeneratorsReady returned: %s", err.Error())
		return
	}

	if !ready {
		return
	}

	// err = qm.GetDependencies(job, parentWorkflowInstance, parent_workflow_input_map, step, context)
	// if err != nil {
	// 	err = fmt.Errorf("(Is_WI_ready) GetDependencies returned: %s", err.Error())
	// }

	// panic("done")

	//var workunit_input_map map[string]cwl.CWLType
	var workunit_input_map cwl.JobDocMap
	workunit_input_map, ok, reason, err = qm.GetStepInputObjects(job, parentWorkflowInstance, parent_workflow_input_map, parent_step, context, "Is_WI_ready")
	if err != nil {
		err = fmt.Errorf("(Is_WI_ready) GetStepInputObjects returned: %s", err.Error())
		return
	}

	//fmt.Println("workunit_input_map:")
	//spew.Dump(workunit_input_map)

	var workunit_input_array cwl.Job_document
	workunit_input_array, err = workunit_input_map.GetArray()

	wi.Inputs = workunit_input_array

	ready = true

	return

}

func (qm *ServerMgr) updateWorkflowInstancesMapTask(wi *WorkflowInstance) (err error) {

	wi_local_id := wi.LocalID
	wiState, _ := wi.GetState(true)

	logger.Debug(3, "(updateWorkflowInstancesMapTask) start: %s state: %s", wi.LocalID, wiState)

	if wiState == WIStatePending {

		jobid := wi.JobID
		//workflow_def_str := wi.Workflow_Definition

		var job *Job
		job, err = GetJob(jobid)
		if err != nil {
			err = fmt.Errorf("(updateWorkflowInstancesMapTask) GetJob failed: %s", err.Error())
			return

		}

		context := job.WorkflowContext

		// get workflow

		var cwl_workflow *cwl.Workflow
		if wi.Workflow == nil {

			cwl_workflow, err = wi.GetWorkflow(context)
			if err != nil {
				err = fmt.Errorf("(updateWorkflowInstancesMapTask) GetWorkflow failed: %s", err.Error())
				return
			}
			if cwl_workflow == nil {
				err = fmt.Errorf("(updateWorkflowInstancesMapTask) a) cwl_workflow == nil")
				return
			}
			wi.Workflow = cwl_workflow
		} else {
			cwl_workflow = wi.Workflow
		}
		if cwl_workflow == nil {
			err = fmt.Errorf("(updateWorkflowInstancesMapTask) b) cwl_workflow == nil")
			return
		}

		if cwl_workflow.Steps == nil {
			err = fmt.Errorf("(updateWorkflowInstancesMapTask) cwl_workflow.Steps == nil")
			return
		}

		// check if workflow_instance is ready

		var ready bool
		var reason string
		ready, reason, err = qm.Is_WI_ready(job, wi)
		if err != nil {
			err = fmt.Errorf("(updateWorkflowInstancesMapTask) qm.Is_WI_ready returned: %s", err.Error())
			return
		}

		if !ready {
			logger.Debug(3, "(updateWorkflowInstancesMapTask) Wi is not ready, reason: %s", reason)
			return
		}

		// for each step create Task or Subworkflow

		if len(wi.Tasks) > 0 {
			spew.Dump(wi.Tasks)
			panic("wi already has tasks " + wi_local_id)
		}

		//subworkflow_str := []string{}

		for i, _ := range cwl_workflow.Steps {

			step := &cwl_workflow.Steps[i]

			stepname_base := path.Base(step.Id)
			//steps_str = append(steps_str, wi_local_id+"/"+stepname_base)
			var process interface{}
			process, _, err = step.GetProcess(context)

			switch process.(type) {
			case *cwl.CommandLineTool, *cwl.ExpressionTool:

				fmt.Printf("(updateWorkflowInstancesMapTask) Creating (CommandLine/Expression) %s\n", wi_local_id+"/"+stepname_base)
				var awe_task *Task
				awe_task, err = NewTask(job, wi_local_id, stepname_base)
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) NewTask returned: %s", err.Error())
					return
				}

				if len(step.Scatter) > 0 {
					awe_task.TaskType = TASK_TYPE_SCATTER
				} else {
					awe_task.TaskType = TASK_TYPE_NORMAL
				}

				awe_task.WorkflowStep = step

				_, err = awe_task.Init(job, jobid)
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) awe_task.Init returned: %s", err.Error())
					return
				}

				logger.Debug(3, "(updateWorkflowInstancesMapTask) adding %s to workflow_instance", awe_task.TaskName)
				err = wi.AddTask(job, awe_task, DbSyncTrue, true)
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) wi.AddTask returned: %s", err.Error())
					return
				}

				err = qm.TaskMap.Add(awe_task, "updateWorkflowInstancesMapTask")
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) qm.TaskMap.Add returned: %s", err.Error())
					return
				}

				//panic("got CommandLineTool")
				// create Task

			//case *cwl.ExpressionTool:
			//	fmt.Println("(updateWorkflowInstancesMapTask) ExpressionTool")
			//create Task

			case *cwl.Workflow:
				//subworkflow_str = append(subworkflow_str, wi_local_id+"/"+stepname_base)
				// create new WorkflowInstance

				subworkflow, ok := process.(*cwl.Workflow)
				if !ok {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) cannot cast to *cwl.Workflow")
					return
				}

				subworkflow_id := subworkflow.GetID()

				fmt.Printf("(updateWorkflowInstancesMapTask) Creating Workflow %s\n", subworkflow_id)

				new_wi_name := wi_local_id + "/" + stepname_base

				// TODO assign inputs
				//var workflow_inputs cwl.Job_document

				//panic("creating new subworkflow " + new_wi_name)
				var new_wi *WorkflowInstance
				new_wi, err = NewWorkflowInstance(new_wi_name, jobid, subworkflow_id, job, wi.LocalID)
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) NewWorkflowInstance returned: %s", err.Error())
					return
				}

				new_wi.Workflow = subworkflow
				new_wi.ParentStep = step
				//new_wi.SetState(WIStatePending, "db_sync_no", false) // updateWorkflowInstancesMapTask
				//AddWorkflowInstance sets steat to WIStatePending
				err = job.AddWorkflowInstance(new_wi, DbSyncTrue, true) // updateWorkflowInstancesMapTask
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) job.AddWorkflowInstance returned: %s", err.Error())
					return
				}

				newWIUniqueID, _ := new_wi.GetID(true)

				err = GlobalWorkflowInstanceMap.Add(newWIUniqueID, new_wi)
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) GlobalWorkflowInstanceMap.Add returned: %s", err.Error())
					return
				}

				err = wi.AddSubworkflow(job, new_wi.LocalID, true)
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) wi.AddSubworkflow returned: %s", err.Error())
					return
				}

			default:
				err = fmt.Errorf("(updateWorkflowInstancesMapTask) type unknown: %s", reflect.TypeOf(process))
				return
			}

			//spew.Dump(cwl_workflow.Steps[i])

		}
		//wi.Subworkflows = subworkflow_str

		// pending -> ready
		err = wi.SetState(WIStateReady, DbSyncTrue, true)
		if err != nil {
			err = fmt.Errorf("(updateWorkflowInstancesMapTask) wi.SetState returned: %s", err.Error())
			return
		}
		//spew.Dump(wi)
		//fmt.Printf("SHOULD BE READY NOW (%p)\n", wi)

		if wi.Inputs != nil && len(wi.Inputs) > 0 {
			//panic("found something...")

			var tasks []*Task
			tasks, err = wi.GetTasks(true)
			if err != nil {
				return
			}

			err = qm.EnqueueTasks(tasks)
			if err != nil {
				err = fmt.Errorf("(updateWorkflowInstancesMapTask) EnqueueTasks returned: %s", err.Error())

				return
			}
		}

		err = wi.SetState(WIStateQueued, DbSyncTrue, true)
		if err != nil {
			err = fmt.Errorf("(updateWorkflowInstancesMapTask) SetState returned: %s", err.Error())
			return
		}

		// update job state
		if wi_local_id == "#main" {

			job_state, _ := job.GetState(true)
			if job_state == JOB_STAT_INIT {
				err = job.SetState(JOB_STAT_QUEUED, []string{JOB_STAT_INIT})
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) job.SetState returned: %s", err.Error())
					return
				}
			}

		}

	}

	return
}

func (qm *ServerMgr) updateWorkflowInstancesMap() (err error) {

	var wis []*WorkflowInstance
	wis, err = GlobalWorkflowInstanceMap.GetWorkflowInstances()
	if err != nil {
		err = fmt.Errorf("(updateWorkflowInstancesMap) ")
		return
	}

	var last_error error

	error_count := 0
	for i, _ := range wis {

		wi := wis[i]

		err = qm.updateWorkflowInstancesMapTask(wi)

		if err != nil {

			last_error = err
			error_count += 1
			err = nil
		}

	}

	if error_count > 0 {
		err = fmt.Errorf("(updateWorkflowInstancesMap) %d errors, last error message: %s", error_count, last_error.Error())
		return
	}

	return
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
			logger.Debug(3, "(ServerMgr ClientHandle %s) send response (maybe workunit) to client via response channel", coReq.fromclient)
		case <-timer.C:
			elapsed_time := time.Since(start_time)
			logger.Error("(ServerMgr ClientHandle %s) timed out after %s ", coReq.fromclient, elapsed_time)
			continue
		}
		logger.Debug(3, "(ServerMgr ClientHandle %s) done", coReq.fromclient)

		if count%100 == 0 { // use modulo to reduce number of log messages
			request_time_elapsed := time.Since(request_start_time)

			logger.Info("(ServerMgr ClientHandle) Responding to work request took %s", request_time_elapsed)
		}
	}
}

func (qm *ServerMgr) NoticeHandle() {
	logger.Info("(ServerMgr NoticeHandle) starting")
	for {
		notice := <-qm.feedback

		id, err := notice.ID.String()
		if err != nil {
			logger.Error("(NoticeHandle) notice.ID invalid: " + err.Error())
			err = nil
			continue
		}

		logger.Debug(3, "(ServerMgr NoticeHandle) got notice: workid=%s, status=%s, clientid=%s", id, notice.Status, notice.WorkerID)

		//fmt.Printf("Notice:")
		//spew.Dump(notice)
		err = qm.handleNoticeWorkDelivered(notice)
		if err != nil {
			err = fmt.Errorf("(NoticeHandle): %s", err.Error())
			logger.Error(err.Error())
			//fmt.Println(err.Error())
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
		suspended_jobs := qm.GetSuspendJobs()
		return jQueueShow{qm.actJobs, suspended_jobs}
	}
	if name == "task" {
		tasks, err := qm.TaskMap.GetTasks()
		if err != nil {
			return err
		}
		return tasks
	}
	if name == "workall" {
		workunits, err := qm.workQueue.all.GetWorkunits()
		if err != nil {
			return err
		}
		return workunits
	}
	if name == "workqueue" {
		workunits, err := qm.workQueue.Queue.GetWorkunits()
		if err != nil {
			return err
		}
		return workunits
	}
	if name == "workcheckout" {
		workunits, err := qm.workQueue.Checkout.GetWorkunits()
		if err != nil {
			return err
		}
		return workunits
	}
	if name == "worksuspend" {
		workunits, err := qm.workQueue.Suspend.GetWorkunits()
		if err != nil {
			return err
		}
		return workunits
	}
	if name == "client" {
		return &qm.clientMap
	}
	return nil
}

//--------suspend job accessor methods-------

func (qm *ServerMgr) lenSusJobs() (l int) {

	l = 0
	jobs, _ := JM.Get_List(true) // TODO error handling

	for i := range jobs {
		job := jobs[i]
		state, _ := job.GetState(true)
		if state == JOB_STAT_SUSPEND {
			l += 1
		}

	}

	//qm.sjLock.RLock()
	//l = len(qm.susJobs)
	//qm.sjLock.RUnlock()
	return
}

//func (qm *ServerMgr) putSusJob(id string) {
//	qm.sjLock.Lock()
//	qm.susJobs[id] = true
//	qm.sjLock.Unlock()
//}

func (qm *ServerMgr) GetSuspendJobs() (sjobs map[string]bool) {

	jobs, _ := JM.Get_List(true) // TODO error handling

	sjobs = make(map[string]bool)

	for i := range jobs {
		job := jobs[i]
		state, _ := job.GetState(true) // TODO error handling
		if state == JOB_STAT_SUSPEND {
			id, _ := job.GetId(true)
			sjobs[id] = true
		}

	}

	// qm.sjLock.RLock()
	// 	defer qm.sjLock.RUnlock()
	// 	sjobs = make(map[string]bool)
	// 	for id, _ := range qm.susJobs {
	// 		sjobs[id] = true
	// 	}
	return
}

//func (qm *ServerMgr) removeSusJob(id string) {
//	qm.sjLock.Lock()
//	delete(qm.susJobs, id)
//	qm.sjLock.Unlock()
//}

func (qm *ServerMgr) isSusJob(id string) (has bool) {

	job, err := GetJob(id)
	if err != nil {
		return
	}

	job_state, err := job.GetState(true)
	if err != nil {
		return
	}

	has = false
	if job_state == JOB_STAT_COMPLETED {
		has = true
	}
	return
	//	qm.sjLock.RLock()
	//	defer qm.sjLock.RUnlock()
	//	if _, ok := qm.susJobs[id]; ok {
	//		has = true
	//	} else {
	//		has = false
	//	}
	//	return
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

func (qm *ServerMgr) isActJob(id string) (ok bool) {
	qm.ajLock.RLock()
	defer qm.ajLock.RUnlock()
	_, ok = qm.actJobs[id]
	return
}

//--------server methods-------

//poll ready tasks and push into workQueue
func (qm *ServerMgr) updateQueue(logTimes bool) (err error) {

	logger.Debug(3, "(updateQueue) wait for lock")
	qm.queueLock.Lock()
	defer qm.queueLock.Unlock()

	logger.Debug(3, "(updateQueue) starting")
	var tasks []*Task
	tasks, err = qm.TaskMap.GetTasks()
	if err != nil {
		return
	}

	if len(tasks) > 0 {
		task := tasks[0]
		job_id := task.JobId
		job, _ := GetJob(job_id)
		job_state, _ := job.GetState(true)
		fmt.Printf("*** job *** %s %s\n", job_id, job_state)
		for wi_id, _ := range job.WorkflowInstancesMap {
			wi := job.WorkflowInstancesMap[wi_id]
			wiState, _ := wi.GetState(true)
			fmt.Printf("WorkflowInstance: %s (%s) remain: %d\n", wi_id, wiState, wi.RemainSteps)

			var wi_tasks []*Task
			wi_tasks, err = wi.GetTasks(true)
			if err != nil {
				return
			}

			if len(wi_tasks) > 0 {
				for j, _ := range wi_tasks {
					task := wi_tasks[j]
					fmt.Printf("  Task %d: %s (wf: %s, state %s)\n", j, task.Id, task.WorkflowInstanceId, task.State)

				}
			} else {
				fmt.Printf("  no tasks\n")
			}
			//if len(wi_tasks) > 20 {
			//	panic("too many tasks!")
			//}

			for _, sw := range wi.Subworkflows {
				fmt.Printf("  Subworkflow: %s\n", sw)
			}

		}

	}

	logger.Debug(3, "(updateQueue) range tasks (%d)", len(tasks))

	threads := 20
	size := len(tasks)

	loopStart := time.Now()
	logger.Debug(3, "(updateQueue) starting loop through TaskMap; threads: %d, TaskMap.Len: %d", threads, size)

	taskChan := make(chan *Task, size)
	queueChan := make(chan bool, size)
	for w := 1; w <= threads; w++ {
		go qm.updateQueueWorker(w, logTimes, taskChan, queueChan)
	}

	//total := 0
	for _, task := range tasks {

		taskIDStr, _ := task.String()
		logger.Debug(3, "(updateQueue) sending task to QueueWorkers: %s", taskIDStr)

		//total += 1
		taskChan <- task
	}
	close(taskChan)

	// count tasks that have been queued (it also is a mean)
	queue := 0
	for i := 1; i <= size; i++ {
		q := <-queueChan
		if q {
			queue += 1

		}
	}
	close(queueChan)
	logger.Debug(3, "(updateQueue) completed loop through TaskMap; # processed: %d, queued: %d, took %s", size, queue, time.Since(loopStart))

	logger.Debug(3, "(updateQueue) range qm.workQueue.Clean()")

	// Remove broken workunits
	for _, workunit := range qm.workQueue.Clean() {
		id := workunit.Id
		job_id := workunit.JobId
		task_id := workunit.TaskName

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

func (qm *ServerMgr) updateQueueWorker(id int, logTimes bool, taskChan <-chan *Task, queueChan chan<- bool) {
	var taskSlow time.Duration = 1 * time.Second
	for task := range taskChan {
		taskStart := time.Now()
		taskIDStr, _ := task.String()

		isQueued, times, skip, err := qm.updateQueueTask(task, logTimes)
		if err != nil {
			jerror := &JobError{
				ClientFailed: "NA",
				WorkFailed:   "NA",
				TaskFailed:   taskIDStr,
				ServerNotes:  "updateQueueTask returned error: " + err.Error(),
				WorkNotes:    "NA",
				AppError:     "NA",
				Status:       JOB_STAT_SUSPEND,
			}
			err = nil

			_ = task.SetState(nil, TASK_STAT_SUSPEND, true)

			jobID := task.JobId

			err = qm.SuspendJob(jobID, jerror)
			if err != nil {
				logger.Error("(updateQueueWorker) SuspendJob failed: jobID=%s; err=%s", jobID, err.Error())
				err = nil
			}
		}
		if skip {
			continue
		}
		if logTimes {
			taskTime := time.Since(taskStart)
			message := fmt.Sprintf("(updateQueue) thread %d processed task: %s, took: %s, is queued %t", id, taskIDStr, taskTime, isQueued)
			if taskTime > taskSlow {
				message += fmt.Sprintf(", times: %+v", times)
			}
			logger.Info(message)
		}
		queueChan <- isQueued
	}
}

func (qm *ServerMgr) updateQueueTask(task *Task, logTimes bool) (isQueued bool, times map[string]time.Duration, skip bool, err error) {
	skip = false
	var task_id Task_Unique_Identifier
	task_id, err = task.GetId("updateQueueTask")
	if err != nil {
		err = nil
		skip = true
		return
	}

	taskIDStr, _ := task_id.String()

	var task_state string
	task_state, err = task.GetState()
	if err != nil {
		err = nil
		skip = true
		return
	}

	switch task_state {
	case TASK_STAT_READY:
		logger.Debug(3, "(updateQueueTask) task %s already has state %s, continue with enqueuing", taskIDStr, task_state)
	case TASK_STAT_INIT, TASK_STAT_PENDING:
		logger.Debug(3, "(updateQueueTask) task %s has state %s, first check if ready", taskIDStr, task_state)

		var task_ready bool
		var reason string
		startIsReady := time.Now()

		if logTimes {
			times = make(map[string]time.Duration)
		}
		task_ready, reason, err = qm.isTaskReady(task_id, task)
		if logTimes {
			times["isTaskReady"] = time.Since(startIsReady)
		}
		if err != nil {
			err = fmt.Errorf("(updateQueueTask) qm.isTaskReady returned: %s", err.Error())
			return
		}
		if !task_ready {
			_ = task.SetTaskNotReadyReason(reason, true)

			logger.Debug(3, "(updateQueueTask) task not ready (%s): qm.isTaskReady returned reason: %s", taskIDStr, reason)
			return
		}

		// get new state
		task_state, err = task.GetState()
		if err != nil {
			return
		}

	default:
		logger.Debug(3, "(updateQueueTask) skipping task %s , it has state %s", taskIDStr, task_state)
		return
	}

	logger.Debug(3, "(updateQueueTask) task: %s (state: %s, added by %s)", taskIDStr, task_state, task.Comment)

	// task_ready
	if task_state != TASK_STAT_READY {
		err = fmt.Errorf("(updateQueueTask) task is not rerady !???!?")
		return
	}
	logger.Debug(3, "(updateQueueTask) task %s is ready now, continue to enqueuing", taskIDStr)

	var job_id string
	job_id, err = task.GetJobId()
	if err != nil {
		return
	}

	var job *Job
	job, err = GetJob(job_id)
	if err != nil {
		return
	}

	startEnQueue := time.Now()
	var teqTimes map[string]time.Duration
	teqTimes, err = qm.taskEnQueue(task_id, task, job, logTimes)
	if logTimes {
		for k, v := range teqTimes {
			times[k] = v
		}
		times["taskEnQueue"] = time.Since(startEnQueue)
	}
	if err != nil {
		xerr := err
		err = nil

		logger.Error("(updateQueueTask) (task_id: %s) suspending task, taskEnQueue returned: %s", taskIDStr, xerr.Error())
		err = task.SetState(nil, TASK_STAT_SUSPEND, true)
		if err != nil {
			return
		}

		job_id, err = task.GetJobId()
		if err != nil {
			return
		}

		var task_str string
		task_str, err = task.String()
		if err != nil {
			return

		}

		jerror := &JobError{
			TaskFailed:  task_str,
			ServerNotes: fmt.Sprintf("failed enqueuing task %s, qm.taskEnQueue returned: %s", taskIDStr, xerr.Error()),
			Status:      JOB_STAT_SUSPEND,
		}
		err = qm.SuspendJob(job_id, jerror)
		if err != nil {
			err = fmt.Errorf("(updateQueueTask) qm.SuspendJob returned: %s", err.Error())
			return
		}
		err = xerr
		return
	}

	isQueued = true

	logger.Debug(3, "(updateQueueTask) task enqueued: %s", taskIDStr)

	return
}

func RemoveWorkFromClient(client *Client, workid Workunit_Unique_Identifier) (err error) {
	err = client.AssignedWork.Delete(workid, true)
	if err != nil {
		return
	}

	work_length, err := client.AssignedWork.Length(true)
	if err != nil {
		return
	}

	if work_length > 0 {

		clientid, _ := client.GetID(true)

		logger.Error("(RemoveWorkFromClient) Client %s still has %d workunits assigned, after delivering one workunit", clientid, work_length)

		assigned_work_ids, err := client.AssignedWork.Get_list(true)
		if err != nil {
			return err
		}
		for _, work_id := range assigned_work_ids {
			_ = client.AssignedWork.Delete(work_id, true)
		}

		work_length, err = client.AssignedWork.Length(true)
		if err != nil {
			return err
		}
		if work_length > 0 {
			logger.Error("(RemoveWorkFromClient) Client still has work assigned, even after everything should have been deleted.")
			return fmt.Errorf("(RemoveWorkFromClient) Client %s still has %d workunits", clientid, work_length)
		}
	}
	return
}

// invoked for every completed workunit
// updates task object:
//  - every workunit will decrease counter task.RemainWork
//  - last workunit will complete the task
// ** only last workunit writes CWL results into task ! **
// ** other workunits contribute only their rank information **
func (qm *ServerMgr) handleWorkStatDone(client *Client, clientid string, task *Task, workid Workunit_Unique_Identifier, notice *Notice) (err error) {
	//log event about work done (WD)

	computetime := notice.ComputeTime

	var work_str string
	work_str, err = workid.String()
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) workid.String() returned: %s", err.Error())
		return
	}
	//workid_string := workid.String()

	logger.Event(event.WORK_DONE, "workid="+work_str+";clientid="+clientid)
	//update client status

	var task_str string
	task_str, err = task.String()
	if err != nil {
		return
	}

	defer func() {
		//done, remove from the workQueue
		qm.workQueue.Delete(workid)
	}()

	if client != nil {
		err = client.IncrementTotalCompleted()
		if err != nil {
			err = fmt.Errorf("(handleWorkStatDone) client.IncrementTotalCompleted returned: %s", err.Error())
			return
		}
	}
	var remain_work int
	remain_work, err = task.IncrementRemainWork(-1, true)
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) client=%s work=%s task.IncrementRemainWork returned: %s", clientid, work_str, err.Error())
		return
	}

	// double check, remain_work should equal # of workunits in workqueue with same job and task ids
	workunits, werr := qm.workQueue.GetAll()
	if werr != nil {
		err = fmt.Errorf("(handleWorkStatDone) unable to get workunit list: %s", werr.Error())
		return
	}

	var workunit_count int
	for _, wu := range workunits {
		if (wu.JobId == workid.JobId) && (wu.TaskName == workid.TaskName) && (wu.Rank != workid.Rank) {
			workunit_count += 1
		}
	}
	if workunit_count != remain_work {
		err = fmt.Errorf("(handleWorkStatDone) client=%s work=%s remainwork (%d) does not match number of workunits in queue (%d)", clientid, work_str, remain_work, workunit_count)
		return
	}

	err = task.IncrementComputeTime(computetime)
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) client=%s work=%s IncrementComputeTime returned: %s", clientid, work_str, err.Error())
		return
	}

	logger.Debug(3, "(handleWorkStatDone) remain_work: %d (%s)", remain_work, work_str)

	if remain_work == 0 {
		err = qm.handleLastWorkunit(clientid, task, task_str, work_str, notice)
		if err != nil {
			err = fmt.Errorf("(handleWorkStatDone) handleLastWorkunit returned: %s", err.Error())
			return
		}
	}

	return
}

// ****************************
// ******* LAST WORKUNIT ******
// ****************************
func (qm *ServerMgr) handleLastWorkunit(clientid string, task *Task, task_str string, work_str string, notice *Notice) (err error) {

	// validate file sizes of all outputs
	err = task.ValidateOutputs() // for AWE1 only
	if err != nil {
		// we create job error object and suspend job
		err_msg := fmt.Sprintf("(handleWorkStatDone) ValidateOutputs returned: %s", err.Error())
		jerror := &JobError{
			ClientFailed: clientid,
			WorkFailed:   work_str,
			TaskFailed:   task_str,
			ServerNotes:  err_msg,
			Status:       JOB_STAT_SUSPEND,
		}
		err = task.SetState(nil, TASK_STAT_SUSPEND, true)
		if err != nil {
			err = fmt.Errorf("(handleWorkStatDone) task.SetState returned: %s", err.Error())
			return
		}
		err = qm.SuspendJob(task.JobId, jerror)
		if err != nil {
			err = fmt.Errorf("(handleWorkStatDone) SuspendJob returned: %s", err.Error())
			return
		}
		err = errors.New(err_msg)
		return
	}

	// **************************************
	// ******* write results into task ******
	// **************************************
	if task.WorkflowStep != nil {
		err = task.SetStepOutput(notice.Results, true)
		if err != nil {
			err = fmt.Errorf("(handleWorkStatDone) task.SetStepOutput returned: %s", err.Error())
			return
		}
	}

	//if task.WorkflowStep == nil {
	//	err = fmt.Errorf("(handleWorkStatDone) task.WorkflowStep == nil")
	//	return
	//}

	var job *Job
	job, err = task.GetJob()
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) GetJob returned: %s", err.Error())
		return
	}

	// iterate over expected outputs

	//var process interface{}
	//process_cached := false

	var wi *WorkflowInstance

	if task.WorkflowStep != nil {
		context := job.WorkflowContext

		if task.Scatter_parent == nil {
			for i, _ := range task.WorkflowStep.Out {
				step_output := &task.WorkflowStep.Out[i]
				basename := path.Base(step_output.Id)

				step_output_array := []cwl.NamedCWLType(*task.StepOutput)

				// find in real outputs
				found := false
				for j, _ := range step_output_array { // []cwl.NamedCWLType
					named := &step_output_array[j]
					actual_output_base := path.Base(named.Id)
					if basename == actual_output_base {
						// add object to context using stepoutput name
						logger.Debug(3, "(handleWorkStatDone) adding %s ...", step_output.Id)
						err = context.Add(step_output.Id, named.Value, "handleWorkStatDone")
						if err != nil {
							err = fmt.Errorf("(handleWorkStatDone) context.Add returned: %s", err.Error())
							return
						}
						found = true
						continue
					}

				}
				if !found {
					var obj cwl.CWLObject
					obj = cwl.NewNull()
					err = context.Add(step_output.Id, obj, "handleWorkStatDone") // TODO: DO NOT DO THIS FOR SCATTER TASKS
					// check if this is an optional output in the tool

					//err = fmt.Errorf("(handleWorkStatDone) expected output not found: %s", basename)
					//return
				}
			}
		}

		var ok bool
		wi, ok, err = task.GetWorkflowInstance()
		if err != nil {
			err = fmt.Errorf("(handleWorkStatDone) task.GetWorkflowInstance returned: %s", err.Error())
			return
		}

		if !ok {
			err = fmt.Errorf("(handleWorkStatDone) Did not get WorkflowInstance from task")
			return
		}
	}
	// err = task.SetState(wi, TASK_STAT_COMPLETED, true)
	// if err != nil {
	// 	err = fmt.Errorf("(handleWorkStatDone) task.SetState returned: %s", err.Error())
	// 	return
	// }

	//log event about task done (TD)
	err = qm.FinalizeTaskPerf(task)
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) FinalizeTaskPerf returned: %s", err.Error())
		return
	}
	logger.Event(event.TASK_DONE, "task_id="+task_str)

	//update the info of the job which the task is belong to, could result in deletion of the
	//task in the task map when the task is the final task of the job to be done.
	err = qm.taskCompleted(wi, task) //task state QUEUED -> COMPLETED
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) taskCompleted returned: %s", err.Error())
		return
	}
	return

}

// handle feedback from a client about the execution of a workunit
// successful workunits are handeled by "handleWorkStatDone()"
func (qm *ServerMgr) handleNoticeWorkDelivered(notice Notice) (err error) {

	logger.Debug(3, "(handleNoticeWorkDelivered) start")
	clientid := notice.WorkerID

	work_id := notice.ID
	task_id := work_id.GetTask()

	job_id := work_id.JobId

	notice_status := notice.Status

	//computetime := notice.ComputeTime
	notes := notice.Notes

	var work_str string
	work_str, err = work_id.String()
	if err != nil {
		err = fmt.Errorf("(handleNoticeWorkDelivered) work_id.String() returned: %s", err.Error())
		return
	}

	logger.Debug(3, "(handleNoticeWorkDelivered) workid: %s status: %s client: %s", work_str, notice_status, clientid)

	// we should not get here, but if we do then return error
	if notice_status == WORK_STAT_DISCARDED {
		logger.Error("(handleNoticeWorkDelivered) [warning] skip status change: workid=%s status=%s", work_str, notice_status)
		return
	}

	var client *Client
	client = nil
	if clientid != "_internal" {
		// *** Get Client

		var ok bool
		client, ok, err = qm.GetClient(clientid, true)
		if err != nil {
			return
		}
		if !ok {
			err = fmt.Errorf("(handleNoticeWorkDelivered) client not found")
			return
		}
		defer RemoveWorkFromClient(client, work_id)
	}
	// *** Get Task
	var task *Task
	var tok bool
	task, tok, err = qm.TaskMap.Get(task_id, true)
	if err != nil {
		return
	}
	if !tok {
		//task not existed, possible when job is deleted before the workunit done
		err = fmt.Errorf("(handleNoticeWorkDelivered) task %s for workunit %s not found", task_id, work_str)
		logger.Error(err.Error())
		qm.workQueue.Delete(work_id)
		return
	}

	reason := ""

	// if notice.Results != nil { // TODO one workunit vs multiple !!!!!!!!!!!!!!!!!!!!!!!!!!!!!
	// 	err = task.SetStepOutput(notice.Results, true)
	// 	if err != nil {
	// 		err = fmt.Errorf("(handleNoticeWorkDelivered) task.SetStepOutput returned: %s", err.Error())
	// 		return
	// 	}

	// 	var job *Job
	// 	job, err = GetJob(job_id)
	// 	if err != nil {
	// 		err = fmt.Errorf("(handleNoticeWorkDelivered) GetJob returned: %s", err.Error())
	// 		return
	// 	}

	// 	context := job.WorkflowContext

	// 	step_output_array := []cwl.NamedCWLType(*task.StepOutput)

	// 	// iterate over expected outputs

	// 	//var process interface{}
	// 	//process_cached := false

	// 	for i, _ := range task.WorkflowStep.Out {
	// 		step_output := &task.WorkflowStep.Out[i]
	// 		basename := path.Base(step_output.Id)

	// 		// find in real outputs
	// 		found := false
	// 		for j, _ := range step_output_array { // []cwl.NamedCWLType
	// 			named := &step_output_array[j]
	// 			actual_output_base := path.Base(named.Id)
	// 			if basename == actual_output_base {
	// 				// add object to context using stepoutput name
	// 				logger.Debug(3, "(handleNoticeWorkDelivered) adding %s ...", step_output.Id)
	// 				err = context.Add(step_output.Id, named.Value, "handleNoticeWorkDelivered")
	// 				if err != nil {
	// 					err = fmt.Errorf("(handleNoticeWorkDelivered) context.Add returned: %s", err.Error())
	// 					return
	// 				}
	// 				found = true
	// 				continue
	// 			}

	// 		}
	// 		if !found {
	// 			var obj cwl.CWLObject
	// 			obj = cwl.NewNull()
	// 			err = context.Add(step_output.Id, obj, "handleNoticeWorkDelivered")
	// 			// check if this is an optional output in the tool

	// 			//err = fmt.Errorf("(handleNoticeWorkDelivered) expected output not found: %s", basename)
	// 			//return
	// 		}
	// 	}

	// }

	// *** Get workunit
	var work *Workunit
	var wok bool
	work, wok, err = qm.workQueue.Get(work_id)
	if err != nil {
		return
	}
	if !wok {
		err = fmt.Errorf("(handleNoticeWorkDelivered) workunit %s not found in workQueue", work_str)
		return
	}
	work_state := work.State

	if work_state != WORK_STAT_CHECKOUT && work_state != WORK_STAT_RESERVED {
		err = fmt.Errorf("(handleNoticeWorkDelivered) workunit %s did not have state WORK_STAT_CHECKOUT or WORK_STAT_RESERVED (state is %s)", work_str, work_state)
		return
	}

	if notice_status == WORK_STAT_SUSPEND {
		reason = "workunit suspended by worker" // TODO add more info from worker
	}

	// *** update state of workunit
	err = qm.workQueue.StatusChange(Workunit_Unique_Identifier{}, work, notice_status, reason)
	if err != nil {
		err = fmt.Errorf("(handleNoticeWorkDelivered) qm.workQueue.StatusChange returned: %s", err.Error())
		return
	}

	err = task.LockNamed("handleNoticeWorkDelivered/noretry")
	if err != nil {
		return
	}
	noretry := task.Info.NoRetry
	task.Unlock()

	var MAX_FAILURE int
	if noretry == true {
		MAX_FAILURE = 1
	} else {
		MAX_FAILURE = conf.MAX_WORK_FAILURE
	}

	var task_state string
	task_state, err = task.GetState()
	if err != nil {
		return
	}

	if task_state == TASK_STAT_FAIL_SKIP {
		// A work unit for this task failed before this one arrived.
		// User set Skip=2 so the task was just skipped. Any subsiquent
		// workunits are just deleted...
		_ = qm.workQueue.Delete(work_id)
		err = fmt.Errorf("(handleNoticeWorkDelivered) workunit %s failed due to skip", work_str)
		return
	}

	logger.Debug(3, "(handleNoticeWorkDelivered) handling status %s", notice_status)
	switch notice_status {
	case WORK_STAT_DONE:
		//      ******************
		//      * WORK_STAT_DONE *
		//      ******************
		err = qm.handleWorkStatDone(client, clientid, task, work_id, &notice)
		if err != nil {
			err = fmt.Errorf("(handleNoticeWorkDelivered) handleWorkStatDone returned: %s", err.Error())
			return
		}
	case WORK_STAT_FAILED_PERMANENT: // (special case !) failed and cannot be recovered

		logger.Event(event.WORK_FAILED, "workid="+work_str+";clientid="+clientid)
		logger.Debug(3, "(handleNoticeWorkDelivered) work failed (status=%s) workid=%s clientid=%s", notice_status, work_str, clientid)
		work.Failed += 1

		//qm.workQueue.StatusChange(Workunit_Unique_Identifier{}, work, WORK_STAT_FAILED_PERMANENT, "")

		err = task.SetState(nil, TASK_STAT_FAILED_PERMANENT, true)
		if err != nil {
			return
		}

		var task_str string
		task_str, err = task.String()
		if err != nil {
			err = fmt.Errorf("(handleNoticeWorkDelivered) task.String returned: %s", err.Error())
			return
		}

		jerror := &JobError{
			ClientFailed: clientid,
			WorkFailed:   work_str,
			TaskFailed:   task_str,
			ServerNotes:  "exit code 42 encountered",
			WorkNotes:    notes,
			AppError:     notice.Stderr,
			Status:       JOB_STAT_FAILED_PERMANENT,
		}
		err = qm.SuspendJob(job_id, jerror)
		if err != nil {
			logger.Error("(handleNoticeWorkDelivered:SuspendJob) job_id=%s; err=%s", job_id, err.Error())
		}
	case WORK_STAT_ERROR: //workunit failed, requeue or put it to suspend list
		logger.Event(event.WORK_FAIL, "workid="+work_str+";clientid="+clientid)
		logger.Debug(3, "(handleNoticeWorkDelivered) work failed (status=%s, notes: %s) workid=%s clientid=%s", notice_status, notes, work_str, clientid)

		work.Failed += 1

		if work.Failed < MAX_FAILURE {
			qm.workQueue.StatusChange(Workunit_Unique_Identifier{}, work, WORK_STAT_QUEUED, "")
			logger.Event(event.WORK_REQUEUE, "workid="+work_str)
		} else {
			//failure time exceeds limit, suspend workunit, task, job
			qm.workQueue.StatusChange(Workunit_Unique_Identifier{}, work, WORK_STAT_SUSPEND, "work.Failed >= MAX_FAILURE")
			logger.Event(event.WORK_SUSPEND, "workid="+work_str)

			if err = task.SetState(nil, TASK_STAT_SUSPEND, true); err != nil {
				return
			}

			var task_str string
			task_str, err = task.String()
			if err != nil {
				err = fmt.Errorf("(handleNoticeWorkDelivered) task.String returned: %s", err.Error())
				return
			}
			jerror := &JobError{
				ClientFailed: clientid,
				WorkFailed:   work_str,
				TaskFailed:   task_str,
				ServerNotes:  fmt.Sprintf("workunit failed %d time(s)", MAX_FAILURE),
				WorkNotes:    notes,
				AppError:     notice.Stderr,
				Status:       JOB_STAT_SUSPEND,
			}
			if err = qm.SuspendJob(job_id, jerror); err != nil {
				logger.Error("(handleNoticeWorkDelivered:SuspendJob) job_id=%s; err=%s", job_id, err.Error())
			}
		}

		// Suspend client if needed
		var client *Client
		var ok bool
		client, ok, err = qm.GetClient(clientid, true)
		if err != nil {
			return
		}
		if !ok {
			err = fmt.Errorf(e.ClientNotFound)
			return
		}

		err = client.AppendSkipwork(work_id, true)
		if err != nil {
			return
		}
		err = client.IncrementTotalFailed(true)
		if err != nil {
			return
		}

		var last_failed int
		last_failed, err = client.IncrementLastFailed(true)
		if err != nil {
			return
		}
		if last_failed >= conf.MAX_CLIENT_FAILURE {
			qm.SuspendClient(clientid, client, "MAX_CLIENT_FAILURE on client reached", true)
		}
	default:
		err = fmt.Errorf("No handler for workunit status '%s' implemented (allowd: %s, %s, %s)", notice_status, WORK_STAT_DONE, WORK_STAT_FAILED_PERMANENT, WORK_STAT_ERROR)
		return
	}
	return
}

// GetJsonStatus _
func (qm *ServerMgr) GetJsonStatus() (status map[string]map[string]int, err error) {
	queuingWork, err := qm.workQueue.Queue.Len()
	if err != nil {
		return
	}
	var outWork int
	outWork, err = qm.workQueue.Checkout.Len()
	if err != nil {
		err = fmt.Errorf("(GetJsonStatus) qm.workQueue.Checkout.Len returtned: %s", err.Error())
		return
	}
	var suspendWork int
	suspendWork, err = qm.workQueue.Suspend.Len()
	if err != nil {
		err = fmt.Errorf("(GetJsonStatus) qm.workQueue.Suspend.Len returtned: %s", err.Error())
		return
	}
	var totalActiveWork int
	totalActiveWork, err = qm.workQueue.Len()
	if err != nil {
		err = fmt.Errorf("(GetJsonStatus) qm.workQueue.Len returtned: %s", err.Error())
		return
	}

	// *** jobs ***
	jobs := make(map[string]int)

	var jobList []*Job
	jobList, err = JM.Get_List(true)
	if err != nil {
		err = fmt.Errorf("(GetJsonStatus) JM.Get_List returtned: %s", err.Error())
		return
	}
	jobs["total"] = len(jobList)

	for _, job := range jobList {

		jobState, terr := job.GetStateTimeout(true, time.Second*1)
		if terr != nil {
			jobState = "unknown"
		}
		jobs[jobState]++

	}

	// *** tasks ***

	tasks := make(map[string]int)

	taskList, err := qm.TaskMap.GetTasks()
	if err != nil {
		return
	}
	tasks["total"] = len(taskList)

	for _, task := range taskList {

		taskState, terr := task.GetStateTimeout(time.Second * 1)

		if terr != nil {
			taskState = "unknown"
		}

		tasks[taskState]++

		// total_task += 1

		// switch task.State {
		// case TASK_STAT_COMPLETED:
		// 	completed_task += 1
		// case TASK_STAT_PENDING:
		// 	pending_task += 1
		// case TASK_STAT_QUEUED:
		// 	queuing_task += 1
		// case TASK_STAT_INPROGRESS:
		// 	started_task += 1
		// case TASK_STAT_SUSPEND:
		// 	suspended_task += 1
		// case TASK_STAT_SKIPPED:
		// 	skipped_task += 1
		// case TASK_STAT_FAIL_SKIP:
		// 	fail_skip_task += 1
		// }
	}
	//total_task -= skipped_task // user doesn't see skipped tasks

	// totalClient := 0
	// busyClient := 0
	// idleClient := 0
	// suspendClient := 0

	clientStates := make(map[string]int)

	clientList, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}

	totalClient := len(clientList)
	for _, client := range clientList {

		clientStates[client.Status]++

		// if ! client.Healty

		// if client.Suspended {
		// 	suspendClient++
		// }
		// if client.Busy {
		// 	busyClient++
		// } else {
		// 	idleClient++
		// }

	}
	clientStates["total"] = totalClient

	//jobs := map[string]int{
	//	"total":     total_job,
	//	"active":    active_jobs,
	//	"suspended": suspend_job,
	//}
	// tasks := map[string]int{
	// 	"total":       total_task,
	// 	"queuing":     queuing_task,
	// 	"in-progress": started_task,
	// 	"pending":     pending_task,
	// 	"completed":   completed_task,
	// 	"suspended":   suspended_task,
	// 	"failed":      fail_skip_task,
	// }
	workunits := map[string]int{
		"total":     totalActiveWork,
		"queuing":   queuingWork,
		"checkout":  outWork,
		"suspended": suspendWork,
	}
	// clients := map[string]int{
	// 	"total":     totalClient,
	// 	"busy":      busyClient,
	// 	"idle":      idleClient,
	// 	"suspended": suspendClient,
	// }
	status = map[string]map[string]int{
		"jobs":      jobs,
		"tasks":     tasks,
		"workunits": workunits,
		"clients":   clientStates,
	}
	return
}

// GetTextStatus _ TODO: this will not reflect all states. Get rid of this, difficult to maintain!
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
// FetchDataToken _
func (qm *ServerMgr) FetchDataToken(work_id Workunit_Unique_Identifier, clientid string) (token string, err error) {

	//precheck if the client is registered
	client, ok, err := qm.GetClient(clientid, true)
	if err != nil {
		return
	}
	if !ok {
		return "", errors.New(e.ClientNotFound)
	}

	is_suspended, err := client.GetSuspended(true)
	if err != nil {
		return
	}

	if is_suspended {
		err = errors.New(e.ClientSuspended)
		return
	}

	jobid := work_id.JobId

	job, err := GetJob(jobid)
	if err != nil {
		return
	}
	token = job.GetDataToken()
	if token == "" {
		var work_str string
		work_str, err = work_id.String()
		if err != nil {
			err = fmt.Errorf("(FetchDataToken) workid.String() returned: %s", err.Error())
			return
		}
		err = errors.New("no data token set for workunit " + work_str)
		return
	}
	return
}

// func (qm *ServerMgr) FetchPrivateEnvs_deprecated(workid string, clientid string) (envs map[string]string, err error) {
// 	//precheck if the client is registered
// 	client, ok, err := qm.GetClient(clientid, true)
// 	if err != nil {
// 		return
// 	}
// 	if !ok {
// 		return nil, errors.New(e.ClientNotFound)
// 	}
// 	client_status, err := client.Get_Status(true)
// 	if err != nil {
// 		return
// 	}
// 	if client_status == CLIENT_STAT_SUSPEND {
// 		return nil, errors.New(e.ClientSuspended)
// 	}
// 	jobid, err := GetJobIdByWorkId(workid)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	job, err := GetJob(jobid)
// 	if err != nil {
// 		return nil, err
// 	}
//
// 	taskid, _ := GetTaskIdByWorkId(workid)
//
// 	idx := -1
// 	for i, t := range job.Tasks {
// 		if t.Id == taskid {
// 			idx = i
// 			break
// 		}
// 	}
// 	envs = job.Tasks[idx].Cmd.Environ.Private
// 	if envs == nil {
// 		return nil, errors.New("no private envs for workunit " + workid)
// 	}
// 	return envs, nil
// }

func (qm *ServerMgr) SaveStdLog(id Workunit_Unique_Identifier, logname string, tmppath string) (err error) {
	savedpath, err := getStdLogPathByWorkId(id, logname)
	if err != nil {
		return err
	}
	os.Rename(tmppath, savedpath)
	return
}

func (qm *ServerMgr) GetReportMsg(id Workunit_Unique_Identifier, logname string) (report string, err error) {
	logpath, err := getStdLogPathByWorkId(id, logname)
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

func getStdLogPathByWorkId(id Workunit_Unique_Identifier, logname string) (savedpath string, err error) {
	jobid := id.JobId

	var logdir string
	logdir, err = getPathByJobId(jobid)
	if err != nil {
		return
	}
	//workid := id.String()
	var work_str string
	work_str, err = id.String()
	if err != nil {
		err = fmt.Errorf("(getStdLogPathByWorkId) id.String() returned: %s", err.Error())
		return
	}

	savedpath = fmt.Sprintf("%s/%s.%s", logdir, work_str, logname)
	return
}

// this is trigggered by user action, either job POST or job resume / recover / resubmit
// func (qm *ServerMgr) EnqueueWorkflowInstancesByJobId(jobid string) (err error) {

// 	logger.Debug(3, "(EnqueueWorkflowInstancesByJobId) starting")
// 	job, err := GetJob(jobid)
// 	if err != nil {
// 		err = fmt.Errorf("(EnqueueWorkflowInstancesByJobId) GetJob returned: %s", err.Error())
// 		return
// 	}

// }

func (qm *ServerMgr) EnqueueWorkflowInstance(wi *WorkflowInstance) (err error) {

	logger.Debug(3, "(EnqueueWorkflowInstance) starting")

	wiUniqueID, _ := wi.GetID(true)

	err = GlobalWorkflowInstanceMap.Add(wiUniqueID, wi)
	if err != nil {
		err = fmt.Errorf("(EnqueueWorkflowInstancesByJobId) GlobalWorkflowInstanceMap.Add returned: %s", err.Error())
		return
	}

	return
}

func (qm *ServerMgr) EnqueueTasks(tasks []*Task) (err error) {
	//logger.Debug(3, "(EnqueueTasksByJobId) starting")
	//job, err := GetJob(jobid)
	//if err != nil {
	//	err = fmt.Errorf("(EnqueueTasksByJobId) GetJob failed: %s", err.Error())
	//	return
	//}

	//fmt.Println("(EnqueueTasksByJobId) job.WorkflowInstances[0]:")
	//spew.Dump(job.WorkflowInstances[0])
	//panic("done")

	//var tasks []*Task
	//tasks, err = job.GetTasks()
	//if err != nil {
	//	err = fmt.Errorf("(EnqueueTasksByJobId) job.GetTasks failed: %s", err.Error())
	//	return
	//}

	task_len := len(tasks)
	logger.Debug(3, "(EnqueueTasks) got %d tasks", task_len)

	// err = job.SetState(JOB_STAT_QUEUING, nil)
	// if err != nil {
	// 	err = fmt.Errorf("(qmgr.taskEnQueue) UpdateJobState: %s", err.Error())
	// 	return
	// }

	//qm.CreateJobPerf(jobid)

	for _, task := range tasks {
		var task_state string
		task_state, err = task.GetState()
		if err != nil {
			return
		}

		if task_state == TASK_STAT_INPROGRESS || task_state == TASK_STAT_QUEUED {
			err = task.SetState(nil, TASK_STAT_READY, true)
			if err != nil {
				return
			}
		} else if task_state == TASK_STAT_SUSPEND {
			err = task.SetState(nil, TASK_STAT_PENDING, true)
			if err != nil {
				return
			}
		}

		// add to qm.TaskMap
		// updateQueue() process will actually enqueue the task
		// TaskMap.Add - makes it a pending task if init, throws error if task already in map with different pointer
		err = qm.TaskMap.Add(task, "EnqueueTasks")
		if err != nil {
			err = fmt.Errorf("(EnqueueTasks) qm.TaskMap.Add() returns: %s", err.Error())
			return
		}
	}

	return
}

// this is trigggered by user action, either job POST or job resume / recover / resubmit
func (qm *ServerMgr) EnqueueTasksByJobId(jobid string, caller string) (err error) {
	logger.Debug(3, "(EnqueueTasksByJobId) starting")
	job, err := GetJob(jobid)
	if err != nil {
		err = fmt.Errorf("(EnqueueTasksByJobId) GetJob failed: %s", err.Error())
		return
	}

	//fmt.Println("(EnqueueTasksByJobId) job.WorkflowInstances[0]:")
	//spew.Dump(job.WorkflowInstances[0])
	//panic("done")

	var tasks []*Task
	tasks, err = job.GetTasks()
	if err != nil {
		err = fmt.Errorf("(EnqueueTasksByJobId) job.GetTasks failed: %s", err.Error())
		return
	}

	task_len := len(tasks)
	logger.Debug(3, "(EnqueueTasksByJobId) got %d tasks", task_len)

	err = job.SetState(JOB_STAT_QUEUING, nil)
	if err != nil {
		err = fmt.Errorf("(qmgr.taskEnQueue) UpdateJobState: %s", err.Error())
		return
	}

	qm.CreateJobPerf(jobid)

	for _, task := range tasks {
		var task_state string
		task_state, err = task.GetState()
		if err != nil {
			return
		}

		if task_state == TASK_STAT_INPROGRESS || task_state == TASK_STAT_QUEUED {
			err = task.SetState(nil, TASK_STAT_READY, true)
			if err != nil {
				return
			}
		} else if task_state == TASK_STAT_SUSPEND {
			err = task.SetState(nil, TASK_STAT_PENDING, true)
			if err != nil {
				return
			}
		}

		// add to qm.TaskMap
		// updateQueue() process will actually enqueue the task
		// TaskMap.Add - makes it a pending task if init, throws error if task already in map with different pointer
		err = qm.TaskMap.Add(task, "EnqueueTasksByJobId/"+caller)
		if err != nil {
			err = fmt.Errorf("(EnqueueTasksByJobId) qm.TaskMap.Add() returns: %s", err.Error())
			return
		}
	}

	var job_state string
	job_state, err = job.GetState(true)
	if err != nil {
		return
	}
	if job_state != JOB_STAT_INPROGRESS {
		err = job.SetState(JOB_STAT_QUEUED, []string{JOB_STAT_INIT, JOB_STAT_SUSPEND, JOB_STAT_QUEUING})
		if err != nil {
			return
		}
	}

	return
}

// used by isTaskReady and is_WI_Ready
// check all WorkflowStepInputs for Source fields and checks if they are available
func (qm *ServerMgr) areSourceGeneratorsReady(step *cwl.WorkflowStep, job *Job, workflow_instance *WorkflowInstance) (ready bool, reason string, err error) {

	logger.Debug(3, "(areSourceGeneratorsReady) start %s", step.Id)

	for _, wsi := range step.In { // WorkflowStepInput

		//input_optional := false
		//if wsi.Default != nil {
		//	input_optional = true
		//}
		logger.Debug(3, "(areSourceGeneratorsReady) step input %s", wsi.Id)
		if wsi.Source == nil {
			logger.Debug(3, "(areSourceGeneratorsReady) step input %s source empty", wsi.Id)
			continue
		}

		source_is_array := false
		source_as_array, source_is_array := wsi.Source.([]interface{})

		if source_is_array {
			logger.Debug(3, "(areSourceGeneratorsReady) step input %s source_is_array", wsi.Id)
			for _, src := range source_as_array { // usually only one
				var src_str string
				var ok bool

				src_str, ok = src.(string)
				if !ok {

					err = fmt.Errorf("src is not a string")
					return ok, reason, err
				}

				// see comments below
				context := job.WorkflowContext
				_, ok, err = context.Get(src_str, true)
				if err != nil {
					err = fmt.Errorf("(areSourceGeneratorsReady) context.Get returned: %s", err.Error())
					return
				}
				if ok {
					continue
				}

				generator := path.Dir(src_str)
				ok, reason, err = qm.isSourceGeneratorReady(job, workflow_instance, generator, false, job.WorkflowContext)
				if err != nil {
					err = fmt.Errorf("(areSourceGeneratorsReady) (type array, src_str: %s) isSourceGeneratorReady returns: %s", src_str, err.Error())
					return
				}
				if !ok {
					reason = fmt.Sprintf("Generator not ready (%s)", reason)
					return
				}

			}
		} else {
			// not source_is_array
			logger.Debug(3, "(areSourceGeneratorsReady) step input %s NOT source_is_array", wsi.Id)
			var src_str string
			var ok bool

			src_str, ok = wsi.Source.(string)
			if !ok {
				err = fmt.Errorf("(areSourceGeneratorsReady) Cannot parse WorkflowStep source: %s", spew.Sdump(wsi.Source))
				return
			}

			//if source is (workflow) input, generator does not need to be ready
			// instead of testing for workflow input, we just test if input already exists
			context := job.WorkflowContext
			_, ok, err = context.Get(src_str, true)
			if err != nil {
				err = fmt.Errorf("(areSourceGeneratorsReady) context.Get returned: %s", err.Error())
				return
			}
			if ok {
				continue
			}

			generator := path.Dir(src_str)
			logger.Debug(3, "(areSourceGeneratorsReady) step input %s using generator %s", wsi.Id, generator)

			ready, reason, err = qm.isSourceGeneratorReady(job, workflow_instance, generator, false, job.WorkflowContext)
			if err != nil {
				err = fmt.Errorf("(areSourceGeneratorsReady) B (type non-array, src_str: %s) isSourceGeneratorReady returns: %s", src_str, err.Error())
				return
			}

			if !ready {
				reason = fmt.Sprintf("Generator not ready (%s)", reason)
				return
			}

		}
		logger.Debug(3, "(areSourceGeneratorsReady) step input %s is ready", wsi.Id)
	}
	ready = true
	logger.Debug(3, "(areSourceGeneratorsReady) finished")
	return
}

// check whether a pending task is ready to enqueue (dependent tasks are all done)
// task is not locked
func (qm *ServerMgr) isTaskReady(task_id Task_Unique_Identifier, task *Task) (ready bool, reason string, err error) {
	ready = false

	reason = "all ok"
	logger.Debug(3, "(isTaskReady) starting")

	task_state, err := task.GetStateNamed("isTaskReady")
	if err != nil {
		return
	}
	logger.Debug(3, "(isTaskReady) task state at start is %s", task_state)

	if task_state == TASK_STAT_READY {
		ready = true
		return
	}

	if task_state == TASK_STAT_INIT || task_state == TASK_STAT_PENDING {
		logger.Debug(3, "(isTaskReady) task state is init or pending, continue:  %s", task_state)
	} else {
		err = fmt.Errorf("(isTaskReady) task has state %s, it does not make sense to test if it is ready", task_state)
		return
	}

	//task_id, err := task.GetId("isTaskReady")
	//if err != nil {
	//	return
	//}

	taskIDStr, _ := task_id.String()

	logger.Debug(3, "(isTaskReady %s)", taskIDStr)

	//skip if the belonging job is suspended
	//jobid, err := task.GetJobId()
	//if err != nil {
	//	return
	//}

	job, err := task.GetJob()
	if err != nil {
		return
	}
	job_state, err := job.GetState(true)
	if err != nil {
		return
	}
	if job_state == JOB_STAT_SUSPEND {
		reason = "job is suspend"
		return
	}

	if task.Info != nil {
		info := task.Info
		if !info.StartAt.IsZero() {

			if info.StartAt.After(time.Now()) {
				// too early
				logger.Debug(3, "(isTaskReady %s) too early to execute (now: %s, StartAt: %s)", taskIDStr, time.Now(), info.StartAt)
				return
			} else {
				logger.Debug(3, "(isTaskReady %s) StartAt field is in the past, can execute now (now: %s, StartAt: %s)", taskIDStr, time.Now(), info.StartAt)
			}
		}
	}

	if task.WorkflowStep != nil {
		// check if CWL-style predecessors are all TASK_STAT_COMPLETED
		// ****** get inputs
		if job == nil {
			err = fmt.Errorf("(isTaskReady) job == nil")
			return
		}

		if job.WorkflowContext == nil {
			err = fmt.Errorf("(isTaskReady) job.WorkflowContext == nil")
			return
		}

		//job_input_map := *job.WorkflowContext.Job_input_map
		//if job_input_map == nil {
		//	err = fmt.Errorf("(isTaskReady) job.CWL_collection.Job_input_map is empty")
		//	return
		//}

		var workflow_instance *WorkflowInstance

		var ok bool
		workflow_instance, ok, err = job.GetWorkflowInstance(task.WorkflowInstanceId, true)
		if err != nil {
			err = fmt.Errorf("(isTaskReady) GetWorkflowInstance returned %s", err.Error())
			return
		}

		if !ok {
			//spew.Dump(job.WorkflowInstancesMap)

			for key, _ := range job.WorkflowInstancesMap {
				fmt.Printf("WorkflowInstancesMap: %s\n", key)
			}

			ready = false
			reason = fmt.Sprintf("(isTaskReady) WorkflowInstance not found: %s", task.WorkflowInstanceId)
			return
		}

		//workflow_input_map := workflow_instance.Inputs.GetMap()
		//workflowInstanceID := workflow_instance.LocalID
		//workflow_def := workflow_instance.Workflow_Definition

		//fmt.Println("WorkflowStep.Id: " + task.WorkflowStep.Id)
		ready, reason, err = qm.areSourceGeneratorsReady(task.WorkflowStep, job, workflow_instance)
		if err != nil {
			err = fmt.Errorf("(isTaskReady) areSourceGeneratorsReady returned: %s", err.Error())
			return
		}
		if !ready {
			reason = fmt.Sprintf("(isTaskReady) areSourceGeneratorsReady returned: %s", reason)
			return
		}
		logger.Debug(3, "(isTaskReady) areSourceGeneratorsReady reports task %s as ready", taskIDStr)
	}

	if task.WorkflowStep == nil {
		// task read lock, check DependsOn list and IO.Origin list
		reason, err = task.ValidateDependants(qm)
		if err != nil {
			err = fmt.Errorf("(isTaskReady) %s", err.Error())
			return
		}
		if reason != "" {
			reason = "ValidateDependants returned: " + reason
			return
		}
	}

	// now we are ready
	err = task.SetState(nil, TASK_STAT_READY, true)
	if err != nil {
		err = fmt.Errorf("(isTaskReady) task.SetState returned: %s", err.Error())
		return
	}
	ready = true

	logger.Debug(3, "(isTaskReady) finished, task %s is ready", taskIDStr)
	return
}

func (qm *ServerMgr) taskEnQueueWorkflow_deprecated(task *Task, job *Job, workflow_input_map cwl.JobDocMap, workflow *cwl.Workflow, logTimes bool) (times map[string]time.Duration, err error) {

	if workflow == nil {
		err = fmt.Errorf("(taskEnQueueWorkflow) workflow == nil !?")
		return
	}

	if len(task.ScatterChildren) > 0 {
		return
	}

	cwl_step := task.WorkflowStep
	task_id := task.Task_Unique_Identifier
	taskIDStr, _ := task_id.String()

	workflow_defintion_id := workflow.Id

	var workflowInstanceID string

	workflowInstanceID = task_id.TaskName

	parent_workflowInstanceID := task.WorkflowInstanceId

	// find inputs
	var task_input_array cwl.Job_document
	var task_input_map cwl.JobDocMap

	context := job.WorkflowContext

	if task.StepInput == nil {

		var workflow_instance *WorkflowInstance
		var ok bool
		workflow_instance, ok, err = job.GetWorkflowInstance(parent_workflowInstanceID, true)
		if err != nil {
			err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) job.GetWorkflowInstance returned: %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) workflow_instance not found")
			return
		}

		var reason string
		task_input_map, ok, reason, err = qm.GetStepInputObjects(job, workflow_instance, workflow_input_map, cwl_step, context, "taskEnQueueWorkflow") // returns map[string]CWLType
		if err != nil {
			err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) GetStepInputObjects returned: %s", err.Error())
			return
		}

		if !ok {
			err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) GetStepInputObjects not ready, reason: %s", reason)
			return
		}

		if len(task_input_map) == 0 {
			err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) A) len(task_input_map) == 0 (%s)", taskIDStr)
			return
		}

		task_input_array, err = task_input_map.GetArray()
		if err != nil {
			err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) task_input_map.GetArray returned: %s", err.Error())
			return
		}
		task.StepInput = &task_input_array
		//task_input_map = task_input_array.GetMap()

	} else {
		task_input_array = *task.StepInput
		task_input_map = task_input_array.GetMap()
		if len(task_input_map) == 0 {
			err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) B) len(task_input_map) == 0 ")
			return
		}
	}

	if strings.HasSuffix(task.TaskName, "/") {
		err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) Slash at the end of TaskName!? %s", task.TaskName)
		return
	}

	// embedded workflows have a uniqe name relative to the parent workflow: e.g #main/steo0/<uuid>
	// stand-alone workflows have no unique name, e.g: #sometool

	//new_sub_workflow := ""
	//fmt.Printf("(taskEnQueueWorkflow) new_sub_workflow: %s - %s\n", task.Parent, task.TaskName)
	//if len(task.Parent) > 0 {
	//	new_sub_workflow = task.Parent + task.TaskName // TaskName starts with #, so we can split later
	//} else {
	//	new_sub_workflow = task.TaskName
	//}

	//new_sub_workflow := workflow_id

	//fmt.Printf("New Subworkflow: %s %s\n", task.Parent, task.TaskName)

	// New WorkflowInstance defined input nd ouput of this subworkflow
	// create tasks
	//var sub_workflow_tasks []*Task
	//sub_workflow_tasks, err = CreateWorkflowTasks(job, workflowInstanceID, workflow.Steps, workflow.Id, &task_id)
	//if err != nil {
	//	err = fmt.Errorf("(taskEnQueueWorkflow) CreateWorkflowTasks returned: %s", err.Error())
	//		return
	//	}

	var wi *WorkflowInstance

	wi, err = NewWorkflowInstance(workflowInstanceID, job.ID, workflow_defintion_id, job, parent_workflowInstanceID)
	if err != nil {
		err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) NewWorkflowInstance returned: %s", err.Error())
		return
	}
	wi.Inputs = task_input_array
	err = wi.SetState(WIStatePending, DbSyncTrue, false)
	if err != nil {
		err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) wi.SetState(WIStatePending returned: %s", err.Error())
		return
	}

	err = job.AddWorkflowInstance(wi, DbSyncTrue, true) // taskEnQueueWorkflow_deprecated
	if err != nil {
		err = fmt.Errorf("(taskEnQueueWorkflow_deprecated) job.AddWorkflowInstance returned: %s", err.Error())
		return
	}

	times = make(map[string]time.Duration)
	//skip_workunit := false

	//var children []string
	// for i := range sub_workflow_tasks {
	// 	sub_task := sub_workflow_tasks[i]
	// 	_, err = sub_task.Init(job, job.ID)
	// 	if err != nil {
	// 		err = fmt.Errorf("(taskEnQueueWorkflow) sub_task.Init() returns: %s", err.Error())
	// 		return
	// 	}

	// 	var sub_task_id Task_Unique_Identifier
	// 	sub_task_id, err = sub_task.GetId("task." + strconv.Itoa(i))
	// 	if err != nil {
	// 		return
	// 	}
	// 	sub_taskIDStr, _ := sub_task_id.String()
	// 	//children = append(children, sub_task_id)

	// 	//err = job.AddTask(sub_task)
	// 	err = wi.AddTask(job, sub_task, DbSyncTrue, true)
	// 	if err != nil {
	// 		err = fmt.Errorf("(taskEnQueueWorkflow) job.AddTask returns: %s", err.Error())
	// 		return
	// 	}

	// 	// add to qm.TaskMap
	// 	// updateQueue() process will actually enqueue the task
	// 	// TaskMap.Add - makes it a pending task if init, throws error if task already in map with different pointer
	// 	err = qm.TaskMap.Add(sub_task, "taskEnQueueWorkflow")
	// 	if err != nil {
	// 		err = fmt.Errorf("(taskEnQueueWorkflow) (subtask: %s) qm.TaskMap.Add() returns: %s", sub_taskIDStr, err.Error())
	// 		return
	// 	}
	// }
	//task.SetWorkflowChildren(qm, children, true)

	return
}

func (qm *ServerMgr) taskEnQueueScatter(workflow_instance *WorkflowInstance, task *Task, job *Job, workflow_input_map cwl.JobDocMap) (notice *Notice, err error) {
	notice = nil
	// TODO store info that this has been evaluated
	cwl_step := task.WorkflowStep
	task_id := task.Task_Unique_Identifier

	count_of_scatter_arrays := len(cwl_step.Scatter)

	scatter_names_map := make(map[string]int, count_of_scatter_arrays)

	for i, name := range cwl_step.Scatter {
		name_base := path.Base(name)
		//fmt.Printf("scatter_names_map, name_base: %s\n", name_base)
		scatter_names_map[name_base] = i
	}

	//  copy

	scatter_method := cwl_step.ScatterMethod
	_ = scatter_method

	scatter_positions := make([]int, count_of_scatter_arrays)
	scatter_source_strings := make([]string, count_of_scatter_arrays) // an array of strings, where each string is a source pointing to an array

	name_to_postiton := make(map[string]int, count_of_scatter_arrays)

	scatter_input_arrays := make([]cwl.Array, count_of_scatter_arrays)

	// search for scatter source arrays
	// fill map name_to_postiton and arrays scatter_positions and scatter_source_strings
	for i, scatter_input_name := range cwl_step.Scatter {

		scatter_input_name_base := path.Base(scatter_input_name)
		//fmt.Printf("scatter_input detected: %s\n", scatter_input_name)

		name_to_postiton[scatter_input_name_base] = i // this just an inverse which is needed later

		scatter_input_source_str := ""
		input_position := -1

		// search for workflow_step_input
		for j, _ := range cwl_step.In {
			workflow_step_input := cwl_step.In[j]

			if path.Base(workflow_step_input.Id) == scatter_input_name_base {
				input_position = j
				break
			}
		}

		// error if workflow_step_input not found
		if input_position == -1 {
			// improve error message
			list_of_inputs := ""
			for j, _ := range cwl_step.In {
				workflow_step_input := cwl_step.In[j]
				list_of_inputs += "," + path.Base(workflow_step_input.Id)
			}

			err = fmt.Errorf("(taskEnQueue) Input %s not found in list of step.Inputs (list: %s)", scatter_input_name_base, list_of_inputs)
			return
		}

		workflow_step_input := cwl_step.In[input_position]
		scatter_input_source := workflow_step_input.Source

		switch scatter_input_source.(type) {
		case string:

			scatter_input_source_str = scatter_input_source.(string)
		case []string, []interface{}:

			var scatter_input_source_array []string

			scatter_input_source_array_if, ok := scatter_input_source.([]interface{})
			if ok {
				scatter_input_source_array = []string{}
				for k, _ := range scatter_input_source_array_if {
					var src_str string
					src_str, ok = scatter_input_source_array_if[k].(string)
					if !ok {
						err = fmt.Errorf("(taskEnQueueScatter) element in source array is not a string")
						return
					}
					scatter_input_source_array = append(scatter_input_source_array, src_str)
				}

			} else {

				scatter_input_source_array = scatter_input_source.([]string)
			}

			scatter_input_arrays[i], ok, err = qm.getCWLSourceArray(workflow_instance, workflow_input_map, job, task_id, scatter_input_source_array, true)
			if err != nil {
				err = fmt.Errorf("(taskEnQueueScatter) getCWLSourceArray returned: %s", err.Error())
				return
			}
			if !ok {
				err = fmt.Errorf("(taskEnQueueScatter) element not found") // should not happen, error would have been thrown
				return
			}
			scatter_input_source_str = "_array_"
		default:
			err = fmt.Errorf("(taskEnQueueScatter) scatter_input_source is not string (%s)", reflect.TypeOf(scatter_input_source))
			return
		}

		scatter_positions[i] = input_position
		scatter_source_strings[i] = scatter_input_source_str

	}

	// get each scatter array and cast into array

	empty_array := false // just for compliance tests
	for i := 0; i < count_of_scatter_arrays; i++ {

		scatter_input := cwl_step.Scatter[i]
		scatter_input_source_str := scatter_source_strings[i]

		if scatter_input_source_str == "_array_" {
			if scatter_input_arrays[i].Len() == 0 {
				empty_array = true
			}

			continue
		}
		// get array (have to cast into array still)
		var scatter_input_object cwl.CWLObject
		var ok bool
		scatter_input_object, ok, _, err = qm.getCWLSource(job, workflow_instance, workflow_input_map, scatter_input_source_str, true, job.WorkflowContext)
		if err != nil {
			err = fmt.Errorf("(taskEnQueueScatter) getCWLSource returned: %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(taskEnQueueScatter) scatter_input %s not found.", scatter_input)
			return
		}

		var scatter_input_array_ptr *cwl.Array
		scatter_input_array_ptr, ok = scatter_input_object.(*cwl.Array)
		if !ok {

			err = fmt.Errorf("(taskEnQueueScatter) scatter_input_object type is not *cwl.Array: %s", reflect.TypeOf(scatter_input_object))
			return
		}

		if scatter_input_array_ptr.Len() == 0 {

			empty_array = true
		}

		scatter_input_arrays[i] = *scatter_input_array_ptr

	}
	scatter_type := ""
	// dotproduct with 2 or more arrays
	if scatter_method == "" || strings.ToLower(scatter_method) == "dotproduct" {
		// requires that all arrays are the same length

		scatter_type = "dot"

	} else if strings.ToLower(scatter_method) == "nested_crossproduct" || strings.ToLower(scatter_method) == "flat_crossproduct" {
		// arrays do not have to be the same length
		// nested_crossproduct and flat_crossproduct differ only in hor results are merged

		// defined counter for iteration over all combinations
		scatter_type = "cross"
	} else {
		err = fmt.Errorf("(taskEnQueueScatter) Scatter type %s unknown", scatter_method)
		return
	}
	// 1. Create template step with scatter inputs removed
	//cwl_step := task.WorkflowStep

	template_task_step := *cwl_step // this should make a copy , not nested copy

	//fmt.Println("template_task_step inital:\n")
	//spew.Dump(template_task_step)

	// remove scatter
	var template_step_in []cwl.WorkflowStepInput
	template_scatter_step_ins := make(map[string]cwl.WorkflowStepInput, count_of_scatter_arrays)
	for i, _ := range cwl_step.In {

		i_input := cwl_step.In[i]
		i_input_id_base := path.Base(i_input.Id)
		//fmt.Printf("i_input_id_base: %s\n", i_input_id_base)
		_, ok := scatter_names_map[i_input_id_base] // skip scatter inputs
		if ok {
			template_scatter_step_ins[i_input_id_base] = i_input // save scatter inputs in template_scatter_step_ins
			continue
		}
		template_step_in = append([]cwl.WorkflowStepInput{i_input}, template_step_in...) // preprend i_input

	}

	//fmt.Println("template_scatter_step_ins:")
	//spew.Dump(template_scatter_step_ins)
	if len(template_scatter_step_ins) == 0 {
		err = fmt.Errorf("(taskEnQueueScatter) no scatter tasks found")
		return
	}

	// overwrite array, keep only non-scatter
	template_task_step.In = template_step_in
	counter := NewSetCounter(count_of_scatter_arrays, scatter_input_arrays, scatter_type)

	if empty_array {

		//task_already_finished = true

		// create dummy Workunit
		var workunit *Workunit
		workunit, err = NewWorkunit(qm, task, 0, job)
		if err != nil {
			err = fmt.Errorf("(taskEnQueueScatter) Creation of fake workunitfailed: %s", err.Error())
			return
		}
		qm.workQueue.Add(workunit)
		err = workunit.SetState(WORK_STAT_CHECKOUT, "internal processing")
		if err != nil {
			err = fmt.Errorf("(taskEnQueueScatter) workunit.SetState failed: %s", err.Error())
			return
		}

		// create empty arrays for output and return notice

		notice = &Notice{}
		notice.WorkerID = "_internal"
		notice.ID = New_Workunit_Unique_Identifier(task.Task_Unique_Identifier, 0)
		notice.Status = WORK_STAT_DONE
		notice.ComputeTime = 0

		notice.Results = &cwl.Job_document{}

		for _, out := range cwl_step.Out {
			//
			out_name := out.Id
			//fmt.Printf("outname: %s\n", out_name)

			new_array := &cwl.Array{}

			//new_out := cwl.NewNamedCWLType(out_name, new_array)
			//spew.Dump(*notice.Results)
			notice.Results = notice.Results.Add(out_name, new_array)
			//spew.Dump(*notice.Results)
		}

		//fmt.Printf("len(notice.Results): %d\n", len(*notice.Results))

		return
	}

	//fmt.Println("template_task_step without scatter:")
	//spew.Dump(template_task_step)

	//if count_of_scatter_arrays == 1 {

	// create tasks
	var children []string
	var new_scatter_tasks []*Task

	counter_running := true

	basename := path.Base(task_id.TaskName)

	parent_id_str := strings.TrimSuffix(task_id.TaskName, "/"+basename)

	if parent_id_str == task_id.TaskName {
		err = fmt.Errorf("(taskEnQueue) parent_id_str == task_id.TaskName")
		return
	}

	for counter_running {

		permutation_instance := ""
		for i := 0; i < counter.NumberOfSets-1; i++ {
			permutation_instance += strconv.Itoa(counter.Counter[i]) + "_"
		}
		permutation_instance += strconv.Itoa(counter.Counter[counter.NumberOfSets-1])

		scatter_task_name := basename + "_scatter" + permutation_instance

		// create task
		var sub_task *Task

		//var parent_id_str string
		// parent_id_str, err = task.GetWorkflowParentStr()
		// if err != nil {
		// 	err = fmt.Errorf("(taskEnQueue) task.GetWorkflowParentStr returned: %s", err.Error())
		// 	return
		// }
		//parent_id_str = "test"
		logger.Debug(3, "(taskEnQueueScatter) New Task: parent: %s and scatter_task_name: %s", parent_id_str, scatter_task_name)

		sub_task, err = NewTask(job, parent_id_str, scatter_task_name)
		if err != nil {
			err = fmt.Errorf("(taskEnQueueScatter) NewTask returned: %s", err.Error())
			return
		}

		sub_task.Scatter_parent = &task.Task_Unique_Identifier
		//awe_task.Scatter_task = true
		_, err = sub_task.Init(job, job.ID)
		if err != nil {
			err = fmt.Errorf("(taskEnQueueScatter) awe_task.Init() returns: %s", err.Error())
			return
		}
		sub_task.TaskType = TASK_TYPE_NORMAL // unless this is a scatter of a scatter...

		// create step
		var new_task_step cwl.WorkflowStep
		//var new_task_step_in []cwl.WorkflowStepInput
		new_task_step = template_task_step // this should make a copy from template, (this is not a nested copy)

		//fmt.Println("new_task_step initial:")
		//spew.Dump(new_task_step)

		// copy scatter inputs

		for input_name := range template_scatter_step_ins {
			input_name_base := path.Base(input_name)
			scatter_input, ok := template_scatter_step_ins[input_name_base]
			if !ok {
				err = fmt.Errorf("(taskEnQueueScatter) %s not in template_scatter_step_ins", input_name_base)
				return
			}

			input_position, ok := name_to_postiton[input_name_base] // input_position points to an array of inputs
			if !ok {
				err = fmt.Errorf("(taskEnQueueScatter) %s not in name_to_postiton map", input_name_base)
				return
			}

			//the_array := scatter_input_array_ptrs[input_position]

			the_index := counter.Counter[input_position]
			scatter_input.Source_index = the_index + 1
			new_task_step.In = append(new_task_step.In, scatter_input)
		}

		fmt.Println("new_task_step with everything:")
		spew.Dump(new_task_step)

		new_task_step.Id = parent_id_str + "/" + scatter_task_name
		sub_task.WorkflowStep = &new_task_step
		children = append(children, scatter_task_name)

		new_task_step.Scatter = nil // []string{}
		new_scatter_tasks = append(new_scatter_tasks, sub_task)

		counter_running = counter.Increment()
	}

	err = task.SetScatterChildren(qm, children, true)
	if err != nil {
		err = fmt.Errorf("(taskEnQueueScatter) task.SetScatterChildren returned: %s", err.Error())
		return
	}

	// add tasks to job and submit
	for i := range new_scatter_tasks {
		sub_task := new_scatter_tasks[i]
		sub_task_id, _ := sub_task.GetId("taskEnQueueScatter")
		sub_taskIDStr, _ := sub_task_id.String()
		logger.Debug(3, "(taskEnQueueScatter) adding %s to workflow_instance", sub_taskIDStr)
		err = workflow_instance.AddTask(job, sub_task, DbSyncTrue, true)
		if err != nil {
			err = fmt.Errorf("(taskEnQueueScatter) job.AddTask returns: %s", err.Error())
			return
		}

		err = qm.TaskMap.Add(sub_task, "taskEnQueueScatter")
		if err != nil {
			sub_taskIDStr := sub_task.Id
			err = fmt.Errorf("(taskEnQueueScatter) sub_taskIDStr=%s qm.TaskMap.Add() returns: %s", sub_taskIDStr, err.Error())
			return
		}

		err = sub_task.SetState(nil, TASK_STAT_READY, true)
		if err != nil {
			sub_taskIDStr := sub_task.Id
			err = fmt.Errorf("(taskEnQueueScatter) sub_taskIDStr=%s sub_task.SetState returns: %s", sub_taskIDStr, err.Error())
			return
		}
	}

	return
}

// happens when task is ready
// prepares task and creates workunits or workflow_instances
// scatter task does not create its own workunit, it just creates new tasks

// workflow: create workflow_instance and tasks
// scatter: create tasks
// commandlinetool: create workunit/s

func (qm *ServerMgr) taskEnQueue(taskID Task_Unique_Identifier, task *Task, job *Job, logTimes bool) (times map[string]time.Duration, err error) {

	taskIDStr, _ := taskID.String()

	var state string
	state, err = task.GetState()
	if err != nil {
		err = fmt.Errorf("(taskEnQueue) Could not get State: %s", err.Error())
		return
	}

	if state != TASK_STAT_READY {
		err = fmt.Errorf("(taskEnQueue) Task state should be TASK_STAT_READY, got state %s", state)
		return
	}

	if task.WorkflowStep != nil {
		logger.Debug(3, "(taskEnQueue) have WorkflowStep")
	} else {
		logger.Debug(3, "(taskEnQueue) DO NOT have WorkflowStep")
	}

	skip_workunit := false

	var task_type string
	task_type, err = task.GetTaskType()
	if err != nil {
		return
	}

	var notice *Notice
	notice = nil

	logger.Debug(3, "(taskEnQueue) have job.WorkflowContext")

	var workflow_instance *WorkflowInstance
	var workflow_input_map cwl.JobDocMap
	if task.WorkflowInstanceId != "" {
		var ok bool
		workflow_instance, ok, err = job.GetWorkflowInstance(task.WorkflowInstanceId, true)
		if err != nil {
			err = fmt.Errorf("(taskEnQueue) GetWorkflowInstance returned %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(taskEnQueue) WorkflowInstance not found: \"%s\"", task.WorkflowInstanceId)
			return
		}

		workflow_input_map = workflow_instance.Inputs.GetMap()
		cwl_step := task.WorkflowStep

		if cwl_step == nil {
			err = fmt.Errorf("(taskEnQueue) task.WorkflowStep is empty")
			return
		}
	}
	//workflow_with_children := false
	//var wfl *cwl.Workflow

	// Detect task_type

	//fmt.Printf("(taskEnQueue) A task_type: %s\n", task_type)

	//if task_type == "" {
	//	err = fmt.Errorf("(taskEnQueue) task_type empty")
	//	return
	//}

	if task.WorkflowInstanceId != "" {
		switch task_type {
		case TASK_TYPE_SCATTER:
			logger.Debug(3, "(taskEnQueue) call taskEnQueueScatter")
			notice, err = qm.taskEnQueueScatter(workflow_instance, task, job, workflow_input_map)
			if err != nil {
				err = fmt.Errorf("(taskEnQueue) taskEnQueueScatter returned: %s", err.Error())
				return
			}

		}
	}
	logger.Debug(2, "(taskEnQueue) task %s has type %s", taskIDStr, task_type)
	if task_type == TASK_TYPE_SCATTER {
		skip_workunit = true
	}

	logger.Debug(2, "(taskEnQueue) trying to enqueue task %s", taskIDStr)

	// if task was flagged by resume, recompute, or resubmit - reset it
	err = task.SetResetTask(job.Info)
	if err != nil {
		err = fmt.Errorf("(taskEnQueue) SetResetTask: %s", err.Error())
		return
	}

	inputeStart := time.Now()
	err = qm.locateInputs(task, job) // only old-style AWE
	if logTimes {
		times["locateInputs"] = time.Since(inputeStart)
	}
	if err != nil {
		err = fmt.Errorf("(taskEnQueue) locateInputs: %s", err.Error())
		return
	}

	// init partition
	indexStart := time.Now()
	err = task.InitPartIndex()
	if logTimes {
		times["InitPartIndex"] = time.Since(indexStart)
	}
	if err != nil {
		err = fmt.Errorf("(taskEnQueue) InitPartitionIndex: %s", err.Error())
		return
	}

	outputStart := time.Now()
	err = qm.createOutputNode(task)
	if logTimes {
		times["createOutputNode"] = time.Since(outputStart)
	}
	if err != nil {
		err = fmt.Errorf("(taskEnQueue) createOutputNode: %s", err.Error())
		return
	}

	if !skip_workunit {
		logger.Debug(3, "(taskEnQueue) create Workunits")
		workunitStart := time.Now()
		err = qm.CreateAndEnqueueWorkunits(task, job)
		if err != nil {
			err = fmt.Errorf("(taskEnQueue) CreateAndEnqueueWorkunits: %s", err.Error())
			return
		}
		if logTimes {
			times["CreateAndEnqueueWorkunits"] = time.Since(workunitStart)
		}
	}
	err = task.SetState(nil, TASK_STAT_QUEUED, true)
	if err != nil {
		return
	}
	err = task.SetCreatedDate(time.Now()) // TODO: this is pretty stupid and useless. May want to use EnqueueDate here ?
	if err != nil {
		return
	}
	err = task.SetStartedDate(time.Now()) //TODO: will be changed to the time when the first workunit is checked out
	if err != nil {
		return
	}

	//updateStart := time.Now()
	// err = qm.taskCompleted(workflow_instance, task) //task status PENDING->QUEUED
	// if logTimes {
	// 	times["taskCompleted"] = time.Since(updateStart)
	// }
	// if err != nil {
	// 	err = fmt.Errorf("(taskEnQueue) qm.taskCompleted: %s", err.Error())
	// 	return
	// }

	// log event about task enqueue (TQ)
	logger.Event(event.TASK_ENQUEUE, fmt.Sprintf("taskid=%s;totalwork=%d", taskIDStr, task.TotalWork))
	err = qm.CreateTaskPerf(task)
	if err != nil {
		err = fmt.Errorf("(taskEnQueue) CreateTaskPerf returned: %s", err.Error())
		return
	}
	logger.Debug(2, "(taskEnQueue) leaving (task=%s)", taskIDStr)

	if notice != nil {
		// This task is not processed by a worker and thus no notice would be sent. For correct completion the server creates and sends an internal Notice.

		//WorkerId    string                     `bson:"worker_id" json:"worker_id" mapstructure:"worker_id"`
		//Results     *cwl.Job_document          `bson:"results" json:"results" mapstructure:"results"`                            // subset of tool_results with Shock URLs

		qm.feedback <- *notice

	}

	return
}

// invoked by taskEnQueue
// main purpose is to copy output io struct of predecessor task to create the input io structs
func (qm *ServerMgr) locateInputs(task *Task, job *Job) (err error) {
	if task.WorkflowStep != nil && job.WorkflowContext != nil {
		//if job.WorkflowContext.Job_input_map == nil {
		//err = fmt.Errorf("job.WorkflowContext.Job_input_map is empty")
		//	return
		//}
	} else {
		// old AWE-style
		err = task.ValidateInputs(qm)
		if err != nil {
			return
		}
		err = task.ValidatePredata()
		if err != nil {
			err = fmt.Errorf("ValidatePredata failed: %s", err.Error())
		}
	}
	return
}

func (qm *ServerMgr) getCWLSourceArray(workflow_instance *WorkflowInstance, workflow_input_map map[string]cwl.CWLType, job *Job, current_task_id Task_Unique_Identifier, src_array []string, error_on_missing_task bool) (obj cwl.Array, ok bool, err error) {

	obj = cwl.Array{}
	ok = false

	for _, src := range src_array {
		var element cwl.CWLType
		var src_ok bool
		element, src_ok, _, err = qm.getCWLSource(job, workflow_instance, workflow_input_map, src, error_on_missing_task, job.WorkflowContext)
		if err != nil {
			err = fmt.Errorf("(getCWLSourceArray) getCWLSource returned: %s", err.Error())
			return
		}
		if !src_ok {
			if error_on_missing_task {
				err = fmt.Errorf("(getCWLSourceArray) Source %s not found", src)
				return
			}

			// return with ok=false, but do not throw error
			ok = false
			return
		}

		obj = append(obj, element)
	}
	ok = true
	return

}

func (qm *ServerMgr) getCWLSourceFromWorkflowInput(workflow_input_map map[string]cwl.CWLType, src_base string) (obj cwl.CWLType, ok bool, err error) {

	//fmt.Println("src_base: " + src_base)
	// search job input
	var this_ok bool
	obj, this_ok = workflow_input_map[src_base]
	if this_ok {
		//fmt.Println("(getCWLSource) found in workflow_input_map: " + src_base)
		ok = true
		return
	}

	ok = false
	return
	// } else {

	// 	// workflow inputs that are missing might be optional, thus Null is returned
	// 	/obj = cwl.NewNull()
	// 	ok = true
	// 	// not found
	// 	return
	// }
	//fmt.Println("(getCWLSource) workflow_input_map:")
	//spew.Dump(workflow_input_map)
}

func (qm *ServerMgr) getCWLSourceFromStepOutput_Tool(job *Job, workflow_instance *WorkflowInstance, workflow_name string, step_name string, output_name string, error_on_missing_task bool) (obj cwl.CWLType, ok bool, reason string, err error) {

	// search task and its output
	if workflow_instance.JobID == "" {
		err = fmt.Errorf("(getCWLSourceFromStepOutput) workflow_instance.JobId empty")
		return
	}
	//workflowInstanceID, _ := workflow_instance.GetId(true)
	workflowInstanceLocalID := workflow_instance.LocalID

	ancestor_task_name_local := workflowInstanceLocalID + "/" + step_name

	ancestor_task_id := Task_Unique_Identifier{}
	ancestor_task_id.JobId = workflow_instance.JobID
	ancestor_task_id.TaskName = ancestor_task_name_local

	ancestor_taskIDStr, _ := ancestor_task_id.String()
	var ancestor_task *Task

	ancestor_task, ok, err = workflow_instance.GetTask(ancestor_task_id, true)
	if err != nil {
		err = fmt.Errorf("(getCWLSourceFromStepOutput) workflow_instance.GetTask returned: %s", err.Error())
		return
	}
	if !ok {

		tasks_str := ""
		if len(workflow_instance.Tasks) > 0 {
			for i, _ := range workflow_instance.Tasks {
				t_str, _ := workflow_instance.Tasks[i].String()
				tasks_str += "," + t_str
			}
		} else {
			tasks_str = "no tasks found"
		}
		reason = fmt.Sprintf("ancestor_task %s not found in workflow_instance %s (tasks found: %s)", ancestor_taskIDStr, workflowInstanceLocalID, tasks_str)

		//spew.Dump(workflow_instance)

		return
	}

	obj, ok, reason, err = ancestor_task.GetStepOutput(output_name)
	if err != nil {
		err = fmt.Errorf("(getCWLSourceFromStepOutput) ancestor_task.GetStepOutput returned: %s", err.Error())
		return
	}
	if !ok {
		reason = fmt.Sprintf("Output %s not found in output of ancestor_task (%s)", ancestor_task_name_local, ancestor_taskIDStr)
		return
	}

	return
}

func (qm *ServerMgr) getCWLSourceFromStepOutput_Workflow(job *Job, workflow_instance *WorkflowInstance, step_name string, output_name string, error_on_missing_task bool) (obj cwl.CWLType, ok bool, reason string, err error) {

	local_name := workflow_instance.LocalID

	subworkflow_name := local_name + "/" + step_name

	var subwi *WorkflowInstance
	subwi, ok, err = job.GetWorkflowInstance(subworkflow_name, true)
	if err != nil {
		err = fmt.Errorf("(getCWLSourceFromStepOutput_Workflow) job.GetWorkflowInstance returned: %s", err.Error())
		return
	}

	if !ok {

		steps := ""
		for _, wi := range job.WorkflowInstancesMap {

			steps += "," + wi.LocalID
		}

		err = fmt.Errorf("(getCWLSourceFromStepOutput_Workflow) step %s (a Workflow) not found (found: %s)", subworkflow_name, steps)
		return
	}

	//var obj cwl.CWLType
	obj, ok, err = subwi.GetOutput(output_name, true)
	if err != nil {
		err = fmt.Errorf("(getCWLSourceFromStepOutput_Workflow) subwi.GetOutput returned: %s", err.Error())
		return
	}

	if !ok {

		outputs := ""
		for i, _ := range subwi.Outputs {
			named := subwi.Outputs[i]
			outputs += "," + named.Id
		}

		err = fmt.Errorf("(getCWLSourceFromStepOutput_Workflow) output %s not found in workflow %s (found: %s)", output_name, subworkflow_name, outputs)
		return
	}

	return
}

// To get StepOutput function has to distinguish between task (CommandLine/Expression-Tool) and Subworkflow
// src = workflow_name / step_name / output_name
func (qm *ServerMgr) getCWLSourceFromStepOutput(job *Job, workflow_instance *WorkflowInstance, workflow_name string, step_name string, output_name string, error_on_missing_task bool) (obj cwl.CWLType, ok bool, reason string, err error) {
	ok = false
	//step_name_abs := workflow_name + "/" + step_name
	workflowInstanceID, _ := workflow_instance.GetID(true)
	//workflowInstanceLocalID := workflow_instance.LocalID

	logger.Debug(3, "(getCWLSourceFromStepOutput) %s / %s / %s (workflowInstanceID: %s)", workflow_name, step_name, output_name, workflowInstanceID)
	_ = workflowInstanceID
	// *** check if workflow_name + "/" + step_name is a subworkflow
	//workflowInstanceName := workflow_name + "/" + step_name

	//var wi *WorkflowInstance

	//wi, ok, err = job.GetWorkflowInstance(workflowInstanceName, true)
	//if err != nil {
	//	err = fmt.Errorf("(getCWLSourceFromStepOutput) job.GetWorkflowInstance returned: %s", err.Error())
	//	return
	//}

	//if ok {
	ok = false

	context := job.WorkflowContext

	// get workflow
	var workflow *cwl.Workflow
	workflow, err = workflow_instance.GetWorkflow(context)

	var step *cwl.WorkflowStep
	step, err = workflow.GetStep(step_name)
	if err != nil {

		steps := ""
		for i, _ := range workflow.Steps {
			s2 := &workflow.Steps[i]
			steps += "," + s2.Id
		}

		err = fmt.Errorf("(getCWLSourceFromStepOutput) Step %s not found (found %s)", step_name, steps)
		return
	}

	// determinre process type
	var process_type string
	process_type, err = step.GetProcessType(context)
	if err != nil {
		err = fmt.Errorf("(getCWLSourceFromStepOutput) step.GetProcessType returned: %s", err.Error())
		return
	}

	switch process_type {
	case "CommandLineTool", "ExpressionTool":
		obj, ok, reason, err = qm.getCWLSourceFromStepOutput_Tool(job, workflow_instance, workflow_name, step_name, output_name, error_on_missing_task)
		if err != nil {
			err = fmt.Errorf("(getCWLSourceFromStepOutput) getCWLSourceFromStepOutput_Tool returned: %s", err.Error())
			return
		}
	case "Workflow":
		obj, ok, reason, err = qm.getCWLSourceFromStepOutput_Workflow(job, workflow_instance, step_name, output_name, error_on_missing_task)
		if err != nil {
			err = fmt.Errorf("(getCWLSourceFromStepOutput) getCWLSourceFromStepOutput_Tool returned: %s", err.Error())
			return
		}
	default:
		err = fmt.Errorf("(getCWLSourceFromStepOutput) process_type %s unknown", process_type)
		return
	}

	//err = fmt.Errorf("(getCWLSourceFromStepOutput) not a subworkflow")
	return

}

func (qm *ServerMgr) GetSourceFromWorkflowInstanceInput(workflow_instance *WorkflowInstance, src string, context *cwl.WorkflowContext, error_on_missing_task bool) (obj cwl.CWLType, ok bool, reason string, err error) {

	ok = false

	fmt.Printf("(GetSourceFromWorkflowInstanceInput) src: %s\n", src)
	src_base := path.Base(src)

	fmt.Printf("(GetSourceFromWorkflowInstanceInput) src_base: %s\n", src_base)
	src_path := strings.TrimSuffix(src, "/"+src_base)

	//src_array := strings.Split(src, "/")
	//src_base := src_array[1]

	if workflow_instance == nil {
		err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) workflow_instance==nil (src: %s)", src)
		return
	}

	if workflow_instance.Inputs == nil {
		if error_on_missing_task {

			err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) workflow_instance.Inputs empty (src: %s)", src)
			return
		}

		err = nil
		msg := fmt.Sprintf("(GetSourceFromWorkflowInstanceInput) workflow_instance.Inputs empty (src: %s)", src)
		logger.Debug(3, msg)
		ok = false
		reason = msg
		return
	}

	obj, ok = workflow_instance.Inputs.Get(src_base)
	if !ok {

		var workflow *cwl.Workflow
		workflow, err = workflow_instance.GetWorkflow(context)
		if err != nil {
			err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) workflow_instance.GetWorkflow returned: %s", err.Error())
			return
		}

		// check if input is optional
		optional := false
		for i, _ := range workflow.Inputs {
			inp := &workflow.Inputs[i]

			for _, input_type := range inp.Type {
				if input_type == cwl.CWLNull {
					optional = true
					break
				}
			}

		}
		if optional {
			reason = "optional"
			ok = false
			return
		}
		if error_on_missing_task {
			//fmt.Println("workflow_instance.Inputs:")
			//spew.Dump(workflow_instance.Inputs)
			//	panic("output not found a)")

			err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) found ancestor_task %s, but output %s not found in workflow_instance.Inputs (was %s)", src_path, src_base, src)
			return
		}

		msg := fmt.Sprintf("(GetSourceFromWorkflowInstanceInput) found ancestor_task %s, but output %s not found in workflow_instance.Inputs", src_path, src_base)
		logger.Debug(3, msg)
		ok = false
		reason = msg
		return
	}

	fmt.Printf("(GetSourceFromWorkflowInstanceInput) found src_base in workflow_instance.Inputs: %s\n", src_base)

	ok = true
	return

}

//
func (qm *ServerMgr) isSourceGeneratorReady(job *Job, workflow_instance *WorkflowInstance, src_generator string, error_on_missing_task bool, context *cwl.WorkflowContext) (ok bool, reason string, err error) {

	ok = false
	//src = strings.TrimPrefix(src, "#main/")

	//src_array := strings.Split(src, "/")
	logger.Debug(3, "(isSourceGeneratorReady) start, src_generator: %s", src_generator)

	var generic_object cwl.CWLObject
	generic_object, ok, err = context.Get(src_generator, true)
	if err != nil {
		err = fmt.Errorf("(isSourceGeneratorReady) context.Get returned: %s", err.Error())
		return
	}
	if !ok {
		reason = fmt.Sprintf("(isSourceGeneratorReady) context.All did not contain %s", src_generator)
		return
	}

	switch generic_object.(type) {
	case *cwl.WorkflowStep:
		logger.Debug(3, "(isSourceGeneratorReady) got WorkflowStep")
		// WorkflowStep does not contain info about state, need workflow_instance
		step_name := src_generator
		workflowInstanceName := path.Dir(src_generator)

		var workflow_instance *WorkflowInstance
		workflow_instance, ok, err = job.GetWorkflowInstance(workflowInstanceName, true)
		if err != nil {
			err = fmt.Errorf("(isSourceGeneratorReady) job.GetWorkflowInstance returned: %s", err.Error())
			return
		}
		if !ok {
			reason = fmt.Sprintf("(isSourceGeneratorReady) workflowInstanceName not found: %s", workflowInstanceName)
			return
		}

		// find task that corresponds to step

		var task *Task
		var tasks []*Task
		tasks, err = workflow_instance.GetTasks(true)
		if err != nil {
			err = fmt.Errorf("(isSourceGeneratorReady) workflow_instance.GetTasks returned: %s", err.Error())
			return
		}
		list_of_tasks := ""
		for i, _ := range tasks {
			t := tasks[i]
			//fmt.Println(t.TaskName)
			list_of_tasks += "," + t.TaskName
			if t.TaskName == step_name {
				task = t
				break
			}
		}

		if task == nil {
			err = fmt.Errorf("(isSourceGeneratorReady) no matching task found: step_name=%s (found : %s)", step_name, list_of_tasks)
			return

		}

		fmt.Printf("(isSourceGeneratorReady) found task\n")

		var task_state string
		task_state, err = task.GetState()
		if task_state != TASK_STAT_COMPLETED {
			ok = false
			task_id, _ := task.GetId("isSourceGeneratorReady")
			taskIDStr, _ := task_id.String()
			reason = fmt.Sprintf("(isSourceGeneratorReady) dependent task %s has state %s", taskIDStr, task_state)
			return
		}
		ok = true

	case *cwl.Workflow:

		workflow := generic_object.(*cwl.Workflow)

		workflow_id := workflow.Id

		if workflow_instance.LocalID != workflow_id {
			err = fmt.Errorf("(isSourceGeneratorReady) workflow_instance.LocalID: %s vs workflow_id %s", workflow_instance.LocalID, workflow_id)
			return
		}

		var wiState string
		wiState, err = workflow_instance.GetState(true)
		if err != nil {
			err = fmt.Errorf("(isSourceGeneratorReady) workflow_instance.GetState returned: %s", err.Error())
			return
		}

		if wiState == WIStateCompleted {
			ok = true
			return
		}

		ok = false
		reason = fmt.Sprintf("(isSourceGeneratorReady) wiState == %s", wiState)
		//wi, job.GetWorkflowInstance(workflow_id, true)

		return
		//workflow := generic_object.(*cwl.Workflow)
		//_ = workflow
		//TODO check state
	default:
		err = fmt.Errorf("(isSourceGeneratorReady) type unknown: %s", reflect.TypeOf(generic_object))
		return

	}

	//fmt.Printf("(isSourceGeneratorReady) got : %s\n", reflect.TypeOf(generic_object))

	//spew.Dump(generic_object)

	return
}

// this retrieves the input from either the (sub-)workflow input, or from the output of another task in the same (sub-)workflow
// error_on_missing_task: when checking if a task is ready, a missing task is not an error, it just means task is not ready,
//    but when getting data this is actually an error.
func (qm *ServerMgr) getCWLSource(job *Job, workflow_instance *WorkflowInstance, workflow_input_map map[string]cwl.CWLType, src string, error_on_missing_task bool, context *cwl.WorkflowContext) (obj cwl.CWLType, ok bool, reason string, err error) {

	ok = false
	//src = strings.TrimPrefix(src, "#main/")

	src_array := strings.Split(src, "/")

	var generic_object cwl.CWLObject
	generic_object, ok, err = context.Get(src, true)
	if err != nil {
		err = fmt.Errorf("(getCWLSource) context.Get returned: %s", err.Error())
		return
	}
	if !ok {
		reason = fmt.Sprintf("(getCWLSource) context.All did not contain %s", src)
		return
	}

	type_str := fmt.Sprintf("%s", reflect.TypeOf(generic_object))

	// var type_str string
	// type_str, err = context.GetType(src)
	// if err != nil {
	// 	err = fmt.Errorf("(getCWLSource) context.GetType returned: %s", err.Error())
	// 	return
	// }

	logger.Debug(3, "(getCWLSource) searching for %s (type: %s)", src, type_str)

	switch type_str {
	case "*cwl.InputParameter", "*cwl.CommandInputParameter": // CommandInputParameter is from CommandLineTool, InputParameter from ExpressionTool
		logger.Debug(3, "(getCWLSource) a workflow input")
		// must be a workflow input, e.g. #main/jobid (workflow, input)
		//src_base := src_array[1]

		obj, ok, reason, err = qm.GetSourceFromWorkflowInstanceInput(workflow_instance, src, context, error_on_missing_task)

		if err != nil {
			err = fmt.Errorf("(getCWLSource) GetSourceFromWorkflowInstanceInput returned: %s", err.Error())
			return
		}

		//if !ok {

		//	fmt.Printf("workflow_instance: (looking for %s)\n", src)
		//	spew.Dump(workflow_instance)
		//	panic("done")
		//}

		return

	case "*cwl.WorkflowStepOutput":
		logger.Debug(3, "(getCWLSource) a step output")
		// must be a step output, e.g. #main/filter/rejected (workflow, step, output)
		workflow_name := strings.Join(src_array[0:len(src_array)-3], "/")
		step_name := src_array[len(src_array)-2]
		output_name := src_array[len(src_array)-1]

		var step_reason string
		obj, ok, step_reason, err = qm.getCWLSourceFromStepOutput(job, workflow_instance, workflow_name, step_name, output_name, error_on_missing_task)
		if err != nil {
			err = fmt.Errorf("(getCWLSource) (%s, %s, %s) getCWLSourceFromStepOutput returned: %s", workflow_name, step_name, output_name, err.Error())
			return
		}
		if !ok {
			reason = "getCWLSourceFromStepOutput returned: " + step_reason
		}
		return

	case "*cwl.File":
		thing := generic_object.(*cwl.File)

		obj = thing
		return
	case "*cwl.Array":
		thing := generic_object.(*cwl.Array)

		obj = thing
		return
	case "*cwl.String":
		thing := generic_object.(*cwl.String)

		obj = thing
		return
	case "*cwl.Int":
		thing := generic_object.(*cwl.Int)

		obj = thing
		return
	case "*cwl.Double":
		thing := generic_object.(*cwl.Double)

		obj = thing
		return
	case "*cwl.Record":
		thing := generic_object.(*cwl.Record)

		obj = thing
		return
	case "*cwl.Boolean":
		thing := generic_object.(*cwl.Boolean)

		obj = thing
		return
	case "*cwl.Null":
		thing := generic_object.(*cwl.Null)

		obj = thing
		return
	}

	err = fmt.Errorf("(getCWLSource) could not parse source: %s, type %s unknown", src, type_str)

	return

}

// Tasks or Subworkflows
func (qm *ServerMgr) GetDependencies(job *Job, workflow_instance *WorkflowInstance, workflow_input_map map[string]cwl.CWLType, workflow_step *cwl.WorkflowStep, context *cwl.WorkflowContext) (err error) {
	if workflow_step.In == nil {
		return
	}

	if len(workflow_step.In) == 0 {

		return
	}
	var ok bool
	src_str := ""

	for _, input := range workflow_step.In {

		id := input.Id
		fmt.Printf("(GetDependencies) id: %s\n", id)

		if input.Source != nil {

			source_is_array := false

			source_as_string := ""
			source_as_array, source_is_array := input.Source.([]interface{})

			if source_is_array {
				fmt.Printf("(GetDependencies) source is a array: %s", spew.Sdump(input.Source))
				if input.Source_index != 0 {
					// from scatter step
					// fmt.Printf("source is a array with Source_index: %s", spew.Sdump(input.Source))
					if input.Source_index > len(source_as_array) {
						err = fmt.Errorf("(GetStepInputObjects) input.Source_index >= len(source_as_array) %d > %d", input.Source_index, len(source_as_array))
						return
					}
					src := source_as_array[input.Source_index-1]
					var src_str string
					//var ok bool
					src_str, ok = src.(string)
					if !ok {
						err = fmt.Errorf("src is not a string")
						return
					}
					_ = src_str
					panic("A1")
				} else {
					cwl_array := cwl.Array{}
					for _, src := range source_as_array { // usually only one
						fmt.Println("src: " + spew.Sdump(src))
						var src_str string
						//var ok bool
						src_str, ok = src.(string)
						if !ok {
							err = fmt.Errorf("src is not a string")
							return
						}
						_ = src_str
					}
					_ = cwl_array
					panic("A2")
				}
				panic("A")

			} else { // NOT source_is_array
				source_as_string, ok = input.Source.(string)
				if !ok {
					err = fmt.Errorf("(GetStepInputObjects) (string) Cannot parse WorkflowStep source: %s", spew.Sdump(input.Source))
					return
				}
				panic("B")
			}
			_ = source_as_string
			_ = src_str
		} else { // input.Source == nil
			if input.Default == nil && input.ValueFrom == "" {
				err = fmt.Errorf("(GetStepInputObjects) sorry, source, Default and ValueFrom are missing") // TODO StepInputExpressionRequirement
				return
			}

			if input.Default != nil {
				panic("D")
				// var default_value cwl.CWLType
				// default_value, err = cwl.NewCWLType(cmd_id, input.Default, context)
				// if err != nil {
				// 	err = fmt.Errorf("(GetStepInputObjects) NewCWLTypeFromInterface(input.Default) returns: %s", err.Error())
				// 	return
				// }

				// if default_value == nil {
				// 	err = fmt.Errorf("(GetStepInputObjects) default_value == nil ")
				// 	return
				// }

				// workunit_input_map[cmd_id] = default_value
			}
			panic("C")
		} // end if input.Source

	} // end for

	return
}

func (qm *ServerMgr) GetStepInputObjects(job *Job, workflow_instance *WorkflowInstance, workflow_input_map map[string]cwl.CWLType, workflow_step *cwl.WorkflowStep, context *cwl.WorkflowContext, caller string) (workunit_input_map cwl.JobDocMap, ok bool, reason string, err error) {

	workunit_input_map = make(map[string]cwl.CWLType) // also used for json
	reason = "undefined"

	fmt.Println("(GetStepInputObjects) workflow_step:")
	spew.Dump(workflow_step)

	if workflow_step.In == nil {
		// empty inputs are ok
		ok = true
		//err = fmt.Errorf("(GetStepInputObjects) workflow_step.In == nil (%s)", workflow_step.Id)
		return
	}

	if len(workflow_step.In) == 0 {
		// empty inputs are ok
		ok = true
		//err = fmt.Errorf("(GetStepInputObjects) len(workflow_step.In) == 0")
		return
	}

	// 1. find all object source and Default
	// 2. make a map copy to be used in javascript, as "inputs"
	// INPUT_LOOP1
	for input_i, input := range workflow_step.In {
		// input is a WorkflowStepInput

		fmt.Printf("(GetStepInputObjects) workflow_step.In: (%d)\n", input_i)
		spew.Dump(workflow_step.In)

		id := input.Id
		//	fmt.Println("(GetStepInputObjects) id: %s", id)
		cmd_id := path.Base(id)

		// get data from Source, Default or valueFrom

		link_merge_method := ""
		if input.LinkMerge != nil {
			link_merge_method = string(*input.LinkMerge)
		} else {
			// default: merge_nested
			link_merge_method = "merge_nested"
		}

		if input.Source != nil {
			fmt.Println("(GetStepInputObjects) input.Source != nil")
			//source_object_array := []cwl.CWLType{}
			//resolve pointers in source

			source_is_array := false

			source_as_string := ""
			source_as_array, source_is_array := input.Source.([]interface{})

			if source_is_array {
				fmt.Printf("(GetStepInputObjects) source is a array: %s", spew.Sdump(input.Source))

				if input.Source_index != 0 {
					// from scatter step
					// fmt.Printf("source is a array with Source_index: %s", spew.Sdump(input.Source))
					if input.Source_index > len(source_as_array) {
						err = fmt.Errorf("(GetStepInputObjects) input.Source_index >= len(source_as_array) %d > %d", input.Source_index, len(source_as_array))
						return
					}
					src := source_as_array[input.Source_index-1]
					var src_str string
					//var ok bool
					src_str, ok = src.(string)
					if !ok {
						err = fmt.Errorf("src is not a string")
						return
					}
					var job_obj cwl.CWLType
					job_obj, ok, _, err = qm.getCWLSource(job, workflow_instance, workflow_input_map, src_str, true, job.WorkflowContext)
					if err != nil {
						err = fmt.Errorf("(GetStepInputObjects) (array) getCWLSource returns: %s", err.Error())
						return
					}
					if !ok {
						err = fmt.Errorf("(GetStepInputObjects) (array) getCWLSource did not find output \"%s\"", src_str)
						return // TODO allow optional ??
					}

					workunit_input_map[cmd_id] = job_obj
				} else {
					// case Source_index == 0

					cwl_array := cwl.Array{}
					for _, src := range source_as_array { // usually only one
						fmt.Println("src: " + spew.Sdump(src))
						var src_str string
						//var ok bool
						src_str, ok = src.(string)
						if !ok {
							err = fmt.Errorf("src is not a string")
							return
						}

						// if ...
						//embedded_workflowInstanceID := "_root/" + strings.Join(src_array[1:len(src_array)-2], "/")

						var job_obj cwl.CWLType
						job_obj, ok, _, err = qm.getCWLSource(job, workflow_instance, workflow_input_map, src_str, true, job.WorkflowContext)
						if err != nil {
							err = fmt.Errorf("(GetStepInputObjects) (array) getCWLSource returns: %s", err.Error())
							return
						}
						if !ok {
							err = fmt.Errorf("(GetStepInputObjects) (array) getCWLSource did not find output \"%s\"", src_str)
							return // TODO allow optional ??
						}

						if link_merge_method == "merge_flattened" {

							job_obj_type := job_obj.GetType()

							if job_obj_type != cwl.CWLArray {
								err = fmt.Errorf("(GetStepInputObjects) merge_flattened, expected array as input, but got %s", job_obj_type)
								return
							}

							var an_array *cwl.Array
							an_array, ok = job_obj.(*cwl.Array)
							if !ok {
								err = fmt.Errorf("got type: %s", reflect.TypeOf(job_obj))
								return
							}

							for i, _ := range *an_array {
								//source_object_array = append(source_object_array, (*an_array)[i])
								cwl_array = append(cwl_array, (*an_array)[i])
							}

						} else if link_merge_method == "merge_nested" {
							//source_object_array = append(source_object_array, job_obj)
							cwl_array = append(cwl_array, job_obj)
						} else {
							err = fmt.Errorf("(GetStepInputObjects) link_merge_method %s not supported", link_merge_method)
							return
						}
						//cwl_array = append(cwl_array, obj)
					}

					workunit_input_map[cmd_id] = &cwl_array

				}
			} else {
				fmt.Printf("(GetStepInputObjects) source is NOT a array: %s", spew.Sdump(input.Source))
				//var ok bool
				source_as_string, ok = input.Source.(string)
				if !ok {
					err = fmt.Errorf("(GetStepInputObjects) (string) Cannot parse WorkflowStep source: %s", spew.Sdump(input.Source))
					return
				}

				var job_obj cwl.CWLType
				//var reason string
				job_obj, ok, reason, err = qm.getCWLSource(job, workflow_instance, workflow_input_map, source_as_string, true, job.WorkflowContext)
				if err != nil {
					err = fmt.Errorf("(GetStepInputObjects) (source_as_string: %s ) getCWLSource returns: %s", source_as_string, err.Error())
					return
				}
				if ok {
					fmt.Printf("(GetStepInputObjects) qm.getCWLSource returned an object\n")
					spew.Dump(job_obj)
					if job_obj.GetType() == cwl.CWLNull {
						//fmt.Println("(GetStepInputObjects) job_obj is cwl.CWLNull")
						//reason = "returned object is null"
						//ok = false
						continue
					} else {
						//fmt.Println("(GetStepInputObjects) job_obj is not cwl.CWLNull")
					}
				}

				if !ok {

					logger.Debug(3, "(GetStepInputObjects) source_as_string %s not found", source_as_string)

					if "#main/step1/output" == source_as_string {
						err = fmt.Errorf("#main/step1/output not found , reason: " + reason + " caller: " + caller)
						return
						//panic("#main/step1/output not found , reason: " + reason + " caller: " + caller)
					}
					logger.Debug(3, "(GetStepInputObjects) qm.getCWLSource did not return an object (reason: %s), now check input.Default", reason)
					if input.Default == nil {
						logger.Debug(1, "(GetStepInputObjects) (string) getCWLSource did not find output (nor a default) that can be used as input \"%s\"", source_as_string)
						//ok = false
						//err = fmt.Errorf("(GetStepInputObjects) getCWLSource did not find source %s and has no Default (reason: %s)", source_as_string, reason)
						continue
					}
					logger.Debug(1, "(GetStepInputObjects) (string) getCWLSource found something \"%s\"", source_as_string)
					job_obj, err = cwl.NewCWLType("", input.Default, context)
					if err != nil {
						err = fmt.Errorf("(GetStepInputObjects) could not use default: %s", err.Error())
						return
					}
					fmt.Println("(GetStepInputObjects) got a input.Default")
					spew.Dump(job_obj)
				}

				//fmt.Printf("(GetStepInputObjects) Source_index: %d\n", input.Source_index)
				if input.Source_index != 0 {
					real_source_index := input.Source_index - 1

					var job_obj_array_ptr *cwl.Array
					job_obj_array_ptr, ok = job_obj.(*cwl.Array)
					if !ok {
						err = fmt.Errorf("(GetStepInputObjects) Array expected but got: %s", reflect.TypeOf(job_obj))
						return
					}
					var job_obj_array cwl.Array
					job_obj_array = *job_obj_array_ptr

					if real_source_index >= len(job_obj_array) {
						err = fmt.Errorf("(GetStepInputObjects) Source_index %d out of bounds, array length: %d", real_source_index, len(job_obj_array))
						return
					}

					var element cwl.CWLType
					element = job_obj_array[real_source_index]
					//fmt.Printf("(GetStepInputObjects) cmd_id=%s element=%s real_source_index=%d\n", cmd_id, element, real_source_index)
					workunit_input_map[cmd_id] = element
				} else {
					workunit_input_map[cmd_id] = job_obj
				}
			}

		} else { //input.Source == nil
			fmt.Println("(GetStepInputObjects) input.Source == nil")

			if input.Default == nil && input.ValueFrom == "" {
				err = fmt.Errorf("(GetStepInputObjects) sorry, source, Default and ValueFrom are missing") // TODO StepInputExpressionRequirement
				return
			}

			if input.Default != nil {
				var default_value cwl.CWLType
				default_value, err = cwl.NewCWLType(cmd_id, input.Default, context)
				if err != nil {
					err = fmt.Errorf("(GetStepInputObjects) NewCWLTypeFromInterface(input.Default) returns: %s", err.Error())
					return
				}

				if default_value == nil {
					err = fmt.Errorf("(GetStepInputObjects) default_value == nil ")
					return
				}

				workunit_input_map[cmd_id] = default_value
			}
		}
		// TODO

	} // end of INPUT_LOOP1
	fmt.Println("(GetStepInputObjects) workunit_input_map after first round:")
	spew.Dump(workunit_input_map)

	// 3. evaluate each ValueFrom field, update results
VALUE_FROM_LOOP:
	for _, input := range workflow_step.In {
		if input.ValueFrom == "" {
			continue VALUE_FROM_LOOP
		}

		id := input.Id
		cmd_id := path.Base(id)

		// from CWL doc: The self value of in the parameter reference or expression must be the value of the parameter(s) specified in the source field, or null if there is no source field.

		// #### Create VM ####
		vm := otto.New()

		// set "inputs"

		//func ToValue(value interface{}) (Value, error)

		//var inputs_value otto.Value
		//inputs_value, err = vm.ToValue(workunit_input_map)
		//if err != nil {
		//	return
		//}

		//fmt.Println("(GetStepInputObjects) workunit_input_map:")
		//spew.Dump(workunit_input_map)

		var inputs_json []byte
		inputs_json, err = json.Marshal(workunit_input_map)
		if err != nil {
			err = fmt.Errorf("(GetStepInputObjects) json.Marshal returns: %s", err.Error())
			return
		}
		logger.Debug(3, "SET inputs=%s\n", inputs_json)

		//err = vm.Set("inputs", workunit_input_map)
		//err = vm.Set("inputs_str", inputs_json)
		//if err != nil {
		//	err = fmt.Errorf("(GetStepInputObjects) vm.Set inputs returns: %s", err.Error())
		//	return
		//}

		var js_self cwl.CWLType
		js_self, ok = workunit_input_map[cmd_id]
		if !ok {
			//err = fmt.Errorf("(GetStepInputObjects) workunit_input %s not found", cmd_id)
			//return
			logger.Warning("(GetStepInputObjects) workunit_input %s not found", cmd_id)
			js_self = cwl.NewNull()
		}

		// TODO check for scatter
		// https://www.commonwl.org/v1.0/Workflow.html#WorkflowStepInput

		if js_self == nil {
			err = fmt.Errorf("(GetStepInputObjects) js_self == nil")
			return
		}

		var self_json []byte
		self_json, err = json.Marshal(js_self)
		if err != nil {
			err = fmt.Errorf("(GetStepInputObjects) json.Marshal returned: %s", err.Error())
			return
		}

		logger.Debug(3, "SET self=%s\n", self_json)

		//err = vm.Set("self", js_self)
		//err = vm.Set("self_str", self_json)
		//if err != nil {
		//	err = fmt.Errorf("(GetStepInputObjects) vm.Set self returns: %s", err.Error())
		//	return
		//}

		//fmt.Printf("input.ValueFrom=%s\n", input.ValueFrom)

		// evaluate $(...) ECMAScript expression
		reg := regexp.MustCompile(`\$\(.+\)`)
		// CWL documentation: http://www.commonwl.org/v1.0/Workflow.html#Expressions

		parsed_str := input.ValueFrom.String()
		//for {

		matches := reg.FindAll([]byte(parsed_str), -1)
		fmt.Printf("()Matches: %d\n", len(matches))
		if len(matches) > 0 {

			concatenate := false
			if len(matches) > 1 {
				concatenate = true
			}

			for _, match := range matches {
				expression_string := bytes.TrimPrefix(match, []byte("$("))
				expression_string = bytes.TrimSuffix(expression_string, []byte(")"))

				javascript_function := fmt.Sprintf("(function(){\n self=%s ; inputs=%s; return %s;\n})()", self_json, inputs_json, expression_string)
				fmt.Printf("%s\n", javascript_function)

				value, xerr := vm.Run(javascript_function)
				if xerr != nil {
					err = fmt.Errorf("(GetStepInputObjects) Javascript complained: A) %s", xerr.Error())
					return
				}
				fmt.Println(reflect.TypeOf(value))

				//if value.IsNumber()
				if concatenate {
					value_str, xerr := value.ToString()
					if xerr != nil {
						err = fmt.Errorf("(GetStepInputObjects) Cannot convert value to string: %s", xerr.Error())
						return
					}
					parsed_str = strings.Replace(parsed_str, string(match), value_str, 1)
				} else {

					var value_returned cwl.CWLType
					var exported_value interface{}
					//https://godoc.org/github.com/robertkrimen/otto#Value.Export
					exported_value, err = value.Export()

					if err != nil {
						err = fmt.Errorf("(GetStepInputObjects)  value.Export() returned: %s", err.Error())
						return
					}
					switch exported_value.(type) {

					case string:
						value_returned = cwl.NewString(exported_value.(string))

					case bool:

						value_returned = cwl.NewBooleanFrombool(exported_value.(bool))

					case int:
						value_returned, err = cwl.NewInt(exported_value.(int), context)
						if err != nil {
							err = fmt.Errorf("(NewCWLType) NewInt: %s", err.Error())
							return
						}
					case float32:
						value_returned = cwl.NewFloat(exported_value.(float32))
					case float64:
						fmt.Println("got a double")
						value_returned = cwl.NewDouble(exported_value.(float64))
					case uint64:
						value_returned, err = cwl.NewInt(exported_value.(int), context)
						if err != nil {
							err = fmt.Errorf("(NewCWLType) NewInt: %s", err.Error())
							return
						}

					case []interface{}: //Array
						err = fmt.Errorf("(GetStepInputObjects) array not supported yet")
						return
					case interface{}: //Object

						value_returned, err = cwl.NewCWLType("", exported_value, context)
						if err != nil {
							//fmt.Println("record:")
							//spew.Dump(exported_value)
							err = fmt.Errorf("(GetStepInputObjects) interface{}, NewCWLType returned: %s", err.Error())
							return
						}

					case nil:
						value_returned = cwl.NewNull()
					default:
						err = fmt.Errorf("(GetStepInputObjects) js return type not supoported: (%s)", reflect.TypeOf(exported_value))
						return
					}

					//fmt.Println("value_returned:")
					//spew.Dump(value_returned)
					workunit_input_map[cmd_id] = value_returned
					continue VALUE_FROM_LOOP
				}
			} // for matches

			//if concatenate
			workunit_input_map[cmd_id] = cwl.NewString(parsed_str)

			continue VALUE_FROM_LOOP
		} // if matches
		//}

		//fmt.Printf("parsed_str: %s\n", parsed_str)

		// evaluate ${...} ECMAScript function body
		reg = regexp.MustCompile(`(?s)\${.+}`) // s-flag is needed to include newlines

		// CWL documentation: http://www.commonwl.org/v1.0/Workflow.html#Expressions

		matches = reg.FindAll([]byte(parsed_str), -1)
		//fmt.Printf("{}Matches: %d\n", len(matches))
		if len(matches) == 0 {
			workunit_input_map[cmd_id] = cwl.NewString(parsed_str)
			continue VALUE_FROM_LOOP
		}

		if len(matches) == 1 {
			match := matches[0]
			expression_string := bytes.TrimPrefix(match, []byte("${"))
			expression_string = bytes.TrimSuffix(expression_string, []byte("}"))

			javascript_function := fmt.Sprintf("(function(){\n self=%s ; inputs=%s; %s \n})()", self_json, inputs_json, expression_string)
			fmt.Printf("%s\n", javascript_function)

			value, xerr := vm.Run(javascript_function)
			if xerr != nil {
				err = fmt.Errorf("Javascript complained: B) %s", xerr.Error())
				return
			}

			value_exported, _ := value.Export()

			fmt.Printf("reflect.TypeOf(value_exported): %s\n", reflect.TypeOf(value_exported))

			var value_cwl cwl.CWLType
			value_cwl, err = cwl.NewCWLType("", value_exported, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkunit) Error parsing javascript VM result value, cwl.NewCWLType returns: %s", err.Error())
				return
			}

			workunit_input_map[cmd_id] = value_cwl
			continue VALUE_FROM_LOOP
		}

		err = fmt.Errorf("(NewWorkunit) ValueFrom contains more than one ECMAScript function body")
		return

	} // end of VALUE_FROM_LOOP

	//fmt.Println("(GetStepInputObjects) workunit_input_map after ValueFrom round:")
	//spew.Dump(workunit_input_map)

	for key, value := range workunit_input_map {
		fmt.Printf("workunit_input_map: %s -> %s (%s)\n", key, value.String(), reflect.TypeOf(value))

	}
	ok = true
	return
}

func (qm *ServerMgr) CreateAndEnqueueWorkunits(task *Task, job *Job) (err error) {
	//logger.Debug(3, "(CreateAndEnqueueWorkunits) starting")
	//fmt.Println("--CreateAndEnqueueWorkunits--")
	//spew.Dump(task)
	workunits, err := task.CreateWorkunits(qm, job)
	if err != nil {
		err = fmt.Errorf("(CreateAndEnqueueWorkunits) error in CreateWorkunits: %s", err.Error())
		return err
	}
	for _, wu := range workunits {
		if err := qm.workQueue.Add(wu); err != nil {
			err = fmt.Errorf("(CreateAndEnqueueWorkunits) error in qm.workQueue.Add: %s", err.Error())
			return err
		}
		id := wu.GetID()
		err = qm.CreateWorkPerf(id)
		if err != nil {
			err = fmt.Errorf("(CreateAndEnqueueWorkunits) error in CreateWorkPerf: %s", err.Error())
			return
		}
	}
	return
}

func (qm *ServerMgr) createOutputNode(task *Task) (err error) {
	err = task.LockNamed("createOutputNode")
	if err != nil {
		return
	}
	defer task.Unlock()

	var modified bool
	for _, io := range task.Outputs {
		if io.Type == "update" {
			// this an update output, it will update an existing shock node and not create a new one (it will update metadata of the shock node)
			if (io.Node == "") || (io.Node == "-") {
				if io.Origin == "" {
					// it may be in inputs
					for _, input := range task.Inputs {
						if io.FileName == input.FileName {
							io.Node = input.Node
							io.Size = input.Size
							io.Url = input.Url
							modified = true
						}
					}
					if (io.Node == "") || (io.Node == "-") {
						// still missing
						err = fmt.Errorf("update output %s in task %s is missing required origin", io.FileName, task.Id)
						return
					}
				} else {
					// find predecessor task
					var preId Task_Unique_Identifier
					preId, err = New_Task_Unique_Identifier(task.JobId, io.Origin)
					if err != nil {
						err = fmt.Errorf("New_Task_Unique_Identifier returned: %s", err.Error())
						return
					}
					var preTaskStr string
					preTaskStr, err = preId.String()
					if err != nil {
						err = fmt.Errorf("task.String returned: %s", err.Error())
						return
					}
					preTask, ok, xerr := qm.TaskMap.Get(preId, true)
					if xerr != nil {
						err = fmt.Errorf("predecessor task %s not found for task %s: %s", preTaskStr, task.Id, xerr.Error())
						return
					}
					if !ok {
						err = fmt.Errorf("predecessor task %s not found for task %s", preTaskStr, task.Id)
						return
					}

					// find predecessor output
					preTaskIO, xerr := preTask.GetOutput(io.FileName)
					if xerr != nil {
						err = fmt.Errorf("unable to get IO for predecessor task %s, file %s: %s", preTask.Id, io.FileName, err.Error())
						return
					}

					// copy if not already done
					if io.Node != preTaskIO.Node {
						io.Node = preTaskIO.Node
						modified = true
					}
					if io.Size != preTaskIO.Size {
						io.Size = preTaskIO.Size
						modified = true
					}
					if io.Url != preTaskIO.Url {
						io.Url = preTaskIO.Url
						modified = true
					}
				}
				logger.Debug(2, "(createOutputNode) outout %s in task %s is an update of node %s", io.FileName, task.Id, io.Node)
			}
		} else {
			// POST empty shock node for this output
			logger.Debug(2, "(createOutputNode) posting output Shock node for file %s in task %s", io.FileName, task.Id)

			sc := shock.ShockClient{Host: io.Host, Token: task.Info.DataToken}
			var nodeid string
			nodeid, err = sc.CreateNode(io.FileName, task.TotalWork)
			if err != nil {
				return
			}
			io.Node = nodeid
			_, err = io.DataUrl()
			if err != nil {
				return
			}
			modified = true
			logger.Debug(2, "(createOutputNode) task %s: output Shock node created, node=%s", task.Id, nodeid)
		}
	}

	if modified {
		err = dbUpdateJobTaskIO(task.JobId, task.WorkflowInstanceId, task.Id, "outputs", task.Outputs)
		if err != nil {
			err = fmt.Errorf("unable to save task outputs to mongodb, task=%s: %s", task.Id, err.Error())
		}
	}
	return
}

//---end of task methods---

// taskCompletedScatter is called for every scatter child, but only the last child performs collection of results
func (qm *ServerMgr) taskCompletedScatter(job *Job, wi *WorkflowInstance, task *Task) (err error) {

	var taskStr string
	taskStr, err = task.String()
	if err != nil {
		err = fmt.Errorf("(taskCompleted_Scatter) task.String returned: %s", err.Error())
		return
	}

	logger.Debug(3, "(taskCompleted_Scatter) %s Scatter_parent exists", taskStr)
	scatterParentID := *task.Scatter_parent
	var scatterParentTask *Task
	var ok bool
	scatterParentTask, ok, err = qm.TaskMap.Get(scatterParentID, true)
	if err != nil {
		err = fmt.Errorf("(taskCompleted_Scatter) qm.TaskMap.Get returned: %s", err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(taskCompleted_Scatter) Scatter_Parent task %s not found", scatterParentID)
		return
	}

	// (taskCompleted_Scatter) get scatter sibblings to see if they are done
	var children []*Task
	children, err = scatterParentTask.GetScatterChildren(wi, qm)
	if err != nil {

		length, _ := qm.TaskMap.Len()

		var tasks []*Task
		tasks, _ = qm.TaskMap.GetTasks()
		for _, task := range tasks {
			fmt.Printf("(taskCompleted_Scatter) got task %s\n", task.Id)
		}

		err = fmt.Errorf("(taskCompleted_Scatter) (scatter) GetScatterChildren returned: %s (total: %d)", err.Error(), length)
		return
	}

	//fmt.Printf("XXX children: %d\n", len(children))

	scatterComplete := true
	for _, childTask := range children {
		var childState string
		childState, err = childTask.GetState()
		if err != nil {
			err = fmt.Errorf("(taskCompleted_Scatter) child_task.GetState returned: %s", err.Error())
			return
		}

		if childState != TASK_STAT_COMPLETED {
			scatterComplete = false
			break
		}
	}

	if !scatterComplete {
		// nothing to do here, scatter is not complete
		return
	}

	ok, err = scatterParentTask.Finalize() // make sure this is the last scatter task
	if err != nil {
		err = fmt.Errorf("(taskCompleted_Scatter) scatter_parent_task.Finalize returned: %s", err.Error())
		return
	}

	if !ok {
		logger.Debug(3, "(taskCompleted_Scatter) somebody else is finalizing")
		// somebody else is finalizing
		return
	}

	// ***************************************
	//           scatter_complete
	// ***************************************

	logger.Debug(3, "(taskCompleted_Scatter) scatter_complete")

	scatterParentStep := scatterParentTask.WorkflowStep

	scatterParentTask.StepOutput = &cwl.Job_document{}

	context := job.WorkflowContext

	//fmt.Printf("XXX start\n")
	for i := range scatterParentStep.Out {
		//fmt.Printf("XXX loop %d\n", i)
		workflowStepOutput := scatterParentStep.Out[i]
		workflowStepOutputID := workflowStepOutput.Id

		workflowStepOutputIDBase := path.Base(workflowStepOutputID)

		outputArray := cwl.Array{}

		for _, childTask := range children {
			//fmt.Printf("XXX inner loop %d\n", i)
			jobDoc := childTask.StepOutput
			var childOutput cwl.CWLType
			childOutput, ok = jobDoc.Get(workflowStepOutputIDBase)
			if !ok {
				//fmt.Printf("XXX job_doc.Get failed\n")
				err = fmt.Errorf("(taskCompleted_Scatter) job_doc.Get failed: %s ", err.Error())
				return
			}
			//fmt.Println("child_output:")
			//spew.Dump(child_output)
			outputArray = append(outputArray, childOutput)
			//fmt.Println("output_array:")
			//spew.Dump(output_array)
		}
		err = context.Add(workflowStepOutputID, &outputArray, "taskCompleted_Scatter")
		if err != nil {
			err = fmt.Errorf("(taskCompleted_Scatter) context.Add returned: %s", err.Error())
			return
		}
		//fmt.Println("final output_array:")
		//spew.Dump(output_array)
		scatterParentTask.StepOutput = scatterParentTask.StepOutput.Add(workflowStepOutputID, &outputArray)

	}

	task = scatterParentTask

	///wi_local_id := task.WorkflowInstanceId
	//var wi *WorkflowInstance
	//if wi_local_id != "" {
	//	wi, ok, err = task.GetWorkflowInstance()

	//}
	// err = task.SetState(wi, TASK_STAT_COMPLETED, true)
	// if err != nil {
	// 	err = fmt.Errorf("(taskCompleted_Scatter) task.SetState returned: %s", err.Error())
	// 	return
	// }

	//log event about task done (TD)
	err = qm.FinalizeTaskPerf(task)
	if err != nil {
		err = fmt.Errorf("(taskCompleted_Scatter) FinalizeTaskPerf returned: %s", err.Error())
		return
	}
	logger.Event(event.TASK_DONE, "task_id="+taskStr)

	//update the info of the job which the task is belong to, could result in deletion of the
	//task in the task map when the task is the final task of the job to be done.
	err = qm.taskCompleted(wi, task) //task state QUEUED -> COMPLETED
	if err != nil {
		err = fmt.Errorf("(taskCompleted_Scatter) updateJobTask returned: %s", err.Error())
		return
	}

	return
}

// completeSubworkflow checks if all steps have completed (should not be required if counters are used)
// invoked by qm.WISetState or
func (qm *ServerMgr) completeSubworkflow(job *Job, workflowInstance *WorkflowInstance) (ok bool, reason string, err error) {

	var wfl *cwl.Workflow

	// if this is a subworkflow, check if sibblings are complete
	//var context *cwl.WorkflowContext

	var wiState string
	wiState, err = workflowInstance.GetState(true)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_instance.GetState returned: %s", err.Error())
		return
	}

	if wiState == WIStateCompleted {
		ok = true
		return
	}

	context := job.WorkflowContext

	workflowInstanceID, _ := workflowInstance.GetID(true)
	logger.Debug(3, "(completeSubworkflow) start: %s", workflowInstanceID)
	// check tasks
	for _, task := range workflowInstance.Tasks {

		var taskState string
		taskState, _ = task.GetState()

		if taskState != TASK_STAT_COMPLETED {
			ok = false
			reason = "(completeSubworkflow) a task is not completed yet"
			return
		}

	}

	// check subworkflows
	for _, subworkflow := range workflowInstance.Subworkflows {

		var subWI *WorkflowInstance
		subWI, ok, err = job.GetWorkflowInstance(subworkflow, true)
		if err != nil {
			err = fmt.Errorf("(completeSubworkflow) job.GetWorkflowInstance returned: %s", err.Error())
			return
		}
		if !ok {
			reason = fmt.Sprintf("(completeSubworkflow) subworkflow %s not found", subworkflow)
			return
		}

		subWIState, _ := subWI.GetState(true)

		if subWIState != WIStateCompleted {
			ok = false
			reason = fmt.Sprintf("(completeSubworkflow) subworkflow %s is not completed", subworkflow)
			return
		}

	}

	// *************
	// subworkflow complete, now collect outputs !
	// *************

	wfl, err = workflowInstance.GetWorkflow(context)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_instance.GetWorkflow returned: %s", err.Error())
		return
	}

	// reminder: at this point all tasks in subworkflow are complete, see above

	workflowInputs := workflowInstance.Inputs

	workflowInputsMap := workflowInputs.GetMap()

	workflowOutputsMap := make(cwl.JobDocMap)

	// collect sub-workflow outputs, put results in workflow_outputs_map

	logger.Debug(3, "(completeSubworkflow) %s len(wfl.Outputs): %d", workflowInstanceID, len(wfl.Outputs))

	for _, output := range wfl.Outputs { // WorkflowOutputParameter http://www.commonwl.org/v1.0/Workflow.html#WorkflowOutputParameter

		outputID := output.Id

		if output.OutputBinding != nil {
			// see http://www.commonwl.org/v1.0/Workflow.html#CommandOutputBinding
			//spew.Dump(output.OutputBinding)
			// import path
			// use https://golang.org/pkg/path/#Match
			// iterate over output files
			//for _, value := range workflow_inputs_map {
			//fmt.Println("key: " + key)

			//	_, ok := value.(*cwl.File)
			//	if !ok {
			//		continue
			//	}

			//fmt.Println("base: " + file.Basename)

			//}

			//panic("ok")
			err = fmt.Errorf("(completeSubworkflow) Workflow output outputbinding not supported yet")
			return
		}

		var expectedTypesRaw []interface{}

		switch output.Type.(type) {
		case []interface{}:
			expectedTypesRaw = output.Type.([]interface{})
		case []cwl.CWLType_Type:

			expectedTypesRawArray := output.Type.([]cwl.CWLType_Type)
			for i, _ := range expectedTypesRawArray {
				expectedTypesRaw = append(expectedTypesRaw, expectedTypesRawArray[i])

			}

		default:
			expectedTypesRaw = append(expectedTypesRaw, output.Type)
			//expected_types_raw = []interface{output.Type}
		}
		expectedTypes := []cwl.CWLType_Type{}

		isOptional := false

		var schemata []cwl.CWLType_Type
		schemata, err = job.WorkflowContext.GetSchemata()
		if err != nil {
			err = fmt.Errorf("(completeSubworkflow) job.CWL_collection.GetSchemata returned: %s", err.Error())
			return
		}

		for _, rawType := range expectedTypesRaw {
			var typeCorrect cwl.CWLType_Type
			typeCorrect, err = cwl.NewCWLType_Type(schemata, rawType, "WorkflowOutput", context)
			if err != nil {
				//spew.Dump(expected_types_raw)
				//fmt.Println("---")
				//spew.Dump(raw_type)

				err = fmt.Errorf("(completeSubworkflow) could not convert element of output.Type into cwl.CWLType_Type: %s", err.Error())
				//fmt.Printf(err.Error())
				//panic("raw_type problem")
				return
			}
			expectedTypes = append(expectedTypes, typeCorrect)
			if typeCorrect == cwl.CWLNull {
				isOptional = true
			}
		}

		// search the outputs and stick them in workflow_outputs_map

		outputSource := output.OutputSource

		switch outputSource.(type) {
		case string:
			outputSourceString := outputSource.(string)
			// example: "#preprocess-fastq.workflow.cwl/rejected2fasta/file"

			var obj cwl.CWLType
			//var ok bool
			//var reason string
			obj, ok, reason, err = qm.getCWLSource(job, workflowInstance, workflowInputsMap, outputSourceString, true, job.WorkflowContext)
			if err != nil {
				err = fmt.Errorf("(completeSubworkflow) A) getCWLSource returns: %s", err.Error())
				return
			}
			skip := false
			if !ok {
				if isOptional {
					skip = true
				} else {
					err = fmt.Errorf("(completeSubworkflow) A) source %s not found by getCWLSource (getCWLSource reason: %s)", outputSourceString, reason)
					return
				}
			}

			if !skip {
				hasType, xerr := cwl.TypeIsCorrect(expectedTypes, obj, context)
				if xerr != nil {
					err = fmt.Errorf("(completeSubworkflow) TypeIsCorrect: %s", xerr.Error())
					return
				}
				if !hasType {
					err = fmt.Errorf("(completeSubworkflow) A) workflow_ouput %s (type: %s), does not match expected types %s", outputID, reflect.TypeOf(obj), expectedTypes)
					return
				}

				workflowOutputsMap[outputID] = obj
			}
		case []string:
			outputSourceArrayOfString := outputSource.([]string)

			if len(outputSourceArrayOfString) == 0 {
				if !isOptional {
					err = fmt.Errorf("(completeSubworkflow) output_source array (%s) is empty, but a required output", outputID)
					return
				}
			}

			output_array := cwl.Array{}

			for _, outputSourceString := range outputSourceArrayOfString {
				var obj cwl.CWLType
				//var ok bool
				obj, ok, _, err = qm.getCWLSource(job, workflowInstance, workflowInputsMap, outputSourceString, true, job.WorkflowContext)
				if err != nil {
					err = fmt.Errorf("(completeSubworkflow) B) (%s) getCWLSource returns: %s", workflowInstanceID, err.Error())
					return
				}

				skip := false
				if !ok {

					if isOptional {
						skip = true
					} else {

						err = fmt.Errorf("(completeSubworkflow) B) (%s) source %s not found", workflowInstanceID, outputSourceString)
						return
					}
				}

				if !skip {
					has_type, xerr := cwl.TypeIsCorrect(expectedTypes, obj, context)
					if xerr != nil {
						err = fmt.Errorf("(completeSubworkflow) TypeIsCorrect: %s", xerr.Error())
						return
					}
					if !has_type {
						err = fmt.Errorf("(completeSubworkflow) B) workflow_ouput %s, does not match expected types %s", outputID, expectedTypes)
						return
					}
					//fmt.Println("obj:")
					//spew.Dump(obj)
					output_array = append(output_array, obj)
				}
			}

			if len(output_array) > 0 {
				workflowOutputsMap[outputID] = &output_array
			} else {
				if !isOptional {
					err = fmt.Errorf("(completeSubworkflow) array with output_id %s is empty, but a required output", outputID)
					return
				}
			}
			//fmt.Println("workflow_outputs_map:")
			//spew.Dump(workflow_outputs_map)

		default:
			err = fmt.Errorf("(completeSubworkflow) output.OutputSource has to be string or []string, but I got type %s", spew.Sdump(outputSource))
			return

		}

	}

	var workflowOutputsArray cwl.Job_document
	workflowOutputsArray, err = workflowOutputsMap.GetArray()
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_outputs_map.GetArray returned: %s", err.Error())
		return
	}

	logger.Debug(3, "(completeSubworkflow) %s len(workflowOutputsArray): %d", workflowInstanceID, len(workflowOutputsArray))

	err = workflowInstance.SetOutputs(workflowOutputsArray, context, true)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_instance.SetOutputs returned: %s", err.Error())
		return
	}

	// stick outputs in context, using correct Step-name (depends on if it is a embedded workflow)
	logger.Debug(3, "(completeSubworkflow) workflow_instance.Outputs: %d", len(workflowInstance.Outputs))
	for output := range workflowInstance.Outputs {
		logger.Debug(3, "(completeSubworkflow) iteration %d", output)
		outputNamed := &workflowInstance.Outputs[output]
		outputNamedBase := path.Base(outputNamed.Id)

		prefix := path.Dir(outputNamed.Id)

		logger.Debug(3, "(completeSubworkflow) old name: %s", outputNamed.Id)

		prefixBase := path.Base(prefix)
		newName := outputNamed.Id
		if len(prefixBase) == 36 { // TODO: find better way of detecting embedded workflow
			// case: emebedded workflow
			prefix = path.Dir(prefix)
			newName = prefix + "/" + outputNamedBase
		}

		//fmt.Printf("new name: %s\n", new_name)
		logger.Debug(3, "(completeSubworkflow) new name: %s", newName)
		err = context.Add(newName, outputNamed.Value, "completeSubworkflow")
		if err != nil {
			err = fmt.Errorf("(completeSubworkflow) context.Add returned: %s", err.Error())
			return
		}

	}

	if workflowInstance.RemainSteps > 0 {
		err = fmt.Errorf("(completeSubworkflow) RemainSteps > 0 cannot complete")
		panic(err.Error())
		return
	}

	err = workflowInstance.SetState(WIStateCompleted, DbSyncTrue, true)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_instance.SetState returned: %s", err.Error())
		return
	}

	// logger.Debug(3, "(completeSubworkflow) job.WorkflowInstancesRemain: (before) %d", job.WorkflowInstancesRemain)
	// err = job.IncrementWorkflowInstancesRemain(-1, DbSyncTrue, true)
	// if err != nil {
	// 	err = fmt.Errorf("(completeSubworkflow) job.IncrementWorkflowInstancesRemain returned: %s", err.Error())
	// 	return
	// }

	// ***** notify parent workflow or job (this workflow might have been the last step)

	workflowInstanceLocalID := workflowInstance.LocalID
	logger.Debug(3, "(completeSubworkflow) completes with workflowInstanceLocalID: %s", workflowInstanceLocalID)

	if workflowInstanceLocalID == "#main" {
		// last workflow_instance -> notify job

		// this was the main workflow, all done!

		err = qm.finalizeJob(job)
		if err != nil {
			err = fmt.Errorf("(completeSubworkflow) qm.finalizeJob returned: %s", err.Error())
			return
		}

		return
	}

	// check if parent needs to be notified
	// update Remain varable and complete if necessary
	logger.Debug(3, "(completeSubworkflow) check parent")
	var parent *WorkflowInstance
	parent, err = workflowInstance.GetParent(true)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_instance.GetParent returned: %s", err.Error())
		return
	}

	// notify parent
	var parentRemain int
	parentRemain, err = parent.IncrementRemainSteps(-1, true)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) parent.IncrementRemainSteps returned: %s", err.Error())
		return
	}

	if parentRemain > 0 {
		// no need to notify
		logger.Debug(3, "(completeSubworkflow) no need to complete parent workflow")
		return
	}

	logger.Debug(3, "(completeSubworkflow) complete parent workflow")

	// complete parent workflow
	var parentResult string
	ok, parentResult, err = qm.completeSubworkflow(job, parent) // recursive call
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) recursive call of qm.completeSubworkflow returned: %s", err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(completeSubworkflow) qm.completeSubworkflow could not complete, reason: %s", parentResult)
		return
	}

	logger.Debug(3, "(completeSubworkflow) done")

	return
}

// invoked when tasks completes
// update job info
// update parent task of a scatter task
func (qm *ServerMgr) taskCompleted(wi *WorkflowInstance, task *Task) (err error) {

	// TODO I am not sure why this function is invoked for every state change....

	var taskState string
	taskState, err = task.GetState()
	if err != nil {
		err = fmt.Errorf("(taskCompleted) task.GetState() returned: %s", err.Error())
		return
	}

	if taskState == TASK_STAT_COMPLETED {
		err = fmt.Errorf("(taskCompleted) task state is alreay TASK_STAT_COMPLETED")
		return
	}

	var job *Job
	job, err = task.GetJob()
	if err != nil {
		err = fmt.Errorf("(taskCompleted) task.GetJob returned: %s", err.Error())
		return
	}

	var taskStr string
	taskStr, err = task.String()
	if err != nil {
		err = fmt.Errorf("(taskCompleted) task.String returned: %s", err.Error())
		return
	}

	// check if this was the last task in a subworkflow

	// check if this task has a parent

	if task.WorkflowStep == nil {
		logger.Debug(3, "(taskCompleted) task.WorkflowStep == nil ")
	} else {
		logger.Debug(3, "(taskCompleted) task.WorkflowStep != nil ")
	}

	logger.Debug(3, "(taskCompleted) task.WorkflowStep != nil (%s)", taskStr)

	if task.Scatter_parent != nil {

		// var wi *WorkflowInstance
		// var ok bool
		// wi, ok, err = task.GetWorkflowInstance()
		// if err != nil {
		// 	err = fmt.Errorf("(taskCompleted) task.GetWorkflowInstance returned: %s", err.Error())
		// 	return
		// }

		// if !ok {
		// 	err = fmt.Errorf("(taskCompleted) task has no WorkflowInstance ?! ")
		// 	return
		// }

		// process scatter children
		err = qm.taskCompletedScatter(job, wi, task)
		if err != nil {
			err = fmt.Errorf("(taskCompleted) taskCompleted_Scatter returned: %s", err.Error())
			return
		}

	} else { // end task.Scatter_parent != nil
		logger.Debug(3, "(taskCompleted) %s  No Scatter_parent", taskStr)
	}

	err = task.SetState(wi, TASK_STAT_COMPLETED, true)
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) task.SetState returned: %s", err.Error())
		return
	}

	// ******************
	// check if workflowInstance needs to be completed

	// workflowInstanceID := task.WorkflowInstanceId

	// var workflowInstance *WorkflowInstance
	// var ok bool
	// workflowInstance, ok, err = job.GetWorkflowInstance(workflowInstanceID, true)
	// if err != nil {
	// 	err = fmt.Errorf("(taskCompleted) job.GetWorkflowInstance returned: %s", err.Error())
	// 	return
	// }

	// if !ok {
	// 	err = fmt.Errorf("(taskCompleted) WorkflowInstance %s not found", err.Error())
	// 	return
	// }

	if wi != nil {

		var subworkflowRemainSteps int
		subworkflowRemainSteps, err = wi.GetRemainSteps(true)
		if err != nil {
			err = fmt.Errorf("(taskCompleted) workflow_instance.GetRemainSteps returned: %s", err.Error())
			return
		}

		//subworkflow_remain_tasks, err = workflow_instance.DecreaseRemainSteps()
		//if err != nil {
		//	err = fmt.Errorf("(taskCompleted) workflow_instance.DecreaseRemainSteps returned: %s", err.Error())
		//	return

		//}

		logger.Debug(3, "(taskCompleted) TASK_STAT_COMPLETED  / remaining steps for subworkflow %s: %d", taskStr, subworkflowRemainSteps)

		//logger.Debug(3, "(taskCompleted) workflow_instance %s remaining tasks: %d (total %d or %d)", workflowInstanceID, subworkflowRemainSteps, workflowInstance.TaskCount(), workflowInstance.TotalTasks)

		if subworkflowRemainSteps > 0 {
			return
		}

		// ******************
		// This is the last task/subworkflow.

		//TODO find a way to lock this

		// subworkflow completed.
		var reason string
		var ok bool
		ok, reason, err = qm.completeSubworkflow(job, wi) // taskCompleted
		if err != nil {
			err = fmt.Errorf("(taskCompleted) completeSubworkflow returned: %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(taskCompleted) completeSubworkflow not ok, reason: %s", reason)
			return
		}

		return
	}

	// old-style AWE job only

	err = job.IncrementRemainTasks(-1)
	if err != nil {
		err = fmt.Errorf("(taskCompleted) IncrementRemainTasks returned: %s", err.Error())
		return
	}

	jobRemainTasks := job.RemainTasks

	if jobRemainTasks > 0 { //#####################################################################
		return
	}

	err = qm.finalizeJob(job)
	if err != nil {
		err = fmt.Errorf("(taskCompleted) qm.finalizeJob returned: %s", err.Error())
		return
	}

	// this is a recursive call !
	// if parent_task_to_complete != nil {

	// 	if parent_task_to_complete.State != TASK_STAT_COMPLETED {
	// 		err = fmt.Errorf("(taskCompleted) qm.taskCompleted(parent_task_to_complete) parent_task_to_complete.State != TASK_STAT_COMPLETED")
	// 		return
	// 	}
	// 	err = qm.taskCompleted(parent_task_to_complete)
	// 	if err != nil {
	// 		err = fmt.Errorf("(taskCompleted) qm.taskCompleted(parent_task_to_complete) returned: %s", err.Error())
	// 		return
	// 	}
	// }

	return
}

func (qm *ServerMgr) finalizeJob(job *Job) (err error) {

	jobState, err := job.GetState(true)
	if err != nil {
		err = fmt.Errorf("(updateJobTask) job.GetState returned: %s", err.Error())
		return
	}
	if jobState == JOB_STAT_COMPLETED {
		err = fmt.Errorf("(updateJobTask) job state is already JOB_STAT_COMPLETED")
		return
	}

	err = job.SetState(JOB_STAT_COMPLETED, nil)
	if err != nil {
		err = fmt.Errorf("(updateJobTask) job.SetState returned: %s", err.Error())
		return
	}

	var jobid string
	jobid, err = job.GetId(true)
	if err != nil {
		err = fmt.Errorf("(updateJobTask) job.GetId returned: %s", err.Error())
		return
	}
	qm.FinalizeJobPerf(jobid)
	qm.LogJobPerf(jobid)
	qm.removeActJob(jobid)
	//delete tasks in task map
	//delete from shock output flagged for deletion

	modified := 0
	for i, task := range job.TaskList() {
		// delete nodes that have been flagged to be deleted
		modified += task.DeleteOutput()
		modified += task.DeleteInput()
		//combined_id := jobid + "_" + task.Id

		id, _ := task.GetId("updateJobTask." + strconv.Itoa(i))

		_, _, err = qm.TaskMap.Delete(id)
		if err != nil {
			err = fmt.Errorf("(updateJobTask) qm.TaskMap.Delete returned: %s", err.Error())
			return
		}
	}

	if modified > 0 {
		// save only is something has changed
		job.Save() // TODO avoid this, try partial updates
	}

	//set expiration from conf if not set
	nullTime := time.Time{}
	var job_expiration time.Time
	job_expiration, err = dbGetJobFieldTime(jobid, "expiration")
	if err != nil {
		err = fmt.Errorf("(updateJobTask) dbGetJobFieldTime returned: %s", err.Error())
		return
	}

	if job_expiration == nullTime {
		expire := conf.GLOBAL_EXPIRE

		var job_info_pipeline string
		job_info_pipeline, err = dbGetJobFieldString(jobid, "info.pipeline")
		if err != nil {
			err = fmt.Errorf("(updateJobTask) dbGetJobFieldTime returned: %s", err.Error())
			return
		}

		if val, ok := conf.PIPELINE_EXPIRE_MAP[job_info_pipeline]; ok {
			expire = val
		}
		if expire != "" {
			err = job.SetExpiration(expire)
			if err != nil {
				err = fmt.Errorf("(updateJobTask) SetExpiration returned: %s", err.Error())
				return
			}
		}
	}
	//log event about job done (JD)
	logger.Event(event.JOB_DONE, "jobid="+job.ID+";name="+job.Info.Name+";project="+job.Info.Project+";user="+job.Info.User)

	return
}

// happens when a client checks out a workunit
//update job/task states from "queued" to "in-progress" once the first workunit is checked out
func (qm *ServerMgr) UpdateJobTaskToInProgress(works []*Workunit) (err error) {
	for _, work := range works {
		//job_was_inprogress := false
		//task_was_inprogress := false
		taskid := work.GetTask()
		jobid := work.JobId

		// get job state
		job, xerr := GetJob(jobid)
		if xerr != nil {
			err = xerr
			return
		}

		job_state, xerr := job.GetState(true)
		if xerr != nil {
			err = xerr
			return
		}

		//update job status
		if job_state != JOB_STAT_INPROGRESS {
			err = job.SetState(JOB_STAT_INPROGRESS, nil)
			if err != nil {
				return
			}
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
			err := task.SetState(nil, TASK_STAT_INPROGRESS, true)
			if err != nil {
				logger.Error("(UpdateJobTaskToInProgress) could not update task %s", taskid)
				continue
			}
			err = qm.UpdateTaskPerfStartTime(task)
			if err != nil {
				logger.Error("(UpdateJobTaskToInProgress) UpdateTaskPerfStartTime: %s", err.Error())
				continue
			}

		}
	}
	return
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

	err = job.SetState(jerror.Status, nil)
	if err != nil {
		return
	}

	// set error struct
	err = job.SetError(jerror)
	if err != nil {
		return
	}

	//suspend queueing workunits
	var workunit_list []*Workunit
	workunit_list, err = qm.workQueue.GetAll()
	if err != nil {
		return
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
		workid := workunit.Workunit_Unique_Identifier
		parentid := workunit.JobId
		//parentid, _ := GetJobIdByWorkId(workid)
		if jobid == parentid {
			qm.workQueue.StatusChange(workid, nil, new_work_state, "see job error")
		}
	}

	//suspend parsed tasks
	for _, task := range job.Tasks {
		var task_state string
		task_state, err = task.GetState()
		if err != nil {
			continue
		}
		if task_state == TASK_STAT_QUEUED || task_state == TASK_STAT_READY || task_state == TASK_STAT_INPROGRESS {
			err = task.SetState(nil, new_task_state, true)
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
	var job *Job
	job, err = GetJob(jobid)
	if err != nil {
		return
	}
	// User must have delete permissions on job or be job owner or be an admin
	rights := job.ACL.Check(u.Uuid)
	if job.ACL.Owner != u.Uuid && rights["delete"] == false && u.Admin == false {
		return errors.New(e.UnAuth)
	}
	if err = job.SetState(JOB_STAT_DELETED, nil); err != nil {
		return
	}
	//delete queueing workunits
	var workunit_list []*Workunit
	workunit_list, err = qm.workQueue.GetAll()
	if err != nil {
		return
	}
	for _, workunit := range workunit_list {
		workid := workunit.Workunit_Unique_Identifier
		workunit_jobid := workid.JobId
		//parentid, _ := GetJobIdByWorkId(workid)
		if jobid == workunit_jobid {
			qm.workQueue.Delete(workid)
		}
	}
	//delete parsed tasks
	for i := 0; i < len(job.TaskList()); i++ {
		//task_id := fmt.Sprintf("%s_%d", jobid, i)
		var task_id Task_Unique_Identifier
		task_id, err = New_Task_Unique_Identifier(jobid, strconv.Itoa(i)) // TODO that will not work
		if err != nil {
			return
		}
		qm.TaskMap.Delete(task_id)
	}
	qm.removeActJob(jobid)
	//qm.removeSusJob(jobid)
	// delete from job map
	if err = JM.Delete(jobid, true); err != nil {
		return
	}
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
	if err := dbjobs.GetAll(q, "info.submittime", "asc", false); err != nil {
		logger.Error("DeleteZombieJobs()->GetAllLimitOffset():" + err.Error())
		return
	}
	for _, dbjob := range *dbjobs {
		if !qm.isActJob(dbjob.ID) {
			if err := qm.DeleteJobByUser(dbjob.ID, u, full); err == nil {
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
		err = errors.New("(ResumeSuspendedJobByUser) failed to load job " + err.Error())
		return
	}

	job_state, err := dbjob.GetState(true)
	if err != nil {
		err = errors.New("(ResumeSuspendedJobByUser) failed to get job state " + err.Error())
		return
	}

	// User must have write permissions on job or be job owner or be an admin
	rights := dbjob.ACL.Check(u.Uuid)
	if dbjob.ACL.Owner != u.Uuid && rights["write"] == false && u.Admin == false {
		err = errors.New(e.UnAuth)
		return
	}

	if job_state != JOB_STAT_SUSPEND {
		err = errors.New("(ResumeSuspendedJobByUser) job " + id + " is not in 'suspend' status")
		return
	}
	logger.Debug(1, "resumeing job=%s, state=%s", id, job_state)

	tasks, err := dbjob.GetTasks()
	if err != nil {
		err = errors.New("(ResumeSuspendedJobByUser) failed to get job tasks " + err.Error())
		return
	}

	for _, task := range tasks {
		task_state, serr := task.GetState()
		if serr != nil {
			err = errors.New("(ResumeSuspendedJobByUser) failed to get task state " + serr.Error())
			return
		}
		if contains(TASK_STATS_RESET, task_state) {
			logger.Debug(1, "(ResumeSuspendedJobByUser/ResetTaskTrue) task=%s, state=%s", task.Id, task_state)
			err = task.ResetTaskTrue("Resume")
			if err != nil {
				err = errors.New("(ResumeSuspendedJobByUser) failed to reset task " + err.Error())
				return
			}
		}
	}

	err = dbjob.IncrementResumed(1)
	if err != nil {
		err = errors.New("(ResumeSuspendedJobByUser) failed to incremenet job resumed " + err.Error())
		return
	}

	err = dbjob.SetState(JOB_STAT_QUEUING, nil)
	if err != nil {
		err = fmt.Errorf("(ResumeSuspendedJobByUser) UpdateJobState: %s", err.Error())
		return
	}
	err = qm.EnqueueTasksByJobId(id, "ResumeSuspendedJobByUser")
	if err != nil {
		err = errors.New("(ResumeSuspendedJobByUser) failed to enqueue job " + err.Error())
		return
	}
	logger.Debug(1, "Resumed job %s", id)
	return
}

//recover a job in db that is missing from queue (caused by server restarting)
func (qm *ServerMgr) RecoverJob(id string, job *Job) (recovered bool, err error) {
	// job by id or object
	if job != nil {
		id = job.ID
	} else {
		job, err = GetJob(id)
		if err != nil {
			err = errors.New("(RecoverJob) failed to load job " + err.Error())
			return
		}
	}

	if qm.isActJob(id) {
		// already acive, skip
		return
	}

	job_state, err := job.GetState(true)
	if err != nil {
		err = errors.New("(RecoverJob) failed to get job state " + err.Error())
		return
	}

	if job_state == JOB_STAT_SUSPEND {
		// just add suspended jobs to in-memory map
		err = JM.Add(job)
		if err != nil {
			err = errors.New("(RecoverJob) JM.Add failed " + err.Error())
			return
		}
	} else {
		if job_state == JOB_STAT_COMPLETED || job_state == JOB_STAT_DELETED || job_state == JOB_STAT_FAILED_PERMANENT {
			// unrecoverable, skip
			return
		}
		tasks, terr := job.GetTasks()
		if terr != nil {
			err = errors.New("(RecoverJob) failed to get job tasks " + terr.Error())
			return
		}
		for _, task := range tasks {
			task_state, serr := task.GetState()
			if serr != nil {
				err = errors.New("(RecoverJob) failed to get task state " + serr.Error())
				return
			}
			if contains(TASK_STATS_RESET, task_state) {
				logger.Debug(1, "(RecoverJob/ResetTaskTrue) task=%s, state=%s", task.Id, task_state)
				err = task.ResetTaskTrue("Recover")
				if err != nil {
					err = errors.New("(RecoverJob) failed to reset task " + err.Error())
					return
				}
			}
		}
		err = job.SetState(JOB_STAT_QUEUING, nil)
		if err != nil {
			err = errors.New("(RecoverJob) UpdateJobState: " + err.Error())
			return
		}
		err = qm.EnqueueTasksByJobId(id, "RecoverJob")
		if err != nil {
			err = errors.New("(RecoverJob) failed to enqueue job: " + err.Error())
			return
		}
	}
	recovered = true
	logger.Debug(1, "(RecoverJob) done job=%s", id)
	return
}

//recover jobs not completed before awe-server restarts
func (qm *ServerMgr) RecoverJobs() (recovered int, total int, err error) {
	//Get jobs to be recovered from db whose states are recoverable
	dbjobs := new(Jobs)
	q := bson.M{}
	q["state"] = bson.M{"$in": JOB_STATS_TO_RECOVER}
	if conf.RECOVER_MAX > 0 {
		logger.Info("Recover %d jobs...", conf.RECOVER_MAX)
		if _, err = dbjobs.GetPaginated(q, conf.RECOVER_MAX, 0, "info.priority", "desc", true); err != nil {
			logger.Error("(RecoverJobs) (GetPaginated) " + err.Error())
			return
		}
	} else {
		logger.Info("Recover all jobs")
		if err = dbjobs.GetAll(q, "info.submittime", "asc", true); err != nil {
			logger.Error("(RecoverJobs) (GetAll) " + err.Error())
			return
		}
	}
	total = dbjobs.Length()
	//Locate the job script and parse tasks for each job
	for _, dbjob := range *dbjobs {
		pipeline := "missing"
		if dbjob.Info != nil {
			pipeline = dbjob.Info.Pipeline
		}
		logger.Debug(1, "recovering %d: job=%s, state=%s, pipeline=%s", recovered+1, dbjob.ID, dbjob.State, pipeline)
		isRecovered, rerr := qm.RecoverJob("", dbjob)
		if rerr != nil {
			logger.Error(fmt.Sprintf("(RecoverJobs) job=%s failed: %s", dbjob.ID, rerr.Error()))
			continue
		}
		if isRecovered {
			recovered += 1
		}
	}
	return
}

//recompute job from specified task stage
func (qm *ServerMgr) RecomputeJob(jobid string, task_stage string) (err error) {
	if qm.isActJob(jobid) {
		err = errors.New("(RecomputeJob) job " + jobid + " is already active")
		return
	}
	//Load job by id
	dbjob, err := GetJob(jobid)
	if err != nil {
		err = errors.New("(RecomputeJob) failed to load job " + err.Error())
		return
	}

	job_state, err := dbjob.GetState(true)
	if err != nil {
		err = errors.New("(RecomputeJob) failed to get job state " + err.Error())
		return
	}

	if job_state != JOB_STAT_COMPLETED && job_state != JOB_STAT_SUSPEND {
		err = errors.New("(RecomputeJob) job " + jobid + " is not in 'completed' or 'suspend' status")
		return
	}
	logger.Debug(1, "recomputing: job=%s, state=%s", jobid, job_state)

	from_task_id := fmt.Sprintf("%s_%s", jobid, task_stage)
	remain_steps := 0
	found := false

	tasks, err := dbjob.GetTasks()
	if err != nil {
		err = errors.New("(RecomputeJob) failed to get job tasks " + err.Error())
		return
	}

	for _, task := range tasks {
		task_str, terr := task.String()
		if terr != nil {
			err = errors.New("(RecomputeJob) failed to get task ID string " + terr.Error())
			return
		}
		if task_str == from_task_id {
			logger.Debug(1, "(RecomputeJob/ResetTaskTrue) task=%s, state=%s", task_str, task.State)
			err = task.ResetTaskTrue("Recompute")
			if err != nil {
				err = errors.New("(RecomputeJob) failed to reset task " + err.Error())
				return
			}
			found = true
			remain_steps += 1
		}
	}
	if !found {
		return errors.New("(RecomputeJob) task not found: " + from_task_id)
	}

	for _, task := range tasks {
		task_str, terr := task.String()
		if terr != nil {
			err = errors.New("(RecomputeJob) failed to get task ID string " + terr.Error())
			return
		}
		is_ancest, aerr := isAncestor(dbjob, task_str, from_task_id)
		if aerr != nil {
			err = errors.New("(RecomputeJob) failed to determine if task is ancestor " + aerr.Error())
			return
		}
		task_state, serr := task.GetState()
		if serr != nil {
			err = errors.New("(RecomputeJob) failed to get task state " + serr.Error())
			return
		}
		if is_ancest || contains(TASK_STATS_RESET, task_state) {
			logger.Debug(1, "(RecomputeJob/ResetTaskTrue) task=%s, state=%s", task_str, task_state)
			err = task.ResetTaskTrue("Recompute")
			if err != nil {
				err = errors.New("(RecomputeJob) failed to reset task " + err.Error())
				return
			}
			remain_steps += 1
		}
	}

	err = dbjob.IncrementResumed(1)
	if err != nil {
		err = errors.New("(RecomputeJob) failed to incremenet job resumed " + err.Error())
		return
	}

	err = dbjob.SetState(JOB_STAT_QUEUING, nil)
	if err != nil {
		err = fmt.Errorf("(RecomputeJob) UpdateJobState: %s", err.Error())
		return
	}
	err = qm.EnqueueTasksByJobId(jobid, "RecomputeJob")
	if err != nil {
		err = errors.New("(RecomputeJob) failed to enqueue job " + err.Error())
		return
	}
	logger.Debug(1, "Recomputed job %s from task %s", jobid, task_stage)
	return
}

//ResubmitJob recompute job from beginning
func (qm *ServerMgr) ResubmitJob(jobid string) (err error) {
	if qm.isActJob(jobid) {
		err = errors.New("(ResubmitJob) job " + jobid + " is already active")
		return
	}
	//Load job by id
	job, err := GetJob(jobid)
	if err != nil {
		err = errors.New("(ResubmitJob) failed to load job " + err.Error())
		return
	}

	job_state, err := job.GetState(true)
	if err != nil {
		err = errors.New("(ResubmitJob) failed to get job state " + err.Error())
		return
	}

	if job_state != JOB_STAT_COMPLETED && job_state != JOB_STAT_SUSPEND {
		err = errors.New("(ResubmitJob) job " + jobid + " is not in 'completed' or 'suspend' status")
		return
	}
	logger.Debug(1, "resubmitting: job=%s, state=%s", jobid, job_state)

	//remain_steps := 0
	tasks, err := job.GetTasks()
	if err != nil {
		err = errors.New("(ResubmitJob) failed to get job tasks " + err.Error())
		return
	}

	for _, task := range tasks {
		logger.Debug(1, "(ResubmitJob/ResetTaskTrue) task=%s, state=%s", task.Id, task.State)
		err = task.ResetTaskTrue("Resubmit")
		if err != nil {
			err = errors.New("(ResubmitJob) failed to reset task " + err.Error())
			return
		}
		//	remain_steps += 1
	}

	err = job.IncrementResumed(1)
	if err != nil {
		err = errors.New("(ResubmitJob) failed to incremenet job resumed " + err.Error())
		return
	}

	err = job.SetState(JOB_STAT_QUEUING, nil)
	if err != nil {
		err = fmt.Errorf("(ResubmitJob) UpdateJobState: %s", err.Error())
		return
	}
	err = qm.EnqueueTasksByJobId(jobid, "ResubmitJob")
	if err != nil {
		err = errors.New("(ResubmitJob) failed to enqueue job " + err.Error())
		return
	}
	logger.Debug(1, "Restarted job %s from beginning", jobid)
	return
}

func isAncestor(job *Job, taskId string, testId string) (result bool, err error) {
	if taskId == testId {
		result = false
		return
	}
	idx := -1
	for i, task := range job.Tasks {
		var task_str string
		task_str, err = task.String()
		if err != nil {
			err = fmt.Errorf("(isAncestor) task.String returned: %s", err.Error())
			return
		}
		if task_str == taskId {
			idx = i
			break
		}
	}
	if idx == -1 {
		result = false
		return
	}

	task := job.Tasks[idx]
	if len(task.DependsOn) == 0 {
		result = false
		return
	}
	if contains(task.DependsOn, testId) {
		result = true
		return
	} else {
		for _, t := range task.DependsOn {
			return isAncestor(job, t, testId)
		}
	}
	result = false
	return
}

//update tokens for in-memory data structures
func (qm *ServerMgr) UpdateQueueToken(job *Job) (err error) {
	//job_id := job.ID
	for _, task := range job.Tasks {
		task_id, _ := task.GetId("UpdateQueueToken")
		mtask, ok, err := qm.TaskMap.Get(task_id, true)
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

func (qm *ServerMgr) CreateTaskPerf(task *Task) (err error) {
	jobid := task.JobId
	//taskid := task.String()
	if perf, ok := qm.getActJob(jobid); ok {
		var task_str string
		task_str, err = task.String()
		if err != nil {
			err = fmt.Errorf("() task.String returned: %s", err.Error())
			return
		}
		perf.Ptasks[task_str] = NewTaskPerf(task_str)
		qm.putActJob(perf)
	}
	return
}

func (qm *ServerMgr) UpdateTaskPerfStartTime(task *Task) (err error) {
	jobid := task.JobId

	if jobperf, ok := qm.getActJob(jobid); ok {
		var task_str string
		task_str, err = task.String()
		if err != nil {
			err = fmt.Errorf("() task.String returned: %s", err.Error())
			return
		}
		if taskperf, ok := jobperf.Ptasks[task_str]; ok {
			now := time.Now().Unix()
			taskperf.Start = now
			qm.putActJob(jobperf)
		}
	}
	return
}

// TODO evaluate err
func (qm *ServerMgr) FinalizeTaskPerf(task *Task) (err error) {
	//jobid, _ := GetJobIdByTaskId(task.Id)
	jobid, err := task.GetJobId()
	if err != nil {
		return
	}
	if jobperf, ok := qm.getActJob(jobid); ok {
		//combined_id := task.String()
		var task_str string
		task_str, err = task.String()
		if err != nil {
			err = fmt.Errorf("() task.String returned: %s", err.Error())
			return
		}

		if taskperf, ok := jobperf.Ptasks[task_str]; ok {
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
	return
}

func (qm *ServerMgr) CreateWorkPerf(id Workunit_Unique_Identifier) (err error) {
	if !conf.PERF_LOG_WORKUNIT {
		return
	}
	//workid := id.String()
	jobid := id.JobId
	jobperf, ok := qm.getActJob(jobid)
	if !ok {
		err = fmt.Errorf("(CreateWorkPerf) job perf not found: %s", jobid)
		return
	}
	var work_str string
	work_str, err = id.String()
	if err != nil {
		err = fmt.Errorf("(CreateWorkPerf) id.String() returned: %s", err.Error())
		return
	}
	jobperf.Pworks[work_str] = NewWorkPerf()
	//fmt.Println("write jobperf.Pworks: " + work_str)
	qm.putActJob(jobperf)

	return
}

func (qm *ServerMgr) FinalizeWorkPerf(id Workunit_Unique_Identifier, reportfile string) (err error) {
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
	jobid := id.JobId
	jobperf, ok := qm.getActJob(jobid)
	if !ok {
		return errors.New("(FinalizeWorkPerf) job perf not found:" + jobid)
	}
	//workid := id.String()
	var work_str string
	work_str, err = id.String()
	if err != nil {
		err = fmt.Errorf("(FinalizeWorkPerf) workid.String() returned: %s", err.Error())
		return
	}
	if _, ok := jobperf.Pworks[work_str]; !ok {
		for key, _ := range jobperf.Pworks {
			fmt.Println("FinalizeWorkPerf jobperf.Pworks: " + key)
		}
		return errors.New("(FinalizeWorkPerf) work perf not found:" + work_str)
	}

	workperf.Queued = jobperf.Pworks[work_str].Queued
	workperf.Done = time.Now().Unix()
	workperf.Resp = workperf.Done - workperf.Queued
	jobperf.Pworks[work_str] = workperf
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

func (qm *ServerMgr) FetchPrivateEnv(id Workunit_Unique_Identifier, clientid string) (env map[string]string, err error) {
	//precheck if the client is registered
	client, ok, err := qm.GetClient(clientid, true)
	if err != nil {
		return
	}
	if !ok {
		return env, errors.New(e.ClientNotFound)
	}

	is_suspended, err := client.GetSuspended(true)
	if err != nil {
		return
	}
	if is_suspended {
		err = errors.New(e.ClientSuspended)
		return
	}
	//jobid := id.JobId
	//taskid := id.TaskName

	//job, err := GetJob(jobid)

	task, ok, err := qm.TaskMap.Get(id.Task_Unique_Identifier, true)
	if err != nil {
		err = fmt.Errorf("(FetchPrivateEnv) qm.TaskMap.Get returned: %s", err.Error())
		return
	}

	if !ok {
		//var task_str string
		//task_str, err = task.String()
		//if err != nil {
		//	err = fmt.Errorf("(FetchPrivateEnv) task.String returned: %s", err.Error())
		//	return
		//}
		err = fmt.Errorf("(FetchPrivateEnv) task not found in qm.TaskMap")
		return
	}

	env = task.Cmd.Environ.Private
	return
	//env, err = dbGetPrivateEnv(jobid, taskid)
	//if err != nil {
	//	return
	//}

	//return
}
