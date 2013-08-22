package main

import (
	"github.com/MG-RAST/AWE/core"
	e "github.com/MG-RAST/AWE/errors"
	. "github.com/MG-RAST/AWE/logger"
	"github.com/jaredwilkening/goweb"
	"net/http"
)

type ClientController struct{}

// POST: /client
func (cr *ClientController) Create(cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)

	// Parse uploaded form
	_, files, err := ParseMultipartForm(cx.Request)
	if err != nil {
		if err.Error() != "request Content-Type isn't multipart/form-data" {
			Log.Error("Error parsing form: " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
			return
		}
	}

	client, err := queueMgr.RegisterNewClient(files)
	if err != nil {
		msg := "Error in registering new client:" + err.Error()
		Log.Error(msg)
		cx.RespondWithErrorMessage(msg, http.StatusBadRequest)
		return
	}

	//log event about client registration (CR)
	Log.Event(EVENT_CLIENT_REGISTRATION, "clientid="+client.Id+";name="+client.Name+";host="+client.Host)

	cx.RespondWithData(client)
	return
}

// GET: /client/{id}
func (cr *ClientController) Read(id string, cx *goweb.Context) {
	// Gather query params
	query := &Query{list: cx.Request.URL.Query()}

	if query.Has("heartbeat") { //handle heartbeat
		hbmsg, err := queueMgr.ClientHeartBeat(id)
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		} else {
			cx.RespondWithData(hbmsg)
		}
		return
	}

	LogRequest(cx.Request) //skip heartbeat in access log

	client, err := queueMgr.GetClient(id)
	if err != nil {
		if err.Error() == e.ClientNotFound {
			cx.RespondWithErrorMessage(e.ClientNotFound, http.StatusBadRequest)
		} else {
			Log.Error("Error in GET client:" + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
		}
		return
	}
	cx.RespondWithData(client)
}

// GET: /client
func (cr *ClientController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	clients := queueMgr.GetAllClients()
	if len(clients) == 0 {
		cx.RespondWithErrorMessage(e.ClientNotFound, http.StatusBadRequest)
		return
	}

	query := &Query{list: cx.Request.URL.Query()}
	filtered := []*core.Client{}
	if query.Has("busy") {
		for _, client := range clients {
			if len(client.Current_work) > 0 {
				filtered = append(filtered, client)
			}
		}
	} else {
		filtered = clients
	}
	cx.RespondWithData(filtered)
}

// PUT: /client/{id} -> status update
func (cr *ClientController) Update(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
}

// PUT: /client
func (cr *ClientController) UpdateMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
}

// DELETE: /client/{id}
func (cr *ClientController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	queueMgr.DeleteClient(id)
	cx.RespondWithData("ok")
}

// DELETE: /client
func (cr *ClientController) DeleteMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
}
