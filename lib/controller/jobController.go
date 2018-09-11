package controller

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"reflect"

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
	mgo "gopkg.in/mgo.v2"
	"gopkg.in/mgo.v2/bson"
	//"os"

	"path"
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
	cwl_file, has_cwl := files["cwl"] // TODO I could overload 'upload'
	job_file, has_job := files["job"] // input data for an CWL workflow

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
			cx.RespondWithErrorMessage("cwl job missing", http.StatusBadRequest)
			return
		}

		workflow_filename := cwl_file.Name

		collection := cwl.NewCWL_collection()

		//1) parse job

		job_stream, err := ioutil.ReadFile(job_file.Path)
		if err != nil {
			cx.RespondWithErrorMessage("error in reading job yaml/json file: "+err.Error(), http.StatusBadRequest)
			return
		}

		//job_str := string(job_stream[:])

		job_input, err := cwl.ParseJob(&job_stream)
		if err != nil {
			logger.Error("ParseJob: " + err.Error())
			cx.RespondWithErrorMessage("error in reading job yaml/json file: "+err.Error(), http.StatusBadRequest)
			return
		}

		//collection.Job_input = job_input

		// 2) parse cwl
		logger.Debug(1, "got CWL")

		// get CWL as byte[]
		yamlstream, err := ioutil.ReadFile(cwl_file.Path)
		if err != nil {
			logger.Error("CWL error: " + err.Error())
			cx.RespondWithErrorMessage("error in reading workflow file: "+err.Error(), http.StatusBadRequest)
			return
		}

		// convert CWL to string
		yaml_str := string(yamlstream[:])

		var schemata []cwl.CWLType_Type
		var object_array []cwl.Named_CWL_object
		var cwl_version cwl.CWLVersion
		var context *cwl.WorkflowContext
		//var namespaces map[string]string
		//var schemas []interface{}
		object_array, cwl_version, schemata, context, _, err = cwl.Parse_cwl_document(yaml_str, "-")
		if err != nil {
			cx.RespondWithErrorMessage("error in parsing cwl workflow yaml file: "+err.Error(), http.StatusBadRequest)
			return
		}

		err = collection.AddArray(object_array)
		if err != nil {
			logger.Error("Parse_cwl_document error: " + err.Error())
			cx.RespondWithErrorMessage("error in adding cwl objects to collection: "+err.Error(), http.StatusBadRequest)
			return
		}
		logger.Debug(1, "Parse_cwl_document done")

		err = collection.AddSchemata(schemata)
		if err != nil {
			cx.RespondWithErrorMessage("error in adding schemata: "+err.Error(), http.StatusBadRequest)
			return
		}

		entrypoint := ""

		var shock_requirement *cwl.ShockRequirement
		shock_requirement = nil

		var cwl_workflow *cwl.Workflow
		if len(collection.Workflows) == 0 {
			if len(object_array) != 1 {
				cx.RespondWithErrorMessage(fmt.Sprintf("Expected exactly one element in object_array, got %d", len(collection.Workflows)), http.StatusBadRequest)
				return
			}
			// This probably is a CommandlineTool or ExpressionTool submission (without workflow)
			// create new Workflow to wrap around the CommandLineTool/ExpressionTool
			entrypoint = "#entrypoint"

			pair := object_array[0]

			runner := pair.Value

			switch runner.(type) {
			case *cwl.Workflow:
				workflow := runner.(*cwl.Workflow)
				workflow.CwlVersion = cwl_version

			case *cwl.CommandLineTool:
				commandlinetool_if := pair.Value

				commandlinetool, ok := commandlinetool_if.(*cwl.CommandLineTool)
				if !ok {

					cx.RespondWithErrorMessage(fmt.Sprintf("(job/create) Error casting CommandLineTool (type: %s)", reflect.TypeOf(commandlinetool_if)), http.StatusBadRequest)
					return
				}

				if shock_requirement == nil {
					shock_requirement, err = cwl.GetShockRequirement(commandlinetool.Requirements)
					if err != nil {
						logger.Debug(1, "(job/create) GetShockRequirement returned: %s", err.Error())
						shock_requirement = nil
					}
				}

				cwl_workflow_instance := cwl.NewWorkflowEmpty()
				cwl_workflow = &cwl_workflow_instance
				cwl_workflow.Id = entrypoint
				cwl_workflow.CwlVersion = cwl_version
				cwl_workflow.Namespaces = context.Namespaces
				new_step := cwl.WorkflowStep{}
				step_id := entrypoint + "/wrapper_step"
				new_step.Id = step_id
				for _, input := range commandlinetool.Inputs { // input is CommandInputParameter

					workflow_input_name := entrypoint + "/" + path.Base(input.Id)

					var workflow_step_input cwl.WorkflowStepInput
					workflow_step_input.Id = step_id + "/" + input.Id
					workflow_step_input.Source = workflow_input_name
					workflow_step_input.Default = input.Default

					//fmt.Println("CommandInputParameter and WorkflowStepInput:")
					//spew.Dump(input)
					//spew.Dump(workflow_step_input)
					new_step.In = append(new_step.In, workflow_step_input)

					var workflow_input_parameter cwl.InputParameter
					workflow_input_parameter.Id = workflow_input_name
					workflow_input_parameter.SecondaryFiles = input.SecondaryFiles
					workflow_input_parameter.Format = input.Format
					workflow_input_parameter.Streamable = input.Streamable
					workflow_input_parameter.InputBinding = input.InputBinding
					workflow_input_parameter.Type = input.Type

					workflow_input_parameter.Default = input.Default

					add_null := false
					if input.Default != nil { // check if this is an optional argument
						add_null = true
					}

					if add_null {
						has_null := false
						for _, t := range workflow_input_parameter.Type {
							if t == cwl.CWL_null {
								has_null = true
								break
							}
						}
						if !has_null {
							workflow_input_parameter.Type = append(workflow_input_parameter.Type, cwl.CWL_null)
						}
					}

					cwl_workflow.Inputs = append(cwl_workflow.Inputs, workflow_input_parameter)
				}

				for _, output := range commandlinetool.Outputs {
					var workflow_step_output cwl.WorkflowStepOutput
					workflow_step_output.Id = step_id + "/" + strings.TrimPrefix(output.Id, "#")

					new_step.Out = append(new_step.Out, workflow_step_output)

					var workflow_output_parameter cwl.WorkflowOutputParameter

					workflow_output_parameter.Id = entrypoint + "/" + path.Base(output.Id)

					workflow_output_parameter.OutputSource = step_id + "/" + path.Base(output.Id)
					workflow_output_parameter.SecondaryFiles = output.SecondaryFiles
					workflow_output_parameter.Format = output.Format
					workflow_output_parameter.Streamable = output.Streamable
					//workflow_output_parameter.OutputBinding = output.OutputBinding
					//workflow_output_parameter.OutputSource = output.OutputSource
					//workflow_output_parameter.LinkMerge = output.LinkMerge
					workflow_output_parameter.Type = output.Type
					cwl_workflow.Outputs = append(cwl_workflow.Outputs, workflow_output_parameter)
				}

				if commandlinetool.Requirements != nil {
					requirements := commandlinetool.Requirements
					for i, _ := range requirements {
						require_type := (requirements)[i].GetClass()
						if require_type == "ShockRequirement" {
							shock_requirement := (requirements)[i]

							cwl_workflow.Requirements, err = cwl.AddRequirement(shock_requirement, requirements)
							if err != nil {
								err = fmt.Errorf("(job/create) AddRequirement returned: %s", err.Error())
								return
							}
						}
					}
				}

				new_step.Run = commandlinetool.Id

				cwl_workflow.Steps = []cwl.WorkflowStep{new_step}

				cwl_workflow_named := cwl.Named_CWL_object{}
				cwl_workflow_named.Id = cwl_workflow.Id
				cwl_workflow_named.Value = cwl_workflow

				object_array = append(object_array, cwl_workflow_named)
				err = collection.Add(entrypoint, cwl_workflow)
				if err != nil {
					cx.RespondWithErrorMessage("collection.Add returned: "+err.Error(), http.StatusBadRequest)
					return
				}

			case *cwl.ExpressionTool:
				expressiontool_if := pair.Value

				expressiontool, ok := expressiontool_if.(*cwl.ExpressionTool)
				if !ok {

					cx.RespondWithErrorMessage(fmt.Sprintf("(job/create) Error casting ExpressionTool (type: %s)", reflect.TypeOf(expressiontool_if)), http.StatusBadRequest)
					return
				}

				if shock_requirement == nil {
					shock_requirement, err = cwl.GetShockRequirement(expressiontool.Requirements)
					if err != nil {
						logger.Debug(1, "(job/create) GetShockRequirement returned: %s", err.Error())
						shock_requirement = nil
					}
				}

				cwl_workflow_instance := cwl.NewWorkflowEmpty()
				cwl_workflow = &cwl_workflow_instance
				cwl_workflow.Id = entrypoint
				cwl_workflow.CwlVersion = cwl_version
				new_step := cwl.WorkflowStep{}
				step_id := entrypoint + "/wrapper_step"
				new_step.Id = step_id
				for _, input := range expressiontool.Inputs { // input is InputParameter

					workflow_input_name := entrypoint + "/" + path.Base(input.Id)

					var workflow_step_input cwl.WorkflowStepInput
					workflow_step_input.Id = step_id + "/" + input.Id
					workflow_step_input.Source = workflow_input_name
					workflow_step_input.Default = input.Default

					//fmt.Println("InputParameter and WorkflowStepInput:")
					//spew.Dump(input)
					//spew.Dump(workflow_step_input)
					new_step.In = append(new_step.In, workflow_step_input)

					var workflow_input_parameter cwl.InputParameter
					workflow_input_parameter.Id = workflow_input_name
					workflow_input_parameter.SecondaryFiles = input.SecondaryFiles
					workflow_input_parameter.Format = input.Format
					workflow_input_parameter.Streamable = input.Streamable
					workflow_input_parameter.InputBinding = input.InputBinding
					workflow_input_parameter.Type = input.Type

					workflow_input_parameter.Default = input.Default

					add_null := false
					if input.Default != nil { // check if this is an optional argument
						add_null = true
					}

					if add_null {
						has_null := false
						for _, t := range workflow_input_parameter.Type {
							if t == cwl.CWL_null {
								has_null = true
								break
							}
						}
						if !has_null {
							workflow_input_parameter.Type = append(workflow_input_parameter.Type, cwl.CWL_null)
						}
					}

					cwl_workflow.Inputs = append(cwl_workflow.Inputs, workflow_input_parameter)
				}

				for _, output := range expressiontool.Outputs { // type: ExpressionToolOutputParameter
					var workflow_step_output cwl.WorkflowStepOutput
					workflow_step_output.Id = step_id + "/" + strings.TrimPrefix(output.Id, "#")

					new_step.Out = append(new_step.Out, workflow_step_output)

					var workflow_output_parameter cwl.WorkflowOutputParameter

					workflow_output_parameter.Id = entrypoint + "/" + path.Base(output.Id)
					workflow_output_parameter.OutputSource = step_id + "/" + path.Base(output.Id)
					workflow_output_parameter.SecondaryFiles = output.SecondaryFiles
					workflow_output_parameter.Format = output.Format
					workflow_output_parameter.Streamable = output.Streamable
					//workflow_output_parameter.OutputBinding = output.OutputBinding
					//workflow_output_parameter.OutputSource = output.OutputSource
					//workflow_output_parameter.LinkMerge = output.LinkMerge
					workflow_output_parameter.Type = output.Type
					cwl_workflow.Outputs = append(cwl_workflow.Outputs, workflow_output_parameter)
				}

				if expressiontool.Requirements != nil {
					requirements := expressiontool.Requirements
					for i, _ := range requirements {
						require_type := (requirements)[i].GetClass()
						if require_type == "ShockRequirement" {
							shock_requirement := (requirements)[i]

							cwl_workflow.Requirements, err = cwl.AddRequirement(shock_requirement, requirements)
							if err != nil {
								err = fmt.Errorf("(job/create) AddRequirement returned: %s", err.Error())
								return
							}
						}
					}
				}

				new_step.Run = expressiontool.Id

				cwl_workflow.Steps = []cwl.WorkflowStep{new_step}

				cwl_workflow_named := cwl.Named_CWL_object{}
				cwl_workflow_named.Id = cwl_workflow.Id
				cwl_workflow_named.Value = cwl_workflow

				object_array = append(object_array, cwl_workflow_named)
				err = collection.Add(entrypoint, cwl_workflow)
				if err != nil {
					cx.RespondWithErrorMessage("collection.Add returned: "+err.Error(), http.StatusBadRequest)
					return
				}
			default:
				cx.RespondWithErrorMessage(fmt.Sprintf("Runner type %s not supported", reflect.TypeOf(runner)), http.StatusBadRequest)

				return
			}
			//spew.Dump(cwl_workflow)

		} else {
			entrypoint = "#main"

			var ok bool
			cwl_workflow, ok = collection.Workflows[entrypoint]
			if !ok {
				cx.RespondWithErrorMessage("Workflow main not found", http.StatusBadRequest)
				return
			}

			shock_requirement, err = cwl.GetShockRequirement(cwl_workflow.Requirements)
			if err != nil {
				logger.Debug(1, "(job/create) GetShockRequirement returned: %s", err.Error())
				shock_requirement = nil
			}

		}

		//fmt.Println("\n\n\n--------------------------------- Steps:\n")
		//for _, step := range cwl_workflow.Steps {
		//	spew.Dump(step)
		//}

		//fmt.Println("\n\n\n--------------------------------- Create AWE Job:\n")
		job, err = core.CWL2AWE(_user, files, job_input, cwl_workflow, &collection, cwl_version, context.Namespaces)
		if err != nil {
			cx.RespondWithErrorMessage("Error: "+err.Error(), http.StatusBadRequest)
			return
		}

		job.Entrypoint = entrypoint
		job.IsCWL = true

		// this ugly conversion is necessary as mongo does not like interface types.
		object_array_of_interface := []interface{}{}
		for i, _ := range object_array {
			object_array_of_interface = append(object_array_of_interface, object_array[i])
		}

		job.CWL_objects = object_array_of_interface
		job.CwlVersion = cwl_version
		job.Namespaces = context.Namespaces
		//job.CWL_collection = &collection
		job.Info.Name = job_file.Name
		job.Info.Pipeline = workflow_filename
		job.Info.ClientGroups = "docker" // TODO this needs to be configured

		if shock_requirement != nil {
			job.CWL_ShockRequirement = shock_requirement
		}

		if job.CwlVersion == "" {
			cx.RespondWithErrorMessage("Error: cwlVersion is empty", http.StatusBadRequest)
			return
		}

		//job.CWL_workflow_interface = cwl_workflow
		//job.CWL_job_input_interface = job_input

		//job.CWL_workflow = cwl_workflow
		//job.CWL_job_input = job_input

		//job.AddWorkflowInstance("<main>", *job_input)

		//job.Set_CWL_workflow_b64(yaml_str)

		//job.Set_CWL_job_input_b64(job_str)

		logger.Debug(1, "CWL2AWE done")

	} else if !has_upload && !has_awf {
		cx.RespondWithErrorMessage("No job script or awf is submitted", http.StatusBadRequest)
		return
	} else {
		// create new uploaded job

		job, err = core.CreateJobUpload(_user, files)

		if err != nil {
			err = fmt.Errorf("(JobController/Create) CreateJobUpload returned: %s", err.Error())
			logger.Error(err.Error())
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
			cx.RespondWithErrorMessage(fmt.Sprintf("(JobController/Create) SetDataToken returned: %s", err.Error()), http.StatusBadRequest)
			return
		}
		logger.Debug(3, "job %s got token", job.Id)
	}

	err = job.Save() // note that the job only goes into mongo, not into memory yet (EnqueueTasksByJobId is dowing that)
	if err != nil {
		cx.RespondWithErrorMessage(fmt.Sprintf("(JobController/Create) job.Save returned: %s", err.Error()), http.StatusBadRequest)
		return
	}

	// make a copy to prevent race conditions
	SR := StandardResponse{
		S: http.StatusOK,
		D: job,
		E: nil,
	}

	//for i, _ := range job.CWL_objects {
	//	spew.Dump(job.CWL_objects[i])
	//}
	//job.CWL_objects = nil

	var response_bytes []byte
	response_bytes, err = json.Marshal(SR)
	if err != nil {
		//fmt.Println("Dump:")
		//spew.Dump(job)

		cx.RespondWithErrorMessage("(JobController/Create) json.Marshal returned: "+err.Error(), http.StatusBadRequest)
		return
	}

	// don't enqueue imports
	if !has_import {
		err = core.QMgr.EnqueueTasksByJobId(job.Id)
		if err != nil {
			err = fmt.Errorf("(JobController/Create) core.QMgr.EnqueueTasksByJobId returned: %s", err.Error())
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
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
	job, err := core.GetJob(id)
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

	job_state, err := job.GetState(true)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

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
		if job_state != "queued" && job_state != "in-progress" {
			cx.RespondWithErrorMessage("job is not queued or in-progress, job state:"+job_state, http.StatusBadRequest)
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

	// check if an adminview is being requested
	if query.Has("adminview") {

		// adminview requires a user
		if u != nil {

			// adminview requires the user to be an admin
			if u.Admin {

				// special is an attribute from the job document chosen via the cgi-param "special"
				// this attribute can be a path in the document, separated by .
				// special attributes do not have to be present in all job documents for this function to work
				special := "info.userattr.bp_count"
				if query.Has("special") {
					special = query.Value("special")
				}

				// call the GetAdminView function, passing along the special attribute
				results, err := core.GetAdminView(special)

				// if there is an error, return it
				if err != nil {
					logger.Error("err " + err.Error())
					cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
					return
				}

				// if there is no error, return the data
				cx.RespondWithData(results)
				return
			} else {
				cx.RespondWithErrorMessage("you need to be an administrator to access this function", http.StatusUnauthorized)
				return
			}
		} else {
			cx.RespondWithErrorMessage("you need to be logged in to access this function", http.StatusUnauthorized)
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
		err := jobs.GetAll(q, order, direction, false)
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

	// This code returns jobs from the in-memory job map, (thus it should be more efficient) but it does not have the same sorting and filtering feature as the mongo code based above.
	// if query.Has("active") {
	//
	// 		jobs, err := core.JM.Get_List(true)
	// 		if err != nil {
	// 			cx.RespondWithErrorMessage("could not get job list: "+err.Error(), http.StatusBadRequest)
	// 			return
	// 		}
	//
	// 		filtered_jobs := core.Jobs{}
	//
	// 		for _, job := range jobs {
	// 			var job_state string
	// 			job_state, err = job.GetState(true)
	// 			if err != nil {
	// 				logger.Error("(JobController/ReadMany/active) Could not get job state")
	// 				continue
	// 			}
	//
	// 			if contains(core.JOB_STATS_ACTIVE, job_state) {
	// 				filtered_jobs = append(filtered_jobs, job)
	// 			}
	//
	// 		}
	//
	// 		//cx.RespondWithPaginatedData(filtered_jobs, limit, offset, len(act_jobs))
	// 		filtered_jobs.RLockRecursive()
	// 		defer filtered_jobs.RUnlockRecursive()
	// 		cx.RespondWithData(filtered_jobs)
	// 		return
	// 	}

	//geting suspended job in the current queue (excluding jobs in db but not in qmgr)
	if query.Has("suspend") {
		err := jobs.GetAll(q, order, direction, false)
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

		//filtered_jobs.RLockRecursive()
		//defer filtered_jobs.RUnlockRecursive()

		cx.RespondWithPaginatedData(filtered_jobs, limit, offset, len(suspend_jobs))
		return
	}

	if query.Has("registered") {
		err := jobs.GetAll(q, order, direction, false)
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
		//paged_jobs.RLockRecursive()
		//defer paged_jobs.RUnlockRecursive()
		cx.RespondWithPaginatedData(paged_jobs, limit, offset, total)
		return
	}

	if query.Has("verbosity") && (query.Value("verbosity") == "minimal") {
		// TODO - have mongo query only return fields needed to populate JobMin struct
		total, err := jobs.GetPaginated(q, limit, offset, order, direction, false)
		if err != nil {
			logger.Error("error: " + err.Error())
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
		minimal_jobs := []core.JobMin{}
		length := jobs.Length()
		for i := 0; i < length; i++ {
			job := jobs.GetJobAt(i)
			job_state, _ := job.GetState(false) // no lock needed

			// create and populate minimal job
			mjob := core.JobMin{}
			mjob.Id = job.Id
			mjob.Name = job.Info.Name
			mjob.SubmitTime = job.Info.SubmitTime
			mjob.CompletedTime = job.Info.CompletedTime
			// get size of input
			var size_sum int64 = 0
			for _, v := range job.Tasks[0].Inputs { // TODO this is stupid, this is MG-RAST specific
				size_sum = size_sum + v.Size
			}
			mjob.Size = size_sum
			// add userattr fields
			if query.Has("userattr") {
				mjob.UserAttr = map[string]interface{}{}
				for _, attr := range query.List("userattr") {
					if val, ok := job.Info.UserAttr[attr]; ok {
						mjob.UserAttr[attr] = val
					}
				}
			}

			// get current total computetime
			for _, task := range job.Tasks {
				mjob.ComputeTime += task.ComputeTime
			}

			if (job_state == core.JOB_STAT_COMPLETED) || (job_state == core.JOB_STAT_DELETED) {

				// if completed or deleted move on, empty task array
				mjob.State = append(mjob.State, job_state)
			} else if job_state == core.JOB_STAT_SUSPEND {
				mjob.State = append(mjob.State, core.JOB_STAT_SUSPEND)
				// get failed task if info available, otherwise empty task array
				if (job.Error != nil) && (job.Error.TaskFailed != "") {
					parts := strings.Split(job.Error.TaskFailed, "_")
					if len(parts) > 1 {
						tid, err := strconv.Atoi(parts[1])
						if err != nil {
							logger.Error("(job resource) verbosity, A job.Error.TaskFailed cannot be parsed (%s)", job.Error.TaskFailed)
						} else {
							mjob.Task = append(mjob.Task, tid)
						}
					} else if len(parts) == 1 {
						tid, err := strconv.Atoi(parts[0])
						if err != nil {
							logger.Error("(job resource) verbosity, B job.Error.TaskFailed cannot be parsed (%s)", job.Error.TaskFailed)
						} else {
							mjob.Task = append(mjob.Task, tid)
						}
					} else {

						logger.Error("(job resource) verbosity, C job.Error.TaskFailed cannot be parsed  (%s)", job.Error.TaskFailed)
					}

				}
			} else {
				// get multiple tasks in state queued or in-progress
				for j, task := range job.Tasks {

					task_state := task.State // no lock needed

					if (task_state == core.TASK_STAT_INPROGRESS) || (task_state == core.TASK_STAT_QUEUED) {
						mjob.State = append(mjob.State, task_state)
						mjob.Task = append(mjob.Task, j)
					}
				}
				// otherwise get oldest pending or init task
				if len(mjob.State) == 0 {
					for j, task := range job.Tasks {

						task_state := task.State // no lock needed

						if (task_state == core.TASK_STAT_PENDING) || (task_state == core.TASK_STAT_INIT) {
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

	total, err := jobs.GetPaginated(q, limit, offset, order, direction, false)
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
	// resume all suspended jobs
	if query.Has("resumeall") {
		num := core.QMgr.ResumeSuspendedJobsByUser(u)
		cx.RespondWithData(fmt.Sprintf("%d suspended jobs resumed", num))
		return
	}
	// recover unfinished jobs in mongodb not in queue, this is for admins only
	if query.Has("recoverall") {
		if conf.ANON_WRITE == true || u.Admin {
			num, _, err := core.QMgr.RecoverJobs()
			if err != nil {
				cx.RespondWithErrorMessage("failed to recover jobs: "+err.Error(), http.StatusBadRequest)
				return
			}
			cx.RespondWithData(fmt.Sprintf("%d missing jobs recovered", num))
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
		}
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
		jerror := &core.JobError{
			ServerNotes: "manually suspended",
			Status:      core.JOB_STAT_SUSPEND,
		}
		if err := core.QMgr.SuspendJob(id, jerror); err != nil {
			cx.RespondWithErrorMessage("fail to suspend job: "+id+" "+err.Error(), http.StatusBadRequest)
			return
		}
		cx.RespondWithData("job suspended: " + id)
		return
	}
	if query.Has("recover") || query.Has("register") { // to recover a job from mongodb missing from queue
		if _, err := core.QMgr.RecoverJob(id, nil); err != nil {
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
