package core

import (
	"errors"
	"fmt"
	e "github.com/MG-RAST/AWE/errors"
	. "github.com/MG-RAST/AWE/logger"
	"os"
	"os/exec"
	"sort"
	"strconv"
	"strings"
	"time"
)

type QueueMgr struct {
	clientMap map[string]*Client
	taskMap   map[string]*Task
	workQueue *WQueue
	reminder  chan bool
	taskIn    chan *Task  //channel for receiving Task (JobController -> qmgr.Handler)
	coReq     chan CoReq  //workunit checkout request (WorkController -> qmgr.Handler)
	coAck     chan CoAck  //workunit checkout item including data and err (qmgr.Handler -> WorkController)
	feedback  chan Notice //workunit execution feedback (WorkController -> qmgr.Handler)
	coSem     chan int    //semaphore for checkout (mutual exclusion between different clients)
}

func NewQueueMgr() *QueueMgr {
	return &QueueMgr{
		clientMap: map[string]*Client{},
		taskMap:   map[string]*Task{},
		workQueue: NewWQueue(),
		reminder:  make(chan bool),
		taskIn:    make(chan *Task, 1024),
		coReq:     make(chan CoReq),
		coAck:     make(chan CoAck),
		feedback:  make(chan Notice),
		coSem:     make(chan int, 1), //non-blocking buffered channel
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
	Workid string
	Status string
}

type WQueue struct {
	workMap map[string]*Workunit
}

func NewWQueue() *WQueue {
	return &WQueue{
		workMap: map[string]*Workunit{},
	}
}

//handle request from JobController to add taskMap
//to-do: remove debug statements
func (qm *QueueMgr) Handle() {
	for {
		select {
		case task := <-qm.taskIn:
			//fmt.Printf("task recived from chan taskIn, id=%s\n", task.Id)
			qm.addTask(task)

		case coReq := <-qm.coReq:
			//fmt.Printf("workunit checkout request received, Req=%v\n", coReq)
			works, err := qm.popWorks(coReq)
			ack := CoAck{workunits: works, err: err}
			qm.coAck <- ack

		case notice := <-qm.feedback:
			//	fmt.Printf("workunit status feedback received, workid=%s, status=%s\n", notice.Workid, notice.Status)
			if err := qm.handleWorkStatusChange(notice); err != nil {
				Log.Error("ERROR: qmgr handleWorkStatusChange(): " + err.Error())
			}

		case <-qm.reminder:
			//fmt.Print("time to update workunit queue....\n")
			qm.updateQueue()
		}
	}
}

func (qm *QueueMgr) Timer() {
	for {
		time.Sleep(10 * time.Second)
		qm.reminder <- true
	}
}

func (qm *QueueMgr) AddTasks(tasks []*Task) (err error) {
	for _, task := range tasks {
		qm.taskIn <- task
	}
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

	//unlock
	<-qm.coSem

	return ack.workunits, ack.err
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

//add task to taskMap
func (qm *QueueMgr) addTask(task *Task) (err error) {
	id := task.Id
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
		//fmt.Printf("taskid=%s state=%s\n", id, task.State)
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
	//qm.workQueue.Show()
	return
}

func (qm *QueueMgr) taskEnQueue(task *Task) (err error) {
	//fmt.Printf("move workunits of task %s to workunit queue\n", task.Id)
	if err := qm.locateInputs(task); err != nil {
		Log.Error("qmgr.taskEnQueue locateInputs:" + err.Error())
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
	supportedApps := qm.clientMap[req.fromclient].Apps
	filtered := qm.workQueue.filterWorkByApps(supportedApps)
	if len(filtered) == 0 {
		return nil, errors.New(e.NoEligibleWorkunitFound)
	}
	works, err = qm.workQueue.getWorks(filtered, req.policy, req.count)
	return
}

func (qm *QueueMgr) handleWorkStatusChange(notice Notice) (err error) {
	workid := notice.Workid
	status := notice.Status
	parts := strings.Split(workid, "_")
	taskid := fmt.Sprintf("%s_%s", parts[0], parts[1])
	rank, err := strconv.Atoi(parts[2])
	if err != nil {
		return errors.New(fmt.Sprintf("invalid workid %s", workid))
	}
	if _, ok := qm.taskMap[taskid]; ok {
		if rank == 0 {
			qm.taskMap[taskid].WorkStatus[rank] = status
		} else {
			qm.taskMap[taskid].WorkStatus[rank-1] = status
		}
		if status == "done" {
			//log event about work done (WD)
			Log.Event(EVENT_WORK_DONE, "workid="+workid)

			qm.taskMap[taskid].RemainWork -= 1
			if qm.taskMap[taskid].RemainWork == 0 {
				qm.taskMap[taskid].State = "completed"

				//log event about task done (TD) 
				Log.Event(EVENT_TASK_DONE, "taskid="+taskid)
				if err = updateJob(qm.taskMap[taskid]); err != nil {
					return
				}
				//delete(qm.taskMap, taskid)

				qm.updateQueue()
			}
		}
	} else {
		return errors.New(fmt.Sprintf("task not existed: %s", taskid))
	}
	return
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

func (wq WQueue) Show() (err error) {
	fmt.Printf("current queuing workunits (%d):\n", wq.Len())
	for key, _ := range wq.workMap {
		fmt.Printf("workunit id: %s\n", key)
	}
	return
}

//create parts
func putParts(host string, nodeid string, numParts int) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	argv = append(argv, "-F")
	argv = append(argv, fmt.Sprintf("parts=%d", numParts))
	target_url := fmt.Sprintf("%s/node/%s", host, nodeid)
	argv = append(argv, target_url)

	cmd := exec.Command("curl", argv...)
	err = cmd.Run()
	if err != nil {
		return
	}
	return
}

//Client functions
func (qm *QueueMgr) RegisterNewClient(params map[string]string, files FormFiles) (client *Client, err error) {
	//if queue is empty, reject client registration
	if qm.workQueue.Len() == 0 {
		return nil, errors.New(e.WorkUnitQueueEmpty)
	}

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

func (qm *QueueMgr) deleteClient(id string) {
	delete(qm.clientMap, id)
}

//job functions
func updateJob(task *Task) (err error) {
	parts := strings.Split(task.Id, "_")
	jobid := parts[0]
	job, err := LoadJob(jobid)
	if err != nil {
		return
	}
	return job.UpdateTask(task)
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

func (wq *WQueue) filterWorkByApps(apps []string) (ids []string) {
	ids = []string{}
	for id, work := range wq.workMap {
		if contains(apps, work.Cmd.Name) {
			ids = append(ids, id)
		}
	}
	return ids
}

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
