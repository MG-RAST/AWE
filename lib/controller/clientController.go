package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"strconv"
	"strings"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"
)

// ClientController _
type ClientController struct{}

// Options OPTIONS: /client
func (cr *ClientController) Options(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithOK()
	return
}

// Create POST: /client - register a new client
func (cr *ClientController) Create(cx *goweb.Context) {
	logger.Debug(3, "POST /client")
	// Log Request and check for Auth
	LogRequest(cx.Request)

	cg, done := GetClientGroup(cx)
	if done {
		return
	}
	// Parse uploaded form

	_, files, err := ParseMultipartForm(cx.Request)
	if err != nil {
		if err.Error() != "request Content-Type isn't multipart/form-data" {
			logger.Error("(ClientController/Create) Error parsing form: " + err.Error())
			cx.RespondWithErrorMessage("(ClientController/Create) Error parsing form: "+err.Error(), http.StatusBadRequest)
			return
		}
	}

	logger.Debug(3, "POST /client, call RegisterNewClient")
	client, err := core.QMgr.RegisterNewClient(files, cg)
	if err != nil {
		msg := "Error in registering new client:" + err.Error()
		logger.Error(msg)
		cx.RespondWithErrorMessage(msg, http.StatusBadRequest)
		return
	}

	//log event about client registration (CR)
	logger.Event(event.CLIENT_REGISTRATION, "clientid="+client.ID+";hostname="+client.Hostname+";hostip="+client.HostIP+";group="+client.Group+";instance_id="+client.InstanceID+";instance_type="+client.InstanceType+";domain="+client.Domain)

	// rlock, err := client.RLockNamed("ClientController/Create")
	// if err != nil {
	// 	msg := "Lock error:" + err.Error()
	// 	logger.Error(msg)
	// 	cx.RespondWithErrorMessage(msg, http.StatusBadRequest)
	// }
	//defer client.RUnlockNamed(rlock)

	rr := core.NewRegistrationResponse()

	cx.RespondWithData(rr)

	return
}

// GET: /client/{id}
func (cr *ClientController) Read(id string, cx *goweb.Context) {
	// Gather query params

	query := &Query{Li: cx.Request.URL.Query()}
	if query.Has("heartbeat") { // OLD heartbeat
		cx.RespondWithErrorMessage("Please update to newer version of awe-worker. Old heartbeat mechanism not supported anymore.", http.StatusBadRequest)
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
	rlock, err := client.RLockNamed("ClientController/Create")
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusInternalServerError)
		return
	}
	defer client.RUnlockNamed(rlock)
	cx.RespondWithData(client)
	return
}

// ReadMany GET: /client
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

	clients, err := core.QMgr.GetAllClientsByUser(u)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusInternalServerError)
		return
	}

	query := &Query{Li: cx.Request.URL.Query()}
	//filtered := []*core.Client{}
	filtered := core.Clients{}

	if query.Has("busy") {
		for _, client := range clients {
			workLength, err := client.CurrentWork.Length(true)
			if err != nil {
				continue
			}
			if workLength > 0 {
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
			status, xerr := client.GetNewStatus(false)
			if xerr != nil {
				continue
			}
			stat := strings.Split(status, "-")
			if status == query.Value("status") {
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
		for _, client := range clients {
			filtered = append(filtered, client)
		}
	}

	filtered.RLockRecursive()
	defer filtered.RUnlockRecursive()

	cx.RespondWithData(filtered)
	return
}

// Update PUT: /client/{id} -> status update
func (cr *ClientController) Update(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	if query.Has("heartbeat") { //handle heartbeat

		cg, done := GetClientGroup(cx)
		if done {
			return
		}

		const maxMemory = 1024

		r := cx.Request
		workerStatusBytes, err := ioutil.ReadAll(r.Body)
		defer r.Body.Close()

		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}

		worker_status := core.WorkerState{}
		err = json.Unmarshal(workerStatusBytes, &worker_status)
		if err != nil {
			err = fmt.Errorf("%s, worker_status_bytes: %s", err, string(workerStatusBytes[:]))
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			//cx.Respond(data interface{}, statusCode int, []string{err.Error()}, cx)
			return
		}
		worker_status.CurrentWork.Init("Current_work")

		hbmsg, err := core.QMgr.ClientHeartBeat(id, cg, worker_status)
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		} else {
			cx.RespondWithData(hbmsg)
		}
		return

	}

	u, done := GetAuthorizedUser(cx)
	if done {
		return
	}

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
		if err := core.QMgr.SuspendClientByUser(id, u, "request by api call"); err != nil {
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
		num, err := core.QMgr.SuspendAllClientsByUser(u, "request by api call")
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		}
		cx.RespondWithData(fmt.Sprintf("%d clients suspended", num))
		return
	}
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// DELETE: /client/{id}
func (cr *ClientController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithErrorMessage("this functionality has been removed from AWE", http.StatusBadRequest)

	// Try to authenticate user.
	//u, err := request.Authenticate(cx.Request)
	//if err != nil && err.Error() != e.NoAuth {
	//	cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
	//	return
	//}

	// If no auth was provided, and anonymous read is allowed, use the public user
	//if u == nil {
	//	if conf.ANON_DELETE == true {
	//		u = &user.User{Uuid: "public"}
	//	} else {
	//		cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
	//		return
	//	}
	//}

	//if err := core.QMgr.DeleteClientByUser(id, u); err != nil {
	//	cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
	//} else {
	//	cx.RespondWithData("client deleted")
	//}
	return
}

// DELETE: /client
func (cr *ClientController) DeleteMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}
