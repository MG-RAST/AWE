package controller

import (
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/vendor/github.com/MG-RAST/golib/goweb"
	"net/http"
)

type QueueController struct{}

// OPTIONS: /queue
func (cr *QueueController) Options(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithOK()
	return
}

// POST: /queue
func (cr *QueueController) Create(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// GET: /queue/{id}
func (cr *QueueController) Read(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// GET: /queue
// get status from queue manager
func (cr *QueueController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Gather query params
	//	query := &Query{list: cx.Request.URL.Query()}

	msg := core.QMgr.ShowStatus()
	cx.RespondWithData(msg)
	return
}

// PUT: /queue/{id} -> status update
func (cr *QueueController) Update(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// PUT: /queue
func (cr *QueueController) UpdateMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// DELETE: /queue/{id}
func (cr *QueueController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// DELETE: /queue
func (cr *QueueController) DeleteMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}
