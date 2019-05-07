package controller

import (
	//"encoding/json"
	"encoding/json"
	"fmt"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"

	//"github.com/MG-RAST/AWE/lib/core/cwl"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"

	//"github.com/davecgh/go-spew/spew"
	"io/ioutil"
	"net/http"
	"sort"
	"strconv"
	"strings"

	mgo "gopkg.in/mgo.v2"
)

// WorkController _
type WorkController struct{}

// OPTIONS: /work
func (cr *WorkController) Options(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithOK()
	return
}

// GET: /work/{id}
// get a workunit by id, read-only
func (cr *WorkController) Read(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	id = DecodeBase64(cx, id)

	work_id, err := core.New_Workunit_Unique_Identifier_FromString(id)
	if err != nil {
		cx.RespondWithErrorMessage("error parsing workunit identifier: "+id+" ("+err.Error()+")", http.StatusBadRequest)
		return
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	if (query.Has("datatoken") || query.Has("privateenv")) && query.Has("client") {
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
		// check that clientgroup auth token matches group of client
		clientid := query.Value("client")
		client, ok, xerr := core.QMgr.GetClient(clientid, true)
		if xerr != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
		if !ok {
			cx.RespondWithErrorMessage(e.ClientNotFound, http.StatusBadRequest)
			return
		}
		if cg != nil && client.Group != cg.Name {
			cx.RespondWithErrorMessage("Clientgroup name in token does not match that in the client configuration.", http.StatusBadRequest)
			return
		}

		if query.Has("datatoken") { //a client is requesting data token for this job

			token, err := core.QMgr.FetchDataToken(work_id, clientid)
			if err != nil {
				cx.RespondWithErrorMessage("error in getting token for job "+id, http.StatusBadRequest)
				return
			}
			//cx.RespondWithData(token)
			RespondTokenInHeader(cx, token)
			return
		}

		if query.Has("privateenv") { //a client is requesting data token for this job
			envs, err := core.QMgr.FetchPrivateEnv(work_id, clientid)
			if err != nil {
				cx.RespondWithErrorMessage("error in getting private environment for job "+id+" :"+err.Error(), http.StatusBadRequest)
				return
			}
			//cx.RespondWithData(token)
			RespondPrivateEnvInHeader(cx, envs)
			return
		}
	}

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

	jobid := work_id.JobId

	//job, err := core.LoadJob(jobid)
	//if err != nil {
	//	cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
	//	return
	//}
	acl, err := core.DBGetJobACL(jobid)
	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("job not found: "+jobid+" "+err.Error(), http.StatusBadRequest)
		}
		return
	}

	// User must have read permissions on job or be job owner or be an admin
	rights := acl.Check(u.Uuid)
	if acl.Owner != u.Uuid && rights["read"] == false && u.Admin == false {
		cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
		return
	}

	if query.Has("report") { //retrieve report: stdout or stderr or worknotes
		reportmsg, err := core.QMgr.GetReportMsg(work_id, query.Value("report"))
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData(reportmsg)
		return
	}

	if err != nil {
		if err.Error() != e.QueueEmpty {
			logger.Error("Err@work_Read:core.QMgr.GetWorkById(): " + err.Error())
		}
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	// Base case respond with workunit in json
	id_wui, err := core.New_Workunit_Unique_Identifier_FromString(id)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}
	workunit, err := core.QMgr.GetWorkByID(id_wui)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}
	cx.RespondWithData(workunit)
	return
}

// GET: /work
// checkout a workunit with earliest submission time
// to-do: to support more options for workunit checkout
func (cr *WorkController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	if !query.Has("client") { //view workunits
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

		// get pagination options
		limit := conf.DEFAULT_PAGE_SIZE
		offset := 0
		order := "info.submittime"
		direction := "desc"
		if query.Has("limit") {
			limit, err = strconv.Atoi(query.Value("limit"))
			if err != nil {
				cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
				return
			}
		}
		if query.Has("offset") {
			offset, err = strconv.Atoi(query.Value("offset"))
			if err != nil {
				cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
				return
			}
		}
		if query.Has("order") {
			order = query.Value("order")
		}
		if query.Has("direction") {
			direction = query.Value("direction")
		}

		var workunits []*core.Workunit
		if query.Has("state") {
			workunits = core.QMgr.ShowWorkunitsByUser(query.Value("state"), u)
		} else {
			workunits = core.QMgr.ShowWorkunitsByUser("", u)
		}

		// if using query syntax then do pagination and sorting
		if query.Has("query") {

			filterByID := ""
			if query.Has("id") {
				filterByID = query.Value("id")
			}

			filterByJobID := ""
			if query.Has("jobid") {
				filterByJobID = query.Value("jobid")
			}

			filtered_work := []*core.Workunit{}
			sorted_work := core.WorkunitsSortby{Order: order, Direction: direction, Workunits: workunits}
			sort.Sort(sorted_work)

			skip := 0
			count := 0
			for _, w := range sorted_work.Workunits {
				if skip < offset {
					skip += 1
					continue
				}
				if filterByID != "" {
					if filterByID != w.Id {
						continue
					}
				}
				if filterByJobID != "" {
					if filterByJobID != w.JobId {
						continue
					}
				}
				filtered_work = append(filtered_work, w)
				count += 1
				if count == limit {
					break
				}
			}
			cx.RespondWithPaginatedData(filtered_work, limit, offset, len(sorted_work.Workunits))

			// for _, w := range sorted_work.Workunits {
			// 	if w.Id == filterByID {
			// 		filtered_work = append(filtered_work, w)
			// 		cx.RespondWithPaginatedData(filtered_work, limit, offset, len(sorted_work.Workunits))
			// 		return
			// 	}
			// }

			// err = fmt.Errorf("workunit with id %s not found", filterByID)
			// cx.RespondWithErrorMessage(err.Error(), http.StatusNotFound)
			return

		} else {
			cx.RespondWithData(workunits)
			return
		}
	}

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

	// check that clientgroup auth token matches group of client
	clientid := query.Value("client")
	client, ok, err := core.QMgr.GetClient(clientid, true)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusInternalServerError)
		return
	}
	logger.Debug(3, "work request with clientid=%s", clientid)
	if !ok {
		cx.RespondWithErrorMessage(e.ClientNotFound, http.StatusBadRequest)
		return
	}

	if cg != nil && client.Group != cg.Name {
		cx.RespondWithErrorMessage("Clientgroup name in token does not match that in the client configuration.", http.StatusBadRequest)
		return
	}

	if core.Service == "proxy" { //drive proxy workStealer to checkout work from server
		core.ProxyWorkChan <- true
	}

	// get available disk space if sent
	availableBytes := int64(-1)
	if query.Has("available") {
		if value, errv := strconv.ParseInt(query.Value("available"), 10, 64); errv == nil {
			availableBytes = value
		}
	}

	//checkout a workunit in FCFS order
	workunits, err := core.QMgr.CheckoutWorkunits("FCFS", clientid, client, availableBytes, 1)

	if err != nil {

		err_str := err.Error()

		err = fmt.Errorf("(ReadMany GET /work) CheckoutWorkunits returns: clientid=%s;available=%d;error=%s", clientid, availableBytes, err.Error())

		if strings.Contains(err_str, e.QueueEmpty) ||
			strings.Contains(err_str, e.QueueSuspend) ||
			strings.Contains(err_str, e.NoEligibleWorkunitFound) ||
			strings.Contains(err_str, e.ClientNotFound) ||
			strings.Contains(err_str, e.ClientSuspended) {

			logger.Debug(3, err.Error())
		} else {
			logger.Error(err.Error())
		}

		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	//log event about workunit checkout (WO)
	workids := []string{}
	for _, work := range workunits {
		workids = append(workids, work.Id)
	}
	logger.Event(event.WORK_CHECKOUT, fmt.Sprintf("workids=%s;clientid=%s;available=%d", strings.Join(workids, ","), clientid, availableBytes))

	// Base case respond with node in json

	if len(workunits) == 0 {
		err = fmt.Errorf("(ReadMany GET /work) No workunits found, clientid=%s", clientid)
		logger.Error(err.Error())
		cx.RespondWithErrorMessage("(ReadMany GET /work) No workunits found", http.StatusBadRequest)
		return
	}

	workunit := workunits[0]
	workunit.State = core.WORK_STAT_RESERVED
	workunit.Client = clientid

	//test, err := json.Marshal(workunit)
	//if err != nil {
	//	panic("did not work")
	//}
	//fmt.Println("workunit: ")
	//fmt.Printf("workunit:\n %s\n", test)

	cx.RespondWithData(workunit)
	return
}

// PUT: /work/{id} -> status update
func (cr *WorkController) Update(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	id = DecodeBase64(cx, id)

	work_id, err := core.New_Workunit_Unique_Identifier_FromString(id)
	if err != nil {
		cx.RespondWithErrorMessage("error parsing workunit identifier: "+id+" ("+err.Error()+")", http.StatusBadRequest)
		return
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}
	if !query.Has("client") {
		cx.RespondWithErrorMessage("This request type requires the client=clientid parameter.", http.StatusBadRequest)
		return
	}

	// Check auth
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

	// check that clientgroup auth token matches group of client
	clientid := query.Value("client")
	client, ok, err := core.QMgr.GetClient(clientid, true)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusInternalServerError)
		return
	}
	if !ok {
		cx.RespondWithErrorMessage(e.ClientNotFound, http.StatusBadRequest)
		return
	}
	if cg != nil && client.Group != cg.Name {
		cx.RespondWithErrorMessage("Clientgroup name in token does not match that in the client configuration.", http.StatusBadRequest)
		return
	}

	// old-style
	var notice *core.Notice
	if query.Has("status") && query.Has("client") { //notify execution result: "done" or "fail"
		//work_id_object, err := core.New_Workunit_Unique_Identifier_FromString(work_id)
		//if err != nil {
		//	cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		//	return
		//}

		notice = &core.Notice{ID: work_id, Status: query.Value("status"), WorkerID: query.Value("client"), Notes: ""}
		// old-style
		if query.Has("computetime") {
			if comptime, err := strconv.Atoi(query.Value("computetime")); err == nil {
				notice.ComputeTime = comptime
			}
		}
	}

	params, files, err := ParseMultipartForm(cx.Request)

	if err != nil {
		cx.RespondWithErrorMessage("error getting form files: "+err.Error(), http.StatusInternalServerError)
		return
	}

	cwl_result_str, ok := params["cwl"]
	if ok {

		//fmt.Printf("cwl_result_str: %s\n", cwl_result_str)

		var notice_if map[string]interface{}
		err = json.Unmarshal([]byte(cwl_result_str), &notice_if)
		if err != nil {

			cx.RespondWithErrorMessage("Could not parse cwl form parameter: "+err.Error(), http.StatusInternalServerError)
			return
		}

		notice, err = core.NewNotice(notice_if, nil)
		if err != nil {
			cx.RespondWithErrorMessage("NewNotice returned: "+err.Error(), http.StatusInternalServerError)
			return
		}

		if notice.Status == "" {
			cx.RespondWithErrorMessage("Status empty!?"+err.Error(), http.StatusInternalServerError)
			return
		}

		//notice = &core.Notice{Id: work_id, Status: cwl_result.Status, WorkerId: cwl_result.WorkerId, Notes: ""}

	}
	//spew.Dump(params)
	//panic("arrrrgggh")

	if notice == nil {
		cx.RespondWithErrorMessage("Could not create a notice", http.StatusInternalServerError)
		return
	}

	// report may also be added for cwl workunit
	if query.Has("report") { // if "report" is specified in query, parse performance statistics or errlog
		if _, ok := files["perf"]; ok {
			err = core.QMgr.FinalizeWorkPerf(work_id, files["perf"].Path)
			if err != nil {
				cx.RespondWithErrorMessage(err.Error(), http.StatusInternalServerError)
				return
			}
		}
		for _, log := range conf.WORKUNIT_LOGS {
			if _, ok := files[log]; ok {
				if log == "worknotes" {
					// add worknotes to notice
					if text, err := ioutil.ReadFile(files[log].Path); err == nil {
						notice.Notes = string(text)
					}
				} else if log == "stderr" {
					// add stderr to notice
					if text, err := ioutil.ReadFile(files[log].Path); err == nil {
						// only save last 5000 chars of string
						err_str := string(text)
						if len(err_str) > 5000 {
							notice.Stderr = string(err_str[len(err_str)-5000:])
						} else {
							notice.Stderr = err_str
						}
					}
				}
				// move / save log file
				core.QMgr.SaveStdLog(work_id, log, files[log].Path)
			}
		}
	}

	core.QMgr.NotifyWorkStatus(*notice)
	//}
	cx.RespondWithData("ok")
	return
}
