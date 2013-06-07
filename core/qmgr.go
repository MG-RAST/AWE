package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	e "github.com/MG-RAST/AWE/errors"
	. "github.com/MG-RAST/AWE/logger"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"
)

type QueueMgr struct {
	clientMap map[string]*Client
	taskMap   map[string]*Task
	actJobs   map[string]*JobPerf
	susJobs   map[string]bool
	workQueue *WQueue
	reminder  chan bool
	jsReq     chan bool   //channel for job submission request (JobController -> qmgr.Handler)
	jsAck     chan string //channel for return an assigned job number in response to jsReq  (qmgr.Handler -> JobController)
	taskIn    chan *Task  //channel for receiving Task (JobController -> qmgr.Handler)
	coReq     chan CoReq  //workunit checkout request (WorkController -> qmgr.Handler)
	coAck     chan CoAck  //workunit checkout item including data and err (qmgr.Handler -> WorkController)
	feedback  chan Notice //workunit execution feedback (WorkController -> qmgr.Handler)
	coSem     chan int    //semaphore for checkout (mutual exclusion between different clients)
	nextJid   string      //next jid that will be assigned to newly submitted job
}

func NewQueueMgr() *QueueMgr {
	return &QueueMgr{
		clientMap: map[string]*Client{},
		taskMap:   map[string]*Task{},
		workQueue: NewWQueue(),
		reminder:  make(chan bool),
		jsReq:     make(chan bool),
		jsAck:     make(chan string),
		taskIn:    make(chan *Task, 1024),
		coReq:     make(chan CoReq),
		coAck:     make(chan CoAck),
		feedback:  make(chan Notice),
		coSem:     make(chan int, 1), //non-blocking buffered channel
		actJobs:   map[string]*JobPerf{},
		susJobs:   map[string]bool{},
		nextJid:   "",
	}
}

type CoReq struct {
	policy     string
	fromclient string
	count      int
}

type CoAck struct {
	workunits []*Workunit
	err       error
}

type Notice struct {
	WorkId   string
	Status   string
	ClientId string
}

type coInfo struct {
	workunit *Workunit
	clientid string
}

type WQueue struct {
	workMap  map[string]*Workunit //all parsed workunits
	wait     map[string]bool      //ids of waiting workunits
	checkout map[string]bool      //ids of workunits being checked out
	suspend  map[string]bool      //ids of suspended workunits
}

func NewWQueue() *WQueue {
	return &WQueue{
		workMap:  map[string]*Workunit{},
		wait:     map[string]bool{},
		checkout: map[string]bool{},
		suspend:  map[string]bool{},
	}
}

//WQueue functions
func (wq WQueue) Len() int {
	return len(wq.workMap)
}

func (wq *WQueue) Add(workunit *Workunit) (err error) {
	if workunit.Id == "" {
		return errors.New("try to push a workunit with an empty id")
	}
	wq.workMap[workunit.Id] = workunit
	wq.wait[workunit.Id] = true
	return nil
}

func (wq *WQueue) Delete(id string) {
	delete(wq.wait, id)
	delete(wq.checkout, id)
	delete(wq.suspend, id)
	delete(wq.workMap, id)
}

func (wq *WQueue) Has(id string) (has bool) {
	if _, ok := wq.workMap[id]; ok {
		has = true
	} else {
		has = false
	}
	return
}

func (wq *WQueue) StatusChange(id string, new_status string) (err error) {
	if _, ok := wq.workMap[id]; !ok {
		return errors.New("WQueue.statusChange: invalid workunit id:" + id)
	}
	//move workunit id between maps. no need to care about the old status because
	//delete function will do nothing if the operated map has no such key.
	if new_status == WORK_STAT_CHECKOUT {
		wq.checkout[id] = true
		delete(wq.wait, id)
		delete(wq.suspend, id)
	} else if new_status == WORK_STAT_QUEUED {
		wq.wait[id] = true
		delete(wq.checkout, id)
		delete(wq.suspend, id)
	} else if new_status == WORK_STAT_SUSPEND {
		wq.suspend[id] = true
		delete(wq.checkout, id)
		delete(wq.wait, id)
	} else {
		return errors.New("WQueue.statusChange: invalid new status:" + new_status)
	}
	wq.workMap[id].State = new_status
	return
}

//select workunits, return a slice of ids based on given queuing policy and requested count
func (wq *WQueue) selectWorkunits(workid []string, policy string, count int) (selected []*Workunit, err error) {
	worklist := []*Workunit{}
	for _, id := range workid {
		worklist = append(worklist, wq.workMap[id])
	}
	//selected = []*Workunit{}
	if policy == "FCFS" {
		sort.Sort(byFCFS{worklist})
	}
	for i := 0; i < count; i++ {
		selected = append(selected, worklist[i])
	}
	return
}

//queuing/prioritizing related functions
type WorkList []*Workunit

func (wl WorkList) Len() int      { return len(wl) }
func (wl WorkList) Swap(i, j int) { wl[i], wl[j] = wl[j], wl[i] }

type byFCFS struct{ WorkList }

func (s byFCFS) Less(i, j int) bool {
	return s.WorkList[i].Info.SubmitTime.Before(s.WorkList[j].Info.SubmitTime)
}

//end of WQueue functions

//start of QueueMgr functions
//to-do: consider separate some independent tasks into another goroutine to handle
func (qm *QueueMgr) Handle() {
	for {
		select {
		case <-qm.jsReq:
			jid := qm.getNextJid()
			qm.jsAck <- jid
			Log.Debug(2, fmt.Sprintf("qmgr:receive a job submission request, assigned jid=%s\n", jid))

		case task := <-qm.taskIn:
			Log.Debug(2, fmt.Sprintf("qmgr:task recived from chan taskIn, id=%s\n", task.Id))
			qm.addTask(task)

		case coReq := <-qm.coReq:
			Log.Debug(2, fmt.Sprintf("qmgr: workunit checkout request received, Req=%v\n", coReq))
			works, err := qm.popWorks(coReq)
			ack := CoAck{workunits: works, err: err}
			qm.coAck <- ack

		case notice := <-qm.feedback:
			Log.Debug(2, fmt.Sprintf("qmgr: workunit feedback received, workid=%s, status=%s, clientid=%s\n", notice.WorkId, notice.Status, notice.ClientId))
			if err := qm.handleWorkStatusChange(notice); err != nil {
				Log.Error("handleWorkStatusChange(): " + err.Error())
			}

		case <-qm.reminder:
			Log.Debug(2, "time to update workunit queue....\n")
			qm.updateQueue()
			if conf.DEV_MODE {
				fmt.Println(qm.ShowStatus())
			}
		}
	}
}

func (qm *QueueMgr) Timer() {
	for {
		time.Sleep(10 * time.Second)
		qm.reminder <- true
	}
}

func (qm *QueueMgr) ClientChecker() {
	for {
		time.Sleep(30 * time.Second)
		for clientid, client := range qm.clientMap {
			if client.Tag == true {
				client.Tag = false
				total_minutes := int(time.Now().Sub(client.RegTime).Minutes())
				hours := total_minutes / 60
				minutes := total_minutes % 60
				client.Serve_time = fmt.Sprintf("%dh%dm", hours, minutes)
			} else {
				//now client must be gone as tag set to false 30 seconds ago and no heartbeat received thereafter
				Log.Event(EVENT_CLIENT_UNREGISTER, "clientid="+clientid+";name="+qm.clientMap[clientid].Name)

				//requeue unfinished workunits associated with the failed client
				workids := qm.getWorkByClient(clientid)
				for _, workid := range workids {
					if qm.workQueue.Has(workid) {
						qm.workQueue.StatusChange(workid, WORK_STAT_QUEUED)
						Log.Event(EVENT_WORK_REQUEUE, "workid="+workid)
					}
				}
				//delete the client from client map
				delete(qm.clientMap, clientid)
			}
		}
	}
}

func (qm *QueueMgr) InitMaxJid() (err error) {
	jidfile := conf.DATA_PATH + "/maxjid"
	if _, err := os.Stat(jidfile); err != nil {
		f, err := os.Create(jidfile)
		if err != nil {
			return err
		}
		f.WriteString("10000")
		qm.nextJid = "10001"
		f.Close()
	} else {
		buf, err := ioutil.ReadFile(jidfile)
		if err != nil {
			return err
		}
		bufstr := strings.TrimSpace(string(buf))

		maxjid, err := strconv.Atoi(bufstr)
		if err != nil {
			return err
		}

		qm.nextJid = strconv.Itoa(maxjid + 1)
	}
	Log.Debug(2, fmt.Sprintf("qmgr:jid initialized, next jid=%s\n", qm.nextJid))
	return
}

func (qm *QueueMgr) AddTasks(jobid string, tasks []*Task) (err error) {
	for _, task := range tasks {
		qm.taskIn <- task
	}
	qm.CreateJobPerf(jobid)
	return
}

func (qm *QueueMgr) CheckoutWorkunits(req_policy string, client_id string, num int) (workunits []*Workunit, err error) {
	//precheck if the client is registered
	if _, hasClient := qm.clientMap[client_id]; !hasClient {
		return nil, errors.New(e.ClientNotFound)
	}
	if qm.clientMap[client_id].Status == CLIENT_STAT_SUSPEND {
		return nil, errors.New(e.ClientSuspended)
	}

	//lock semephore, at one time only one client's checkout request can be served
	qm.coSem <- 1

	req := CoReq{policy: req_policy, fromclient: client_id, count: num}
	qm.coReq <- req
	ack := <-qm.coAck

	if ack.err == nil {
		for _, work := range ack.workunits {
			qm.clientMap[client_id].Total_checkout += 1
			qm.clientMap[client_id].Current_work[work.Id] = true
		}
	}

	//unlock
	<-qm.coSem

	return ack.workunits, ack.err
}

func (qm *QueueMgr) ShowWorkunits() (workunits []*Workunit) {
	for _, work := range qm.workQueue.workMap {
		workunits = append(workunits, work)
	}
	return workunits
}

func (qm *QueueMgr) JobRegister() (jid string, err error) {
	qm.jsReq <- true
	jid = <-qm.jsAck

	if jid == "" {
		return "", errors.New("failed to assign a job number for the newly submitted job")
	}
	return jid, nil
}

func (qm *QueueMgr) GetWorkById(id string) (workunit *Workunit, err error) {
	if workunit, hasid := qm.workQueue.workMap[id]; hasid {
		return workunit, nil
	}
	return nil, errors.New(fmt.Sprintf("no workunit found with id %s", id))
}

func (qm *QueueMgr) NotifyWorkStatus(notice Notice) {
	qm.feedback <- notice
	return
}

func (qm *QueueMgr) DeleteJob(jobid string) (err error) {
	job, err := LoadJob(jobid)
	if err != nil {
		return
	}
	if err := job.UpdateState(JOB_STAT_DELETED); err != nil {
		return err
	}
	//delete queueing workunits
	for workid, _ := range qm.workQueue.workMap {
		if jobid == strings.Split(workid, "_")[0] {
			qm.workQueue.Delete(workid)
		}
	}
	//delete parsed tasks
	for i := 0; i < len(job.TaskList()); i++ {
		task_id := fmt.Sprintf("%s_%d", jobid, i)
		delete(qm.taskMap, task_id)
	}
	qm.DeleteJobPerf(jobid)
	delete(qm.susJobs, jobid)

	return
}

func (qm *QueueMgr) DeleteSuspendedJobs() (num int) {
	suspendjobs := qm.GetSuspendJobs()
	for id, _ := range suspendjobs {
		if err := qm.DeleteJob(id); err == nil {
			num += 1
		}
	}
	return
}

func (qm *QueueMgr) SuspendJob(jobid string) (err error) {
	job, err := LoadJob(jobid)
	if err != nil {
		return
	}
	if err := job.UpdateState(JOB_STAT_SUSPEND); err != nil {
		return err
	}
	qm.DeleteJobPerf(jobid)
	qm.susJobs[jobid] = true

	//suspend queueing workunits
	for workid, _ := range qm.workQueue.workMap {
		if jobid == strings.Split(workid, "_")[0] {
			qm.workQueue.StatusChange(workid, WORK_STAT_SUSPEND)
		}
	}
	//suspend parsed tasks
	for i := 0; i < len(job.TaskList()); i++ {
		task_id := fmt.Sprintf("%s_%d", jobid, i)
		if _, ok := qm.taskMap[task_id]; ok {
			qm.taskMap[task_id].State = TASK_STAT_SUSPEND
		}
	}
	qm.DeleteJobPerf(jobid)

	Log.Event(EVENT_JOB_SUSPEND, "jobid="+jobid)

	return
}

//add task to taskMap
func (qm *QueueMgr) addTask(task *Task) (err error) {
	id := task.Id
	if task.State == TASK_STAT_COMPLETED { //for job recovery from db
		qm.taskMap[id] = task
		return
	}
	if task.Skip == 1 && task.Skippable() {
		task.State = TASK_STAT_SKIPPED
		qm.taskMap[id] = task
		qm.taskEnQueue(task)
		return
	}

	task.State = TASK_STAT_PENDING
	qm.taskMap[id] = task
	if len(task.DependsOn) == 0 {
		qm.taskEnQueue(task)
	}
	return
}

//delete task from taskMap
func (qm *QueueMgr) deleteTasks(tasks []*Task) (err error) {
	return
}

//poll ready tasks and push into workQueue
func (qm *QueueMgr) updateQueue() (err error) {

	for _, task := range qm.taskMap {
		ready := false
		if task.State == TASK_STAT_PENDING {
			ready = true
			for _, predecessor := range task.DependsOn {
				if _, haskey := qm.taskMap[predecessor]; haskey {
					if qm.taskMap[predecessor].State != TASK_STAT_COMPLETED &&
						qm.taskMap[predecessor].State != TASK_STAT_SKIPPED &&
						qm.taskMap[predecessor].State != TASK_STAT_FAIL_SKIP {
						ready = false
					}
				}
			}
		}
		if ready {
			if err := qm.taskEnQueue(task); err != nil {
				continue
			}
		}
	}
	return
}

func (qm *QueueMgr) taskEnQueue(task *Task) (err error) {

	if task.Skip == 1 && task.Skippable() { // user wants to skip this task, checking if task is skippable
		task.RemainWork = 0
		// Not sure if this is needed
		qm.CreateTaskPerf(task.Id)
		qm.FinalizeTaskPerf(task.Id)
		//
		Log.Event(EVENT_TASK_SKIPPED, "taskid="+task.Id)
		//update job and queue info. Skipped task behaves as finished tasks
		if err = qm.updateJob(task); err != nil {
			Log.Error("qmgr.taskEnQueue updateJob: " + err.Error())
			return
		}
		qm.updateQueue()
		return
	} // if not, we proceed normally

	//fmt.Printf("move workunits of task %s to workunit queue\n", task.Id)
	if err := qm.locateInputs(task); err != nil {
		Log.Error("qmgr.taskEnQueue locateInputs:" + err.Error())
		return err
	}

	//init partition
	if err := task.InitPartIndex(); err != nil {
		Log.Error("qmgr.taskEnQueue InitPartitionIndex:" + err.Error())
		return err
	}

	if err := qm.createOutputNode(task); err != nil {
		Log.Error("qmgr.taskEnQueue createOutputNode:" + err.Error())
		return err
	}
	if err := qm.parseTask(task); err != nil {
		Log.Error("qmgr.taskEnQueue parseTask:" + err.Error())
		return err
	}
	task.State = TASK_STAT_QUEUED

	//log event about task enqueue (TQ)
	Log.Event(EVENT_TASK_ENQUEUE, fmt.Sprintf("taskid=%s;totalwork=%d", task.Id, task.TotalWork))
	qm.CreateTaskPerf(task.Id)
	return
}

func (qm *QueueMgr) locateInputs(task *Task) (err error) {
	jobid := strings.Split(task.Id, "_")[0]
	for name, io := range task.Inputs {
		if io.Node == "-" {
			preId := fmt.Sprintf("%s_%s", jobid, io.Origin)
			if preTask, ok := qm.taskMap[preId]; ok {
				if preTask.State == TASK_STAT_SKIPPED ||
					preTask.State == TASK_STAT_FAIL_SKIP {
					// For now we know that skipped tasks have
					// just one input and one output. So we know
					// that we just need to change one file (this
					// may change in the future)
					locateSkippedInput(qm, preTask, io)
				} else {
					outputs := preTask.Outputs
					if outio, ok := outputs[name]; ok {
						io.Node = outio.Node
					}
				}
			}
		}
		if io.Node == "-" {
			return errors.New(fmt.Sprintf("error in locate input for task %s, %s", task.Id, name))
		}
		io.GetFileSize()
	}
	return
}

func (qm *QueueMgr) parseTask(task *Task) (err error) {
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

func (qm *QueueMgr) createOutputNode(task *Task) (err error) {
	outputs := task.Outputs
	for _, io := range outputs {
		nodeid, err := PostNode(io, task.TotalWork)
		if err != nil {
			return err
		}
		io.Node = nodeid
		//fmt.Printf("%s, output Shock node created, id=%s\n", name, io.Node)
	}
	return
}

func (qm *QueueMgr) popWorks(req CoReq) (works []*Workunit, err error) {
	filtered := qm.filterWorkByClient(req.fromclient)
	if len(filtered) == 0 {
		return nil, errors.New(e.NoEligibleWorkunitFound)
	}
	works, err = qm.workQueue.selectWorkunits(filtered, req.policy, req.count)
	if err == nil { //get workunits successfully, put them into coWorkMap
		for _, work := range works {
			qm.workQueue.StatusChange(work.Id, WORK_STAT_CHECKOUT)
		}
	}
	return
}

//handle feedback from a client about the execution of a workunit
func (qm *QueueMgr) handleWorkStatusChange(notice Notice) (err error) {
	workid := notice.WorkId
	status := notice.Status
	clientid := notice.ClientId
	parts := strings.Split(workid, "_")
	taskid := fmt.Sprintf("%s_%s", parts[0], parts[1])
	jobid := parts[0]
	rank, err := strconv.Atoi(parts[2])
	if err != nil {
		return errors.New(fmt.Sprintf("invalid workid %s", workid))
	}
	if _, ok := qm.clientMap[clientid]; ok {
		delete(qm.clientMap[clientid].Current_work, workid)
	}
	if task, ok := qm.taskMap[taskid]; ok {
		if task.State == TASK_STAT_FAIL_SKIP {
			// A work unit for this task failed before this one arrived.
			// User set Skip=2 so the task was just skipped. Any subsiquent
			// workunits are just deleted...
			qm.workQueue.Delete(workid)
		} else {

			qm.updateTaskWorkStatus(taskid, rank, status)
			if status == WORK_STAT_DONE {
				//log event about work done (WD)
				Log.Event(EVENT_WORK_DONE, "workid="+workid+";clientid="+clientid)

				//update client status
				if client, ok := qm.clientMap[clientid]; ok {
					client.Total_completed += 1
					client.Last_failed = 0 //reset last consecutive failures
				} else {
					//it happens when feedback is sent after server restarted and before client re-registered
				}
				task.RemainWork -= 1
				if task.RemainWork == 0 {
					task.State = TASK_STAT_COMPLETED
					for _, output := range task.Outputs {
						output.GetFileSize()
					}
					//log event about task done (TD)
					qm.FinalizeTaskPerf(taskid)
					Log.Event(EVENT_TASK_DONE, "taskid="+taskid)
					//update the info of the job which the task is belong to, could result in deletion of the
					//task in the task map when the task is the final task of the job to be done.
					if err = qm.updateJob(task); err != nil {
						return
					}
					qm.updateQueue()
				}
				//done, remove from the workQueue
				qm.workQueue.Delete(workid)
			} else if status == WORK_STAT_FAIL { //workunit failed, requeue or put it to suspend list

				Log.Event(EVENT_WORK_FAIL, "workid="+workid+";clientid="+clientid)
				if qm.workQueue.Has(workid) {
					qm.workQueue.workMap[workid].Failed += 1
					if task.Skip == 2 && task.Skippable() { // user wants to skip task
						task.RemainWork = 0 // not doing anything else...
						task.State = TASK_STAT_FAIL_SKIP
						for _, output := range task.Outputs {
							output.GetFileSize()
						}
						qm.FinalizeTaskPerf(taskid)
						// log event about task skipped
						Log.Event(EVENT_TASK_SKIPPED, "taskid="+taskid)
						//update the info of the job which the task is belong to, could result in deletion of the
						//task in the task map when the task is the final task of the job to be done.
						if err = qm.updateJob(task); err != nil {
							return
						}
						qm.updateQueue()
					} else if qm.workQueue.workMap[workid].Failed < conf.MAX_WORK_FAILURE {
						qm.workQueue.StatusChange(workid, WORK_STAT_QUEUED)
						Log.Event(EVENT_WORK_REQUEUE, "workid="+workid)
					} else { //failure time exceeds limit, suspend workunit, task, job
						qm.workQueue.StatusChange(workid, WORK_STAT_SUSPEND)
						Log.Event(EVENT_WORK_SUSPEND, "workid="+workid)
						qm.updateTaskWorkStatus(taskid, rank, WORK_STAT_SUSPEND)
						qm.taskMap[taskid].State = TASK_STAT_SUSPEND
						if err := qm.SuspendJob(jobid); err != nil {
							Log.Error("error returned by SuspendJOb()" + err.Error())
						}
					}
				}
				if client, ok := qm.clientMap[clientid]; ok {
					client.Skip_work = append(client.Skip_work, workid)
					client.Total_failed += 1
					client.Last_failed += 1 //last consecutive failures
					if client.Last_failed == conf.MAX_CLIENT_FAILURE {
						client.Status = CLIENT_STAT_SUSPEND
					}
				}
			}
		}
	} else { //task not existed, possible when job is deleted before the workunit done
		qm.workQueue.Delete(workid)
	}
	return
}

func (qm *QueueMgr) updateTaskWorkStatus(taskid string, rank int, newstatus string) {
	if _, ok := qm.taskMap[taskid]; !ok {
		return
	}
	if rank == 0 {
		qm.taskMap[taskid].WorkStatus[rank] = newstatus
	} else {
		qm.taskMap[taskid].WorkStatus[rank-1] = newstatus
	}
	return
}

func (qm *QueueMgr) getNextJid() (jid string) {
	jid = qm.nextJid
	jidfile := conf.DATA_PATH + "/maxjid"
	ioutil.WriteFile(jidfile, []byte(jid), 0644)
	qm.nextJid = jidIncr(qm.nextJid)
	return jid
}

// show functions used in debug
func (qm *QueueMgr) ShowWorkQueue() {
	fmt.Printf("current queuing workunits (%d):\n", qm.workQueue.Len())
	for key, _ := range qm.workQueue.workMap {
		fmt.Printf("workunit id: %s\n", key)
	}
	return
}

func (qm *QueueMgr) ShowTasks() {
	fmt.Printf("current active tasks  (%d):\n", len(qm.taskMap))
	for key, task := range qm.taskMap {
		fmt.Printf("workunit id: %s, status:%s\n", key, task.State)
	}
}

func (qm *QueueMgr) ShowStatus() string {
	total_task := len(qm.taskMap)
	queuing_work := len(qm.workQueue.wait)
	out_work := len(qm.workQueue.checkout)
	suspend_work := len(qm.workQueue.suspend)
	total_active_work := len(qm.workQueue.workMap)
	queuing_task := 0
	pending_task := 0
	completed_task := 0
	suspended_task := 0
	skipped_task := 0
	fail_skip_task := 0
	for _, task := range qm.taskMap {
		if task.State == TASK_STAT_COMPLETED {
			completed_task += 1
		} else if task.State == TASK_STAT_PENDING {
			pending_task += 1
		} else if task.State == TASK_STAT_QUEUED {
			queuing_task += 1
		} else if task.State == TASK_STAT_SUSPEND {
			suspended_task += 1
		} else if task.State == TASK_STAT_SKIPPED {
			skipped_task += 1
		} else if task.State == TASK_STAT_FAIL_SKIP {
			fail_skip_task += 1
		}
	}
	total_task -= skipped_task // user doesn't see skipped tasks
	in_progress_job := len(qm.actJobs)
	suspend_job := len(qm.susJobs)
	total_job := in_progress_job + suspend_job
	busy_client := 0
	idle_client := 0
	suspend_client := 0
	for _, client := range qm.clientMap {
		if client.Status == CLIENT_STAT_SUSPEND {
			suspend_client += 1
		} else if client.IsBusy() {
			busy_client += 1
		} else {
			idle_client += 1
		}
	}

	statMsg := "++++++++AWE server queue status++++++++\n" +
		fmt.Sprintf("total jobs ............... %d\n", total_job) +
		fmt.Sprintf("    in-progress:      (%d)\n", in_progress_job) +
		fmt.Sprintf("    suspended:        (%d)\n", suspend_job) +
		fmt.Sprintf("total tasks .............. %d\n", total_task) +
		fmt.Sprintf("    queuing:          (%d)\n", queuing_task) +
		fmt.Sprintf("    pending:          (%d)\n", pending_task) +
		fmt.Sprintf("    completed:        (%d)\n", completed_task) +
		fmt.Sprintf("    suspended:        (%d)\n", suspended_task) +
		fmt.Sprintf("    failed & skipped: (%d)\n", fail_skip_task) +
		fmt.Sprintf("total workunits .......... %d\n", total_active_work) +
		fmt.Sprintf("    queuing:          (%d)\n", queuing_work) +
		fmt.Sprintf("    checkout:         (%d)\n", out_work) +
		fmt.Sprintf("    suspended:        (%d)\n", suspend_work) +
		fmt.Sprintf("total clients ............ %d\n", len(qm.clientMap)) +
		fmt.Sprintf("    busy:             (%d)\n", busy_client) +
		fmt.Sprintf("    idle:             (%d)\n", idle_client) +
		fmt.Sprintf("    suspend:          (%d)\n", suspend_client) +
		fmt.Sprintf("---last update: %s\n\n", time.Now())
	return statMsg
}

//Client functions
func (qm *QueueMgr) RegisterNewClient(files FormFiles) (client *Client, err error) {
	//if queue is empty (no task is queuing or pending), reject client registration
	if _, ok := files["profile"]; ok {
		client, err = NewProfileClient(files["profile"].Path)
		os.Remove(files["profile"].Path)
	} else {
		client = NewClient()
	}
	if err != nil {
		return nil, err
	}
	qm.clientMap[client.Id] = client

	if len(client.Current_work) > 0 { //re-registered client
		// move already checked-out workunit from waiting queue (workMap) to checked-out list (coWorkMap)
		for workid, _ := range client.Current_work {
			if qm.workQueue.Has(workid) {
				qm.workQueue.StatusChange(workid, WORK_STAT_CHECKOUT)
			}
		}
	}
	return
}

func (qm *QueueMgr) GetClient(id string) (client *Client, err error) {
	if client, ok := qm.clientMap[id]; ok {
		return client, nil
	}
	return nil, errors.New(e.ClientNotFound)
}

func (qm *QueueMgr) GetAllClients() []*Client {
	var clients []*Client
	for _, client := range qm.clientMap {
		clients = append(clients, client)
	}
	return clients
}

func (qm *QueueMgr) ClientHeartBeat(id string) (client *Client, err error) {
	if client, ok := qm.clientMap[id]; ok {
		qm.clientMap[id].Tag = true
		Log.Debug(3, "HeartBeatFrom:"+"clientid="+id+",name="+qm.clientMap[id].Name)
		return client, nil
	}
	return nil, errors.New(e.ClientNotFound)
}

func (qm *QueueMgr) DeleteClient(id string) {
	delete(qm.clientMap, id)
}

func (qm *QueueMgr) filterWorkByClient(clientid string) (ids []string) {
	client := qm.clientMap[clientid]
	for id, _ := range qm.workQueue.wait {
		if _, ok := qm.workQueue.workMap[id]; !ok {
			continue
		}
		work := qm.workQueue.workMap[id]
		//skip works that are in the client's skip-list
		if contains(client.Skip_work, work.Id) {
			continue
		}
		//skip works that have dedicate client groups which this client doesn't belong to
		if len(work.Info.ClientGroups) > 0 {
			eligible_groups := strings.Split(work.Info.ClientGroups, ",")
			if !contains(eligible_groups, client.Group) {
				continue
			}
		}
		//append works whos apps are supported by the client
		if contains(client.Apps, work.Cmd.Name) {
			ids = append(ids, id)
		}
	}
	return ids
}

func (qm *QueueMgr) getWorkByClient(clientid string) (ids []string) {
	if client, ok := qm.clientMap[clientid]; ok {
		for id, _ := range client.Current_work {
			ids = append(ids, id)
		}
	}
	return
}

//job functions
func (qm *QueueMgr) updateJob(task *Task) (err error) {
	parts := strings.Split(task.Id, "_")
	jobid := parts[0]
	job, err := LoadJob(jobid)
	if err != nil {
		return
	}
	remainTasks, err := job.UpdateTask(task)
	if err != nil {
		return err
	}

	if remainTasks == 0 { //job done
		qm.FinalizeJobPerf(jobid)
		qm.LogJobPerf(jobid)
		qm.DeleteJobPerf(jobid)
		//delete tasks in task map
		for _, task := range job.TaskList() {
			delete(qm.taskMap, task.Id)
		}
		//log event about job done (JD)
		Log.Event(EVENT_JOB_DONE, "jobid="+job.Id)
	}
	return
}

func (qm *QueueMgr) GetActiveJobs() map[string]*JobPerf {
	return qm.actJobs
}

func (qm *QueueMgr) GetSuspendJobs() map[string]bool {
	return qm.susJobs
}

//recover jobs not completed before awe-server restarts
func (qm *QueueMgr) RecoverJobs() (err error) {
	//Get jobs to be recovered from db whose states are "submitted"
	dbjobs := new(Jobs)
	q := bson.M{}
	q["state"] = JOB_STAT_SUBMITTED
	lim := 1000
	off := 0
	if err := dbjobs.GetAllLimitOffset(q, lim, off); err != nil {
		Log.Error("RecoverJobs()->GetAllLimitOffset():" + err.Error())
		return err
	}
	//Locate the job script and parse tasks for each job
	jobct := 0
	for _, dbjob := range *dbjobs {
		qm.AddTasks(dbjob.Id, dbjob.TaskList())
		jobct += 1
	}
	qm.updateQueue()
	fmt.Printf("%d unfinished jobs recovered\n", jobct)
	return
}

//perf related
func (qm *QueueMgr) CreateJobPerf(jobid string) {
	qm.actJobs[jobid] = NewJobPerf(jobid)
}

func (qm *QueueMgr) FinalizeJobPerf(jobid string) {

	if perf, ok := qm.actJobs[jobid]; ok {
		now := time.Now().Unix()
		perf.End = now
		perf.Resp = now - perf.Queued
	}
	return
}

func (qm *QueueMgr) CreateTaskPerf(taskid string) {
	jobid := getParentJobId(taskid)
	if perf, ok := qm.actJobs[jobid]; ok {
		perf.Ptasks[taskid] = NewTaskPerf(taskid)
	}
}

func (qm *QueueMgr) FinalizeTaskPerf(taskid string) {
	jobid := getParentJobId(taskid)
	if jobperf, ok := qm.actJobs[jobid]; ok {
		if taskperf, ok := jobperf.Ptasks[taskid]; ok {
			now := time.Now().Unix()
			taskperf.End = now
			taskperf.Resp = now - taskperf.Queued
			//qm.actJobs[jobid].Ptasks[taskid] = taskperf

			if task, ok := qm.taskMap[taskid]; ok {
				for _, io := range task.Inputs {
					taskperf.InFileSizes = append(taskperf.InFileSizes, io.Size)
				}
				for _, io := range task.Outputs {
					taskperf.OutFileSizes = append(taskperf.OutFileSizes, io.Size)
				}
			}

			return
		}
	}
}

func (qm *QueueMgr) CreateWorkPerf(workid string) {
	if !conf.PERF_LOG_WORKUNIT {
		return
	}
	jobid := getParentJobId(workid)
	if _, ok := qm.actJobs[jobid]; !ok {
		return
	}
	qm.actJobs[jobid].Pworks[workid] = NewWorkPerf(workid)
}

func (qm *QueueMgr) FinalizeWorkPerf(workid string, reportfile string) (err error) {
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
	if _, ok := qm.actJobs[jobid]; !ok {
		return errors.New("job perf not found:" + jobid)
	}
	if _, ok := qm.actJobs[jobid].Pworks[workid]; !ok {
		return errors.New("work perf not found:" + workid)
	}
	workperf.Queued = qm.actJobs[jobid].Pworks[workid].Queued
	workperf.Done = time.Now().Unix()
	workperf.Resp = workperf.Done - workperf.Queued
	qm.actJobs[jobid].Pworks[workid] = workperf
	os.Remove(reportfile)
	return
}

func (qm *QueueMgr) LogJobPerf(jobid string) {
	if perf, ok := qm.actJobs[jobid]; ok {
		perfstr, _ := json.Marshal(perf)
		Log.Perf(string(perfstr))
	}
}

func (qm *QueueMgr) DeleteJobPerf(jobid string) {
	delete(qm.actJobs, jobid)
}

//misc local functions

func contains(list []string, elem string) bool {
	for _, t := range list {
		if t == elem {
			return true
		}
	}
	return false
}

func jidIncr(jid string) (newjid string) {
	if jidint, err := strconv.Atoi(jid); err == nil {
		jidint += 1
		return strconv.Itoa(jidint)
	}
	return jid
}

func locateSkippedInput(qm *QueueMgr, task *Task, ret_io *IO) {
	jobid := strings.Split(task.Id, "_")[0]
	for name, io := range task.Inputs { // Really just one entry
		if io.Node == "-" {
			preId := fmt.Sprintf("%s_%s", jobid, io.Origin)
			if preTask, ok := qm.taskMap[preId]; ok {
				if preTask.State == TASK_STAT_SKIPPED ||
					preTask.State == TASK_STAT_FAIL_SKIP {
					// recursive call
					locateSkippedInput(qm, preTask, ret_io)
				} else {
					outputs := preTask.Outputs
					if outio, ok := outputs[name]; ok {
						ret_io.Node = outio.Node
					}
				}
			}
		} else {
			ret_io.Node = io.Node
		}
	}
}
