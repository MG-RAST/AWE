package controller

import (
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/golib/goweb"
	"net/http"
)

type AwfController struct{}

// GET: /awf/{name}
// get a workflow by name, read-only
func (cr *AwfController) Read(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	// Load workunit by id
	workflow, err := core.AwfMgr.GetWorkflow(id)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}
	// Base case respond with workunit in json
	cx.RespondWithData(workflow)
	return
}

// GET: /awf
// get all loaded workflows
func (cr *AwfController) ReadMany(cx *goweb.Context) {
	// Gather query params
	workflows := core.AwfMgr.GetAllWorkflows()
	cx.RespondWithData(workflows)
	return
}
