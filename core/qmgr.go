package core

import (
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
	actJobs   map[string]bool //false:job submitted, true: job in progress (at least one task in queue)
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
		actJobs:   map[string]bool{},
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
	workMap   map[string]*Workunit //workunits waiting in the queue
	coWorkMap map[string]coInfo    //workunits being checked out yet not done
}

func NewWQueue() *WQueue {
	return &WQueue{
		workMap:   map[string]*Workunit{},
		coWorkMap: map[string]coInfo{},
	}
}

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
			fmt.Println(qm.ShowStatus())
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
			} else {
				//now client must be gone as tag set to false 30 seconds ago and no heartbeat received thereafter
				//delete the client from client map
				Log.Event(EVENT_CLIENT_UNREGISTER, "clientid="+clientid+",name="+qm.clientMap[clientid].Name)
				delete(qm.clientMap, clientid)

				//requeue unfinished workunits associated with the failed client
				workids := qm.getWorkByClient(clientid)
				for _, workid := range workids {
					if coinfo, ok := qm.workQueue.coWorkMap[workid]; ok {
						qm.workQueue.workMap[workid] = coinfo.workunit
						delete(qm.workQueue.coWorkMap, workid)
						Log.Event(EVENT_WORK_REQUEUE, "workid="+workid)
					}
				}
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
	qm.actJobs[jobid] = true
	return
}

func (qm *QueueMgr) CheckoutWorkunits(req_policy string, client_id string, num int) (workunits []*Workunit, err error) {
	//precheck if teh client is registered
	if _, hasClient := qm.clientMap[client_id]; !hasClient {
		return nil, errors.New("invalid client id: " + client_id)
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
			delete(qm.workQueue.workMap, workid)
		}
	}
	//delete parsed tasks
	for i := 0; i < len(job.TaskList()); i++ {
		task_id := fmt.Sprintf("%s_%d", jobid, i)
		delete(qm.taskMap, task_id)
	}
	delete(qm.actJobs, jobid)
	return
}

//add task to taskMap
func (qm *QueueMgr) addTask(task *Task) (err error) {
	id := task.Id
	if task.State == "completed" { //for job recovery from db
		qm.taskMap[id] = task
		return
	}
	task.State = "pending"
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
		if task.State == "pending" {
			ready = true
			for _, predecessor := range task.DependsOn {
				if _, haskey := qm.taskMap[predecessor]; haskey {
					if qm.taskMap[predecessor].State != "completed" {
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
	//fmt.Printf("move workunits of task %s to workunit queue\n", task.Id)
	if err := qm.locateInputs(task); err != nil {
		Log.Error("qmgr.taskEnQueue locateInputs:" + err.Error())
		return err
	}

	if task.TotalWork > 0 {
		if err := task.InitPartIndex(); err != nil {
			Log.Error("qmgr.taskEnQueue InitPartitionIndex:" + err.Error())
			return err
		}
	}

	if err := qm.createOutputNode(task); err != nil {
		Log.Error("qmgr.taskEnQueue createOutputNode:" + err.Error())
		return err
	}
	if err := qm.parseTask(task); err != nil {
		Log.Error("qmgr.taskEnQueue parseTask:" + err.Error())
		return err
	}
	task.State = "queued"

	//log event about task enqueue (TQ)
	Log.Event(EVENT_TASK_ENQUEUE, "taskid="+task.Id)

	return
}

func (qm *QueueMgr) locateInputs(task *Task) (err error) {
	jobid := strings.Split(task.Id, "_")[0]
	for name, io := range task.Inputs {
		if io.Node == "-" {
			preId := fmt.Sprintf("%s_%s", jobid, io.Origin)
			if preTask, ok := qm.taskMap[preId]; ok {
				outputs := preTask.Outputs
				if outio, ok := outputs[name]; ok {
					io.Node = outio.Node
				}
			}
		}
		if io.Node == "-" {
			return errors.New(fmt.Sprintf("error in locate input for task %s, %s", task.Id, name))
		}
	}
	return
}

func (qm *QueueMgr) parseTask(task *Task) (err error) {
	workunits, err := task.ParseWorkunit()
	if err != nil {
		return err
	}
	for _, wu := range workunits {
		if err := qm.workQueue.Push(wu); err != nil {
			return err
		}
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
	works, err = qm.workQueue.getWorks(filtered, req.policy, req.count)

	if err == nil { //get workunits successfully, put them into coWorkMap
		for _, work := range works {
			coinfo := coInfo{workunit: work, clientid: req.fromclient}
			qm.workQueue.coWorkMap[work.Id] = coinfo
		}
	}
	return
}

func (qm *QueueMgr) handleWorkStatusChange(notice Notice) (err error) {
	workid := notice.WorkId
	status := notice.Status
	clientid := notice.ClientId
	parts := strings.Split(workid, "_")
	taskid := fmt.Sprintf("%s_%s", parts[0], parts[1])
	rank, err := strconv.Atoi(parts[2])
	if err != nil {
		return errors.New(fmt.Sprintf("invalid workid %s", workid))
	}
	if _, ok := qm.clientMap[clientid]; ok {
		delete(qm.clientMap[clientid].Current_work, workid)
	}

	if _, ok := qm.taskMap[taskid]; ok {
		if rank == 0 {
			qm.taskMap[taskid].WorkStatus[rank] = status
		} else {
			qm.taskMap[taskid].WorkStatus[rank-1] = status
		}
		if status == "done" {
			//log event about work done (WD)
			Log.Event(EVENT_WORK_DONE, "workid="+workid+";clientid="+clientid)

			//update client status
			if _, ok := qm.clientMap[clientid]; ok {
				qm.clientMap[clientid].Total_completed += 1
			}

			qm.taskMap[taskid].RemainWork -= 1
			if qm.taskMap[taskid].RemainWork == 0 {
				qm.taskMap[taskid].State = "completed"

				//log event about task done (TD) 
				Log.Event(EVENT_TASK_DONE, "taskid="+taskid)
				if err = qm.updateJob(qm.taskMap[taskid]); err != nil {
					return
				}
				qm.updateQueue()
			}
			delete(qm.workQueue.coWorkMap, workid)
		} else if status == "fail" { //requeue failed workunit
			Log.Event(EVENT_WORK_FAIL, "workid="+workid)
			if coinfo, ok := qm.workQueue.coWorkMap[workid]; ok {
				qm.workQueue.workMap[workid] = coinfo.workunit
				delete(qm.workQueue.coWorkMap, workid)
				client := qm.clientMap[coinfo.clientid]
				client.SkipWorks = append(client.SkipWorks, workid)
				client.Total_failed += 1
				Log.Event(EVENT_WORK_REQUEUE, "workid="+workid)
			}
		}
	} else { //task not existed, possible when job is deleted before the workunit done
		delete(qm.workQueue.coWorkMap, workid)
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
	queuing_work := len(qm.workQueue.workMap)
	out_work := len(qm.workQueue.coWorkMap)
	pending := 0
	completed := 0
	queuing := 0
	for _, task := range qm.taskMap {
		if task.State == "completed" {
			completed += 1
		} else if task.State == "pending" {
			pending += 1
		} else if task.State == "queued" {
			queuing += 1
		}
	}

	statMsg := "+++++AWE server queue status+++++\n" +
		fmt.Sprintf("total jobs ......... %d\n", len(qm.actJobs)) +
		fmt.Sprintf("total tasks ........ %d\n", total_task) +
		fmt.Sprintf("    queuing:    (%d)\n", queuing) +
		fmt.Sprintf("    pending:    (%d)\n", pending) +
		fmt.Sprintf("    completed:  (%d)\n", completed) +
		fmt.Sprintf("total workunits .... %d\n", queuing_work+out_work) +
		fmt.Sprintf("    queuing:  (%d)\n", queuing_work) +
		fmt.Sprintf("    checkout: (%d)\n", out_work) +
		fmt.Sprintf("total clients ...... %d\n", len(qm.clientMap)) +
		fmt.Sprintf("---last update: %s\n\n", time.Now())
	return statMsg
}

//WQueue functions

func (wq WQueue) Len() int {
	return len(wq.workMap)
}

func (wq *WQueue) Push(workunit *Workunit) (err error) {
	if workunit.Id == "" {
		return errors.New("try to push a workunit with an empty id")
	}
	wq.workMap[workunit.Id] = workunit
	return nil
}

//Client functions
func (qm *QueueMgr) RegisterNewClient(params map[string]string, files FormFiles) (client *Client, err error) {
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
	if _, ok := qm.clientMap[id]; ok {
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
	for id, work := range qm.workQueue.workMap {
		//skip works that are in the client's skip-list
		if contains(client.SkipWorks, work.Id) {
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
	for id, coinfo := range qm.workQueue.coWorkMap {
		if coinfo.clientid == clientid {
			ids = append(ids, id)
		}
	}
	return ids
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
	if remainTasks == 0 {
		delete(qm.actJobs, jobid)
		//delete tasks in task map
		for _, task := range job.TaskList() {
			delete(qm.taskMap, task.Id)
		}
	}
	return
}

func (qm *QueueMgr) GetActiveJobs() map[string]bool {
	return qm.actJobs
}

//queue related functions
type WorkList []*Workunit

func (wl WorkList) Len() int      { return len(wl) }
func (wl WorkList) Swap(i, j int) { wl[i], wl[j] = wl[j], wl[i] }

type byFCFS struct{ WorkList }

func (s byFCFS) Less(i, j int) bool {
	return s.WorkList[i].Info.SubmitTime.Before(s.WorkList[j].Info.SubmitTime)
}

func (wq *WQueue) getWorks(workid []string, policy string, count int) (works []*Workunit, err error) {
	worklist := []*Workunit{}
	for _, id := range workid {
		worklist = append(worklist, wq.workMap[id])
	}

	works = []*Workunit{}

	if policy == "FCFS" {
		sort.Sort(byFCFS{worklist})
	}
	for i := 0; i < count; i++ {
		works = append(works, worklist[i])
		delete(wq.workMap, worklist[i].Id)
	}

	return
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
