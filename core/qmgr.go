package core

import (
	"container/heap"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/core/pqueue"
	e "github.com/MG-RAST/AWE/errors"
	. "github.com/MG-RAST/AWE/logger"
	"io/ioutil"
	"net/http"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

type QueueMgr struct {
	clientMap map[string]*Client
	taskMap   map[string]*Task
	workQueue WQueue
	reminder  chan bool
	taskIn    chan *Task   //channel for receiving Task (JobController -> qmgr.Handler)
	coReq     chan string  //workunit checkout request (WorkController -> qmgr.Handler)
	coAck     chan AckItem //workunit checkout item including data and err (qmgr.Handler -> WorkController)
	feedback  chan Notice  //workunit execution feedback (WorkController -> qmgr.Handler)
	coSem     chan int     //semaphore for checkout (mutual exclusion between different clients)
}

func NewQueueMgr() *QueueMgr {
	return &QueueMgr{
		clientMap: map[string]*Client{},
		taskMap:   map[string]*Task{},
		workQueue: NewWQueue(),
		reminder:  make(chan bool),
		taskIn:    make(chan *Task, 1024),
		coReq:     make(chan string),
		coAck:     make(chan AckItem),
		feedback:  make(chan Notice),
		coSem:     make(chan int, 1), //non-blocking buffered channel
	}
}

type ShockResponse struct {
	Code int       `bson:"S" json:"S"`
	Data ShockNode `bson:"D" json:"D"`
	Errs []string  `bson:"E" json:"E"`
}

type ShockNode struct {
	Id         string            `bson:"id" json:"id"`
	Version    string            `bson:"version" json:"version"`
	File       shockfile         `bson:"file" json:"file"`
	Attributes interface{}       `bson:"attributes" json:"attributes"`
	Indexes    map[string]string `bson:"indexes" json:"indexes"`
}

type shockfile struct {
	Name         string            `bson:"name" json:"name"`
	Size         int64             `bson:"size" json:"size"`
	Checksum     map[string]string `bson:"checksum" json:"checksum"`
	Format       string            `bson:"format" json:"format"`
	Virtual      bool              `bson:"virtual" json:"virtual"`
	VirtualParts []string          `bson:"virtual_parts" json:"virtual_parts"`
}

type AckItem struct {
	workunit *Workunit
	err      error
}

type Notice struct {
	Workid string
	Status string
}

type WQueue struct {
	workMap map[string]*Workunit
	pQue    pqueue.PriorityQueue
}

func NewWQueue() WQueue {
	return WQueue{
		workMap: map[string]*Workunit{},
		pQue:    make(pqueue.PriorityQueue, 0, 10000),
	}
}

//handle request from JobController to add taskMap
//to-do: remove debug statements
func (qm *QueueMgr) Handle() {
	for {
		select {
		case task := <-qm.taskIn:
			fmt.Printf("task recived from chan taskIn, id=%s\n", task.Id)
			qm.addTask(task)

		case coReq := <-qm.coReq:
			fmt.Printf("workunit checkout request received, policy=%s\n", coReq)

			var wu *Workunit
			var err error

			segs := strings.Split(coReq, ":")
			policy := segs[0]

			if policy == "FCFS" {
				wu, err = qm.workQueue.PopWorkFCFS()
			} else if policy == "ById" {
				wu, err = qm.workQueue.PopWorkByID(segs[1])
			} else {
				err = errors.New("bad checkout policy")
				continue
			}

			ack := AckItem{workunit: wu, err: err}
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

func (qm *QueueMgr) GetWorkByFCFS() (workunit *Workunit, err error) {
	//lock semephore, at one time only one client's checkout request can be served 
	qm.coSem <- 1

	qm.coReq <- "FCFS"
	ack := <-qm.coAck

	//unlock
	<-qm.coSem

	return ack.workunit, ack.err
}

func (qm *QueueMgr) GetWorkById(id string) (workunit *Workunit, err error) {
	qm.coSem <- 1

	qm.coReq <- fmt.Sprintf("ById:%s", id)
	ack := <-qm.coAck

	<-qm.coSem

	return ack.workunit, ack.err
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
	fmt.Printf("%s: try to move tasks to workunit queue...\n", time.Now())
	for id, task := range qm.taskMap {
		fmt.Printf("taskid=%s state=%s\n", id, task.State)
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
	qm.workQueue.Show()
	return
}

func (qm *QueueMgr) taskEnQueue(task *Task) (err error) {
	fmt.Printf("move workunits of task %s to workunit queue\n", task.Id)
	if err := qm.locateInputs(task); err != nil {
		Log.Error("ERROR: qmgr.taskEnQueue: " + err.Error())
		return err
	}
	if err := qm.createOutputNode(task); err != nil {
		Log.Error("ERROR: qmgr.taskEnQueue: " + err.Error())
		return err
	}
	if err := qm.parseTask(task); err != nil {
		Log.Error("ERROR: qmgr.taskEnQueue: " + err.Error())
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
	for name, io := range outputs {
		nodeid, err := postNode(io, task.TotalWork)
		if err != nil {
			return err
		}
		io.Node = nodeid
		fmt.Printf("%s, output Shock node created, id=%s\n", name, io.Node)
	}
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
				Log.Event(EVENT_TASK_DONE, "taskid="+workid)

				if err = updateJob(qm.taskMap[taskid]); err != nil {
					return
				}
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

	score := priorityFunction(workunit)

	queueItem := pqueue.NewItem(workunit.Id, score)

	heap.Push(&wq.pQue, queueItem)

	return nil
}

func (wq WQueue) Show() (err error) {
	fmt.Printf("current queuing workunits (%d):\n", wq.Len())
	for key, _ := range wq.workMap {
		fmt.Printf("workunit id: %s\n", key)
	}
	return
}

//pop a workunit by specified workunit id
//to-do: support returning workunit based on a given job id
func (wq WQueue) PopWorkByID(id string) (workunit *Workunit, err error) {
	if workunit, hasid := wq.workMap[id]; hasid {
		delete(wq.workMap, id)
		return workunit, nil
	}
	return nil, errors.New(fmt.Sprintf("no workunit found with id %s", id))
}

//pop a workunit in FCFS order
//to-do: update workunit state to "checked-out"
func (wq *WQueue) PopWorkFCFS() (workunit *Workunit, err error) {
	if wq.Len() == 0 {
		return nil, errors.New(e.WorkUnitQueueEmpty)
	}

	//some workunit might be checked out by id, thus keep popping
	//pQue until find an Id with queued workunit
	for {
		item := heap.Pop(&wq.pQue).(*pqueue.Item)
		id := item.Value()
		if workunit, hasid := wq.workMap[id]; hasid {
			delete(wq.workMap, id)
			return workunit, nil
		}
	}
	return
}

//curently support calculate priority score by FCFS
//to-do: make prioritizing policy configurable
func priorityFunction(workunit *Workunit) int64 {
	return 1 - workunit.Info.SubmitTime.Unix()
}

//create a shock node for output
func postNode(io *IO, numParts int) (nodeid string, err error) {
	var res *http.Response
	shockurl := fmt.Sprintf("%s/node", io.Host)
	res, err = http.Post(shockurl, "", strings.NewReader(""))
	if err != nil {
		return "", err
	}

	jsonstream, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	response := new(ShockResponse)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return "", err
	}
	if len(response.Errs) > 0 {
		return "", errors.New(strings.Join(response.Errs, ","))
	}

	shocknode := &response.Data
	nodeid = shocknode.Id

	if numParts > 1 {
		putParts(io.Host, nodeid, numParts)
	}

	fmt.Printf("posted a node: %s\n", nodeid)
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
func (qm *QueueMgr) RegisterNewClient() (client *Client, err error) {
	//if queue is empty, reject client registration
	if qm.workQueue.Len() == 0 {
		return nil, errors.New(e.WorkUnitQueueEmpty)
	}
	client = NewClient()
	qm.clientMap[client.Id] = client
	fmt.Printf("registered a new client:%s, current total clients: %d\n", client.Id, len(qm.clientMap))
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
