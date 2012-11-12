package core

import (
	"container/heap"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/core/pqueue"
	"time"
)

type QueueMgr struct {
	taskMap   map[string]*Task
	workQueue WQueue
	taskIn    chan Task
	reminder  chan bool
	//	wuReq   chan int
	//	wuAck   chan int
}

func NewQueueMgr() *QueueMgr {
	return &QueueMgr{
		taskMap:   map[string]*Task{},
		workQueue: NewWQueue(),
		taskIn:    make(chan Task, 1024),
		reminder:  make(chan bool),
		//		wuReq:     make(chan int, 1024),
		//		wuAck:     make(chan int, 1024),
	}
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
		case <-qm.reminder:
			fmt.Print("time to move tasks....\n")
			qm.moveTasks()
		}
	}
}

func (qm *QueueMgr) Timer() {
	for {
		time.Sleep(10 * time.Second)
		qm.reminder <- true
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
			qm.parseTask(task)
			task.State = "queued"
		}
	}
	qm.workQueue.Show()
	return
}

func (qm *QueueMgr) parseTask(task *Task) (workunits []*Workunit, err error) {
	workunits, err = task.ParseWorkunit()
	if err != nil {
		return nil, err
	}
	for _, wu := range workunits {
		if err := qm.workQueue.Push(wu); err != nil {
			return nil, err
		}
	}
	return
}

func (qm *QueueMgr) GetWorkById(id string) (workunit *Workunit, err error) {
	return qm.workQueue.PopWorkByID(id)
}

func (qm *QueueMgr) GetWorkByFCFS() (workunit *Workunit, err error) {
	return qm.workQueue.PopWorkFCFS()
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
