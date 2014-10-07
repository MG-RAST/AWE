package controller

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/foreign/taverna"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"
	"github.com/MG-RAST/golib/mgo"
	"github.com/MG-RAST/golib/mgo/bson"
	"net/http"
	"strconv"
	"strings"
	"time"
)

type JobController struct{}

// OPTIONS: /job
func (cr *JobController) Options(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithOK()
	return
}

// POST: /job
func (cr *JobController) Create(cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous write is allowed, use the public user
	if u == nil {
		if conf.ANON_WRITE == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	// Parse uploaded form
	params, files, err := ParseMultipartForm(cx.Request)

	if err != nil {
		if err.Error() == "request Content-Type isn't multipart/form-data" {
			cx.RespondWithErrorMessage("No job file is submitted", http.StatusBadRequest)
		} else {
			// Some error other than request encoding. Theoretically
			// could be a lost db connection between user lookup and parsing.
			// Blame the user, Its probaby their fault anyway.
			logger.Error("Error parsing form: " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
		}
		return
	}

	_, has_upload := files["upload"]
	_, has_awf := files["awf"]

	if !has_upload && !has_awf {
		cx.RespondWithErrorMessage("No job script or awf is submitted", http.StatusBadRequest)
		return
	}

	//send job submission request and get back an assigned job number (jid)
	var jid string
	jid, err = core.QMgr.JobRegister()
	if err != nil {
		logger.Error("Err@job_Create:GetNextJobNum: " + err.Error())
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	var job *core.Job
	job, err = core.CreateJobUpload(u, params, files, jid)
	if err != nil {
		logger.Error("Err@job_Create:CreateJobUpload: " + err.Error())
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	if token, err := request.RetrieveToken(cx.Request); err == nil {
		job.SetDataToken(token)
	}

	core.QMgr.EnqueueTasksByJobId(job.Id, job.TaskList())

	//log event about job submission (JB)
	logger.Event(event.JOB_SUBMISSION, "jobid="+job.Id+";jid="+job.Jid+";name="+job.Info.Name+";project="+job.Info.Project+";user="+job.Info.User)
	cx.RespondWithData(job)
	return
}

// GET: /job/{id}
func (cr *JobController) Read(id string, cx *goweb.Context) {
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

	// Load job by id
	job, err := core.LoadJob(id)

	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			// logger.Error("Err@job_Read:LoadJob: " + id + ":" + err.Error())
			cx.RespondWithErrorMessage("job not found:"+id, http.StatusBadRequest)
		}
		return
	}

	// User must have read permissions on job or be job owner or be an admin
	rights := job.Acl.Check(u.Uuid)
	if job.Acl.Owner != u.Uuid && rights["read"] == false && u.Admin == false {
		cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
		return
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	if query.Has("perf") {
		//Load job perf by id
		perf, err := core.LoadJobPerf(id)
		if err != nil {
			if err == mgo.ErrNotFound {
				cx.RespondWithNotFound()
			} else {
				// In theory the db connection could be lost between
				// checking user and load but seems unlikely.
				logger.Error("Err@LoadJobPerf: " + id + ":" + err.Error())
				cx.RespondWithErrorMessage("job perf stats not found:"+id, http.StatusBadRequest)
			}
			return
		}
		cx.RespondWithData(perf)
		return //done with returning perf, no need to load job further.
	}

	if core.QMgr.IsJobRegistered(id) {
		job.Registered = true
	} else {
		job.Registered = false
	}

	if query.Has("export") {
		target := query.Value("export")
		if target == "" {
			cx.RespondWithErrorMessage("lacking stage id from which the recompute starts", http.StatusBadRequest)
			return
		} else if target == "taverna" {
			wfrun, err := taverna.ExportWorkflowRun(job)
			if err != nil {
				cx.RespondWithErrorMessage("failed to export job to taverna workflowrun:"+id, http.StatusBadRequest)
				return
			}
			cx.RespondWithData(wfrun)
			return
		}
	}

	// Base case respond with job in json
	cx.RespondWithData(job)
	return
}

// GET: /job
// To do:
// - Iterate job queries
func (cr *JobController) ReadMany(cx *goweb.Context) {
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

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	// Setup query and jobs objects
	q := bson.M{}
	jobs := core.Jobs{}

	// Add authorization checking to query if the user is not an admin
	if u.Admin == false {
		q["$or"] = []bson.M{bson.M{"acl.read": "public"}, bson.M{"acl.read": u.Uuid}, bson.M{"acl.owner": u.Uuid}, bson.M{"acl": bson.M{"$exists": "false"}}}
	}

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

	// Gather params to make db query. Do not include the
	// following list.
	skip := map[string]int{"limit": 1,
		"offset":     1,
		"query":      1,
		"recent":     1,
		"order":      1,
		"direction":  1,
		"active":     1,
		"suspend":    1,
		"registered": 1,
		"verbosity":  1,
		"userattr":   1,
	}
	if query.Has("query") {
		const shortForm = "2006-01-02"
		date_query := bson.M{}
		for key, val := range query.All() {
			_, s := skip[key]
			if !s {
				// special case for date range, either full date-time or just date
				if (key == "date_start") || (key == "date_end") {
					opr := "$gte"
					if key == "date_end" {
						opr = "$lt"
					}
					if t_long, err := time.Parse(time.RFC3339, val[0]); err != nil {
						if t_short, err := time.Parse(shortForm, val[0]); err != nil {
							cx.RespondWithErrorMessage("Invalid datetime format: "+val[0], http.StatusBadRequest)
							return
						} else {
							date_query[opr] = t_short
						}
					} else {
						date_query[opr] = t_long
					}
				} else {
					// handle either multiple values for key, or single comma-spereated value
					if len(val) == 1 {
						queryvalues := strings.Split(val[0], ",")
						q[key] = bson.M{"$in": queryvalues}
					} else if len(val) > 1 {
						q[key] = bson.M{"$in": val}
					}
				}
			}
		}
		// add submittime and completedtime range query
		if len(date_query) > 0 {
			q["$or"] = []bson.M{bson.M{"info.submittime": date_query}, bson.M{"info.completedtime": date_query}}
		}
	} else if query.Has("active") {
		q["state"] = bson.M{"$in": core.JOB_STATS_ACTIVE}
	} else if query.Has("suspend") {
		q["state"] = core.JOB_STAT_SUSPEND
	} else if query.Has("registered") {
		q["state"] = bson.M{"$in": core.JOB_STATS_REGISTERED}
	}

	//getting real active (in-progress) job (some jobs are in "submitted" states but not in the queue,
	//because they may have failed and not recovered from the mongodb).
	if query.Has("active") {
		err := jobs.GetAll(q, order, direction)
		if err != nil {
			logger.Error("err " + err.Error())
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}

		filtered_jobs := []core.Job{}
		act_jobs := core.QMgr.GetActiveJobs()
		length := jobs.Length()

		skip := 0
		count := 0
		for i := 0; i < length; i++ {
			job := jobs.GetJobAt(i)
			if _, ok := act_jobs[job.Id]; ok {
				if skip < offset {
					skip += 1
					continue
				}
				job.Registered = true
				filtered_jobs = append(filtered_jobs, job)
				count += 1
				if count == limit {
					break
				}
			}
		}
		cx.RespondWithPaginatedData(filtered_jobs, limit, offset, len(act_jobs))
		return
	}

	//geting suspended job in the current queue (excluding jobs in db but not in qmgr)
	if query.Has("suspend") {
		err := jobs.GetAll(q, order, direction)
		if err != nil {
			logger.Error("err " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
			return
		}

		filtered_jobs := []core.Job{}
		suspend_jobs := core.QMgr.GetSuspendJobs()
		length := jobs.Length()

		skip := 0
		count := 0
		for i := 0; i < length; i++ {
			job := jobs.GetJobAt(i)
			if _, ok := suspend_jobs[job.Id]; ok {
				if skip < offset {
					skip += 1
					continue
				}
				job.Registered = true
				filtered_jobs = append(filtered_jobs, job)
				count += 1
				if count == limit {
					break
				}
			}
		}
		cx.RespondWithPaginatedData(filtered_jobs, limit, offset, len(suspend_jobs))
		return
	}

	if query.Has("registered") {
		err := jobs.GetAll(q, order, direction)
		if err != nil {
			logger.Error("err " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
			return
		}

		paged_jobs := []core.Job{}
		registered_jobs := []core.Job{}
		length := jobs.Length()

		total := 0
		for i := 0; i < length; i++ {
			job := jobs.GetJobAt(i)
			if core.QMgr.IsJobRegistered(job.Id) {
				job.Registered = true
				registered_jobs = append(registered_jobs, job)
				total += 1
			}
		}
		count := 0
		for i := offset; i < len(registered_jobs); i++ {
			paged_jobs = append(paged_jobs, registered_jobs[i])
			count += 1
			if count == limit {
				break
			}
		}
		cx.RespondWithPaginatedData(paged_jobs, limit, offset, total)
		return
	}

	if query.Has("verbosity") && (query.Value("verbosity") == "minimal") {
		// TODO - have mongo query only return fields needed to populate JobMin struct
		total, err := jobs.GetPaginated(q, limit, offset, order, direction)
		if err != nil {
			logger.Error("err " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
			return
		}
		minimal_jobs := []core.JobMin{}
		length := jobs.Length()
		for i := 0; i < length; i++ {
			job := jobs.GetJobAt(i)
			// create and populate minimal job
			mjob := core.JobMin{}
			mjob.Id = job.Id
			mjob.Name = job.Info.Name
			mjob.SubmitTime = job.Info.SubmitTime
			mjob.CompletedTime = job.Info.CompletedTime
			// get size of input
			var size_sum int64 = 0
			for _, v := range job.Tasks[0].Inputs {
				size_sum = size_sum + v.Size
			}
			mjob.Size = size_sum
			// add userattr fields
			if query.Has("userattr") {
				mjob.UserAttr = map[string]string{}
				for _, attr := range query.List("userattr") {
					if val, ok := job.Info.UserAttr[attr]; ok {
						mjob.UserAttr[attr] = val
					}
				}
			}

			if (job.State == "completed") || (job.State == "deleted") {
				// if completed or deleted move on
				mjob.State = job.State
				mjob.Task = -1
			} else if job.State == "suspend" {
				// get failed task
				mjob.State = "suspend"
				parts := strings.Split(job.LastFailed, "_")
				if (len(parts) == 2) || (len(parts) == 3) {
					if tid, err := strconv.Atoi(parts[1]); err != nil {
						mjob.Task = -1
					} else {
						mjob.Task = tid
					}
				} else {
					mjob.Task = -1
				}
			} else {
				// determine most recent task of states (or oldest for pending or init)
				state := map[string]int{}
				for j := 0; j < len(job.Tasks); j++ {
					task := job.Tasks[j]
					if (task.State == "pending") || (task.State == "init") {
						if _, ok := state[task.State]; !ok {
							state[task.State] = j
						}
					} else {
						state[task.State] = j
					}
				}
				// fancy logic to get "current" state and its task
				ordered := [4]string{"in-progress", "queued", "pending", "init"}
				for s := range ordered {
					if val, ok := state[ordered[s]]; ok {
						mjob.State = ordered[s]
						mjob.Task = val
						break
					}
				}
				// catch-all for bad state
				if mjob.State == "" {
					mjob.State = "unknown"
					mjob.Task = -1
				}
			}
			minimal_jobs = append(minimal_jobs, mjob)
		}
		cx.RespondWithPaginatedData(minimal_jobs, limit, offset, total)
		return
	}

	total, err := jobs.GetPaginated(q, limit, offset, order, direction)
	if err != nil {
		logger.Error("err " + err.Error())
		cx.RespondWithError(http.StatusBadRequest)
		return
	}
	filtered_jobs := []core.Job{}
	length := jobs.Length()
	for i := 0; i < length; i++ {
		job := jobs.GetJobAt(i)
		if core.QMgr.IsJobRegistered(job.Id) {
			job.Registered = true
		} else {
			job.Registered = false
		}
		filtered_jobs = append(filtered_jobs, job)
	}
	cx.RespondWithPaginatedData(filtered_jobs, limit, offset, total)
	return
}

// PUT: /job
func (cr *JobController) UpdateMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous write is allowed, use the public user
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
	if query.Has("resumeall") { //resume the suspended job
		num := core.QMgr.ResumeSuspendedJobsByUser(u)
		cx.RespondWithData(fmt.Sprintf("%d suspended jobs resumed", num))
		return
	}
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// PUT: /job/{id} -> used for job manipulation
func (cr *JobController) Update(id string, cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous write is allowed, use the public user
	if u == nil {
		if conf.ANON_WRITE == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	// Load job by id
	job, err := core.LoadJob(id)

	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			// logger.Error("Err@job_Read:LoadJob: " + id + ":" + err.Error())
			cx.RespondWithErrorMessage("job not found:"+id, http.StatusBadRequest)
		}
		return
	}

	// User must have write permissions on job or be job owner or be an admin
	rights := job.Acl.Check(u.Uuid)
	if job.Acl.Owner != u.Uuid && rights["write"] == false && u.Admin == false {
		cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
		return
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}
	if query.Has("resume") { // to resume a suspended job
		if err := core.QMgr.ResumeSuspendedJob(id); err != nil {
			cx.RespondWithErrorMessage("fail to resume job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job resumed: " + id)
		return
	}
	if query.Has("suspend") { // to suspend an in-progress job
		if err := core.QMgr.SuspendJob(id, "manually suspended", ""); err != nil {
			cx.RespondWithErrorMessage("fail to suspend job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job suspended: " + id)
		return
	}
	if query.Has("resubmit") || query.Has("reregister") { // to re-submit a job from mongodb
		if err := core.QMgr.ResubmitJob(id); err != nil {
			cx.RespondWithErrorMessage("fail to resubmit job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job resubmitted: " + id)
		return
	}
	if query.Has("recompute") { // to recompute a job from task i, the successive/downstream tasks of i will all be computed
		stage := query.Value("recompute")
		if stage == "" {
			cx.RespondWithErrorMessage("lacking stage id from which the recompute starts", http.StatusBadRequest)
			return
		}
		if err := core.QMgr.RecomputeJob(id, stage); err != nil {
			cx.RespondWithErrorMessage("fail to recompute job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job recompute started: " + id)
		return
	}
	if query.Has("clientgroup") { // change the clientgroup attribute of the job
		newgroup := query.Value("clientgroup")
		if newgroup == "" {
			cx.RespondWithErrorMessage("lacking groupname", http.StatusBadRequest)
			return
		}
		if err := core.QMgr.UpdateGroup(id, newgroup); err != nil {
			cx.RespondWithErrorMessage("fail to update group for job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job group updated: " + id + " to " + newgroup)
		return
	}
	if query.Has("priority") { // change the clientgroup attribute of the job
		priority_str := query.Value("priority")
		if priority_str == "" {
			cx.RespondWithErrorMessage("lacking priority value (0-3)", http.StatusBadRequest)
			return
		}
		priority, err := strconv.Atoi(priority_str)
		if err != nil {
			cx.RespondWithErrorMessage("need int for priority value (0-3) "+err.Error(), http.StatusBadRequest)
			return
		}
		if err := core.QMgr.UpdatePriority(id, priority); err != nil {
			cx.RespondWithErrorMessage("fail to priority for job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job priority updated: " + id + " to " + priority_str)
		return
	}

	if query.Has("settoken") { // set data token
		token, err := request.RetrieveToken(cx.Request)
		if err != nil {
			cx.RespondWithErrorMessage("fail to retrieve token for job, pls set token in header: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}

		job, err := core.LoadJob(id)
		if err != nil {
			if err == mgo.ErrNotFound {
				cx.RespondWithNotFound()
			} else {
				logger.Error("Err@job_Read:LoadJob: " + id + ":" + err.Error())
				cx.RespondWithErrorMessage("job not found:"+id, http.StatusBadRequest)
			}
			return
		}
		job.SetDataToken(token)
		cx.RespondWithData("data token set for job: " + id)
		return
	}

	cx.RespondWithData("requested job operation not supported")
	return
}

// DELETE: /job/{id}
func (cr *JobController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
	}

	// If no auth was provided, and anonymous delete is allowed, use the public user
	if u == nil {
		if conf.ANON_DELETE == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	if err = core.QMgr.DeleteJobByUser(id, u); err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
			return
		} else if err.Error() == e.UnAuth {
			cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
			return
		} else {
			cx.RespondWithErrorMessage("fail to delete job: "+id, http.StatusBadRequest)
			return
		}
	}

	cx.RespondWithData("job deleted: " + id)
	return
}

// DELETE: /job?suspend, /job?zombie
func (cr *JobController) DeleteMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous delete is allowed, use the public user
	if u == nil {
		if conf.ANON_DELETE == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}
	if query.Has("suspend") {
		num := core.QMgr.DeleteSuspendedJobsByUser(u)
		cx.RespondWithData(fmt.Sprintf("deleted %d suspended jobs", num))
	} else if query.Has("zombie") {
		num := core.QMgr.DeleteZombieJobsByUser(u)
		cx.RespondWithData(fmt.Sprintf("deleted %d zombie jobs", num))
	} else {
		cx.RespondWithError(http.StatusNotImplemented)
	}
	return
}
