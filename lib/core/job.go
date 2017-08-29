package core

import (
	b64 "encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
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

var JOB_STATS_ACTIVE = []string{JOB_STAT_QUEUED, JOB_STAT_INPROGRESS}
var JOB_STATS_REGISTERED = []string{JOB_STAT_QUEUING, JOB_STAT_QUEUED, JOB_STAT_INPROGRESS, JOB_STAT_SUSPEND}
var JOB_STATS_TO_RECOVER = []string{JOB_STAT_INIT, JOB_STAT_QUEUED, JOB_STAT_INPROGRESS, JOB_STAT_SUSPEND}

type JobError struct {
	ClientFailed string `bson:"clientfailed" json:"clientfailed"`
	WorkFailed   string `bson:"workfailed" json:"workfailed"`
	TaskFailed   string `bson:"taskfailed" json:"taskfailed"`
	ServerNotes  string `bson:"servernotes" json:"servernotes"`
	WorkNotes    string `bson:"worknotes" json:"worknotes"`
	AppError     string `bson:"apperror" json:"apperror"`
	Status       string `bson:"status" json:"status"`
}

type Job struct {
	JobRaw `bson:",inline"`
	Tasks  []*Task `bson:"tasks" json:"tasks"`
}

type JobRaw struct {
	RWMutex
	Id                      string              `bson:"id" json:"id"` // uuid
	Acl                     acl.Acl             `bson:"acl" json:"-"`
	Info                    *Info               `bson:"info" json:"info"`
	Script                  script              `bson:"script" json:"-"`
	State                   string              `bson:"state" json:"state"`
	Registered              bool                `bson:"registered" json:"registered"`
	RemainTasks             int                 `bson:"remaintasks" json:"remaintasks"`
	Expiration              time.Time           `bson:"expiration" json:"expiration"` // 0 means no expiration
	UpdateTime              time.Time           `bson:"updatetime" json:"updatetime"`
	Error                   *JobError           `bson:"error" json:"error"`         // error struct exists when in suspended state
	Resumed                 int                 `bson:"resumed" json:"resumed"`     // number of times the job has been resumed from suspension
	ShockHost               string              `bson:"shockhost" json:"shockhost"` // this is a fall-back default if not specified at a lower level
	CWL_job_input_interface interface{}         `bson:"cwl_job_input" json:"cwl_job_input`
	CWL_workflow_interface  interface{}         `bson:"cwl_workflow" json:"cwl_workflow`
	CWL_collection          *cwl.CWL_collection `bson:"-" json:"-" yaml:"-" mapstructure:"-"`
	CWL_job_input           *cwl.Job_document   `bson:"-" json:"-" yaml:"-" mapstructure:"-"`
	CWL_workflow            *cwl.Workflow       `bson:"-" json:"-" yaml:"-" mapstructure:"-"`
}

// Deprecated JobDep struct uses deprecated TaskDep struct which uses the deprecated IOmap.  Maintained for backwards compatibility.
// Jobs that cannot be parsed into the Job struct, but can be parsed into the JobDep struct will be translated to the new Job struct.
// (=deprecated=)
type JobDep struct {
	JobRaw `bson:",inline"`
	Tasks  []*TaskDep `bson:"tasks" json:"tasks"`
}

type JobMin struct {
	Id            string            `bson:"id" json:"id"`
	Name          string            `bson:"name" json:"name"`
	Size          int64             `bson:"size" json:"size"`
	SubmitTime    time.Time         `bson:"submittime" json:"submittime"`
	CompletedTime time.Time         `bson:"completedtime" json:"completedtime"`
	ComputeTime   int               `bson:"computetime" json:"computetime"`
	Task          []int             `bson:"task" json:"task"`
	State         []string          `bson:"state" json:"state"`
	UserAttr      map[string]string `bson:"userattr" json:"userattr"`
}

type JobLog struct {
	Id         string     `bson:"id" json:"id"`
	State      string     `bson:"state" json:"state"`
	UpdateTime time.Time  `bson:"updatetime" json:"updatetime"`
	Error      *JobError  `bson:"error" json:"error"`
	Resumed    int        `bson:"resumed" json:"resumed"`
	Tasks      []*TaskLog `bson:"tasks" json:"tasks"`
}

func NewJobRaw() (job *JobRaw) {
	r := &JobRaw{
		Info: NewInfo(),
		Acl:  acl.Acl{},
	}
	r.RWMutex.Init("Job")
	return r
}

func NewJob() (job *Job) {
	r_job := NewJobRaw()
	job = &Job{JobRaw: *r_job}
	return
}

func NewJobDep() (job *JobDep) {
	r_job := NewJobRaw()
	job = &JobDep{JobRaw: *r_job}
	return
}

// this has to be called after Unmarshalling from JSON
func (job *Job) Init() (changed bool, err error) {
	changed = false
	job.RWMutex.Init("Job")

	if job.State == "" {
		job.State = JOB_STAT_INIT
		changed = true
	}
	job.Registered = true

	if job.Id == "" {
		job.setId() //uuid for the job
		logger.Debug(3, "(Job.Init) Set JobID: %s", job.Id)
		changed = true
	} else {
		logger.Debug(3, "(Job.Init)  Already have JobID: %s", job.Id)
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

	old_remaintasks := job.RemainTasks
	job.RemainTasks = 0

	for _, task := range job.Tasks {
		if task.Id == "" {
			// suspend and create error
			logger.Error("(job.Init) task.Id empty, job %s broken?", job.Id)
			//task.Id = job.Id + "_" + uuid.New()
			task.Id = uuid.New()
			job.State = JOB_STAT_SUSPEND
			job.Error = &JobError{
				ServerNotes: "task.Id was empty",
				TaskFailed:  task.Id,
			}
			changed = true
		}
		t_changed, xerr := task.Init(job)
		if xerr != nil {
			err = xerr
			return
		}
		if t_changed {
			changed = true
		}
		if task.State != TASK_STAT_COMPLETED {
			job.RemainTasks += 1
		}
	}

	// try to fix inconsistent state
	if job.RemainTasks != old_remaintasks {
		changed = true
	}

	// try to fix inconsistent state
	if job.RemainTasks == 0 && job.State != JOB_STAT_COMPLETED {
		job.State = JOB_STAT_COMPLETED
		logger.Debug(3, "fixing state to JOB_STAT_COMPLETED")
		changed = true
	}

	// fix job.Info.CompletedTime
	if job.State == JOB_STAT_COMPLETED && job.Info.CompletedTime.IsZero() {
		// better now, than never:
		job.Info.CompletedTime = time.Now()
		changed = true
	}

	// try to fix inconsistent state
	if job.RemainTasks > 0 && job.State == JOB_STAT_COMPLETED {
		job.State = JOB_STAT_QUEUED
		logger.Debug(3, "fixing state to JOB_STAT_QUEUED")
		changed = true
	}

	if len(job.Tasks) == 0 {
		err = errors.New("(job.Init) invalid job script: task list empty")
		return
	}

	// check that input FileName is not repeated within an individual task
	for _, task := range job.Tasks {
		inputFileNames := make(map[string]bool)
		for _, io := range task.Inputs {
			if _, exists := inputFileNames[io.FileName]; exists {
				err = errors.New("(job.Init) invalid inputs: task " + task.Id + " contains multiple inputs with filename=" + io.FileName)
				return
			}
			inputFileNames[io.FileName] = true
		}
	}

	var workflow *cwl.Workflow

	if job.CWL_workflow_interface != nil {
		workflow, err = cwl.NewWorkflow(job.CWL_workflow_interface)
		if err != nil {
			return
		}
		job.CWL_workflow = workflow
	}

	var job_input *cwl.Job_document
	if job.CWL_job_input_interface != nil {
		job_input, err = cwl.NewJob_document(job.CWL_job_input_interface)
		if err != nil {
			return
		}
		job.CWL_job_input = job_input
	}

	// read from base64 string
	// if job.CWL_collection == nil && job.CWL_job_input_b64 != "" {
	// 		new_collection := cwl.NewCWL_collection()
	//
	// 		//1) parse job
	// 		job_input_byte_array, xerr := b64.StdEncoding.DecodeString(job.CWL_job_input_b64)
	// 		if xerr != nil {
	// 			err = fmt.Errorf("(job.Init) error decoding CWL_job_input_b64: %s", xerr.Error())
	// 			return
	// 		}
	//
	// 		job_input, xerr := cwl.ParseJob(&job_input_byte_array)
	// 		if xerr != nil {
	// 			err = fmt.Errorf("(job.Init) error in reading job yaml/json file: " + xerr.Error())
	// 			return
	// 		}
	// 		new_collection.Job_input = job_input
	//
	// 		// 2) parse workflow document
	//
	// 		workflow_byte_array, xerr := b64.StdEncoding.DecodeString(job.CWL_workflow_b64)
	//
	// 		err = cwl.Parse_cwl_document(&new_collection, string(workflow_byte_array[:]))
	// 		if err != nil {
	// 			err = fmt.Errorf("(job.Init) Parse_cwl_document error: " + err.Error())
	//
	// 			return
	// 		}
	//
	// 		cwl_workflow, ok := new_collection.Workflows["#main"]
	// 		if !ok {
	//
	// 			err = fmt.Errorf("(job.Init) Workflow main not found")
	// 			return
	// 		}
	//
	// 		job.CWL_collection = &new_collection
	// 		_ = cwl_workflow
	// 	}

	return
}

func (job *Job) RLockRecursive() {
	for _, task := range job.Tasks {
		task.RLockAnon()
	}
}

func (job *Job) RUnlockRecursive() {
	for _, task := range job.Tasks {
		task.RUnlockAnon()
	}
}

//set job's uuid
func (job *Job) setId() {
	job.Id = uuid.New()
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

func (job *Job) SaveToDisk() (err error) {
	var job_path string
	job_path, err = job.Path()
	if err != nil {
		err = fmt.Errorf("Save() Path error: %v", err)
		return
	}
	bsonPath := path.Join(job_path, job.Id+".bson")
	os.Remove(bsonPath)
	logger.Debug(1, "Save() bson.Marshal next: %s", job.Id)
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

// takes a yaml string as input, may change that later
// func (job *Job) Set_CWL_workflow_b64(yaml_str string) {
//
// 	job.CWL_workflow_b64 = b64.StdEncoding.EncodeToString([]byte(yaml_str))
//
// 	return
// }

// func (job *Job) Set_CWL_job_input_b64(yaml_str string) {
//
// 	job.CWL_job_input_b64 = b64.StdEncoding.EncodeToString([]byte(yaml_str))
//
// 	return
// }

// func Serialize_b64(input interface{}) (serialized string, err error) {
// 	json_byte, xerr := json.Marshal(input)
// 	if xerr != nil {
// 		err = fmt.Errorf("(job.Save) json.Marshal(input) failed: %s", xerr.Error())
// 		return
// 	}
// 	serialized = b64.StdEncoding.EncodeToString(json_byte)
// 	return
// }

func (job *Job) Save() (err error) {
	if job.Id == "" {
		err = fmt.Errorf("(job.Save()) job id empty")
		return
	}
	logger.Debug(1, "(job.Save()) saving job: %s", job.Id)

	job.UpdateTime = time.Now()
	err = job.SaveToDisk()
	if err != nil {
		err = fmt.Errorf("(job.Save()) SaveToDisk failed: %s", err.Error())
		return
	}

	logger.Debug(1, "(job.Save()) dbUpsert next: %s", job.Id)
	//spew.Dump(job)

	err = dbUpsert(job)
	if err != nil {
		err = fmt.Errorf("(job.Save()) dbUpsert failed (job_id=%s) error=%s", job.Id, err.Error())
		return
	}
	logger.Debug(1, "(job.Save()) job saved: %s", job.Id)
	return
}

func (job *Job) Delete() (err error) {
	if err = dbDelete(bson.M{"id": job.Id}, conf.DB_COLL_JOBS); err != nil {
		return err
	}
	if err = job.Rmdir(); err != nil {
		return err
	}
	logger.Event(event.JOB_FULL_DELETE, "jobid="+job.Id)
	return
}

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

func (job *Job) Rmdir() (err error) {
	var path string
	path, err = job.Path()
	if err != nil {
		return
	}
	return os.RemoveAll(path)
}

func (job *Job) SetFile(file FormFile) (err error) {
	var path string
	path, err = job.FilePath()
	os.Rename(file.Path, path)
	job.Script.Name = file.Name
	return
}

//---Path functions
func (job *Job) Path() (path string, err error) {
	return getPathByJobId(job.Id)
}

func (job *Job) FilePath() (path string, err error) {
	if job.Script.Path != "" {
		path = job.Script.Path
		return
	}
	path, err = getPathByJobId(job.Id)
	if err != nil {
		return
	}
	path = path + "/" + job.Id + ".script"
	return
}

func getPathByJobId(id string) (path string, err error) {
	if len(id) < 6 {
		err = fmt.Errorf("Job-Id format wrong: \"%s\"", id)
		return
	}
	path = fmt.Sprintf("%s/%s/%s/%s/%s", conf.DATA_PATH, id[0:2], id[2:4], id[4:6], id)
	return
}

func (job *Job) GetTasks() (tasks []*Task, err error) {
	tasks = []*Task{}

	read_lock, err := job.RLockNamed("GetTasks")
	if err != nil {
		return
	}
	defer job.RUnlockNamed(read_lock)

	for _, task := range job.Tasks {
		tasks = append(tasks, task)
	}
	return
}

func (job *Job) GetState(do_lock bool) (state string, err error) {
	if do_lock {
		read_lock, xerr := job.RLockNamed("GetState")
		if xerr != nil {
			err = xerr
			return
		}
		defer job.RUnlockNamed(read_lock)
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

//---Field update functions

func (job *Job) SetState(newState string, oldstates []string) (err error) {
	err = job.LockNamed("SetState")
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
			err = fmt.Errorf("(UpdateJobState) old state %s does not match one of the required ones (required: %s)", job_state, oldstates_str)
			return
		}
	}

	err = dbUpdateJobFieldString(job.Id, "state", newState)
	if err != nil {
		return
	}
	job.State = newState

	// set time if completed
	switch newState {
	case JOB_STAT_COMPLETED:
		newTime := time.Now()
		err = dbUpdateJobFieldTime(job.Id, "info.completedtime", newTime)
		if err != nil {
			return
		}
		job.Info.CompletedTime = newTime
	case JOB_STAT_INPROGRESS:
		time_now := time.Now()
		jobid := job.Id
		err = job.Info.SetStartedTime(jobid, time_now)
		if err != nil {
			return
		}

	}

	// unset error if not suspended
	if (newState != JOB_STAT_SUSPEND) && (job.Error != nil) {
		err = dbUpdateJobFieldNull(job.Id, "error")
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
	spew.Dump(newError)

	update_value := bson.M{"error": newError}
	err = dbUpdateJobFields(job.Id, update_value)
	if err != nil {
		return
	}
	job.Error = newError
	return
}

func (job *Job) GetRemainTasks() (remain_tasks int, err error) {
	remain_tasks = job.RemainTasks
	return
}

func (job *Job) SetRemainTasks(remain_tasks int) (err error) {
	err = job.LockNamed("SetRemainTasks")
	if err != nil {
		return
	}
	defer job.Unlock()

	if remain_tasks == job.RemainTasks {
		return
	}
	err = dbUpdateJobFieldInt(job.Id, "remaintasks", remain_tasks)
	if err != nil {
		return
	}
	job.RemainTasks = remain_tasks
	return
}

func (job *Job) IncrementRemainTasks(inc int) (err error) {
	err = job.LockNamed("IncrementRemainTasks")
	if err != nil {
		return
	}
	defer job.Unlock()

	newRemainTask := job.RemainTasks + inc
	err = dbUpdateJobFieldInt(job.Id, "remaintasks", newRemainTask)
	if err != nil {
		return
	}
	job.RemainTasks = newRemainTask
	return
}

func (job *Job) IncrementResumed(inc int) (err error) {
	err = job.LockNamed("IncrementResumed")
	if err != nil {
		return
	}
	defer job.Unlock()

	newResumed := job.Resumed + inc
	err = dbUpdateJobFieldInt(job.Id, "resumed", newResumed)
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

	err = dbUpdateJobFieldString(job.Id, "info.clientgroups", clientgroups)
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

	err = dbUpdateJobFieldInt(job.Id, "info.priority", priority)
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

	err = dbUpdateJobFieldString(job.Id, "info.pipeline", pipeline)
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
	err = dbUpdateJobFieldString(job.Id, "info.token", token)
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
		err = dbUpdateJobFieldBoolean(job.Id, "info.auth", true)
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
	err = dbUpdateJobFieldTime(job.Id, "expiration", newExpiration)
	if err != nil {
		return
	}
	job.Expiration = newExpiration
	return
}

func (job *Job) GetDataToken() (token string) {
	return job.Info.DataToken
}

func (job *Job) GetPrivateEnv(taskid string) (env map[string]string) {
	for _, task := range job.Tasks {
		if taskid == task.Id {
			return task.Cmd.Environ.Private
		}
	}
	return
}

func (job *Job) GetJobLogs() (jlog *JobLog, err error) {
	jlog = new(JobLog)
	jlog.Id = job.Id
	jlog.State = job.State
	jlog.UpdateTime = job.UpdateTime
	jlog.Error = job.Error
	jlog.Resumed = job.Resumed
	for _, task := range job.Tasks {
		jlog.Tasks = append(jlog.Tasks, task.GetTaskLogs())
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
