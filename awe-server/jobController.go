package main

import (
	"github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/logger"
	e "github.com/MG-RAST/Shock/errors"
	"github.com/jaredwilkening/goweb"
	"labix.org/v2/mgo/bson"
	"net/http"
	"strconv"
)

type JobController struct{}

func handleAuthError(err error, cx *goweb.Context) {
	switch err.Error() {
	case e.MongoDocNotFound:
		cx.RespondWithErrorMessage("Invalid username or password", http.StatusBadRequest)
		return
	case e.InvalidAuth:
		cx.RespondWithErrorMessage("Invalid Authorization header", http.StatusBadRequest)
		return
	}
	Log.Error("Error at Auth: " + err.Error())
	cx.RespondWithError(http.StatusInternalServerError)
	return
}

// POST: /job
func (cr *JobController) Create(cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)

	// Parse uploaded form 
	params, files, err := ParseMultipartForm(cx.Request)

	if err != nil {
		if err.Error() == "request Content-Type isn't multipart/form-data" {
			cx.RespondWithErrorMessage("No job file is submitted", http.StatusBadRequest)
		} else {
			// Some error other than request encoding. Theoretically 
			// could be a lost db connection between user lookup and parsing.
			// Blame the user, Its probaby their fault anyway.
			Log.Error("Error parsing form: " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
		}
		return
	}
	// Create job
	_, hasupload := files["upload"]

	if !hasupload {
		cx.RespondWithErrorMessage("No job script is submitted", http.StatusBadRequest)
		return
	}

	//send job submissin request and get back an assigned job number (jid)
	var jid string
	jid, err = queueMgr.JobRegister()
	if err != nil {
		Log.Error("Err@job_Create:GetNextJobNum: " + err.Error())
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	var job *core.Job
	job, err = core.CreateJobUpload(params, files, jid)
	if err != nil {
		Log.Error("Err@job_Create:CreateJobUpload: " + err.Error())
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	queueMgr.AddTasks(job.TaskList())

	//log event about job submission (JB)
	Log.Event(EVENT_JOB_SUBMISSION, "jobid="+job.Id)

	cx.RespondWithData(job)
	return
}

// GET: /job/{id}
func (cr *JobController) Read(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	// Load job by id
	job, err := core.LoadJob(id)
	if err != nil {
		if err.Error() == e.MongoDocNotFound {
			cx.RespondWithNotFound()
			return
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			Log.Error("Err@job_Read:LoadJob: " + err.Error())
			cx.RespondWithError(http.StatusInternalServerError)
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

	// Gather query params
	query := &Query{list: cx.Request.URL.Query()}

	// Setup query and jobs objects
	q := bson.M{}
	jobs := new(core.Jobs)

	// Gather params to make db query. Do not include the
	// following list.	
	skip := map[string]int{"limit": 1, "skip": 1, "query": 1}
	if query.Has("query") {
		for key, val := range query.All() {
			_, s := skip[key]
			if !s {
				q[key] = val[0]
			}
		}
	}

	// Limit and skip. Set default if both are not specified
	if query.Has("limit") || query.Has("skip") {
		var lim, off int
		if query.Has("limit") {
			lim, _ = strconv.Atoi(query.Value("limit"))
		} else {
			lim = 100
		}
		if query.Has("skip") {
			off, _ = strconv.Atoi(query.Value("skip"))
		} else {
			off = 0
		}
		// Get jobs from db
		err := jobs.GetAllLimitOffset(q, lim, off)
		if err != nil {
			Log.Error("err " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
			return
		}
	} else {
		// Get nodes from db
		err := jobs.GetAll(q)
		if err != nil {
			Log.Error("err " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
			return
		}
	}

	cx.RespondWithData(jobs)
	return
}

// DELETE: /job/{id}
func (cr *JobController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	if err := queueMgr.DeleteJob(id); err != nil {
		cx.RespondWithErrorMessage("fail to delete job: "+id, http.StatusBadRequest)
		return
	}
	cx.RespondWithData("job deleted: " + id)
	return
}
