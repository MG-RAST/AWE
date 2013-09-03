package core

import ()

type Workflow struct {
	WfInfo     awf_info          `bson:"workflow_info" json:"workflow_info"`
	JobInfo    awf_jobinfo       `bson:"job_info" json:"job_info"`
	RawInputs  map[string]string `bson:"raw_inputs" json:"raw_inputs"`
	Variables  map[string]string `bson:"variables" json:"variables"`
	DataServer string            `bson:"data_server" json:"data_server"`
	Tasks      []*awf_task       `bson:"tasks" json:"tasks"`
}

type awf_info struct {
	Name        string `bson:"name" json:"name"`
	Author      string `bson:"author" json:"author"`
	UpdateDate  string `bson:"update_date" json:"update_date"`
	Description string `bson:"description" json:"description"`
}

type awf_jobinfo struct {
	Name    string `bson:"jobname" json:"jobname"`
	Project string `bson:"project" json:"project"`
	User    string `bson:"user" json:"user"`
	Queue   string `bson:"queue" json:"queue"`
}

type awf_task struct {
	TaskId    int            `bson:"taskid" json:"taskid"`
	Cmd       *Command       `bson:"cmd" json:"cmd"`
	DependsOn []int          `bson:"dependsOn" json:"dependsOn"`
	Inputs    map[string]int `bson:"inputs" json:"inputs"`
	Outputs   []string       `bson:"outputs" json:"outputs"`
	Splits    int            `bson:"splits" json:"splits"`
}
