package core

import (
	"container/heap"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/core/pqueue"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type QueueMgr struct {
	taskMap   map[string]*Task
	workQueue WQueue
	taskIn    chan Task    //channel for receiving Task (JobController -> qmgr.Handler)
	coReq     chan string  //workunit checkout request (WorkController -> qmgr.Handler)
	coAck     chan AckItem //workunit checkout item including data and err (qmgr.Handler -> WorkController)
	coSem     chan int     //semaphore for checkout (mutual exclusion between different clients)
}

func NewQueueMgr() *QueueMgr {
	return &QueueMgr{
		taskMap:   map[string]*Task{},
		workQueue: NewWQueue(),
		taskIn:    make(chan Task, 1024),
		coReq:     make(chan string),
		coAck:     make(chan AckItem),
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
			qm.moveTasks()

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
				continue
			}

			ack := AckItem{workunit: wu, err: err}
			qm.coAck <- ack

		case <-time.After(10 * time.Second):
			fmt.Print("time to move tasks....\n")
			qm.moveTasks()
		}
	}
}

//send task to QueueMgr (called by other goroutins)
func (qm *QueueMgr) AddTasks(tasks []Task) (err error) {
	for _, task := range tasks {
		qm.taskIn <- task
	}
	return
}

//add task to taskMap
func (qm *QueueMgr) addTask(task Task) (err error) {
	id := task.Id
	task.State = "pending"
	qm.taskMap[id] = &task
	return
}

//delete task from taskMap
func (qm *QueueMgr) deleteTasks(tasks []Task) (err error) {
	return
}

//poll ready tasks and push into workQueue
func (qm *QueueMgr) moveTasks() (err error) {
	fmt.Printf("%s: try to move tasks to workunit queue...\n", time.Now())
	for id, task := range qm.taskMap {
		fmt.Printf("taskid=%s state=%s\n", id, task.State)
		ready := false
		if task.State == "pending" {
			ready = true
			for _, predecessor := range task.DependsOn {
				if _, haskey := qm.taskMap[predecessor]; haskey {
					ready = false
				}
			}
		}
		if ready {
			fmt.Printf("move workunits of task %s to workunit queue\n", id)
			if err := qm.createOutputNode(task); err != nil {
				fmt.Printf("error in createOutputNode(): %v\n", err)
				continue
			}
			if err := qm.parseTask(task); err != nil {
				fmt.Printf("error in parseTask(): %v\n", err)
				continue
			}

			task.State = "queued"
		}
	}
	qm.workQueue.Show()
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

func (qm *QueueMgr) createOutputNode(task *Task) (err error) {
	outputs := task.Outputs
	for name, io := range outputs {

		url := fmt.Sprintf("%s/node", io.Host)
		bodyType := ""
		body := strings.NewReader("")
		res, err := http.Post(url, bodyType, body)

		if err != nil {
			return err
		}

		jsonstream, err := ioutil.ReadAll(res.Body)
		res.Body.Close()

		response := new(ShockResponse)

		if err := json.Unmarshal(jsonstream, response); err != nil {
			return err
		}

		if len(response.Errs) > 0 {
			return errors.New(strings.Join(response.Errs, ","))
		}

		shocknode := &response.Data

		io.Node = shocknode.Id

		fmt.Printf("%s, output Shock node created, id=%s\n", name, shocknode.Id)
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
		return nil, errors.New("workunit queue is empty")
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
