package controller

import (
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/util"
	"github.com/jaredwilkening/goweb"
	"io/ioutil"
	"net/http"
	"strings"
)

type WorkController struct{}

// GET: /work/{id}
// get a workunit by id, read-only
func (cr *WorkController) Read(id string, cx *goweb.Context) {
	util.LogRequest(cx.Request)

	// Load workunit by id
	workunit, err := core.QMgr.GetWorkById(id)

	if err != nil {
		if err.Error() != e.QueueEmpty {
			logger.Error("Err@work_Read:core.QMgr.GetWorkById(): " + err.Error())
		}
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}
	// Base case respond with workunit in json
	cx.RespondWithData(workunit)
	return
}

// GET: /work
// checkout a workunit with earliest submission time
// to-do: to support more options for workunit checkout
func (cr *WorkController) ReadMany(cx *goweb.Context) {

	// Gather query params
	query := &util.Query{Li: cx.Request.URL.Query()}

	if !query.Has("client") { //view workunits
		var workunits []*core.Workunit
		if query.Has("state") {
			workunits = core.QMgr.ShowWorkunits(query.Value("state"))
		} else {
			workunits = core.QMgr.ShowWorkunits("")
		}
		cx.RespondWithData(workunits)
		return
	}

	//checkout a workunit in FCFS order
	clientid := query.Value("client")
	workunits, err := core.QMgr.CheckoutWorkunits("FCFS", clientid, 1)

	if err != nil {
		if err.Error() != e.QueueEmpty && err.Error() != e.NoEligibleWorkunitFound {
			logger.Error("Err@work_ReadMany:core.QMgr.GetWorkByFCFS(): " + err.Error() + ";client=" + clientid)
		}
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	//log access info only when the queue is not empty, save some log
	util.LogRequest(cx.Request)

	//log event about workunit checkout (WO)
	workids := []string{}
	for _, work := range workunits {
		workids = append(workids, work.Id)
	}

	logger.Event(event.WORK_CHECKOUT,
		"workids="+strings.Join(workids, ","),
		"clientid="+clientid)

	// Base case respond with node in json
	cx.RespondWithData(workunits[0])
	return
}

// PUT: /work/{id} -> status update
func (cr *WorkController) Update(id string, cx *goweb.Context) {
	// Log Request and check for Auth
	util.LogRequest(cx.Request)
	// Gather query params
	query := &util.Query{Li: cx.Request.URL.Query()}

	if query.Has("status") && query.Has("client") { //notify execution result: "done" or "fail"
		notice := core.Notice{WorkId: id, Status: query.Value("status"), ClientId: query.Value("client"), Notes: ""}
		if query.Has("report") { // if "report" is specified in query, parse performance statistics or errlog
			if _, files, err := util.ParseMultipartForm(cx.Request); err == nil {
				if _, ok := files["perf"]; ok {
					core.QMgr.FinalizeWorkPerf(id, files["perf"].Path)
				}
				if _, ok := files["notes"]; ok {
					if notes, err := ioutil.ReadFile(files["notes"].Path); err == nil {
						notice.Notes = string(notes)
					}
				}
			}
		}
		core.QMgr.NotifyWorkStatus(notice)
	}
	cx.RespondWithData("ok")
	return
}
