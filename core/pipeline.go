package core

import ()

type Pipeline struct {
	Info  *Info  `bson:"info" json:"info"`
	Tasks []Task `bson:"tasks" json:"tasks"`
}

func NewPipeline() *Pipeline {
	return &Pipeline{
		Info:  NewInfo(),
		Tasks: []Task{},
	}
}
