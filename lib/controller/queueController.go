package controller

import (
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/golib/goweb"
	"net/http"
)

type QueueController struct{}

var queueTypes = []string{"job", "task", "work", "client"}

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
	query := &Query{Li: cx.Request.URL.Query()}

	// unathenticated queue status, numbers only
	if query.Empty() {
		statusText := core.QMgr.GetTextStatus()
		cx.RespondWithData(statusText)
		return
	}
	if query.Has("json") {
		statusJson := core.QMgr.GetJsonStatus()
		cx.RespondWithData(statusJson)
		return
	}

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}
	// must be admin user
	if u == nil || u.Admin == false {
		cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
		return
	}
	// check if valid queue type requested
	for _, q := range queueTypes {
		if query.Has(q) {
			queueData := core.QMgr.GetQueue(q)
			cx.RespondWithData(queueData)
			return
		}
	}
	// build set of running jobs based on clientgroup
	if query.Has("clientgroup") {
		if query.Value("clientgroup") == "" {
			cx.RespondWithErrorMessage("missing required clientgroup name", http.StatusBadRequest)
			return
		}
		cg, err := core.LoadClientGroupByName(query.Value("clientgroup"))
		if err != nil {
			cx.RespondWithErrorMessage("unable to retrieve clientgroup '"+query.Value("clientgroup")+"': "+err.Error(), http.StatusBadRequest)
			return
		}
		if cg == nil {
			cx.RespondWithErrorMessage("clientgroup '"+query.Value("clientgroup")+"' does not exist", http.StatusBadRequest)
			return
		}
		// User must have read permissions on clientgroup or be clientgroup owner or be an admin or the clientgroup is publicly readable.
		// The other possibility is that public read of clientgroups is enabled and the clientgroup is publicly readable.
		rights := cg.Acl.Check(u.Uuid)
		public_rights := cg.Acl.Check("public")
		if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || rights["read"] == true || u.Admin == true || public_rights["read"] == true)) ||
			(u.Uuid == "public" && conf.ANON_CG_READ == true && public_rights["read"] == true) {
			// get running jobs for clients for clientgroup
			jobs := []*core.Job{}
			for _, client := range core.QMgr.GetAllClients() {
				if client.Group == cg.Name {
					client.Current_work_lock.RLock()
					for wid, _ := range client.Current_work {
						jid, _ := core.GetJobIdByWorkId(wid)
						if job, err := core.LoadJob(jid); err == nil {
							jobs = append(jobs, job)
						}
					}
					client.Current_work_lock.RUnlock()
				}
			}
			cx.RespondWithData(jobs)
			return
		}
		cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
		return
	}

	cx.RespondWithErrorMessage("requested queue operation not supported", http.StatusBadRequest)
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

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}
	// must be admin user
	if u == nil || u.Admin == false {
		cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
		return
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	if query.Has("resume") {
		core.QMgr.ResumeQueue()
		logger.Event(event.QUEUE_RESUME, "user="+u.Username)
		cx.RespondWithData("work queue resumed")
		return
	}
	if query.Has("suspend") {
		core.QMgr.SuspendQueue()
		logger.Event(event.QUEUE_SUSPEND, "user="+u.Username)
		cx.RespondWithData("work queue suspended")
		return
	}

	cx.RespondWithErrorMessage("requested queue operation not supported", http.StatusBadRequest)
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
