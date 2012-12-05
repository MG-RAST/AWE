package main

import (
	"github.com/MG-RAST/AWE/core"
	e "github.com/MG-RAST/AWE/errors"
	. "github.com/MG-RAST/AWE/logger"
	"github.com/jaredwilkening/goweb"
	"net/http"
)

type WorkController struct{}

// GET: /work/{id}
// get a workunit by id
func (cr *WorkController) Read(id string, cx *goweb.Context) {
	// Load workunit by id
	workunit, err := queueMgr.GetWorkById(id)

	if err != nil {
		if err.Error() != e.WorkUnitQueueEmpty {
			Log.Error("Err@work_Read:QueueMgr.GetWorkById(): " + err.Error())
		}
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}
	// Base case respond with node in json	
	cx.RespondWithData(workunit)
	return
}

// GET: /work
// checkout a workunit with earliest submission time
// to-do: to support more options for workunit checkout 
func (cr *WorkController) ReadMany(cx *goweb.Context) {
	//checkout a workunit in FCFS order
	workunit, err := queueMgr.GetWorkByFCFS()

	if err != nil {
		Log.Error("Err@work_ReadMany:QueueMgr.GetWorkByFCFS(): " + err.Error())
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}
	// Base case respond with node in json	
	cx.RespondWithData(workunit)
	return
}

// PUT: /work/{id} -> status update
func (cr *WorkController) Update(id string, cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)

	// Gather query params
	query := &Query{list: cx.Request.URL.Query()}

	if query.Has("status") {
		notice := core.Notice{Workid: id, Status: query.Value("status")}
		queueMgr.NotifyWorkStatus(notice)
	}
	return
}
