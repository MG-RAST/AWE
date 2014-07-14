package core

import (
	"time"
)

type JobPerf struct {
	Id     string               `bson:"id" json:"id"`
	Queued int64                `bson:"queued" json:"queued"`
	Start  int64                `bson:"start" json:"start"`
	End    int64                `bson:"end" json:"end"`
	Resp   int64                `bson:"resp" json:"resp"` //End - Queued
	Ptasks map[string]*TaskPerf `bson:"task_stats" json:"task_stats"`
	Pworks map[string]*WorkPerf `bson:"work_stats" json:"work_stats"`
}

type TaskPerf struct {
	Queued       int64   `bson:"queued" json:"queued"`
	Start        int64   `bson:"start" json:"start"`
	End          int64   `bson:"end" json:"end"`
	Resp         int64   `bson:"resp" json:"resp"` //End -Queued
	InFileSizes  []int64 `bson:"size_infile" json:"size_infile"`
	OutFileSizes []int64 `bson:"size_outfile" json:"size_outfile"`
}

type WorkPerf struct {
	Queued      int64   `bson:"queued" json:"queued"`                   // WQ (queued at server or client, depending on who creates it)
	Done        int64   `bson:"done" json:"done"`                       // WD (done at server)
	Resp        int64   `bson:"resp" json:"resp"`                       // Done - Queued (server metric)
	Checkout    int64   `bson:"checkout" json:"checkout"`               // checkout at client
	Deliver     int64   `bson:"deliver" json:"deliver"`                 // done at client
	ClientResp  int64   `bson:"clientresp" json:"clientresp"`           // Deliver - Checkout (client metric)
	PreDataIn   float64 `bson:"time_predata_in" json:"time_predata_in"` // time in seconds for downloading prerequisite data at client
	DataIn      float64 `bson:"time_data_in" json:"time_data_in"`       // time in seconds for input data move-in at client
	DataOut     float64 `bson:"time_data_out" json:"time_data_out"`     // time in seconds for output data move-out at client
	Runtime     int64   `bson:"runtime" json:"runtime"`                 // time in seconds for computation at client
	MaxMemUsage int64   `bson:"max_mem_usage" json:"max_mem_usage"`     // maxium memery consumption
	ClientId    string  `bson:"client_id" json:"client_id"`
	PreDataSize int64   `bson:"size_predata" json:"size_predata"` //predata moved over network
	InFileSize  int64   `bson:"size_infile" json:"size_infile"`   //input file moved over network
	OutFileSize int64   `bson:"size_outfile" json:"size_outfile"` //outpuf file moved over network
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
