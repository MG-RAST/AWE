package controller

import ()

type ServerController struct {
	Awf    *AwfController
	Client *ClientController
	Job    *JobController
	Queue  *QueueController
	Work   *WorkController
}

func NewServerController() *ServerController {
	return &ServerController{
		Awf:    new(AwfController),
		Client: new(ClientController),
		Job:    new(JobController),
		Queue:  new(QueueController),
		Work:   new(WorkController),
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
