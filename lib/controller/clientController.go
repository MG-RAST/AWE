package controller

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"
	"net/http"
	"strconv"
	"strings"
)

type ClientController struct{}

// OPTIONS: /client
func (cr *ClientController) Options(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithOK()
	return
}

// POST: /client - register a new client
func (cr *ClientController) Create(cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)

	cg, err := request.AuthenticateClientGroup(cx.Request)
	if err != nil {
		if err.Error() == e.NoAuth || err.Error() == e.UnAuth || err.Error() == e.InvalidAuth {
			if conf.CLIENT_AUTH_REQ == true {
				cx.RespondWithError(http.StatusUnauthorized)
				return
			}
		} else {
			logger.Error("Err@AuthenticateClientGroup: " + err.Error())
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

	client, err := core.QMgr.RegisterNewClient(files, cg)
	if err != nil {
		msg := "Error in registering new client:" + err.Error()
		logger.Error(msg)
		cx.RespondWithErrorMessage(msg, http.StatusBadRequest)
		return
	}

	//log event about client registration (CR)
	logger.Event(event.CLIENT_REGISTRATION, "clientid="+client.Id+";name="+client.Name+";host="+client.Host+";group="+client.Group+";instance_id="+client.InstanceId+";instance_type="+client.InstanceType+";domain="+client.Domain)

	cx.RespondWithData(client)
	return
}

// GET: /client/{id}
func (cr *ClientController) Read(id string, cx *goweb.Context) {
	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	if query.Has("heartbeat") { //handle heartbeat
		cg, err := request.AuthenticateClientGroup(cx.Request)
		if err != nil {
			if err.Error() == e.NoAuth || err.Error() == e.UnAuth || err.Error() == e.InvalidAuth {
				if conf.CLIENT_AUTH_REQ == true {
					cx.RespondWithError(http.StatusUnauthorized)
					return
				}
			} else {
				logger.Error("Err@AuthenticateClientGroup: " + err.Error())
				cx.RespondWithError(http.StatusInternalServerError)
				return
			}
		}

		hbmsg, err := core.QMgr.ClientHeartBeat(id, cg)
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		} else {
			cx.RespondWithData(hbmsg)
		}
		return
	}

	LogRequest(cx.Request) //skip heartbeat in access log

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous read is allowed, use the public user
	if u == nil {
		if conf.ANON_READ == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	client, err := core.QMgr.GetClientByUser(id, u)
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
	return
}

// GET: /client
func (cr *ClientController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous read is allowed, use the public user
	if u == nil {
		if conf.ANON_READ == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	clients := core.QMgr.GetAllClientsByUser(u)

	query := &Query{Li: cx.Request.URL.Query()}
	filtered := []*core.Client{}
	if query.Has("busy") {
		for _, client := range clients {
			if client.Current_work_length() > 0 {
				filtered = append(filtered, client)
			}
		}
	} else if query.Has("group") {
		for _, client := range clients {
			if client.Group == query.Value("group") {
				filtered = append(filtered, client)
			}
		}
	} else if query.Has("status") {
		for _, client := range clients {
			stat := strings.Split(client.Status, "-")
			if client.Status == query.Value("status") {
				filtered = append(filtered, client)
			} else if (len(stat) == 2) && (stat[1] == query.Value("status")) {
				filtered = append(filtered, client)
			}
		}
	} else if query.Has("app") {
		for _, client := range clients {
			for _, app := range client.Apps {
				if app == query.Value("app") {
					filtered = append(filtered, client)
				}
			}
		}
	} else {
		filtered = clients
	}
	cx.RespondWithData(filtered)
	return
}

// PUT: /client/{id} -> status update
func (cr *ClientController) Update(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous read is allowed, use the public user
	if u == nil {
		if conf.ANON_WRITE == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}
	if query.Has("subclients") { //update the number of subclients for a proxy
		if count, err := strconv.Atoi(query.Value("subclients")); err != nil {
			cx.RespondWithError(http.StatusNotImplemented)
		} else {
			core.QMgr.UpdateSubClientsByUser(id, count, u)
			cx.RespondWithData("ok")
		}
		return
	}
	if query.Has("suspend") { //resume the suspended client
		if err := core.QMgr.SuspendClientByUser(id, u); err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		} else {
			cx.RespondWithData("client suspended")
		}
		return
	}
	if query.Has("resume") { //resume the suspended client
		if err := core.QMgr.ResumeClientByUser(id, u); err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		} else {
			cx.RespondWithData("client resumed")
		}
		return
	}
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// PUT: /client
func (cr *ClientController) UpdateMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous read is allowed, use the public user
	if u == nil {
		if conf.ANON_WRITE == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}
	if query.Has("resumeall") { //resume the suspended client
		num := core.QMgr.ResumeSuspendedClientsByUser(u)
		cx.RespondWithData(fmt.Sprintf("%d suspended clients resumed", num))
		return
	}
	if query.Has("suspendall") { //resume the suspended client
		num := core.QMgr.SuspendAllClientsByUser(u)
		cx.RespondWithData(fmt.Sprintf("%d clients suspended", num))
		return
	}
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// DELETE: /client/{id}
func (cr *ClientController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous read is allowed, use the public user
	if u == nil {
		if conf.ANON_DELETE == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	if err := core.QMgr.DeleteClientByUser(id, u); err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
	} else {
		cx.RespondWithData("client deleted")
	}
	return
}

// DELETE: /client
func (cr *ClientController) DeleteMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}
