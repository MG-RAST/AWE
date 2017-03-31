package controller

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/foreign/taverna"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"
	"github.com/davecgh/go-spew/spew"
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"net/http"
	//"os"
	"encoding/json"
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
	_user, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous write is allowed, use the public user
	if _user == nil {
		if conf.ANON_WRITE == true {
			_user = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	// Parse uploaded form
	_, files, err := ParseMultipartForm(cx.Request)

	if err != nil {
		if err.Error() == "request Content-Type isn't multipart/form-data" {
			cx.RespondWithErrorMessage("No job file is submitted", http.StatusBadRequest)
		} else {
			// Some error other than request encoding. Theoretically
			// could be a lost db connection between user lookup and parsing.
			// Blame the user, Its probaby their fault anyway.
			logger.Error("(JobController/Create) Error parsing form: " + err.Error())
			cx.RespondWithErrorMessage("(JobController/Create) Error parsing form: "+err.Error(), http.StatusBadRequest)
		}
		return
	}

	_, has_import := files["import"]
	_, has_upload := files["upload"]
	_, has_awf := files["awf"]
	_, has_cwl := files["cwl"] // TODO I could overload 'upload'
	_, has_job := files["job"] // input data for an CWL workflow

	var job *core.Job
	job = nil

	if has_import {
		// import a job document
		job, err = core.CreateJobImport(_user, files["import"])
		if err != nil {
			logger.Error("Err@job_Create:CreateJobImport: " + err.Error())
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
		logger.Event(event.JOB_IMPORT, "jobid="+job.Id+";name="+job.Info.Name+";project="+job.Info.Project+";user="+job.Info.User)
	} else if has_cwl {

		if !has_job {
			logger.Error("job missing")
			cx.RespondWithErrorMessage("job missing", http.StatusBadRequest)
			return
		}

		collection := cwl.NewCWL_collection()

		// 1) parse job
		err = cwl.ParseJob(&collection, files["job"].Path)
		if err != nil {
			logger.Debug(1, "ParseJob: "+err.Error())
			cx.RespondWithErrorMessage("error in reading job yaml/json file: "+err.Error(), http.StatusBadRequest)
			return
		}
		// 2) parse cwl
		logger.Debug(1, "got CWL")

		// get CWL as byte[]
		yamlstream, err := ioutil.ReadFile(files["cwl"].Path)
		if err != nil {
			logger.Debug(1, "CWL error: "+err.Error())
			cx.RespondWithErrorMessage("error in reading workflow file: "+err.Error(), http.StatusBadRequest)
			return
		}

		// convert CWL to string
		yaml_str := string(yamlstream[:])

		err = cwl.Parse_cwl_document(&collection, yaml_str)
		if err != nil {
			logger.Debug(1, "CWL error"+err.Error())
			cx.RespondWithErrorMessage("error in parsing job yaml file: "+err.Error(), http.StatusBadRequest)
			return
		}
		logger.Debug(1, "Parse_cwl_document done")

		cwl_workflow, ok := collection.Workflows["main"]
		if !ok {

			cx.RespondWithErrorMessage("Workflow main not found", http.StatusBadRequest)
			return
		}

		fmt.Println("\n\n\n--------------------------------- Steps:\n")
		for _, step := range cwl_workflow.Steps {
			spew.Dump(step)
		}

		fmt.Println("\n\n\n--------------------------------- Create AWE Job:\n")
		job, err = core.CWL2AWE(_user, files, cwl_workflow, &collection)
		if err != nil {
			cx.RespondWithErrorMessage("Error: "+err.Error(), http.StatusBadRequest)
			return
		}
		logger.Debug(1, "CWL2AWE done")

		//fmt.Println("\n\n\n--------------------------------- Done... now respond...")
		//cx.RespondWithData(job)
		//return

	} else if !has_upload && !has_awf {
		cx.RespondWithErrorMessage("No job script or awf is submitted", http.StatusBadRequest)
		return
	} else {
		// create new uploaded job

		job, err = core.CreateJobUpload(_user, files)

		if err != nil {
			logger.Error("Err@job_Create:CreateJobUpload: " + err.Error())
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
		logger.Event(event.JOB_SUBMISSION, "jobid="+job.Id+";name="+job.Info.Name+";project="+job.Info.Project+";user="+job.Info.User)
	}

	token, err := request.RetrieveToken(cx.Request)
	if err != nil {
		logger.Debug(3, "job %s no token", job.Id)
	} else {
		err = job.SetDataToken(token)
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
		logger.Debug(3, "job %s got token", job.Id)

	}

	err = job.Save()
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	// make a copy to prevent race conditions
	SR := StandardResponse{
		S: http.StatusOK,
		D: job,
		E: nil,
	}

	var response_bytes []byte
	response_bytes, err = json.Marshal(SR)

	// don't enqueue imports
	if !has_import {
		core.QMgr.EnqueueTasksByJobId(job.Id)
	}

	//cx.RespondWithData(job)
	cx.ResponseWriter.WriteHeader(http.StatusOK)
	cx.ResponseWriter.Write(response_bytes)

	//cx.WriteResponse(string(job_bytes[:]), http.StatusOK)
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

	//job, ok, err := core.QMgr.JobMap.Get(id)
	job, err := core.GetJob(id)
	//job, err := core.LoadJob(id)

	if err != nil {
		cx.RespondWithErrorMessage("job not found:"+id+" "+err.Error(), http.StatusBadRequest)
		return
	}

	// User must have read permissions on job or be job owner or be an admin
	rights := job.Acl.Check(u.Uuid)
	prights := job.Acl.Check("public")
	if job.Acl.Owner != u.Uuid && rights["read"] == false && u.Admin == false && prights["read"] == false {
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

	if query.Has("position") {
		if job.State != "queued" && job.State != "in-progress" {
			cx.RespondWithErrorMessage("job is not queued or in-progress, job state:"+job.State, http.StatusBadRequest)
			return
		}

		// Retrieve the job's approximate position in the queue (this is a rough estimate since jobs are not actually in a queue)
		q := bson.M{}
		qState := bson.M{}    // query job state
		qPriority := bson.M{} // query job priority
		qCgroup := bson.M{}   // query job clietgroup

		qState["$or"] = []bson.M{bson.M{"state": core.JOB_STAT_INIT}, bson.M{"state": core.JOB_STAT_QUEUED}, bson.M{"state": core.JOB_STAT_INPROGRESS}}
		qPriority["$or"] = []bson.M{bson.M{"info.priority": bson.M{"$gt": job.Info.Priority}}, bson.M{"$and": []bson.M{bson.M{"info.priority": job.Info.Priority}, bson.M{"info.submittime": bson.M{"$lt": job.Info.SubmitTime}}}}}

		var cgroups []bson.M
		for _, value := range strings.Split(job.Info.ClientGroups, ",") {
			cgroups = append(cgroups, bson.M{"info.clientgroups": bson.M{"$regex": value}})
		}
		qCgroup["$or"] = cgroups
		q["$and"] = []bson.M{qState, qPriority, qCgroup}

		if count, err := core.GetJobCount(q); err != nil {
			cx.RespondWithErrorMessage("error retrieving job position in queue", http.StatusInternalServerError)
		} else {
			m := make(map[string]int)
			m["position"] = count + 1
			cx.RespondWithData(m)
		}
		return
	}

	if query.Has("report") {
		jobLogs, err := job.GetJobLogs()
		if err != nil {
			logger.Error("Err@GetJobLogs: " + id + ":" + err.Error())
			cx.RespondWithErrorMessage("job logs not found: "+id, http.StatusBadRequest)
		}
		cx.RespondWithData(jobLogs)
		return
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

	job.RLockRecursive()
	defer job.RUnlockRecursive()

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

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	// Setup query and jobs objects
	q := bson.M{}
	jobs := core.Jobs{}

	if u != nil {
		// Add authorization checking to query if the user is not an admin
		if u.Admin == false {
			q["$or"] = []bson.M{bson.M{"acl.read": "public"}, bson.M{"acl.read": u.Uuid}, bson.M{"acl.owner": u.Uuid}, bson.M{"acl": bson.M{"$exists": "false"}}}
		}
	} else {
		// User is anonymous
		if conf.ANON_READ {
			// select on only jobs that are publicly readable
			q["acl.read"] = "public"
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
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
	skip := map[string]int{
		"limit":      1,
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
		"distinct":   1,
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

		filtered_jobs := core.Jobs{}
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
		filtered_jobs.RLockRecursive()
		defer filtered_jobs.RUnlockRecursive()
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

		filtered_jobs := core.Jobs{}
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

		filtered_jobs.RLockRecursive()
		defer filtered_jobs.RUnlockRecursive()

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

		paged_jobs := core.Jobs{}
		registered_jobs := core.Jobs{}
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
		paged_jobs.RLockRecursive()
		defer paged_jobs.RUnlockRecursive()
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
				// if completed or deleted move on, empty task array
				mjob.State = append(mjob.State, job.State)
			} else if job.State == "suspend" {
				// get failed task if info available, otherwise empty task array
				mjob.State = append(mjob.State, "suspend")
				parts := strings.Split(job.LastFailed, "_")
				if (len(parts) == 2) || (len(parts) == 3) {
					if tid, err := strconv.Atoi(parts[1]); err == nil {
						mjob.Task = append(mjob.Task, tid)
					}
				}
			} else {
				// get multiple tasks in state queued or in-progress
				for j := 0; j < len(job.Tasks); j++ {
					task := job.Tasks[j]
					task_state, xerr := task.GetState()
					if xerr != nil {
						continue
					}

					if (task_state == "in-progress") || (task_state == "queued") {
						mjob.State = append(mjob.State, task_state)
						mjob.Task = append(mjob.Task, j)
					}
				}
				// otherwise get oldest pending or init task
				if len(mjob.State) == 0 {
					for j := 0; j < len(job.Tasks); j++ {
						task := job.Tasks[j]
						task_state, xerr := task.GetState()
						if xerr != nil {
							continue
						}
						if (task_state == "pending") || (task_state == "init") {
							mjob.State = append(mjob.State, task_state)
							mjob.Task = append(mjob.Task, j)
							break
						}
					}
				}
			}
			minimal_jobs = append(minimal_jobs, mjob)
		}

		cx.RespondWithPaginatedData(minimal_jobs, limit, offset, total)
		return
	}

	if query.Has("distinct") {
		dField := query.Value("distinct")
		if !core.HasInfoField(dField) {
			cx.RespondWithErrorMessage("unable to run distinct query on non-indexed info field: "+dField, http.StatusBadRequest)
		}
		results, err := core.DbFindDistinct(q, dField)
		if err != nil {
			logger.Error("err " + err.Error())
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData(results)
		return
	}

	total, err := jobs.GetPaginated(q, limit, offset, order, direction)
	if err != nil {
		logger.Error("err " + err.Error())
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}
	filtered_jobs := core.Jobs{}
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

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	// Load job by id
	var job *core.Job
	if query.Has("clientgroup") || query.Has("priority") || query.Has("pipeline") || query.Has("expiration") || query.Has("settoken") {
		job, err = core.GetJob(id)
		if err != nil {
			if err == mgo.ErrNotFound {
				cx.RespondWithNotFound()
			} else {
				// In theory the db connection could be lost between
				// checking user and load but seems unlikely.
				// logger.Error("Err@job_Read:LoadJob: " + id + ":" + err.Error())
				cx.RespondWithErrorMessage("job not found:"+id+" "+err.Error(), http.StatusBadRequest)
			}
			return
		}
	}
	// User must have write permissions on job or be job owner or be an admin
	acl, err := core.DBGetJobAcl(id)
	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("job not found: "+id+" "+err.Error(), http.StatusBadRequest)
		}
		return
	}

	rights := acl.Check(u.Uuid)
	if acl.Owner != u.Uuid && rights["write"] == false && u.Admin == false {
		cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
		return
	}

	if query.Has("resume") { // to resume a suspended job
		if err := core.QMgr.ResumeSuspendedJobByUser(id, u); err != nil {
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
	if query.Has("recover") || query.Has("register") { // to recover a job from mongodb missing from queue
		if err := core.QMgr.RecoverJob(id); err != nil {
			cx.RespondWithErrorMessage("fail to recover job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job recovered: " + id)
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
		cx.RespondWithData("job recompute started at task " + stage + ": " + id)
		return
	}
	if query.Has("resubmit") { // to recompute a job from the beginning, all tasks will be computed
		if err := core.QMgr.ResubmitJob(id); err != nil {
			cx.RespondWithErrorMessage("fail to resubmit job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job resubmitted: " + id)
		return
	}
	if query.Has("clientgroup") { // change the clientgroup attribute of the job
		newgroup := query.Value("clientgroup")
		if newgroup == "" {
			cx.RespondWithErrorMessage("lacking clientgroup name", http.StatusBadRequest)
			return
		}
		if err := job.SetClientgroups(newgroup); err != nil {
			cx.RespondWithErrorMessage("failed to update group for job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job group updated: " + id + " to " + newgroup)
		return
	}
	if query.Has("priority") { // change the priority attribute of the job
		priority_str := query.Value("priority")
		if priority_str == "" {
			cx.RespondWithErrorMessage("lacking priority value", http.StatusBadRequest)
			return
		}
		priority, err := strconv.Atoi(priority_str)
		if err != nil {
			cx.RespondWithErrorMessage("priority value must be an integer"+err.Error(), http.StatusBadRequest)
			return
		}
		if err := job.SetPriority(priority); err != nil {
			cx.RespondWithErrorMessage("failed to set the priority for job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job priority updated: " + id + " to " + priority_str)
		return
	}
	if query.Has("pipeline") { // change the pipeline attribute of the job
		pipeline := query.Value("pipeline")
		if pipeline == "" {
			cx.RespondWithErrorMessage("lacking pipeline value", http.StatusBadRequest)
			return
		}
		if err := job.SetPipeline(pipeline); err != nil {
			cx.RespondWithErrorMessage("failed to set the pipeline for job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job pipeline updated: " + id + " to " + pipeline)
		return
	}
	if query.Has("expiration") { // change the expiration attribute of the job, does not get reaped until in completed state
		expire := query.Value("expiration")
		if expire == "" {
			cx.RespondWithErrorMessage("lacking expiration value", http.StatusBadRequest)
			return
		}
		if err := job.SetExpiration(expire); err != nil {
			cx.RespondWithErrorMessage("failed to set the expiration for job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("expiration '" + job.Expiration.String() + "' set for job: " + id)
		return
	}
	if query.Has("settoken") { // set data token
		token, err := request.RetrieveToken(cx.Request)
		if err != nil {
			cx.RespondWithErrorMessage("fail to retrieve token for job, pls set token in header: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}

		err = job.SetDataToken(token)

		if err != nil {
			cx.RespondWithErrorMessage("failed to set the token for job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		err = job.Save()
		if err != nil {
			cx.RespondWithErrorMessage("failed to save job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}

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

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}
	full := false
	if query.Has("full") {
		full = true
	}

	if err = core.QMgr.DeleteJobByUser(id, u, full); err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
			return
		} else if err.Error() == e.UnAuth {
			cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
			return
		} else {
			cx.RespondWithErrorMessage("fail to delete job "+id+" "+err.Error(), http.StatusBadRequest)
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
	full := false
	if query.Has("full") {
		full = true
	}
	if query.Has("suspend") {
		num := core.QMgr.DeleteSuspendedJobsByUser(u, full)
		cx.RespondWithData(fmt.Sprintf("deleted %d suspended jobs", num))
	} else if query.Has("zombie") {
		num := core.QMgr.DeleteZombieJobsByUser(u, full)
		cx.RespondWithData(fmt.Sprintf("deleted %d zombie jobs", num))
	} else {
		cx.RespondWithError(http.StatusNotImplemented)
	}
	return
}
