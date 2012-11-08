package core

import (
	"errors"
	"fmt"
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

type WQueue map[string]*Workunit

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

//WQueue functions

func NewWQueue() WQueue {
	return map[string]*Workunit{}
}

func (wq WQueue) Len() int {
	return len(wq)
}

func (wq *WQueue) Push(workunit *Workunit) (err error) {
	if workunit.Id == "" {
		return errors.New("try to push a workunit with an empty id")
	}
	(*wq)[workunit.Id] = workunit
	return nil
}

func (wq WQueue) Show() (err error) {
	fmt.Printf("current queuing workunits (%d):\n", wq.Len())
	for key, _ := range wq {
		fmt.Printf("workunit id: %s\n", key)
	}
	return
}
