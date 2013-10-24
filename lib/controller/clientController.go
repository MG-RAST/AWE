package controller

import (
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/golib/goweb"
	"net/http"
	"strconv"
)

type ClientController struct{}

// POST: /client
func (cr *ClientController) Create(cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)

	_, err := request.Authenticate(cx.Request)
	if err != nil {
		if err.Error() == e.NoAuth || err.Error() == e.UnAuth {
			cx.RespondWithError(http.StatusUnauthorized)
			return
		} else {
			logger.Error("Err@user_Read: " + err.Error())
			cx.RespondWithError(http.StatusInternalServerError)
			return
		}
	}

	// Parse uploaded form
	_, files, err := ParseMultipartForm(cx.Request)
	if err != nil {
		if err.Error() != "request Content-Type isn't multipart/form-data" {
			logger.Error("Error parsing form: " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
			return
		}
	}

	client, err := core.QMgr.RegisterNewClient(files)
	if err != nil {
		msg := "Error in registering new client:" + err.Error()
		logger.Error(msg)
		cx.RespondWithErrorMessage(msg, http.StatusBadRequest)
		return
	}

	//log event about client registration (CR)
	logger.Event(event.CLIENT_REGISTRATION, "clientid="+client.Id+";name="+client.Name+";host="+client.Host)

	cx.RespondWithData(client)
	return
}

// GET: /client/{id}
func (cr *ClientController) Read(id string, cx *goweb.Context) {
	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	if query.Has("heartbeat") { //handle heartbeat
		hbmsg, err := core.QMgr.ClientHeartBeat(id)
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		} else {
			cx.RespondWithData(hbmsg)
		}
		return
	}

	LogRequest(cx.Request) //skip heartbeat in access log

	client, err := core.QMgr.GetClient(id)
	if err != nil {
		if err.Error() == e.ClientNotFound {
			cx.RespondWithErrorMessage(e.ClientNotFound, http.StatusBadRequest)
		} else {
			logger.Error("Error in GET client:" + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
		}
		return
	}
	cx.RespondWithData(client)
}

// GET: /client
func (cr *ClientController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	clients := core.QMgr.GetAllClients()
	if len(clients) == 0 {
		cx.RespondWithErrorMessage(e.ClientNotFound, http.StatusBadRequest)
		return
	}

	query := &Query{Li: cx.Request.URL.Query()}
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

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}
	if query.Has("subclients") { // to resume a suspended job
		if count, err := strconv.Atoi(query.Value("subclients")); err != nil {
			cx.RespondWithError(http.StatusNotImplemented)
		} else {
			core.QMgr.UpdateSubClients(id, count)
			cx.RespondWithData("ok")
		}
		return
	}
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
	core.QMgr.DeleteClient(id)
	cx.RespondWithData("ok")
}

// DELETE: /client
func (cr *ClientController) DeleteMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
}
