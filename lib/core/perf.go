package core

import (
	"time"
)

type JobPerf struct {
	Id     string
	Queued int64
	//Start  int64
	End    int64
	Resp   int64 //End - Queued
	Ptasks map[string]*TaskPerf
	Pworks map[string]*WorkPerf
}

type TaskPerf struct {
	Size         int64
	Queued       int64
	End          int64
	Resp         int64 //End -Queued
	InFileSizes  []int64
	OutFileSizes []int64
}

type WorkPerf struct {
	Queued     int64 // WQ (queued at server or client, depending on who creates it)
	Done       int64 // WD (done at server)
	Resp       int64 // Done - Queued (server metric)
	Checkout   int64 // checkout at client
	Deliver    int64 // done at client
	ClientResp int64 // Deliver - Checkout (client metric)
	DataIn     int64 // input data move-in at client
	DataOut    int64 // output data move-out at client
	Runtime    int64 // computing time at client
	ClientId   string
}

func NewJobPerf(id string) *JobPerf {
	return &JobPerf{
		Id:     id,
		Queued: time.Now().Unix(),
		Ptasks: map[string]*TaskPerf{},
		Pworks: map[string]*WorkPerf{},
	}
}

func NewTaskPerf(id string) *TaskPerf {
	return &TaskPerf{
		Queued:       time.Now().Unix(),
		InFileSizes:  []int64{},
		OutFileSizes: []int64{},
	}
}

func NewWorkPerf(id string) *WorkPerf {
	return &WorkPerf{
		Queued: time.Now().Unix(),
	}
}
