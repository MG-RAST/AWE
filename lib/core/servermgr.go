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

type ServerMgr struct {
	CQMgr
	queueLock      sync.Mutex //only update one at a time
	lastUpdate     time.Time
	lastUpdateLock sync.RWMutex
	TaskMap        TaskMap
	ajLock         sync.RWMutex
	actJobs        map[string]*JobPerf
}

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

func (qm *ServerMgr) Lock()    {}
func (qm *ServerMgr) Unlock()  {}
func (qm *ServerMgr) RLock()   {}
func (qm *ServerMgr) RUnlock() {}

func (qm *ServerMgr) UpdateQueueLoop() {
	// TODO this may not be dynamic enough for small amounts of workunits, as they always have to wait
	for {

		err := qm.updateWorkflowInstancesMap()
		if err != nil {
			logger.Error("(UpdateQueueLoop) updateQueue returned: %s", err.Error())
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
		elapsed := time.Since(start)         // type Duration
		elapsed_seconds := elapsed.Seconds() // type float64

		var sleeptime time.Duration
		if elapsed_seconds <= 1 {
			sleeptime = 1 * time.Second // wait at least 1 second
		} else if elapsed_seconds > 5 && elapsed_seconds < 30 {
			sleeptime = elapsed
		} else {
			sleeptime = 30 * time.Second // wait at most 30 seconds
		}

		logger.Debug(3, "(UpdateQueueLoop) elapsed: %s (sleeping for %s)", elapsed, sleeptime)

		time.Sleep(sleeptime)

	}
}

func (qm *ServerMgr) Is_WI_ready(job *Job, wi *WorkflowInstance) (ready bool, reason string, err error) {

	if wi.Inputs != nil {
		ready = true
		return
	}

	cwl_workflow := wi.Workflow
	if cwl_workflow == nil {
		err = fmt.Errorf("Is_WI_ready wi.Workflow==nil")
		return
	}

	fmt.Println("cwl_workflow.Inputs:")
	spew.Dump(cwl_workflow.Inputs)

	fmt.Println("wi.LocalId: " + wi.LocalId)
	parent_workflow_instance_name := path.Dir(wi.LocalId)

	if parent_workflow_instance_name == "." {
		return
	}

	fmt.Println("parent_workflow_instance_name: " + parent_workflow_instance_name)

	var parent_workflow_instance *WorkflowInstance
	var ok bool
	parent_workflow_instance, ok, err = job.GetWorkflowInstance(parent_workflow_instance_name, true)
	if err != nil {
		err = fmt.Errorf("(Is_WI_ready) job.GetWorkflowInstance returned: %s", err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(Is_WI_ready) parent workflow_instance %s not found", parent_workflow_instance_name)
		return
	}

	_ = parent_workflow_instance
	_ = ok

	step := wi.ParentStep

	if step == nil {
		err = fmt.Errorf("(Is_WI_ready)  wi.ParentStep==nil")
		return
	}

	parent_workflow_input_map := parent_workflow_instance.Inputs.GetMap()

	context := job.WorkflowContext

	//var workunit_input_map map[string]cwl.CWLType
	var workunit_input_map cwl.JobDocMap
	workunit_input_map, ok, reason, err = qm.GetStepInputObjects(job, parent_workflow_instance, parent_workflow_input_map, step, context)
	if err != nil {
		err = fmt.Errorf("(NewWorkunit) GetStepInputObjects returned: %s", err.Error())
		return
	}

	fmt.Println("workunit_input_map:")
	spew.Dump(workunit_input_map)

	var workunit_input_array cwl.Job_document
	workunit_input_array, err = workunit_input_map.GetArray()

	wi.Inputs = workunit_input_array

	// for _, step_input := range step.In {
	// 	fmt.Println("step_input:")
	// 	spew.Dump(step_input)

	// 	step_input_id := step_input.Id
	// 	step_input_source := step_input.Source
	// 	step_input_source_array := strings.Split(step_input_source, "/")

	// 	if len(step_input_id_array) == 3 { // step output

	// 	}

	// 	step_input_id_base := path.Base(step_input_id)

	// 	fmt.Println("parent_workflow_instance.Tasks: " + string(len(parent_workflow_instance.Tasks)))
	// 	for _, p_task := range parent_workflow_instance.Tasks {
	// 		fmt.Println("p_task: " + p_task.Id)

	// 	}

	// }

	//fmt.Println("step:")
	//spew.Dump(step)

	ready = true

	return

}

func (qm *ServerMgr) updateWorkflowInstancesMapTask(wi *WorkflowInstance) (err error) {

	wi_local_id := wi.LocalId
	wi_state, _ := wi.GetState(true)

	logger.Debug(3, "(updateWorkflowInstancesMapTask) start: %s state: %s", wi.LocalId, wi_state)

	if wi_state == WI_STAT_PENDING {

		jobid := wi.JobId
		//workflow_def_str := wi.Workflow_Definition

		var job *Job
		job, err = GetJob(jobid)
		if err != nil {
			err = fmt.Errorf("(updateWorkflowInstancesMapTask) GetJob failed: %s", err.Error())
			return

		}

		context := job.WorkflowContext

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

		// TODO ? Create a Is_WI_ready() function
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

		// if wi.Inputs == nil {
		// 	logger.Debug(3, "(updateWorkflowInstancesMapTask) is not ready: %s has no inputs (nil)", wi.LocalId)
		// 	return
		// }

		// if len(wi.Inputs) == 0 {
		// 	logger.Debug(3, "(updateWorkflowInstancesMapTask) is not ready: %s has no inputs (empty)", wi.LocalId)
		// 	return
		// }

		// state is pending, might have to create tasks from steps

		subworkflow_str := []string{}

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

				err = wi.AddTask(awe_task, "db_sync_yes", true)
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
				subworkflow_str = append(subworkflow_str, wi_local_id+"/"+stepname_base)
				// create new WorkflowInstance

				subworkflow, ok := process.(*cwl.Workflow)
				if !ok {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) cannot cast to *cwl.Workflow")
					return
				}

				subworkflow_id := subworkflow.GetId()

				fmt.Printf("(updateWorkflowInstancesMapTask) Creating Workflow %s\n", subworkflow_id)

				new_wi_name := wi_local_id + "/" + stepname_base

				// TODO assign inputs
				//var workflow_inputs cwl.Job_document

				//panic("creating new subworkflow " + new_wi_name)
				var new_wi *WorkflowInstance
				new_wi, err = NewWorkflowInstance(new_wi_name, jobid, subworkflow_id, job)
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) NewWorkflowInstance returned: %s", err.Error())
					return
				}

				new_wi.Workflow = subworkflow
				new_wi.ParentStep = step
				new_wi.SetState(WI_STAT_PENDING, "db_sync_true", false)

				err = job.AddWorkflowInstance(new_wi, "db_sync_yes", true)
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) job.AddWorkflowInstance returned: %s", err.Error())
					return
				}

				err = GlobalWorkflowInstanceMap.Add(new_wi)
				if err != nil {
					err = fmt.Errorf("(updateWorkflowInstancesMapTask) GlobalWorkflowInstanceMap.Add returned: %s", err.Error())
					return
				}

			default:
				err = fmt.Errorf("(updateWorkflowInstancesMapTask) type unknown: %s", reflect.TypeOf(process))
				return
			}

			spew.Dump(cwl_workflow.Steps[i])

		}
		wi.Subworkflows = subworkflow_str
		wi.RemainSteps = len(cwl_workflow.Steps)
		wi.SetState(WI_STAT_READY, "db_sync_true", true)

		if wi.Inputs != nil && len(wi.Inputs) > 0 {
			//panic("found something...")

			err = qm.EnqueueTasks(wi.Tasks)
			if err != nil {
				err = fmt.Errorf("(updateWorkflowInstancesMapTask) EnqueueTasks returned: %s", err.Error())

				return
			}
		}

		err = wi.SetState(WI_STAT_QUEUED, "db_sync_yes", true)
		if err != nil {
			err = fmt.Errorf("(updateWorkflowInstancesMapTask) SetState returned: %s", err.Error())
			return
		}

		if wi_local_id == "_root" {

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
	for _, wi := range wis {

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

		id, err := notice.Id.String()
		if err != nil {
			logger.Error("(NoticeHandle) notice.Id invalid: " + err.Error())
			err = nil
			continue
		}

		logger.Debug(3, "(ServerMgr NoticeHandle) got notice: workid=%s, status=%s, clientid=%s", id, notice.Status, notice.WorkerId)

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
		fmt.Printf("*** job ***\n")
		for wi_id, _ := range job.WorkflowInstancesMap {
			wi := job.WorkflowInstancesMap[wi_id]
			wi_state, _ := wi.GetState(true)
			fmt.Printf("WorkflowInstance: %s (%s)\n", wi_id, wi_state)

			for j, _ := range wi.Tasks {
				task := wi.Tasks[j]
				fmt.Printf("  Task %d: %s (wf: %s, state %s)\n", j, task.Id, task.WorkflowInstanceId, task.State)

			}

		}

	}

	logger.Debug(3, "(updateQueue) range tasks (%d)", len(tasks))

	threads := 20
	size, _ := qm.TaskMap.Len()
	loopStart := time.Now()
	logger.Info("(updateQueue) starting loop through TaskMap; threads: %d, TaskMap.Len: %d", threads, size)

	taskChan := make(chan *Task, size)
	queueChan := make(chan bool, size)
	for w := 1; w <= threads; w++ {
		go qm.updateQueueWorker(w, logTimes, taskChan, queueChan)
	}

	total := 0
	for _, task := range tasks {

		task_id_str, _ := task.String()
		logger.Debug(3, "(updateQueue) sending task to QueueWorkers: %s", task_id_str)

		total += 1
		taskChan <- task
	}
	close(taskChan)

	queue := 0
	for i := 1; i <= size; i++ {
		q := <-queueChan
		if q {
			queue += 1

		}
	}
	close(queueChan)
	logger.Info("(updateQueue) completed loop through TaskMap; # processed: %d, queued: %d, took %s", total, queue, time.Since(loopStart))

	logger.Debug(3, "(updateQueue) range qm.workQueue.Clean()")
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
		task_id_str, _ := task.String()
		isQueued, times, err := qm.updateQueueTask(task, logTimes)
		if err != nil {
			logger.Error("(updateQueue) updateQueueTask task: %s error: %s", task_id_str, err.Error())
		}
		if logTimes {
			taskTime := time.Since(taskStart)
			message := fmt.Sprintf("(updateQueue) thread %d processed task: %s, took: %s, is queued %t", id, task_id_str, taskTime, isQueued)
			if taskTime > taskSlow {
				message += fmt.Sprintf(", times: %+v", times)
			}
			logger.Info(message)
		}
		queueChan <- isQueued
	}
}

func (qm *ServerMgr) updateQueueTask(task *Task, logTimes bool) (isQueued bool, times map[string]time.Duration, err error) {
	var task_id Task_Unique_Identifier
	task_id, err = task.GetId("updateQueueTask")
	if err != nil {
		return
	}

	task_id_str, _ := task_id.String()

	var task_state string
	task_state, err = task.GetState()
	if err != nil {
		return
	}

	if !(task_state == TASK_STAT_INIT || task_state == TASK_STAT_PENDING || task_state == TASK_STAT_READY) {
		logger.Debug(3, "(updateQueueTask) skipping task %s , it has state %s", task_id_str, task_state)
		return
	}

	logger.Debug(3, "(updateQueueTask) task: %s", task_id_str)
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
		logger.Debug(3, "(updateQueueTask) task not ready: %s reason: %s", task_id_str, reason)
		return
	}

	// task_ready
	logger.Debug(3, "(updateQueueTask) task ready: %s", task_id_str)

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
	teqTimes, xerr := qm.taskEnQueue(task_id, task, job, logTimes)
	if logTimes {
		for k, v := range teqTimes {
			times[k] = v
		}
		times["taskEnQueue"] = time.Since(startEnQueue)
	}
	if xerr != nil {
		logger.Error("(updateQueueTask) taskEnQueue returned: %s", xerr.Error())
		_ = task.SetState(TASK_STAT_SUSPEND, true)

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
			ServerNotes: fmt.Sprintf("failed enqueuing task %s, err=%s", task_id_str, xerr.Error()),
			Status:      JOB_STAT_SUSPEND,
		}
		if err = qm.SuspendJob(job_id, jerror); err != nil {
			return
		}
		err = xerr
		return
	} else {
		isQueued = true
	}
	logger.Debug(3, "(updateQueueTask) task enqueued: %s", task_id_str)

	return
}

func RemoveWorkFromClient(client *Client, workid Workunit_Unique_Identifier) (err error) {
	err = client.Assigned_work.Delete(workid, true)
	if err != nil {
		return
	}

	work_length, err := client.Assigned_work.Length(true)
	if err != nil {
		return
	}

	if work_length > 0 {

		clientid, _ := client.Get_Id(true)

		logger.Error("(RemoveWorkFromClient) Client %s still has %d workunits assigned, after delivering one workunit", clientid, work_length)

		assigned_work_ids, err := client.Assigned_work.Get_list(true)
		if err != nil {
			return err
		}
		for _, work_id := range assigned_work_ids {
			_ = client.Assigned_work.Delete(work_id, true)
		}

		work_length, err = client.Assigned_work.Length(true)
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

func (qm *ServerMgr) handleWorkStatDone(client *Client, clientid string, task *Task, workid Workunit_Unique_Identifier, computetime int) (err error) {
	//log event about work done (WD)

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
		err = client.Increment_total_completed()
		if err != nil {
			err = fmt.Errorf("(handleWorkStatDone) client.Increment_total_completed returned: %s", err.Error())
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

	if remain_work > 0 {
		return
	}

	// ******* LAST WORKUNIT ******

	// validate file sizes of all outputs
	err = task.ValidateOutputs()
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
		err = task.SetState(TASK_STAT_SUSPEND, true)
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

	err = task.SetState(TASK_STAT_COMPLETED, true)
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) task.SetState returned: %s", err.Error())
		return
	}

	//log event about task done (TD)
	err = qm.FinalizeTaskPerf(task)
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) FinalizeTaskPerf returned: %s", err.Error())
		return
	}
	logger.Event(event.TASK_DONE, "task_id="+task_str)

	//update the info of the job which the task is belong to, could result in deletion of the
	//task in the task map when the task is the final task of the job to be done.
	err = qm.taskCompleted(task) //task state QUEUED -> COMPLETED
	if err != nil {
		err = fmt.Errorf("(handleWorkStatDone) updateJobTask returned: %s", err.Error())
		return
	}
	return
}

//handle feedback from a client about the execution of a workunit
func (qm *ServerMgr) handleNoticeWorkDelivered(notice Notice) (err error) {

	clientid := notice.WorkerId

	work_id := notice.Id
	task_id := work_id.GetTask()

	job_id := work_id.JobId

	notice_status := notice.Status

	computetime := notice.ComputeTime
	notes := notice.Notes

	var work_str string
	work_str, err = work_id.String()
	if err != nil {
		err = fmt.Errorf("(handleNoticeWorkDelivered) work_id.String() returned: %s", err.Error())
		return
	}

	logger.Debug(3, "(handleNoticeWorkDelivered) workid: %s status: %s client: %s", work_str, notice_status, clientid)

	// we should not get here, but if we do than end
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

	if notice.Results != nil { // TODO one workunit vs multiple !!!!!!!!!!!!!!!!!!!!!!!!!!!!!
		err = task.SetStepOutput(notice.Results, true)
		if err != nil {
			err = fmt.Errorf("(handleNoticeWorkDelivered) task.SetStepOutput returned: %s", err.Error())
			return
		}
	}

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
		qm.workQueue.Delete(work_id)
		err = fmt.Errorf("(handleNoticeWorkDelivered) workunit %s failed due to skip", work_str)
		return
	}

	logger.Debug(3, "(handleNoticeWorkDelivered) handling status %s", notice_status)
	if notice_status == WORK_STAT_DONE {
		err = qm.handleWorkStatDone(client, clientid, task, work_id, computetime)
		if err != nil {
			err = fmt.Errorf("(handleNoticeWorkDelivered) handleWorkStatDone returned: %s", err.Error())
			return
		}
	} else if notice_status == WORK_STAT_FAILED_PERMANENT { // (special case !) failed and cannot be recovered

		logger.Event(event.WORK_FAILED, "workid="+work_str+";clientid="+clientid)
		logger.Debug(3, "(handleNoticeWorkDelivered) work failed (status=%s) workid=%s clientid=%s", notice_status, work_str, clientid)
		work.Failed += 1

		qm.workQueue.StatusChange(Workunit_Unique_Identifier{}, work, WORK_STAT_FAILED_PERMANENT, "")

		if err = task.SetState(TASK_STAT_FAILED_PERMANENT, true); err != nil {
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
		if err = qm.SuspendJob(job_id, jerror); err != nil {
			logger.Error("(handleNoticeWorkDelivered:SuspendJob) job_id=%s; err=%s", job_id, err.Error())
		}
	} else if notice_status == WORK_STAT_ERROR { //workunit failed, requeue or put it to suspend list
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

			if err = task.SetState(TASK_STAT_SUSPEND, true); err != nil {
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

		err = client.Append_Skip_work(work_id, true)
		if err != nil {
			return
		}
		err = client.Increment_total_failed(true)
		if err != nil {
			return
		}

		var last_failed int
		last_failed, err = client.Increment_last_failed(true)
		if err != nil {
			return
		}
		if last_failed >= conf.MAX_CLIENT_FAILURE {
			qm.SuspendClient(clientid, client, "MAX_CLIENT_FAILURE on client reached", true)
		}
	} else {
		err = fmt.Errorf("No handler for workunit status '%s' implemented (allowd: %s, %s, %s)", notice_status, WORK_STAT_DONE, WORK_STAT_FAILED_PERMANENT, WORK_STAT_ERROR)
		return
	}
	return
}

func (qm *ServerMgr) GetJsonStatus() (status map[string]map[string]int, err error) {
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

	//active_jobs := 0
	//active_jobs := qm.lenActJobs()
	//suspend_job := qm.lenSusJobs()
	//total_job := active_jobs + suspend_job

	//total_task := 0
	//queuing_task := 0
	//started_task := 0
	//pending_task := 0
	//completed_task := 0
	//suspended_task := 0
	//skipped_task := 0
	//fail_skip_task := 0

	// *** jobs ***
	jobs := make(map[string]int)

	var job_list []*Job
	job_list, err = JM.Get_List(true)
	jobs["total"] = len(job_list)

	for _, job := range job_list {

		job_state, _ := job.GetState(true)

		jobs[job_state] += 1

	}

	// *** tasks ***

	tasks := make(map[string]int)

	task_list, err := qm.TaskMap.GetTasks()
	if err != nil {
		return
	}
	tasks["total"] = len(task_list)

	for _, task := range task_list {

		task_state, _ := task.GetState()

		tasks[task_state] += 1

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

	total_client := 0
	busy_client := 0
	idle_client := 0
	suspend_client := 0

	client_list, err := qm.clientMap.GetClients()
	if err != nil {
		return
	}

	for _, client := range client_list {
		total_client += 1

		if client.Suspended {
			suspend_client += 1
		}
		if client.Busy {
			busy_client += 1
		} else {
			idle_client += 1
		}
	}

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
func (qm *ServerMgr) FetchDataToken(work_id Workunit_Unique_Identifier, clientid string) (token string, err error) {

	//precheck if the client is registered
	client, ok, err := qm.GetClient(clientid, true)
	if err != nil {
		return
	}
	if !ok {
		return "", errors.New(e.ClientNotFound)
	}

	is_suspended, err := client.Get_Suspended(true)
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

	err = GlobalWorkflowInstanceMap.Add(wi)
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
			task.SetState(TASK_STAT_READY, true)
		} else if task_state == TASK_STAT_SUSPEND {
			task.SetState(TASK_STAT_PENDING, true)
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
func (qm *ServerMgr) EnqueueTasksByJobId(jobid string) (err error) {
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
			task.SetState(TASK_STAT_READY, true)
		} else if task_state == TASK_STAT_SUSPEND {
			task.SetState(TASK_STAT_PENDING, true)
		}

		// add to qm.TaskMap
		// updateQueue() process will actually enqueue the task
		// TaskMap.Add - makes it a pending task if init, throws error if task already in map with different pointer
		err = qm.TaskMap.Add(task, "EnqueueTasksByJobId")
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
	logger.Debug(3, "(isTaskReady) task state is %s", task_state)

	if task_state == TASK_STAT_READY {
		ready = true
		return
	}

	if task_state == TASK_STAT_INIT || task_state == TASK_STAT_PENDING {
		logger.Debug(3, "(isTaskReady) task state is %s", task_state)
	} else {
		err = fmt.Errorf("(isTaskReady) task has state %s, it does not make sense to test if it is ready", task_state)
		return
	}

	//task_id, err := task.GetId("isTaskReady")
	//if err != nil {
	//	return
	//}

	task_id_str, _ := task_id.String()

	logger.Debug(3, "(isTaskReady %s)", task_id_str)

	//skip if the belonging job is suspended
	jobid, err := task.GetJobId()
	if err != nil {
		return
	}

	job, err := GetJob(jobid)
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
				logger.Debug(3, "(isTaskReady %s) too early to execute (now: %s, StartAt: %s)", task_id_str, time.Now(), info.StartAt)
				return
			} else {
				logger.Debug(3, "(isTaskReady %s) StartAt field is in the past, can execute now (now: %s, StartAt: %s)", task_id_str, time.Now(), info.StartAt)
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

		workflow_input_map := workflow_instance.Inputs.GetMap()
		workflow_instance_id := workflow_instance.LocalId
		workflow_def := workflow_instance.Workflow_Definition

		//fmt.Println("WorkflowStep.Id: " + task.WorkflowStep.Id)
		for _, wsi := range task.WorkflowStep.In { // WorkflowStepInput
			if wsi.Source == nil {
				if wsi.Default != nil { // input is optional, anyway....
					continue
				}
				if wsi.ValueFrom != "" { // input is optional, anyway....
					continue
				}
				reason = fmt.Sprintf("(isTaskReady) No Source, Default or ValueFrom found")
				return

			} else {
				source_is_array := false
				source_as_array, source_is_array := wsi.Source.([]interface{})

				if source_is_array {
					for _, src := range source_as_array { // usually only one
						var src_str string
						var ok bool

						src_str, ok = src.(string)
						if !ok {
							err = fmt.Errorf("src is not a string")
							return
						}

						_, ok, _, err = qm.getCWLSource(job, workflow_instance, workflow_input_map, src_str, false, job.WorkflowContext)
						if err != nil {
							err = fmt.Errorf("(isTaskReady) (type array, src_str: %s) getCWLSource returns: %s", src_str, err.Error())
							return
						}
						if !ok {
							reason = fmt.Sprintf("Source CWL object (type array) %s not found", src_str)
							return
						}
					}
				} else {
					var src_str string
					var ok bool

					src_str, ok = wsi.Source.(string)
					if !ok {
						err = fmt.Errorf("(isTaskReady) Cannot parse WorkflowStep source: %s", spew.Sdump(wsi.Source))
						return
					}
					fmt.Println("----------------------------")
					spew.Dump(workflow_instance)
					fmt.Println("----------------------------")

					fmt.Println("workflow_def: " + workflow_def)
					fmt.Println("src_str: " + src_str)
					fmt.Println("workflow_instance_id: " + workflow_instance_id)
					fmt.Println("task_id: " + task_id_str)

					src_str = strings.TrimPrefix(src_str, workflow_def)
					src_str = strings.TrimPrefix(src_str, "/")

					src_str_array := strings.Split(src_str, "/")
					src_str_array_len := len(src_str_array)
					if src_str_array_len == 1 {
						_, ok, err = qm.getCWLSourceFromWorkflowInput_deprecated(workflow_input_map, src_str)
						if err != nil {
							err = fmt.Errorf("(isTaskReady) (type non-array, src_str: %s) getCWLSource returned: %s", src_str, err.Error())
							return
						}
						if !ok {

							reason = fmt.Sprintf("Source CWL object not found (type non-array, len: %d) src_str=%s", src_str_array_len, src_str)
							return
						}
						continue
					} else if src_str_array_len == 2 {
						// src = workflow_instance_id / src_str_array[0] / src_str_array[1]
						var step_reason string
						_, ok, step_reason, err = qm.getCWLSourceFromStepOutput(job, workflow_instance, workflow_instance_id, src_str_array[0], src_str_array[1], false)
						if err != nil {
							err = fmt.Errorf("(isTaskReady) (type non-array, src_str: %s) getCWLSource returns: %s", src_str, err.Error())
							return
						}
						if !ok {

							reason = fmt.Sprintf("Source CWL object not found (type non-array, len: %d) %s/%s/%s (%s) reason: %s", src_str_array_len, workflow_instance_id, src_str_array[0], src_str_array[1], src_str, step_reason)
							return
						}
						continue

					} else {
						panic("source: " + src_str + " " + string(len(src_str_array)))

					}

					src_str = workflow_instance_id + "/" + src_str

					fmt.Println("new src_str: " + src_str)

					_, ok, _, err = qm.getCWLSource(job, workflow_instance, workflow_input_map, src_str, false, job.WorkflowContext)
					if err != nil {
						err = fmt.Errorf("(isTaskReady) (type non-array, src_str: %s) getCWLSource returns: %s", src_str, err.Error())
						return
					}
					if !ok {

						reason = fmt.Sprintf("Source CWL object (type non-array) %s not found", src_str)
						return
					}
				}
			}
		}
	}

	if task.WorkflowStep == nil {
		// task read lock, check DependsOn list and IO.Origin list
		reason, err = task.ValidateDependants(qm)
		if err != nil {
			err = fmt.Errorf("(isTaskReady) %s", err.Error())
			return
		}
		if reason != "" {
			return
		}
	}

	// now we are ready
	err = task.SetState(TASK_STAT_READY, true)
	if err != nil {
		err = fmt.Errorf("(isTaskReady) task.SetState returned: %s", err.Error())
		return
	}
	ready = true

	logger.Debug(3, "(isTaskReady) finished, task %s is ready", task_id_str)
	return
}

func (qm *ServerMgr) taskEnQueueWorkflow(task *Task, job *Job, workflow_input_map cwl.JobDocMap, workflow *cwl.Workflow, logTimes bool) (times map[string]time.Duration, err error) {

	if workflow == nil {
		err = fmt.Errorf("(taskEnQueueWorkflow) workflow == nil !?")
		return
	}

	cwl_step := task.WorkflowStep
	task_id := task.Task_Unique_Identifier
	task_id_str, _ := task_id.String()

	workflow_defintion_id := workflow.Id

	var workflow_instance_id string

	workflow_instance_id = task_id.TaskName

	// find inputs
	var task_input_array cwl.Job_document
	var task_input_map cwl.JobDocMap

	context := job.WorkflowContext

	if task.StepInput == nil {

		var workflow_instance *WorkflowInstance
		var ok bool
		workflow_instance, ok, err = job.GetWorkflowInstance(task.WorkflowInstanceId, true)
		if err != nil {
			err = fmt.Errorf("(taskEnQueueWorkflow) job.GetWorkflowInstance returned: %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(taskEnQueueWorkflow) workflow_instance not found")
			return
		}

		var reason string
		task_input_map, ok, reason, err = qm.GetStepInputObjects(job, workflow_instance, workflow_input_map, cwl_step, context) // returns map[string]CWLType
		if err != nil {
			err = fmt.Errorf("(taskEnQueueWorkflow) GetStepInputObjects returned: %s", err.Error())
			return
		}

		if !ok {
			err = fmt.Errorf("(taskEnQueueWorkflow) GetStepInputObjects not ready, reason: %s", reason)
			return
		}

		if len(task_input_map) == 0 {
			err = fmt.Errorf("(taskEnQueueWorkflow) A) len(task_input_map) == 0 (%s)", task_id_str)
			return
		}

		task_input_array, err = task_input_map.GetArray()
		if err != nil {
			err = fmt.Errorf("(taskEnQueueWorkflow) task_input_map.GetArray returned: %s", err.Error())
			return
		}
		task.StepInput = &task_input_array
		//task_input_map = task_input_array.GetMap()

	} else {
		task_input_array = *task.StepInput
		task_input_map = task_input_array.GetMap()
		if len(task_input_map) == 0 {
			err = fmt.Errorf("(taskEnQueueWorkflow) B) len(task_input_map) == 0 ")
			return
		}
	}

	if strings.HasSuffix(task.TaskName, "/") {
		err = fmt.Errorf("(taskEnQueueWorkflow) Slash at the end of TaskName!? %s", task.TaskName)
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
	var sub_workflow_tasks []*Task
	sub_workflow_tasks, err = CreateWorkflowTasks(job, workflow_instance_id, workflow.Steps, workflow.Id, &task_id)
	if err != nil {
		err = fmt.Errorf("(taskEnQueueWorkflow) CreateWorkflowTasks returned: %s", err.Error())
		return
	}

	var wi *WorkflowInstance

	wi, err = NewWorkflowInstance(workflow_instance_id, job.Id, workflow_defintion_id, job)
	if err != nil {
		err = fmt.Errorf("(AddWorkflowInstance) NewWorkflowInstance returned: %s", err.Error())
		return
	}
	wi.Inputs = task_input_array

	err = job.AddWorkflowInstance(wi, "db_sync_yes", true)
	if err != nil {
		err = fmt.Errorf("(taskEnQueueWorkflow) job.AddWorkflowInstance returned: %s", err.Error())
		return
	}
	//wi, err = job.AddWorkflowInstance(workflow_instance_id, workflow_defintion_id, task_input_array, len(workflow.Steps), sub_workflow_tasks)
	//if err != nil {
	//	err = fmt.Errorf("(taskEnQueueWorkflow) job.AddWorkflowInstance returned: %s", err.Error())
	//	return
	//}

	err = job.IncrementRemainTasks(len(sub_workflow_tasks))
	if err != nil {
		err = fmt.Errorf("(taskEnQueueWorkflow) job.IncrementRemainTasks returned: %s", err.Error())
		return
	}

	times = make(map[string]time.Duration)
	//skip_workunit := false

	children := []Task_Unique_Identifier{}
	for i := range sub_workflow_tasks {
		sub_task := sub_workflow_tasks[i]
		_, err = sub_task.Init(job, job.Id)
		if err != nil {
			err = fmt.Errorf("(taskEnQueueWorkflow) sub_task.Init() returns: %s", err.Error())
			return
		}

		var sub_task_id Task_Unique_Identifier
		sub_task_id, err = sub_task.GetId("task." + strconv.Itoa(i))
		if err != nil {
			return
		}
		sub_task_id_str, _ := sub_task_id.String()
		children = append(children, sub_task_id)

		//err = job.AddTask(sub_task)
		err = wi.AddTask(sub_task, "db_sync_yes", true)
		if err != nil {
			err = fmt.Errorf("(taskEnQueueWorkflow) job.AddTask returns: %s", err.Error())
			return
		}

		// add to qm.TaskMap
		// updateQueue() process will actually enqueue the task
		// TaskMap.Add - makes it a pending task if init, throws error if task already in map with different pointer
		err = qm.TaskMap.Add(sub_task, "taskEnQueueWorkflow")
		if err != nil {
			err = fmt.Errorf("(taskEnQueueWorkflow) (subtask: %s) qm.TaskMap.Add() returns: %s", sub_task_id_str, err.Error())
			return
		}
	}
	task.SetChildren(qm, children, true)

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
						err = fmt.Errorf("(taskEnQueue) element in source array is not a string")
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
		var scatter_input_object cwl.CWL_object
		var ok bool
		scatter_input_object, ok, _, err = qm.getCWLSource(job, workflow_instance, workflow_input_map, scatter_input_source_str, true, job.WorkflowContext)
		if err != nil {
			err = fmt.Errorf("(taskEnQueue) getCWLSource returned: %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(taskEnQueue) scatter_input %s not found.", scatter_input)
			return
		}

		var scatter_input_array_ptr *cwl.Array
		scatter_input_array_ptr, ok = scatter_input_object.(*cwl.Array)
		if !ok {

			err = fmt.Errorf("(taskEnQueue) scatter_input_object type is not *cwl.Array: %s", reflect.TypeOf(scatter_input_object))
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
		err = fmt.Errorf("(taskEnQueue) Scatter type %s unknown", scatter_method)
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

	fmt.Println("template_scatter_step_ins:")
	spew.Dump(template_scatter_step_ins)
	if len(template_scatter_step_ins) == 0 {
		err = fmt.Errorf("(taskEnQueue) no scatter tasks found")
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
			err = fmt.Errorf("(taskEnQueue) Creation of fake workunitfailed: %s", err.Error())
			return
		}
		qm.workQueue.Add(workunit)
		err = workunit.SetState(WORK_STAT_CHECKOUT, "internal processing")
		if err != nil {
			err = fmt.Errorf("(taskEnQueue) workunit.SetState failed: %s", err.Error())
			return
		}

		// create empty arrays for output and return notice

		notice = &Notice{}
		notice.WorkerId = "_internal"
		notice.Id = New_Workunit_Unique_Identifier(task.Task_Unique_Identifier, 0)
		notice.Status = WORK_STAT_DONE
		notice.ComputeTime = 0

		notice.Results = &cwl.Job_document{}

		for _, out := range cwl_step.Out {
			//
			out_name := out.Id
			//fmt.Printf("outname: %s\n", out_name)

			new_array := &cwl.Array{}

			//new_out := cwl.NewNamedCWLType(out_name, new_array)
			spew.Dump(*notice.Results)
			notice.Results = notice.Results.Add(out_name, new_array)
			spew.Dump(*notice.Results)
		}

		//fmt.Printf("len(notice.Results): %d\n", len(*notice.Results))

		return
	}

	fmt.Println("template_task_step without scatter:")
	spew.Dump(template_task_step)

	//if count_of_scatter_arrays == 1 {

	// create tasks
	var children []Task_Unique_Identifier
	var new_scatter_tasks []*Task

	counter_running := true

	for counter_running {

		permutation_instance := ""
		for i := 0; i < counter.NumberOfSets-1; i++ {
			permutation_instance += strconv.Itoa(counter.Counter[i]) + "_"
		}
		permutation_instance += strconv.Itoa(counter.Counter[counter.NumberOfSets-1])

		scatter_task_name := task_id.TaskName + "_scatter" + permutation_instance

		// create task
		var awe_task *Task

		var parent_id_str string
		// parent_id_str, err = task.GetWorkflowParentStr()
		// if err != nil {
		// 	err = fmt.Errorf("(taskEnQueue) task.GetWorkflowParentStr returned: %s", err.Error())
		// 	return
		// }
		parent_id_str = "test"
		logger.Debug(3, "(taskEnQueue) New Task: parent: %s and scatter_task_name: %s", parent_id_str, scatter_task_name)

		awe_task, err = NewTask(job, parent_id_str, scatter_task_name)
		if err != nil {
			err = fmt.Errorf("(taskEnQueue) NewTask returned: %s", err.Error())
			return
		}

		awe_task.Scatter_parent = &task.Task_Unique_Identifier
		//awe_task.Scatter_task = true
		_, err = awe_task.Init(job, job.Id)
		if err != nil {
			err = fmt.Errorf("(taskEnQueue) awe_task.Init() returns: %s", err.Error())
			return
		}

		// create step
		var new_task_step cwl.WorkflowStep
		//var new_task_step_in []cwl.WorkflowStepInput
		new_task_step = template_task_step // this should make a copy from template, (this is not a nested copy)

		fmt.Println("new_task_step initial:")
		spew.Dump(new_task_step)

		// copy scatter inputs

		for input_name := range template_scatter_step_ins {
			input_name_base := path.Base(input_name)
			scatter_input, ok := template_scatter_step_ins[input_name_base]
			if !ok {
				err = fmt.Errorf("(taskEnQueue) %s not in template_scatter_step_ins", input_name_base)
				return
			}

			input_position, ok := name_to_postiton[input_name_base] // input_position points to an array of inputs
			if !ok {
				err = fmt.Errorf("(taskEnQueue) %s not in name_to_postiton map", input_name_base)
				return
			}

			//the_array := scatter_input_array_ptrs[input_position]

			the_index := counter.Counter[input_position]
			scatter_input.Source_index = the_index + 1
			new_task_step.In = append(new_task_step.In, scatter_input)
		}

		fmt.Println("new_task_step with everything:")
		spew.Dump(new_task_step)

		new_task_step.Id = scatter_task_name
		awe_task.WorkflowStep = &new_task_step
		children = append(children, awe_task.Task_Unique_Identifier)

		new_task_step.Scatter = nil // []string{}
		new_scatter_tasks = append(new_scatter_tasks, awe_task)

		counter_running = counter.Increment()
	}

	task.SetChildren(qm, children, true)
	err = job.IncrementRemainTasks(len(new_scatter_tasks))
	if err != nil {
		return
	}

	// add tasks to job and submit
	for i := range new_scatter_tasks {
		sub_task := new_scatter_tasks[i]

		err = job.AddTask(sub_task)
		if err != nil {
			err = fmt.Errorf("(taskEnQueue) job.AddTask returns: %s", err.Error())
			return
		}

		err = qm.TaskMap.Add(sub_task, "taskEnQueue")
		if err != nil {
			err = fmt.Errorf("(taskEnQueue) qm.TaskMap.Add() returns: %s", err.Error())
			return
		}

		err = sub_task.SetState(TASK_STAT_PENDING, true)
		if err != nil {
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

func (qm *ServerMgr) taskEnQueue(task_id Task_Unique_Identifier, task *Task, job *Job, logTimes bool) (times map[string]time.Duration, err error) {

	task_id_str, _ := task_id.String()

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

	if job.WorkflowContext != nil {
		logger.Debug(3, "(taskEnQueue) have job.WorkflowContext")

		var workflow_instance *WorkflowInstance
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

		workflow_input_map := workflow_instance.Inputs.GetMap()
		cwl_step := task.WorkflowStep

		if cwl_step == nil {
			err = fmt.Errorf("(taskEnQueue) task.WorkflowStep is empty")
			return
		}

		//workflow_with_children := false
		//var wfl *cwl.Workflow

		// Detect task_type

		fmt.Printf("(taskEnQueue) A task_type: %s\n", task_type)

		if task_type == "" {
			err = fmt.Errorf("(taskEnQueue) task_type empty")
			return
		}

		switch task_type {
		case TASK_TYPE_SCATTER:
			logger.Debug(3, "(taskEnQueue) call taskEnQueueScatter")
			notice, err = qm.taskEnQueueScatter(workflow_instance, task, job, workflow_input_map)
			if err != nil {
				err = fmt.Errorf("(taskEnQueue) taskEnQueueScatter returned: %s", err.Error())
				return
			}

		}

	} else {
		logger.Debug(3, "(taskEnQueue) DOES NOT have job.CWL_collection")
	}

	logger.Debug(2, "(taskEnQueue) task %s has type %s", task_id_str, task_type)
	if task_type == TASK_TYPE_SCATTER {
		skip_workunit = true
	}

	logger.Debug(2, "(taskEnQueue) trying to enqueue task %s", task_id_str)

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
	err = task.SetState(TASK_STAT_QUEUED, true)
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

	updateStart := time.Now()
	err = qm.taskCompleted(task) //task status PENDING->QUEUED
	if logTimes {
		times["taskCompleted"] = time.Since(updateStart)
	}
	if err != nil {
		err = fmt.Errorf("(taskEnQueue) qm.taskCompleted: %s", err.Error())
		return
	}

	// log event about task enqueue (TQ)
	logger.Event(event.TASK_ENQUEUE, fmt.Sprintf("taskid=%s;totalwork=%d", task_id_str, task.TotalWork))
	qm.CreateTaskPerf(task)

	logger.Debug(2, "(taskEnQueue) leaving (task=%s)", task_id_str)

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

func (qm *ServerMgr) getCWLSourceFromWorkflowInput_deprecated(workflow_input_map map[string]cwl.CWLType, src_base string) (obj cwl.CWLType, ok bool, err error) {

	//fmt.Println("src_base: " + src_base)
	// search job input
	var this_ok bool
	obj, this_ok = workflow_input_map[src_base]
	if this_ok {
		//fmt.Println("(getCWLSource) found in workflow_input_map: " + src_base)
		ok = true
		return

	} else {

		// workflow inputs that are missing might be optional, thus Null is returned
		obj = cwl.NewNull()
		ok = true
		// not found
		return
	}
	//fmt.Println("(getCWLSource) workflow_input_map:")
	//spew.Dump(workflow_input_map)
}

// src = workflow_name / step_name / output_name
func (qm *ServerMgr) getCWLSourceFromStepOutput(job *Job, workflow_instance *WorkflowInstance, workflow_name string, step_name string, output_name string, error_on_missing_task bool) (obj cwl.CWLType, ok bool, reason string, err error) {
	ok = false
	//step_name_abs := workflow_name + "/" + step_name
	workflow_instance_id, _ := workflow_instance.GetId(true)
	workflow_instance_local_id := workflow_instance.LocalId
	//if workflow_instance_id != workflow_name {
	//	err = fmt.Errorf("(getCWLSourceFromStepOutput) workflow_instance_id != workflow_name ???")
	//	return
	//}


		TODO: is is not clear if step is a task (implenented here) or a workflow output

	//src := workflow_name + "/" + step_name + "/" + output_name
	logger.Debug(3, "(getCWLSourceFromStepOutput) %s / %s / %s (workflow_instance_id: %s)", workflow_name, step_name, output_name, workflow_instance_id)
	// *** check if workflow_name + "/" + step_name is a subworkflow
	//workflow_instance_name := workflow_name + "/" + step_name

	//var wi *WorkflowInstance

	//wi, ok, err = job.GetWorkflowInstance(workflow_instance_name, true)
	//if err != nil {
	//	err = fmt.Errorf("(getCWLSourceFromStepOutput) job.GetWorkflowInstance returned: %s", err.Error())
	//	return
	//}

	//if ok {
	ok = false
	// source is a workflow output
	//obj, ok, err = workflow_instance.GetOutput(output_name, true)
	//if err != nil {
	//	err = fmt.Errorf("(getCWLSourceFromStepOutput) wi.GetOutput returned: %s", err.Error())
	//	return
	//}

	///if !ok {
	//	reason = fmt.Sprintf("Output %s not found in output of workflow_instance", output_name)
	//	return
	//}

	//	ok = true
	//	return

	//}

	// search task and its output
	if workflow_instance.JobId == "" {
		err = fmt.Errorf("(getCWLSourceFromStepOutput) workflow_instance.JobId empty")
		return
	}

	ancestor_task_name_local := workflow_instance_local_id + "/" + step_name

	ancestor_task_id := Task_Unique_Identifier{}
	ancestor_task_id.JobId = workflow_instance.JobId
	ancestor_task_id.TaskName = ancestor_task_name_local

	ancestor_task_id_str, _ := ancestor_task_id.String()
	var ancestor_task *Task

	ancestor_task, ok, err = workflow_instance.GetTask(ancestor_task_id)
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
			tasks_str = workflow_instance_local_id + " has no tasks"
		}
		reason = fmt.Sprintf("ancestor_task %s not found (found %s)", ancestor_task_id_str, tasks_str)
		return
	}

	obj, ok, reason, err = ancestor_task.GetStepOutput(output_name)
	if err != nil {
		err = fmt.Errorf("(getCWLSourceFromStepOutput) ancestor_task.GetStepOutput returned: %s", err.Error())
		return
	}
	if !ok {
		reason = fmt.Sprintf("Output %s not found in output of ancestor_task", ancestor_task_name_local)
		return
	}

	//err = fmt.Errorf("(getCWLSourceFromStepOutput) not a subworkflow")
	return

}

func (qm *ServerMgr) GetSourceFromWorkflowInstanceInput(workflow_instance *WorkflowInstance, src string, error_on_missing_task bool) (obj cwl.CWLType, ok bool, err error) {

	ok = false

	fmt.Printf("(GetSourceFromWorkflowInstanceInput) src: %s\n", src)
	src_base := path.Base(src)

	fmt.Printf("(GetSourceFromWorkflowInstanceInput) src_base: %s\n", src_base)
	src_path := strings.TrimSuffix(src, "/"+src_base)

	//src_array := strings.Split(src, "/")
	//src_base := src_array[1]

	obj, err = workflow_instance.Inputs.Get(src_base)
	if err != nil {

		if error_on_missing_task {
			fmt.Println("workflow_instance.Inputs:")
			spew.Dump(workflow_instance.Inputs)
			//	panic("output not found a)")

			err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) found ancestor_task %s, but output %s not found in workflow_instance.Inputs (was %s)", src_path, src_base, src)
			return
		}

		err = nil
		logger.Debug(3, "(GetSourceFromWorkflowInstanceInput) found ancestor_task %s, but output %s not found in workflow_instance.Inputs", src_path, src_base)
		ok = false
		return
	}
	ok = true
	return

	//}

	// 2) find task / task output

	//ancestor_task_id := current_task_id

	// src_path + src_base
	step_name := src_path
	fmt.Printf("(GetSourceFromWorkflowInstanceInput) step_name: %s\n", step_name)

	//workflow_name := path.Base(step_name)
	//workflow_name = strings.TrimSuffix(workflow_name, "/")
	//fmt.Printf("(getCWLSource) workflow_name: %s\n", workflow_name)
	ancestor_task_id := Task_Unique_Identifier{}
	ancestor_task_id.JobId = workflow_instance.JobId
	ancestor_task_id.TaskName = step_name

	var ancestor_task *Task
	ancestor_task, ok, err = workflow_instance.GetTask(ancestor_task_id)
	if err != nil {
		err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) workflow_instance.GetTask returned: %s", err.Error())
		return
	}
	//ancestor_task_id.TaskName = step_name

	//var ancestor_task *Task
	// var local_ok bool
	// ancestor_task, local_ok, err = qm.TaskMap.Get(ancestor_task_id, true)
	// if err != nil {
	// 	err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) qm.TaskMap.Get returned: %s", err.Error())
	// 	return
	// }
	if !ok {
		if error_on_missing_task {
			err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) ancestor_task %s not found ", ancestor_task_id)
			return
		}
		logger.Debug(3, "(GetSourceFromWorkflowInstanceInput) ancestor_task %s not found ", ancestor_task_id)
		ok = false
		return
	}

	if ancestor_task == nil {
		err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) did not find predecessor task %s for task %s", ancestor_task_id, src) // this should not happen, taskReady makes sure everything is available
		return
	}

	if ancestor_task.StepOutput == nil {

		if error_on_missing_task {
			spew.Dump(workflow_instance)
			panic("output not found b)")
			err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) found ancestor_task %s, but not outputs found in StepOutput (%s)", src_path, src_base)
			return
		}
		err = nil
		logger.Debug(3, "(GetSourceFromWorkflowInstanceInput) found ancestor_task %s, but not outputs found in StepOutput (%s)", src_path, src_base)
		ok = false
		return
	}

	logger.Debug(3, "(GetSourceFromWorkflowInstanceInput) len(ancestor_task.StepOutput): %d", len(*ancestor_task.StepOutput))

	var reason string
	obj, ok, reason, err = ancestor_task.GetStepOutput(src_base)
	if err != nil {
		err = fmt.Errorf("(GetSourceFromWorkflowInstanceInput) ancestor_task.GetStepOutput returned: %s", err.Error())
		return
	}

	if !ok {
		logger.Debug(3, "(GetSourceFromWorkflowInstanceInput) step output not found, reason: %s", reason)
		return
	}

	return
}

// this retrieves the input from either the (sub-)workflow input, or from the output of another task in the same (sub-)workflow
// error_on_missing_task: when checking if a task is ready, a missing task is not an error, it just means task is not ready,
//    but when getting data this is actually an error.
func (qm *ServerMgr) getCWLSource(job *Job, workflow_instance *WorkflowInstance, workflow_input_map map[string]cwl.CWLType, src string, error_on_missing_task bool, context *cwl.WorkflowContext) (obj cwl.CWLType, ok bool, reason string, err error) {

	ok = false
	//src = strings.TrimPrefix(src, "#main/")

	logger.Debug(3, "(getCWLSource) searching for %s", src)

	src_array := strings.Split(src, "/")
	if len(src_array) == 2 {
		logger.Debug(3, "(getCWLSource) a workflow input")
		// must be a workflow input, e.g. #main/jobid (workflow, input)
		//src_base := src_array[1]

		obj, ok, err = qm.GetSourceFromWorkflowInstanceInput(workflow_instance, src, error_on_missing_task)

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

	} else if len(src_array) == 3 {
		logger.Debug(3, "(getCWLSource) a step output")
		// must be a step output, e.g. #main/filter/rejected (workflow, step, output)
		workflow_name := src_array[0]
		step_name := src_array[1]
		output_name := src_array[2]

		var step_reason string
		obj, ok, step_reason, err = qm.getCWLSourceFromStepOutput(job, workflow_instance, workflow_name, step_name, output_name, error_on_missing_task)
		if !ok {
			reason = "getCWLSourceFromStepOutput returned: " + step_reason
		}
		return

	} else if len(src_array) >= 4 {
		logger.Debug(3, "(getCWLSource) a step input?")
		panic("(getCWLSource) src: " + src)
		//context := job.WorkflowContext

		var real_obj cwl.CWL_object
		real_obj, ok = context.All[src]

		if !ok {

			fmt.Printf("(getCWLSource) src: %s\n", src)
			src_base := path.Base(src)

			fmt.Printf("(getCWLSource) src_base: %s\n", src_base)
			src_path := strings.TrimSuffix(src, "/"+src_base)

			fmt.Printf("(getCWLSource) src_path: %s\n", src_path)

			// 1) search matching WorkflowInstance
			//var wi *WorkflowInstance

			//wi, ok, err = job.GetWorkflowInstance(src_path, true)
			//if err != nil {
			//	err = fmt.Errorf("(getCWLSource) GetWorkflowInstance returned: %s", err.Error())
			//	return
			//}
			//if ok {
			// found workflow instance
			obj, err = workflow_instance.Outputs.Get(src_base)
			if err != nil {
				if error_on_missing_task {
					err = fmt.Errorf("(getCWLSource) found ancestor_task %s, but output %s not found", src_path, src_base)
					return
				}
				err = nil
				logger.Debug(3, "(getCWLSource) found ancestor_task %s, but output %s not found", src_path, src_base)
				ok = false
				return
			}
			return

			//}

			// 2) find task / task output

			//ancestor_task_id := current_task_id
			ancestor_task_id := Task_Unique_Identifier{}
			ancestor_task_id.JobId = workflow_instance.JobId
			// src_path + src_base
			step_name := src_path
			fmt.Printf("(getCWLSource) step_name: %s\n", step_name)

			//workflow_name := path.Base(step_name)
			//workflow_name = strings.TrimSuffix(workflow_name, "/")
			//fmt.Printf("(getCWLSource) workflow_name: %s\n", workflow_name)

			ancestor_task_id.TaskName = step_name

			var ancestor_task *Task
			var local_ok bool
			ancestor_task, local_ok, err = qm.TaskMap.Get(ancestor_task_id, true)
			if err != nil {
				err = fmt.Errorf("(getCWLSource) qm.TaskMap.Get returned: %s", err.Error())
				return
			}
			if !local_ok {
				if error_on_missing_task {
					err = fmt.Errorf("(getCWLSource) ancestor_task %s not found ", ancestor_task_id)
					return
				}
				logger.Debug(3, "(getCWLSource) ancestor_task %s not found ", ancestor_task_id)
				ok = false
				return
			}

			if ancestor_task == nil {
				err = fmt.Errorf("(getCWLSource) did not find predecessor task %s for task %s", ancestor_task_id, src) // this should not happen, taskReady makes sure everything is available
				return
			}

			if ancestor_task.StepOutput == nil {
				if error_on_missing_task {
					err = fmt.Errorf("(getCWLSource) found ancestor_task %s, but not outputs found (%s)", src_path, src_base)
					return
				}
				err = nil
				logger.Debug(3, "(getCWLSource) found ancestor_task %s, but not outputs found (%s)", src_path, src_base)
				ok = false
				return
			}

			logger.Debug(3, "(getCWLSource) len(ancestor_task.StepOutput): %d", len(*ancestor_task.StepOutput))

			obj, ok, reason, err = ancestor_task.GetStepOutput(src_base)
			if err != nil {
				err = fmt.Errorf("(getCWLSource) ancestor_task.GetStepOutput returned: %s", err.Error())
				return
			}

			if ok {
				return
			}

			logger.Debug(3, "(getCWLSource) step output not found")
			ok = false

			return
		}
		obj = real_obj.(cwl.CWLType)
		// search context for object
		// if onject is not in context, use a flag in context object to indicate collection of objects

		// how find task that would generate that object ???

		//workflow_name := src_array[0]
		//step_name := src_array[1]
		//output_name := src_array[2]
		//_ = output_name

	} else {
		err = fmt.Errorf("(getCWLSource) could not parse source: %s", src) // workflow, step, run

	}

	return
}

func (qm *ServerMgr) GetStepInputObjects(job *Job, workflow_instance *WorkflowInstance, workflow_input_map map[string]cwl.CWLType, workflow_step *cwl.WorkflowStep, context *cwl.WorkflowContext) (workunit_input_map cwl.JobDocMap, ok bool, reason string, err error) {

	workunit_input_map = make(map[string]cwl.CWLType) // also used for json

	fmt.Println("(GetStepInputObjects) workflow_step:")
	spew.Dump(workflow_step)

	if workflow_step.In == nil {
		err = fmt.Errorf("(GetStepInputObjects) workflow_step.In == nil (%s)", workflow_step.Id)
		return
	}

	if len(workflow_step.In) == 0 {
		err = fmt.Errorf("(GetStepInputObjects) len(workflow_step.In) == 0")
		return
	}

	// 1. find all object source and Default
	// 2. make a map copy to be used in javascript, as "inputs"
	// INPUT_LOOP1
	for _, input := range workflow_step.In {
		// input is a WorkflowStepInput

		//fmt.Printf("(GetStepInputObjects) workflow_step.In: (%d)\n", input_i)
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
			//fmt.Println("(GetStepInputObjects) input.Source != nil")
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

							if job_obj_type != cwl.CWL_array {
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
					err = fmt.Errorf("(GetStepInputObjects) (string) getCWLSource returns: %s", err.Error())
					return
				}
				if ok {

					if job_obj.GetType() == cwl.CWL_null {
						//fmt.Println("(GetStepInputObjects) job_obj is cwl.CWL_null")
						ok = false
					} else {
						//fmt.Println("(GetStepInputObjects) job_obj is not cwl.CWL_null")
					}
				}

				if !ok {
					fmt.Println("(GetStepInputObjects) check input.Default")
					if input.Default == nil {
						//logger.Debug(1, "(GetStepInputObjects) (string) getCWLSource did not find output (nor a default) that can be used as input \"%s\"", source_as_string)
						err = fmt.Errorf("(GetStepInputObjects) getCWLSource did not find source %s and has no Default (reason: %s)", source_as_string, reason)
						continue
					}
					job_obj, err = cwl.NewCWLType("", input.Default, context)
					if err != nil {
						err = fmt.Errorf("(GetStepInputObjects) could not use default: %s", err.Error())
						return
					}
					//fmt.Println("(GetStepInputObjects) got a input.Default")
					//spew.Dump(job_obj)
				}
				//workunit_input_map[cmd_id] = job_obj
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
	//fmt.Println("(GetStepInputObjects) workunit_input_map after first round:\n")
	//spew.Dump(workunit_input_map)

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

		fmt.Println("(GetStepInputObjects) workunit_input_map:")
		spew.Dump(workunit_input_map)

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
						value_returned = cwl.NewInt(exported_value.(int), context)
					case float32:
						value_returned = cwl.NewFloat(exported_value.(float32))
					case float64:
						fmt.Println("got a double")
						value_returned = cwl.NewDouble(exported_value.(float64))
					case uint64:
						value_returned = cwl.NewInt(exported_value.(int), context)

					case []interface{}: //Array
						err = fmt.Errorf("(GetStepInputObjects) array not supported yet")
						return
					case interface{}: //Object

						value_returned, err = cwl.NewCWLType("", exported_value, context)
						if err != nil {
							fmt.Println("record:")
							spew.Dump(exported_value)
							err = fmt.Errorf("(GetStepInputObjects) interface{}, NewCWLType returned: %s", err.Error())
							return
						}

					case nil:
						value_returned = cwl.NewNull()
					default:
						err = fmt.Errorf("(GetStepInputObjects) js return type not supoported: (%s)", reflect.TypeOf(exported_value))
						return
					}

					fmt.Println("value_returned:")
					spew.Dump(value_returned)
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

	fmt.Println("(GetStepInputObjects) workunit_input_map after ValueFrom round:")
	spew.Dump(workunit_input_map)

	//for key, value := range workunit_input_map {
	//	fmt.Printf("workunit_input_map: %s -> %s\n", key, value.String())

	//}

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
		id := wu.GetId()
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

func (qm *ServerMgr) taskCompleted_Scatter(task *Task) (err error) {

	var task_str string
	task_str, err = task.String()
	if err != nil {
		err = fmt.Errorf("(taskCompleted_Scatter) task.String returned: %s", err.Error())
		return
	}

	logger.Debug(3, "(taskCompleted_Scatter) %s Scatter_parent exists", task_str)
	scatter_parent_id := *task.Scatter_parent
	var scatter_parent_task *Task
	var ok bool
	scatter_parent_task, ok, err = qm.TaskMap.Get(scatter_parent_id, true)
	if err != nil {
		err = fmt.Errorf("(taskCompleted_Scatter) qm.TaskMap.Get returned: %s", err.Error())
		return
	}
	if !ok {
		err = fmt.Errorf("(taskCompleted_Scatter) Scatter_Parent task %s not found", scatter_parent_id)
		return
	}

	// (taskCompleted_Scatter) get scatter sibblings to see if they are done
	var children []*Task
	children, err = scatter_parent_task.GetChildren(qm)
	if err != nil {

		length, _ := qm.TaskMap.Len()

		var tasks []*Task
		tasks, _ = qm.TaskMap.GetTasks()
		for _, task := range tasks {
			fmt.Printf("(taskCompleted_Scatter) got task %s\n", task.Id)
		}

		err = fmt.Errorf("(taskCompleted_Scatter) (scatter) GetChildren returned: %s (total: %d)", err.Error(), length)
		return
	}

	//fmt.Printf("XXX children: %d\n", len(children))

	scatter_complete := true
	for _, child_task := range children {
		var child_state string
		child_state, err = child_task.GetState()
		if err != nil {
			err = fmt.Errorf("(taskCompleted_Scatter) child_task.GetState returned: %s", err.Error())
			return
		}

		if child_state != TASK_STAT_COMPLETED {
			scatter_complete = false
			break
		}
	}

	if !scatter_complete {
		// nothing to do here, scatter is not complete
		return
	}

	// // (updateJobTask) scatter_complete
	ok, err = scatter_parent_task.Finalize() // make sure this is the last scatter task
	if err != nil {
		err = fmt.Errorf("(taskCompleted_Scatter) scatter_parent_task.Finalize returned: %s", err.Error())
		return
	}

	if !ok {
		// somebody else is finalizing
		return
	}

	scatter_parent_step := scatter_parent_task.WorkflowStep

	scatter_parent_task.StepOutput = &cwl.Job_document{}

	//fmt.Printf("XXX start\n")
	for i, _ := range scatter_parent_step.Out {
		//fmt.Printf("XXX loop %d\n", i)
		workflow_step_output := scatter_parent_step.Out[i]
		workflow_step_output_id := workflow_step_output.Id

		workflow_step_output_id_base := path.Base(workflow_step_output_id)

		output_array := cwl.Array{}

		for _, child_task := range children {
			//fmt.Printf("XXX inner loop %d\n", i)
			job_doc := child_task.StepOutput
			var child_output cwl.CWLType
			child_output, err = job_doc.Get(workflow_step_output_id_base)
			if err != nil {
				//fmt.Printf("XXX job_doc.Get failed\n")
				err = fmt.Errorf("(taskCompleted_Scatter) job_doc.Get failed: %s ", err.Error())
				return
			}
			fmt.Println("child_output:")
			spew.Dump(child_output)
			output_array = append(output_array, child_output)
			fmt.Println("output_array:")
			spew.Dump(output_array)
		}
		fmt.Println("final output_array:")
		spew.Dump(output_array)
		scatter_parent_task.StepOutput = scatter_parent_task.StepOutput.Add(workflow_step_output_id, &output_array)

	}

	fmt.Println("scatter_parent_task.StepOutput:")
	spew.Dump(scatter_parent_task.StepOutput)
	//panic("scatter done")
	//count_a, _ := job.GetRemainTasks()
	//fmt.Printf("GetRemainTasks job A1: %d\n", count_a)

	// set TASK_STAT_COMPLETED
	//err = scatter_parent_task.SetState(TASK_STAT_COMPLETED, true)
	//if err != nil {
	//		return
	//	}

	task = scatter_parent_task

	err = task.SetState(TASK_STAT_COMPLETED, true)
	if err != nil {
		err = fmt.Errorf("(taskCompleted_Scatter) task.SetState returned: %s", err.Error())
		return
	}

	return
}

func (qm *ServerMgr) completeSubworkflow(job *Job, workflow_instance *WorkflowInstance) (ok bool, reason string, err error) {

	var wfl *cwl.Workflow

	// if this is a subworkflow, check if sibblings are complete
	//var context *cwl.WorkflowContext

	var wi_state string
	wi_state, err = workflow_instance.GetState(true)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_instance.GetState returned: %s", err.Error())
		return
	}

	if wi_state == WI_STAT_COMPLETED {
		ok = true
		return
	}

	context := job.WorkflowContext

	workflow_instance_id, _ := workflow_instance.GetId(true)

	// check tasks
	for _, task := range workflow_instance.Tasks {

		task_state, _ := task.GetState()

		if task_state != TASK_STAT_COMPLETED {
			ok = false
			reason = "a task is not completed yet"
			return
		}

	}

	// check subworkflows
	for _, subworkflow := range workflow_instance.Subworkflows {

		var sub_wi *WorkflowInstance
		sub_wi, ok, err = job.GetWorkflowInstance(subworkflow, true)
		if err != nil {
			err = fmt.Errorf("(completeSubworkflow) job.GetWorkflowInstance returned: %s", err.Error())
			return
		}
		if !ok {
			reason = fmt.Sprintf("(completeSubworkflow) subworkflow %s not found", subworkflow)
			return
		}

		sub_wi_state, _ := sub_wi.GetState(true)

		if sub_wi_state != WI_STAT_COMPLETED {
			ok = false
			reason = fmt.Sprintf("(completeSubworkflow) subworkflow %s is not completed", subworkflow)
			return
		}

	}

	// subworkflow complete, now collect outputs !

	//TODO

	wfl, err = workflow_instance.GetWorkflow(context)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_instance.GetWorkflow returned: %s", err.Error())
		return
	}

	// reminder: at this point all tasks in subworkflow are complete, see above

	workflow_inputs := workflow_instance.Inputs
	//workflow_inputs := nil

	workflow_inputs_map := workflow_inputs.GetMap()
	//workflow_inputs_map =

	workflow_outputs_map := make(cwl.JobDocMap)

	// collect sub-workflow outputs, put results in workflow_outputs_map

	for _, output := range wfl.Outputs { // WorkflowOutputParameter http://www.commonwl.org/v1.0/Workflow.html#WorkflowOutputParameter

		output_id := output.Id

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

		var expected_types_raw []interface{}

		switch output.Type.(type) {
		case []interface{}:
			expected_types_raw = output.Type.([]interface{})
		case []cwl.CWLType_Type:

			expected_types_raw_array := output.Type.([]cwl.CWLType_Type)
			for i, _ := range expected_types_raw_array {
				expected_types_raw = append(expected_types_raw, expected_types_raw_array[i])

			}

		default:
			expected_types_raw = append(expected_types_raw, output.Type)
			//expected_types_raw = []interface{output.Type}
		}
		expected_types := []cwl.CWLType_Type{}

		is_optional := false

		var schemata []cwl.CWLType_Type
		schemata, err = job.WorkflowContext.GetSchemata()
		if err != nil {
			err = fmt.Errorf("(completeSubworkflow) job.CWL_collection.GetSchemata returned: %s", err.Error())
			return
		}

		for _, raw_type := range expected_types_raw {
			var type_correct cwl.CWLType_Type
			type_correct, err = cwl.NewCWLType_Type(schemata, raw_type, "WorkflowOutput", context)
			if err != nil {
				spew.Dump(expected_types_raw)
				fmt.Println("---")
				spew.Dump(raw_type)

				err = fmt.Errorf("(completeSubworkflow) could not convert element of output.Type into cwl.CWLType_Type: %s", err.Error())
				//fmt.Printf(err.Error())
				//panic("raw_type problem")
				return
			}
			expected_types = append(expected_types, type_correct)
			if type_correct == cwl.CWL_null {
				is_optional = true
			}
		}

		// search the outputs and stick them in workflow_outputs_map

		output_source := output.OutputSource

		switch output_source.(type) {
		case string:
			outputSourceString := output_source.(string)
			// example: "#preprocess-fastq.workflow.cwl/rejected2fasta/file"

			var obj cwl.CWLType
			//var ok bool
			//var reason string
			obj, ok, reason, err = qm.getCWLSource(job, workflow_instance, workflow_inputs_map, outputSourceString, true, job.WorkflowContext)
			if err != nil {
				err = fmt.Errorf("(completeSubworkflow) A) getCWLSource returns: %s", err.Error())
				return
			}
			skip := false
			if !ok {
				if is_optional {
					skip = true
				} else {
					err = fmt.Errorf("(completeSubworkflow) A) source %s not found by getCWLSource (getCWLSource reason: %s)", outputSourceString, reason)
					return
				}
			}

			if !skip {
				has_type, xerr := cwl.TypeIsCorrect(expected_types, obj, context)
				if xerr != nil {
					err = fmt.Errorf("(completeSubworkflow) TypeIsCorrect: %s", xerr.Error())
					return
				}
				if !has_type {
					err = fmt.Errorf("(completeSubworkflow) A) workflow_ouput %s (type: %s), does not match expected types %s", output_id, reflect.TypeOf(obj), expected_types)
					return
				}

				workflow_outputs_map[output_id] = obj
			}
		case []string:
			outputSourceArrayOfString := output_source.([]string)

			if len(outputSourceArrayOfString) == 0 {
				if !is_optional {
					err = fmt.Errorf("(completeSubworkflow) output_source array (%s) is empty, but a required output", output_id)
					return
				}
			}

			output_array := cwl.Array{}

			for _, outputSourceString := range outputSourceArrayOfString {
				var obj cwl.CWLType
				//var ok bool
				obj, ok, _, err = qm.getCWLSource(job, workflow_instance, workflow_inputs_map, outputSourceString, true, job.WorkflowContext)
				if err != nil {
					err = fmt.Errorf("(completeSubworkflow) B) (%s) getCWLSource returns: %s", workflow_instance_id, err.Error())
					return
				}

				skip := false
				if !ok {

					if is_optional {
						skip = true
					} else {

						err = fmt.Errorf("(completeSubworkflow) B) (%s) source %s not found", workflow_instance_id, outputSourceString)
						return
					}
				}

				if !skip {
					has_type, xerr := cwl.TypeIsCorrect(expected_types, obj, context)
					if xerr != nil {
						err = fmt.Errorf("(completeSubworkflow) TypeIsCorrect: %s", xerr.Error())
						return
					}
					if !has_type {
						err = fmt.Errorf("(completeSubworkflow) B) workflow_ouput %s, does not match expected types %s", output_id, expected_types)
						return
					}
					fmt.Println("obj:")
					spew.Dump(obj)
					output_array = append(output_array, obj)
				}
			}

			if len(output_array) > 0 {
				workflow_outputs_map[output_id] = &output_array
			} else {
				if !is_optional {
					err = fmt.Errorf("(completeSubworkflow) array with output_id %s is empty, but a required output", output_id)
					return
				}
			}
			fmt.Println("workflow_outputs_map:")
			spew.Dump(workflow_outputs_map)

		default:
			err = fmt.Errorf("(completeSubworkflow) output.OutputSource has to be string or []string, but I got type %s", spew.Sdump(output_source))
			return

		}

	}

	var workflow_outputs_array cwl.Job_document
	workflow_outputs_array, err = workflow_outputs_map.GetArray()
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_outputs_map.GetArray returned: %s", err.Error())
		return
	}
	workflow_instance.Outputs = workflow_outputs_array

	err = workflow_instance.SetState(WI_STAT_COMPLETED, "db_sync_yes", true)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) workflow_instance.SetState returned: %s", err.Error())
		return
	}

	logger.Debug(3, "(completeSubworkflow) job.WorkflowInstancesRemain: (before) %d", job.WorkflowInstancesRemain)
	err = job.IncrementWorkflowInstancesRemain(-1, "db_sync_yes", true)
	if err != nil {
		err = fmt.Errorf("(completeSubworkflow) job.IncrementWorkflowInstancesRemain returned: %s", err.Error())
		return
	}

	// ***** notify parent workflow (this wokrflow might have been the last step)

	workflow_instance_local_id := workflow_instance.LocalId
	fmt.Printf("(completeSubworkflow) completes with workflow_instance_local_id: %s\n", workflow_instance_local_id)

	if workflow_instance_local_id == "_root" {
		// notify job

		// this was the main workflow, all done!

		err = job.SetState(JOB_STAT_COMPLETED, nil)
		if err != nil {
			err = fmt.Errorf("(completeSubworkflow) job.SetState returned: %s", err.Error())
			return
		}

	} else {
		// notify parent
		var parent_id string
		parent_id = path.Dir(workflow_instance_local_id)

		fmt.Printf("(completeSubworkflow) completes with parent_id: %s\n", parent_id)

		var parent *WorkflowInstance

		parent, ok, err = job.GetWorkflowInstance(parent_id, true)

		if err != nil {
			err = fmt.Errorf("(completeSubworkflow) job.GetWorkflowInstance returned: %s", err.Error())
			return
		}

		if !ok {
			err = fmt.Errorf("(completeSubworkflow) parent workflow instance %s not found", parent_id)
			return

		}

		fmt.Printf("(completeSubworkflow) got parent\n")

		//var parent_ok bool
		var parent_result string
		_, parent_result, err = qm.completeSubworkflow(job, parent)
		if err != nil {
			err = fmt.Errorf("(completeSubworkflow) recursive call of qm.completeSubworkflow returned: %s", err.Error())
			return
		}
		//if parent_id == "_root" {
		fmt.Printf("(completeSubworkflow) parent_result: %s", parent_result)
		//	panic("done")
		//}

	}

	return
}

// update job info when a task in that job changed to a new state
// update parent task of a scatter task
// this is invoked everytime a task changes state, but only when job_remainTasks==0 and state is not yet JOB_STAT_COMPLETED it will complete the job
func (qm *ServerMgr) taskCompleted(task *Task) (err error) {

	// TODO I am not sure why this function is invoked for every state change....

	var task_state string
	task_state, err = task.GetState()
	if err != nil {
		err = fmt.Errorf("(taskCompleted) task.GetState() returned: %s", err.Error())
		return
	}

	if task_state != TASK_STAT_COMPLETED {
		return
	}

	jobid, err := task.GetJobId()
	if err != nil {
		err = fmt.Errorf("(taskCompleted) task.GetJobId() returned: %s", err.Error())
		return
	}
	var job *Job
	job, err = GetJob(jobid)
	if err != nil {
		err = fmt.Errorf("(taskCompleted) GetJob returned: %s", err.Error())
		return
	}

	var job_remainTasks int
	job_remainTasks, err = job.GetRemainTasks() // TODO deprecated !?
	if err != nil {
		err = fmt.Errorf("(taskCompleted) job.GetRemainTasks() returned: %s", err.Error())
		return
	}

	var task_str string
	task_str, err = task.String()
	if err != nil {
		err = fmt.Errorf("(taskCompleted) task.String returned: %s", err.Error())
		return
	}

	logger.Debug(2, "(taskCompleted) remaining tasks for job %s: %d (task_state: %s)", task_str, job_remainTasks, task_state)

	// check if this was the last task in a subworkflow

	// check if this task has a parent

	if task.WorkflowStep == nil {
		logger.Debug(3, "(taskCompleted) task.WorkflowStep == nil ")
	} else {
		logger.Debug(3, "(taskCompleted) task.WorkflowStep != nil ")
	}

	var parent_task_to_complete *Task
	parent_task_to_complete = nil
	// (taskCompleted) CWL Task completes
	if task.WorkflowStep != nil {
		// this task belongs to a subworkflow // TODO every task should belong to a subworkflow
		logger.Debug(3, "(taskCompleted) task.WorkflowStep != nil (%s)", task_str)

		if task.Scatter_parent != nil {
			// process scatter children
			err = qm.taskCompleted_Scatter(task)
			if err != nil {
				err = fmt.Errorf("(taskCompleted) taskCompleted_Scatter returned: %s", err.Error())
				return
			}

		} else { // end task.Scatter_parent != nil
			logger.Debug(3, "(taskCompleted) %s  No Scatter_parent", task_str)
		}

		//count, _ := job.GetRemainTasks()

		//fmt.Printf("GetRemainTasks job B: %d\n", count)

		workflow_instance_id := task.WorkflowInstanceId

		var workflow_instance *WorkflowInstance
		var ok bool
		workflow_instance, ok, err = job.GetWorkflowInstance(workflow_instance_id, true)
		if err != nil {
			err = fmt.Errorf("(taskCompleted) job.GetWorkflowInstance returned: %s", err.Error())
			return
		}

		if !ok {
			err = fmt.Errorf("(taskCompleted) WorkflowInstance %s not found", err.Error())
			return
		}

		var subworkflow_remain_tasks int
		subworkflow_remain_tasks, err = workflow_instance.DecreaseRemainSteps()
		if err != nil {
			err = fmt.Errorf("(taskCompleted) workflow_instance.DecreaseRemainSteps returned: %s", err.Error())
			return

		}
		// var subworkflow_remain_tasks int

		// subworkflow_remain_tasks, err = job.Decrease_WorkflowInstance_RemainTasks(parent_id_str, task_str)
		// if err != nil {
		// 	//fmt.Printf("ERROR: (taskCompleted) Decrease_WorkflowInstance_RemainTasks returned: %s\n", err.Error())
		// 	err = fmt.Errorf("(taskCompleted) WorkflowInstanceDecreaseRemainTasks returned: %s", err.Error())
		// 	return
		// }
		//fmt.Printf("GetRemainTasks subworkflow C: %d\n", subworkflow_remain_tasks)

		logger.Debug(3, "(taskCompleted) TASK_STAT_COMPLETED  / remaining tasks for subworkflow %s: %d", task_str, subworkflow_remain_tasks)

		logger.Debug(3, "(taskCompleted) workflow_instance %s remaining tasks: %d (total %d or %d)", workflow_instance_id, subworkflow_remain_tasks, workflow_instance.TaskCount(), workflow_instance.TotalTasks)

		if subworkflow_remain_tasks > 0 {
			return
		}

		// subworkflow completed.
		var reason string
		ok, reason, err = qm.completeSubworkflow(job, workflow_instance)
		if err != nil {
			err = fmt.Errorf("(taskCompleted) completeSubworkflow returned: %s", err.Error())
			return
		}
		if !ok {
			err = fmt.Errorf("(taskCompleted) completeSubworkflow not ok, reason: %s", reason)
			return
		}

	} // end of: if task_state == TASK_STAT_COMPLETED && task.WorkflowInstanceId != ""

	if job.IsCWL {

		logger.Debug(3, "(taskCompleted) job.WorkflowInstancesRemain: %d", job.WorkflowInstancesRemain)

		if job.WorkflowInstancesRemain > 0 {
			return
		}
	} else {
		logger.Debug(3, "(taskCompleted) job_remainTasks: %d", job_remainTasks)
		if job_remainTasks > 0 { //#####################################################################
			return
		}
	}

	err = qm.finalizeJob(job, jobid)

	if err != nil {
		err = fmt.Errorf("(taskCompleted) qm.finalizeJob returned: %s", err.Error())
		return
	}

	// this is a recursive call !
	if parent_task_to_complete != nil {

		if parent_task_to_complete.State != TASK_STAT_COMPLETED {
			err = fmt.Errorf("(taskCompleted) qm.taskCompleted(parent_task_to_complete) parent_task_to_complete.State != TASK_STAT_COMPLETED")
			return
		}
		err = qm.taskCompleted(parent_task_to_complete)
		if err != nil {
			err = fmt.Errorf("(taskCompleted) qm.taskCompleted(parent_task_to_complete) returned: %s", err.Error())
			return
		}
	}

	return
}

func (qm *ServerMgr) finalizeJob(job *Job, jobid string) (err error) {

	job_state, err := job.GetState(true)
	if err != nil {
		err = fmt.Errorf("(updateJobTask) job.GetState returned: %s", err.Error())
		return
	}
	if job_state == JOB_STAT_COMPLETED {
		err = fmt.Errorf("(updateJobTask) job state is already JOB_STAT_COMPLETED")
		return
	}

	err = job.SetState(JOB_STAT_COMPLETED, nil)
	if err != nil {
		err = fmt.Errorf("(updateJobTask) job.SetState returned: %s", err.Error())
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
	logger.Event(event.JOB_DONE, "jobid="+job.Id+";name="+job.Info.Name+";project="+job.Info.Project+";user="+job.Info.User)

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
			err := task.SetState(TASK_STAT_INPROGRESS, true)
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
			err = task.SetState(new_task_state, true)
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
	rights := job.Acl.Check(u.Uuid)
	if job.Acl.Owner != u.Uuid && rights["delete"] == false && u.Admin == false {
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
		err = errors.New("(ResumeSuspendedJobByUser) failed to load job " + err.Error())
		return
	}

	job_state, err := dbjob.GetState(true)
	if err != nil {
		err = errors.New("(ResumeSuspendedJobByUser) failed to get job state " + err.Error())
		return
	}

	// User must have write permissions on job or be job owner or be an admin
	rights := dbjob.Acl.Check(u.Uuid)
	if dbjob.Acl.Owner != u.Uuid && rights["write"] == false && u.Admin == false {
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
	err = qm.EnqueueTasksByJobId(id)
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
		id = job.Id
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
		err = qm.EnqueueTasksByJobId(id)
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
		logger.Debug(1, "recovering %d: job=%s, state=%s, pipeline=%s", recovered+1, dbjob.Id, dbjob.State, pipeline)
		isRecovered, rerr := qm.RecoverJob("", dbjob)
		if rerr != nil {
			logger.Error(fmt.Sprintf("(RecoverJobs) job=%s failed: %s", dbjob.Id, rerr.Error()))
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
	remaintasks := 0
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
			remaintasks += 1
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
			remaintasks += 1
		}
	}

	err = dbjob.IncrementResumed(1)
	if err != nil {
		err = errors.New("(RecomputeJob) failed to incremenet job resumed " + err.Error())
		return
	}
	err = dbjob.SetRemainTasks(remaintasks)
	if err != nil {
		err = errors.New("(RecomputeJob) failed to set job remain_tasks " + err.Error())
		return
	}

	err = dbjob.SetState(JOB_STAT_QUEUING, nil)
	if err != nil {
		err = fmt.Errorf("(RecomputeJob) UpdateJobState: %s", err.Error())
		return
	}
	err = qm.EnqueueTasksByJobId(jobid)
	if err != nil {
		err = errors.New("(RecomputeJob) failed to enqueue job " + err.Error())
		return
	}
	logger.Debug(1, "Recomputed job %s from task %s", jobid, task_stage)
	return
}

//recompute job from beginning
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

	remaintasks := 0
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
		remaintasks += 1
	}

	err = job.IncrementResumed(1)
	if err != nil {
		err = errors.New("(ResubmitJob) failed to incremenet job resumed " + err.Error())
		return
	}
	err = job.SetRemainTasks(remaintasks)
	if err != nil {
		err = errors.New("(ResubmitJob) failed to set job remain_tasks " + err.Error())
		return
	}

	err = job.SetState(JOB_STAT_QUEUING, nil)
	if err != nil {
		err = fmt.Errorf("(ResubmitJob) UpdateJobState: %s", err.Error())
		return
	}
	err = qm.EnqueueTasksByJobId(jobid)
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
	//job_id := job.Id
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

	is_suspended, err := client.Get_Suspended(true)
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
		var task_str string
		task_str, err = task.String()
		if err != nil {
			err = fmt.Errorf("(FetchPrivateEnv) task.String returned: %s", err.Error())
			return
		}
		err = fmt.Errorf("(FetchPrivateEnv) task %s not found in qm.TaskMap", task_str)
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
