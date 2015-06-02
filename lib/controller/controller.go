package controller

import (
	"github.com/MG-RAST/AWE/vendor/github.com/MG-RAST/golib/goweb"
)

type ServerController struct {
	Awf              *AwfController
	Client           *ClientController
	ClientGroup      *ClientGroupController
	ClientGroupAcl   map[string]goweb.ControllerFunc
	ClientGroupToken goweb.ControllerFunc
	Job              *JobController
	JobAcl           map[string]goweb.ControllerFunc
	Queue            *QueueController
	Work             *WorkController
}

func NewServerController() *ServerController {
	return &ServerController{
		Awf:              new(AwfController),
		Client:           new(ClientController),
		ClientGroup:      new(ClientGroupController),
		ClientGroupAcl:   map[string]goweb.ControllerFunc{"base": ClientGroupAclController, "typed": ClientGroupAclControllerTyped},
		ClientGroupToken: ClientGroupTokenController,
		Job:              new(JobController),
		JobAcl:           map[string]goweb.ControllerFunc{"base": JobAclController, "typed": JobAclControllerTyped},
		Queue:            new(QueueController),
		Work:             new(WorkController),
	}
}

type ProxyController struct {
	Client *ClientController
	Work   *WorkController
}

func NewProxyController() *ProxyController {
	return &ProxyController{
		Client: new(ClientController),
		Work:   new(WorkController),
	}
}
