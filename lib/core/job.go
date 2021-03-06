package core

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	rwmutex "github.com/MG-RAST/go-rwmutex"

	//cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	uuid "github.com/MG-RAST/golib/go-uuid/uuid"
	"gopkg.in/mgo.v2/bson"

	"strconv"
	"strings"
	"time"
)

const (
	JOB_STAT_INIT             = "init"        // inital state
	JOB_STAT_QUEUING          = "queuing"     // transition from "init" to "queued"
	JOB_STAT_QUEUED           = "queued"      // all tasks have been added to taskmap
	JOB_STAT_INPROGRESS       = "in-progress" // a first task went into state in-progress
	JOB_STAT_COMPLETED        = "completed"
	JOB_STAT_SUSPEND          = "suspend"
	JOB_STAT_FAILED_PERMANENT = "failed-permanent" // this sepcific error state can be trigger by the workflow software
	JOB_STAT_DELETED          = "deleted"
)

var JOB_STATS_ACTIVE = []string{JOB_STAT_QUEUING, JOB_STAT_QUEUED, JOB_STAT_INPROGRESS}
var JOB_STATS_REGISTERED = []string{JOB_STAT_QUEUING, JOB_STAT_QUEUED, JOB_STAT_INPROGRESS, JOB_STAT_SUSPEND}
var JOB_STATS_TO_RECOVER = []string{JOB_STAT_INIT, JOB_STAT_QUEUING, JOB_STAT_QUEUED, JOB_STAT_INPROGRESS, JOB_STAT_SUSPEND}

// JobError _
type JobError struct {
	ClientFailed string `bson:"clientfailed" json:"clientfailed,omitempty"`
	WorkFailed   string `bson:"workfailed" json:"workfailed,omitempty"`
	TaskFailed   string `bson:"taskfailed" json:"taskfailed,omitempty"`
	ServerNotes  string `bson:"servernotes" json:"servernotes,omitempty"`
	WorkNotes    string `bson:"worknotes" json:"worknotes,omitempty"`
	AppError     string `bson:"apperror" json:"apperror,omitempty"`
	Status       string `bson:"status" json:"status,omitempty"`
}

// Job _
type Job struct {
	JobRaw `bson:",inline"`
	Tasks  []*Task `bson:"tasks" json:"tasks"`
}

// JobRaw _
type JobRaw struct {
	rwmutex.RWMutex
	ID                      string                       `bson:"id" json:"id"` // uuid
	ACL                     acl.Acl                      `bson:"acl" json:"-"`
	Info                    *Info                        `bson:"info" json:"info"`
	Script                  script                       `bson:"script" json:"-"`
	State                   string                       `bson:"state" json:"state"`
	Registered              bool                         `bson:"registered" json:"registered"`
	RemainTasks             int                          `bson:"remaintasks" json:"remaintasks"` // old-style AWE
	RemainSteps             int                          `bson:"remainsteps" json:"remainteps"`
	Expiration              time.Time                    `bson:"expiration" json:"expiration"` // 0 means no expiration
	UpdateTime              time.Time                    `bson:"updatetime" json:"updatetime"`
	Error                   *JobError                    `bson:"error" json:"error"`         // error struct exists when in suspended state
	Resumed                 int                          `bson:"resumed" json:"resumed"`     // number of times the job has been resumed from suspension
	ShockHost               string                       `bson:"shockhost" json:"shockhost"` // this is a fall-back default if not specified at a lower level
	IsCWL                   bool                         `bson:"is_cwl" json:"is_cwl"`
	CWL_job_input           interface{}                  `bson:"cwl_job_input" json:"cwl_job_input"` // has to be an array for mongo (id as key would not work)
	CWL_ShockRequirement    *cwl.ShockRequirement        `bson:"cwl_shock_requirement" json:"cwl_shock_requirement"`
	CWL_workflow            *cwl.Workflow                `bson:"-" json:"-" yaml:"-" mapstructure:"-"`
	WorkflowInstancesMap    map[string]*WorkflowInstance `bson:"-" json:"-" yaml:"-" mapstructure:"-"`
	WorkflowInstancesRemain int                          `bson:"workflow_instances_remain" json:"workflow_instances_remain"`
	Entrypoint              string                       `bson:"entrypoint" json:"entrypoint"` // name of main workflow (typically has name #main or #entrypoint)
	Root                    string                       `bson:"root" json:"root"`             // UUID of root workflow instance
	WorkflowContext         *cwl.WorkflowContext         `bson:"context" json:"context" yaml:"context" mapstructure:"context"`
}

// GetID _
func (job *JobRaw) GetID(doReadLock bool) (id string, err error) {
	if doReadLock {
		readLock, xerr := job.RLockNamed("String")
		if xerr != nil {
			err = xerr
			return
		}
		defer job.RUnlockNamed(readLock)
	}

	id = job.ID
	return
}

// AddWorkflowInstance _
func (job *Job) AddWorkflowInstance(wi *WorkflowInstance, dbSync bool, writeLock bool) (err error) {
	//fmt.Printf("(AddWorkflowInstance) id: %s\n", wi.LocalID)
	if writeLock {
		err = job.LockNamed("AddWorkflowInstance")
		if err != nil {
			return
		}
		defer job.Unlock()
	}

	if job.WorkflowInstancesMap == nil {
		job.WorkflowInstancesMap = make(map[string]*WorkflowInstance)
	} else {

		_, hasWI := job.WorkflowInstancesMap[wi.LocalID]
		if hasWI {

			//err = fmt.Errorf("(AddWorkflowInstance) WorkflowInstance %s already in map !", wi.LocalID)
			return
		}
	}

	job.WorkflowInstancesMap[wi.LocalID] = wi

	if len(wi.Tasks) > 0 {
		err = fmt.Errorf("(AddWorkflowInstance) already has tasks !?")
		return
	}

	err = wi.SetStateNoSync(WIStatePending, true) // AddWorkflowInstance
	if err != nil {
		err = fmt.Errorf("(AddWorkflowInstance) wi.SetStateNoSync returned: %s (wi.LocalID: %s)", err.Error(), wi.LocalID)
		return
	}

	if dbSync == DbSyncTrue {

		err = dbInsert(wi)
		if err != nil {
			err = fmt.Errorf("(AddWorkflowInstance) dbUpsert(wi) returned: %s", err.Error())
			return
		}
	}

	// err = job.IncrementWorkflowInstancesRemain(1, db_sync, false)
	// if err != nil {
	// 	err = fmt.Errorf("(AddWorkflowInstance) job.IncrementWorkflowInstancesRemain returned: %s", err.Error())
	// 	return
	// }

	//logger.Debug(3, "(AddWorkflowInstance) wi.LocalId: %s , old_state: %s, db_sync: %s", wi.LocalId, old_state, db_sync)

	return
}

// func (job *Job) GetWorkflowInstanceIndex(id string, context *cwl.WorkflowContext, doReadLock bool) (index int, err error) {
// 	if doReadLock {
// 		readLock, xerr := job.RLockNamed("GetWorkflowInstanceIndex")
// 		if xerr != nil {
// 			err = xerr
// 			return
// 		}
// 		defer job.RUnlockNamed(readLock)
// 	}
// 	if id == "" {
// 		id = "_main"
// 	}

// 	var wi_int interface{}

// 	for index, wi_int = range job.WorkflowInstances {
// 		var element_wi WorkflowInstance
// 		element_wi, err = NewWorkflowInstanceFromInterface(wi_int, context)
// 		if err != nil {
// 			err = fmt.Errorf("(GetWorkflowInstance) object was not a WorkflowInstance !? %s", err.Error())
// 			return
// 		}
// 		if element_wi.Id == id {
// 			//job.WorkflowInstancesMap[id] = &element_wi
// 			//wi = &element_wi
// 			return
// 		}

// 	}

// 	err = fmt.Errorf("(GetWorkflowInstance) WorkflowInstance %s not found", id)
// 	return

// }

// GetWorkflowInstance _
func (job *Job) GetWorkflowInstance(id string, doReadLock bool) (wi *WorkflowInstance, ok bool, err error) {
	if doReadLock {
		readLock, xerr := job.RLockNamed("GetWorkflowInstance")
		if xerr != nil {
			err = xerr
			return
		}
		defer job.RUnlockNamed(readLock)
	}
	//if id == "" {
	//	id = "_main"
	//}

	wi, ok = job.WorkflowInstancesMap[id]
	if !ok {
		return
	}

	return
}

// func (job *Job) Set_WorkflowInstance_Outputs(id string, outputs cwl.Job_document, context *cwl.WorkflowContext) (err error) {
// 	err = job.LockNamed("Set_WorkflowInstance_Outputs")
// 	if err != nil {
// 		return
// 	}
// 	defer job.Unlock()

// 	if id == "" {
// 		id = "_main"
// 	}

// 	err = dbUpdateJobWorkflow_instancesFieldOutputs(job.ID, id, outputs)
// 	if err != nil {
// 		err = fmt.Errorf("(Set_WorkflowInstance_Outputs) dbUpdateJobWorkflow_instancesFieldOutputs returned: %s", err.Error())
// 		return
// 	}

// 	var index int
// 	index, err = job.GetWorkflowInstanceIndex(id, context, false)
// 	if err != nil {
// 		err = fmt.Errorf("(Set_WorkflowInstance_Outputs) GetWorkflowInstanceIndex returned: %s", err.Error())
// 		return
// 	}

// 	var workflow_instance WorkflowInstance
// 	workflow_instance_if := job.WorkflowInstances[index]
// 	workflow_instance, err = NewWorkflowInstanceFromInterface(workflow_instance_if, context)
// 	if err != nil {
// 		err = fmt.Errorf("(Set_WorkflowInstance_Outputs) NewWorkflowInstanceFromInterface returned: %s", err.Error())
// 		return
// 	}

// 	workflow_instance.Outputs = outputs

// 	job.WorkflowInstances[index] = workflow_instance
// 	job.WorkflowInstancesMap[id] = &workflow_instance

// 	return
// }

// Deprecated JobDep struct uses deprecated TaskDep struct which uses the deprecated IOmap.  Maintained for backwards compatibility.
// Jobs that cannot be parsed into the Job struct, but can be parsed into the JobDep struct will be translated to the new Job struct.
// (=deprecated=)
type JobDep struct {
	JobRaw `bson:",inline"`
	Tasks  []*TaskDep `bson:"tasks" json:"tasks"`
}

// JobMin _
type JobMin struct {
	ID            string                 `bson:"id" json:"id"`
	Name          string                 `bson:"name" json:"name"`
	Size          int64                  `bson:"size" json:"size"`
	SubmitTime    time.Time              `bson:"submittime" json:"submittime"`
	CompletedTime time.Time              `bson:"completedtime" json:"completedtime"`
	ComputeTime   int                    `bson:"computetime" json:"computetime"`
	Task          []int                  `bson:"task" json:"task"`
	State         []string               `bson:"state" json:"state"`
	UserAttr      map[string]interface{} `bson:"userattr" json:"userattr"`
}

// JobLog _
type JobLog struct {
	ID         string     `bson:"id" json:"id"`
	State      string     `bson:"state" json:"state"`
	UpdateTime time.Time  `bson:"updatetime" json:"updatetime"`
	Error      *JobError  `bson:"error" json:"error"`
	Resumed    int        `bson:"resumed" json:"resumed"`
	Tasks      []*TaskLog `bson:"tasks" json:"tasks"`
}

// NewJobRaw _
func NewJobRaw() (job *JobRaw) {
	r := &JobRaw{
		Info: NewInfo(),
		ACL:  acl.Acl{},
	}
	r.RWMutex.Init("Job")
	return r
}

// NewJob _
func NewJob() (job *Job) {
	//r_job := NewJobRaw()
	job = &Job{}
	//job.JobRaw = *r_job
	job.JobRaw = *NewJobRaw()
	return
}

// NewJobDep _
func NewJobDep() (job *JobDep) {
	//r_job := NewJobRaw()
	//job = &JobDep{JobRaw: *r_job}
	job = &JobDep{}
	job.JobRaw = *NewJobRaw()
	return
}

// Init this has to be called after Unmarshalling from JSON
func (job *Job) Init() (changed bool, err error) {
	changed = false
	job.RWMutex.Init("Job")

	if job.State == "" {
		job.State = JOB_STAT_INIT
		changed = true
	}
	job.Registered = true

	if job.ID == "" {
		job.setID() //uuid for the job
		logger.Debug(3, "(Job.Init) Set JobID: %s", job.ID)
		changed = true
	} else {
		logger.Debug(3, "(Job.Init)  Already have JobID: %s", job.ID)
	}

	if job.Info == nil {
		logger.Error("job.Info == nil")
		job.Info = NewInfo()
	}

	if job.Info.SubmitTime.IsZero() {
		job.Info.SubmitTime = time.Now()
		changed = true
	}

	if job.Info.Priority < conf.BasePriority {
		job.Info.Priority = conf.BasePriority
		changed = true
	}

	context := job.WorkflowContext

	if context != nil && context.IfObjects == nil {
		if context.Graph == nil {
			err = fmt.Errorf("(job.Init) job.WorkflowContext.Graph == nil")
			return
		}

		if len(context.Graph) == 0 {
			err = fmt.Errorf("(job.Init) len(job.WorkflowContext.Graph) == 0")
			return
		}

		err = context.Init(job.Entrypoint)
		if err != nil {
			err = fmt.Errorf("(job.Init) context.Init() returned: %s", err.Error())
			return
		}
	}

	oldRemainTasks := job.RemainTasks // job.Init
	newRemainTasks := 0

	for _, task := range job.Tasks {
		if task.ID == "" {
			// suspend and create error
			logger.Error("(job.Init) task.Id empty, job %s broken?", job.ID)
			//task.Id = job.ID + "_" + uuid.New()
			task.ID = uuid.New()
			job.State = JOB_STAT_SUSPEND
			job.Error = &JobError{
				ServerNotes: "task.Id was empty",
				TaskFailed:  task.ID,
			}
			changed = true
		}
		t_changed, xerr := task.Init(job, job.ID)
		if xerr != nil {
			err = fmt.Errorf("(job.Init) task.Init returned: %s", xerr.Error())
			return
		}
		if t_changed {
			changed = true
		}
		if task.State != TASK_STAT_COMPLETED {
			newRemainTasks++ // job.Init
		}
	}

	// try to fix inconsistent state
	if !job.IsCWL && (newRemainTasks != oldRemainTasks) { // job.Init
		job.RemainTasks = newRemainTasks
		changed = true
	}

	// try to fix inconsistent state
	//if job.RemainSteps > 0 && job.State == JOB_STAT_COMPLETED {
	//	job.State = JOB_STAT_QUEUED
	//	logger.Debug(3, "fixing state to JOB_STAT_QUEUED")
	//	changed = true
	//}

	//if len(job.Tasks) == 0 {
	//	err = errors.New("(job.Init) invalid job script: task list empty")
	//	return
	//}

	// check that input FileName is not repeated within an individual task
	for _, task := range job.Tasks {
		inputFileNames := make(map[string]bool)
		for _, io := range task.Inputs {
			if _, exists := inputFileNames[io.FileName]; exists {
				var task_str string
				task_str, err = task.String()
				if err != nil {
					return
				}
				err = fmt.Errorf("(job.Init) invalid inputs: task %s contains multiple inputs with filename=%s", task_str, io.FileName)
				return
			}
			inputFileNames[io.FileName] = true
		}
	}

	//var workflow *cwl.Workflow

	if job.IsCWL {

		entrypoint := job.Entrypoint
		var cwlWorkflow *cwl.Workflow
		cwlWorkflow, err = context.GetWorkflow(entrypoint)
		//cwl_workflow, ok := context.Workflows[entrypoint]
		if err != nil {
			err = fmt.Errorf("(job.Init) Workflow \"%s\" not found: %s", entrypoint, err.Error())

			//for key, _ := range context.Workflows {
			//	fmt.Printf("(job.Init) Workflows key: %s\n", key)
			//}
			for key, _ := range context.Objects {
				fmt.Printf("(job.Init) All key: %s\n", key)
			}
			return
		}

		job.WorkflowContext = context
		job.CWL_workflow = cwlWorkflow

	}

	return
}

// GetRemainTasks _
func (job *Job) GetRemainTasks() (remainTasks int, err error) {
	remainTasks = job.RemainTasks
	return
}

// SetRemainTasks _
func (job *Job) SetRemainTasks(remainTasks int) (err error) {
	err = job.LockNamed("SetRemainTasks")
	if err != nil {
		return
	}
	defer job.Unlock()

	if remainTasks == job.RemainTasks {
		return
	}
	err = dbUpdateJobFieldInt(job.ID, "remaintasks", remainTasks)
	if err != nil {
		return
	}
	job.RemainTasks = remainTasks
	return
}

// IncrementRemainTasks _
func (job *Job) IncrementRemainTasks(inc int) (err error) {
	err = job.LockNamed("IncrementRemainTasks")
	if err != nil {
		return
	}
	defer job.Unlock()

	logger.Debug(3, "(IncrementRemainTasks) called with inc=%d", inc)

	newRemainTask := job.RemainTasks + inc
	logger.Debug(3, "(IncrementRemainTasks) new value of RemainTasks: %d", newRemainTask)
	err = dbUpdateJobFieldInt(job.ID, "remaintasks", newRemainTask)
	if err != nil {
		return
	}
	job.RemainTasks = newRemainTask
	return
}

// RLockRecursive _
func (job *Job) RLockRecursive() {
	for _, task := range job.Tasks {
		task.RLockAnon()
	}
}

// RUnlockRecursive _
func (job *Job) RUnlockRecursive() {
	for _, task := range job.Tasks {
		task.RUnlockAnon()
	}
}

//setID set job's uuid
func (job *Job) setID() {
	job.ID = uuid.New()
	return
}

type script struct {
	Name string `bson:"name" json:"name"`
	Type string `bson:"type" json:"type"`
	Path string `bson:"path" json:"-"`
}

//---Script upload (e.g. field="upload")
func (job *Job) UpdateFile(files FormFiles, field string) (err error) {
	_, isRegularUpload := files[field]
	if isRegularUpload {
		if err = job.SetFile(files[field]); err != nil {
			return err
		}
		delete(files, field)
	}
	return
}

// SaveToDisk _
func (job *Job) SaveToDisk() (err error) {
	var job_path string
	job_path, err = job.Path()
	if err != nil {
		err = fmt.Errorf("Save() Path error: %v", err)
		return
	}
	bsonPath := path.Join(job_path, job.ID+".bson")
	os.Remove(bsonPath)
	logger.Debug(1, "Save() bson.Marshal next: %s", job.ID)
	nbson, err := bson.Marshal(job)
	if err != nil {
		err = errors.New("error in Marshal in job.Save(), error=" + err.Error())
		return
	}
	// this is incase job path does not exist, ignored if it does
	err = job.Mkdir()
	if err != nil {
		err = errors.New("error creating dir in job.Save(), error=" + err.Error())
		return
	}
	err = ioutil.WriteFile(bsonPath, nbson, 0644)
	if err != nil {
		err = errors.New("error writing file in job.Save(), error=" + err.Error())
		return
	}
	return
}

// Deserialize_b64 _
func Deserialize_b64(encoding string, target interface{}) (err error) {
	byte_array, err := b64.StdEncoding.DecodeString(encoding)
	if err != nil {
		err = fmt.Errorf("(Deserialize_b64) DecodeString error: %s", err.Error())
		return
	}

	err = json.Unmarshal(byte_array, target)
	if err != nil {
		return
	}

	return
}

// Save _
func (job *Job) Save() (err error) {

	if job.ID == "" {
		err = fmt.Errorf("(job.Save()) job id empty")
		return
	}
	logger.Debug(1, "(job.Save()) saving job: %s", job.ID)

	job.UpdateTime = time.Now()
	err = job.SaveToDisk()
	if err != nil {
		err = fmt.Errorf("(job.Save()) SaveToDisk failed: %s", err.Error())
		return
	}

	logger.Debug(1, "(job.Save()) dbUpsert next: %s", job.ID)
	//spew.Dump(job)

	err = dbUpsert(job)
	if err != nil {
		err = fmt.Errorf("(job.Save()) dbUpsert failed (job_id=%s) error=%s", job.ID, err.Error())
		return
	}
	logger.Debug(1, "(job.Save()) job saved: %s", job.ID)
	return
}

// Delete _
func (job *Job) Delete() (err error) {
	if err = dbDelete(bson.M{"id": job.ID}, conf.DB_COLL_JOBS); err != nil {
		return err
	}
	if err = job.Rmdir(); err != nil {
		return err
	}
	logger.Event(event.JOB_FULL_DELETE, "jobid="+job.ID)
	return
}

// Mkdir _
func (job *Job) Mkdir() (err error) {
	var path string
	path, err = job.Path()
	if err != nil {
		return
	}
	err = os.MkdirAll(path, 0777)
	if err != nil {
		err = fmt.Errorf("Could not run os.MkdirAll (path: %s) %s", path, err.Error())
		return
	}
	return
}

// Rmdir _
func (job *Job) Rmdir() (err error) {
	var path string
	path, err = job.Path()
	if err != nil {
		return
	}
	return os.RemoveAll(path)
}

// SetFile _
func (job *Job) SetFile(file FormFile) (err error) {
	var path string
	path, err = job.FilePath()
	os.Rename(file.Path, path)
	job.Script.Name = file.Name
	return
}

// Path ---Path functions
func (job *Job) Path() (path string, err error) {
	return getPathByJobID(job.ID)
}

// FilePath _
func (job *Job) FilePath() (path string, err error) {
	if job.Script.Path != "" {
		path = job.Script.Path
		return
	}
	path, err = getPathByJobID(job.ID)
	if err != nil {
		return
	}
	path = path + "/" + job.ID + ".script"
	return
}

func getPathByJobID(id string) (path string, err error) {
	if len(id) < 6 {
		err = fmt.Errorf("Job-Id format wrong: \"%s\"", id)
		return
	}
	path = fmt.Sprintf("%s/%s/%s/%s/%s", conf.DATA_PATH, id[0:2], id[2:4], id[4:6], id)
	return
}

// GetTasks get tasks form from all subworkflows in the job
func (job *Job) GetTasks() (tasks []*Task, err error) {
	tasks = []*Task{}

	readLock, err := job.RLockNamed("GetTasks")
	if err != nil {
		return
	}
	defer job.RUnlockNamed(readLock)

	if job.IsCWL {
		logger.Debug(3, "(GetTasks) iscwl len(job.WorkflowInstancesMap): %d", len(job.WorkflowInstancesMap))
		for _, wi := range job.WorkflowInstancesMap {

			var wi_tasks []*Task
			wi_tasks, err = wi.GetTasks(true)
			if err != nil {
				return
			}

			logger.Debug(3, "(GetTasks) wi_tasks: %d", len(wi_tasks))
			for _, task := range wi_tasks {
				tasks = append(tasks, task)
			}
		}

	} else {
		logger.Debug(3, "(GetTasks) is not cwl")
		for _, task := range job.Tasks {
			tasks = append(tasks, task)
		}
	}
	return
}

func (job *Job) GetState(do_lock bool) (state string, err error) {
	if do_lock {
		readLock, xerr := job.RLockNamed("GetState")
		if xerr != nil {
			err = xerr
			return
		}
		defer job.RUnlockNamed(readLock)
	}
	state = job.State
	return
}

func (job *Job) GetStateTimeout(do_lock bool, timeout time.Duration) (state string, err error) {
	if do_lock {
		readLock, xerr := job.RLockNamedTimeout("GetStateTimeout", timeout)
		if xerr != nil {
			err = xerr
			return
		}
		defer job.RUnlockNamed(readLock)
	}
	state = job.State
	return
}

//---Task functions
func (job *Job) TaskList() []*Task {
	return job.Tasks
}

func (job *Job) NumTask() int {
	return len(job.Tasks)
}

func (job *Job) AddTask(task *Task) (err error) {
	err = job.LockNamed("job/AddTask")
	if err != nil {
		return
	}
	defer job.Unlock()

	id := job.ID

	err = dbPushJobTask(id, task)
	if err != nil {
		return
	}

	job.Tasks = append(job.Tasks, task)
	return
}

//---Field update functions

func (job *Job) SetState(newState string, oldstates []string) (err error) {
	err = job.LockNamed("job/SetState")
	if err != nil {
		return
	}
	defer job.Unlock()

	job_state := job.State

	if job_state == newState {
		return
	}

	if len(oldstates) > 0 {
		matched := false
		for _, oldstate := range oldstates {
			if oldstate == job_state {
				matched = true
				break
			}
		}
		if !matched {
			oldstates_str := strings.Join(oldstates, ",")
			err = fmt.Errorf("(SetState) newState: %s, old state %s does not match one of the required ones (required: %s)", newState, job_state, oldstates_str)
			return
		}
	}

	err = dbUpdateJobFieldString(job.ID, "state", newState)
	if err != nil {
		return
	}
	job.State = newState

	// set time if completed
	switch newState {
	case JOB_STAT_COMPLETED:
		newTime := time.Now()
		err = dbUpdateJobFieldTime(job.ID, "info.completedtime", newTime)
		if err != nil {
			return
		}
		job.Info.CompletedTime = newTime
	case JOB_STAT_INPROGRESS:
		time_now := time.Now()
		jobid := job.ID
		err = job.Info.SetStartedTime(jobid, time_now)
		if err != nil {
			return
		}

	}

	// unset error if not suspended
	if (newState != JOB_STAT_SUSPEND) && (job.Error != nil) {
		err = dbUpdateJobFieldNull(job.ID, "error")
		if err != nil {
			return
		}
		job.Error = nil
	}
	return
}

func (job *Job) SetError(newError *JobError) (err error) {
	err = job.LockNamed("SetError")
	if err != nil {
		return
	}
	defer job.Unlock()
	//spew.Dump(newError)

	update_value := bson.M{"error": newError}
	err = dbUpdateJobFields(job.ID, update_value)
	if err != nil {
		return
	}
	job.Error = newError
	return
}

func (job *Job) IncrementResumed(inc int) (err error) {
	err = job.LockNamed("IncrementResumed")
	if err != nil {
		return
	}
	defer job.Unlock()

	newResumed := job.Resumed + inc
	err = dbUpdateJobFieldInt(job.ID, "resumed", newResumed)
	if err != nil {
		return
	}
	job.Resumed = newResumed
	return
}

func (job *Job) SetClientgroups(clientgroups string) (err error) {
	err = job.LockNamed("SetClientgroups")
	if err != nil {
		return
	}
	defer job.Unlock()

	err = dbUpdateJobFieldString(job.ID, "info.clientgroups", clientgroups)
	if err != nil {
		return
	}
	job.Info.ClientGroups = clientgroups
	return
}

func (job *Job) SetPriority(priority int) (err error) {
	err = job.LockNamed("SetPriority")
	if err != nil {
		return
	}
	defer job.Unlock()

	err = dbUpdateJobFieldInt(job.ID, "info.priority", priority)
	if err != nil {
		return
	}
	job.Info.Priority = priority
	return
}

func (job *Job) SetPipeline(pipeline string) (err error) {
	err = job.LockNamed("SetPipeline")
	if err != nil {
		return
	}
	defer job.Unlock()

	err = dbUpdateJobFieldString(job.ID, "info.pipeline", pipeline)
	if err != nil {
		return
	}
	job.Info.Pipeline = pipeline
	return
}

func (job *Job) SetDataToken(token string) (err error) {
	err = job.LockNamed("SetDataToken")
	if err != nil {
		return
	}
	defer job.Unlock()

	if job.Info.DataToken == token {
		return
	}
	// update toekn in info
	err = dbUpdateJobFieldString(job.ID, "info.token", token)
	if err != nil {
		return
	}
	job.Info.DataToken = token

	// update token in IO structs
	err = QMgr.UpdateQueueToken(job)
	if err != nil {
		return
	}

	// set using auth if not before
	if !job.Info.Auth {
		err = dbUpdateJobFieldBoolean(job.ID, "info.auth", true)
		if err != nil {
			return
		}
		job.Info.Auth = true
	}
	return
}

func (job *Job) SetExpiration(expire string) (err error) {
	err = job.LockNamed("SetExpiration")
	if err != nil {
		return
	}
	defer job.Unlock()

	parts := ExpireRegex.FindStringSubmatch(expire)
	if len(parts) == 0 {
		return errors.New("expiration format '" + expire + "' is invalid")
	}
	var expireTime time.Duration
	expireNum, _ := strconv.Atoi(parts[1])
	currTime := time.Now()

	switch parts[2] {
	case "M":
		expireTime = time.Duration(expireNum) * time.Minute
	case "H":
		expireTime = time.Duration(expireNum) * time.Hour
	case "D":
		expireTime = time.Duration(expireNum*24) * time.Hour
	}

	newExpiration := currTime.Add(expireTime)
	err = dbUpdateJobFieldTime(job.ID, "expiration", newExpiration)
	if err != nil {
		return
	}
	job.Expiration = newExpiration
	return
}

func (job *Job) GetDataToken() (token string) {
	return job.Info.DataToken
}

func (job *Job) GetPrivateEnv(taskid string) (env map[string]string, err error) {
	for _, task := range job.Tasks {
		var task_str string
		task_str, err = task.String()
		if err != nil {
			return
		}
		if taskid == task_str {
			env = task.Cmd.Environ.Private
			return
		}
	}
	return
}

func (job *Job) GetJobLogs() (jlog *JobLog, err error) {
	jlog = new(JobLog)
	jlog.ID = job.ID
	jlog.State = job.State
	jlog.UpdateTime = job.UpdateTime
	jlog.Error = job.Error
	jlog.Resumed = job.Resumed
	for _, task := range job.Tasks {
		var tl *TaskLog
		tl, err = task.GetTaskLogs()
		if err != nil {
			return
		}
		jlog.Tasks = append(jlog.Tasks, tl)
	}
	return
}

func ReloadFromDisk(path string) (err error) {
	id := filepath.Base(path)
	jobbson, err := ioutil.ReadFile(path + "/" + id + ".bson")
	if err != nil {
		return
	}
	job := NewJob()
	err = bson.Unmarshal(jobbson, &job)
	if err == nil {
		if err = dbUpsert(job); err != nil {
			return err
		}
	}
	return
}
