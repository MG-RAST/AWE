package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"gopkg.in/mgo.v2/bson"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strconv"
	//"strings"
	"time"
)

const (
	JOB_STAT_INIT       = "init"
	JOB_STAT_QUEUED     = "queued"
	JOB_STAT_INPROGRESS = "in-progress"
	JOB_STAT_COMPLETED  = "completed"
	JOB_STAT_SUSPEND    = "suspend"
	JOB_STAT_FAILED     = "failed" // this sepcific error state can be trigger by the workflow software
	JOB_STAT_DELETED    = "deleted"
)

var JOB_STATS_ACTIVE = []string{JOB_STAT_QUEUED, JOB_STAT_INPROGRESS}
var JOB_STATS_REGISTERED = []string{JOB_STAT_QUEUED, JOB_STAT_INPROGRESS, JOB_STAT_SUSPEND}
var JOB_STATS_TO_RECOVER = []string{JOB_STAT_INIT, JOB_STAT_QUEUED, JOB_STAT_INPROGRESS, JOB_STAT_SUSPEND}

type JobRaw struct {
	RWMutex
	Id string `bson:"id" json:"id"`
	//Tasks       []*Task   `bson:"tasks" json:"tasks"`
	Acl         acl.Acl   `bson:"acl" json:"-"`
	Info        *Info     `bson:"info" json:"info"`
	Script      script    `bson:"script" json:"-"`
	State       string    `bson:"state" json:"state"`
	Registered  bool      `bson:"registered" json:"registered"`
	RemainTasks int       `bson:"remaintasks" json:"remaintasks"`
	Expiration  time.Time `bson:"expiration" json:"expiration"` // 0 means no expiration
	UpdateTime  time.Time `bson:"updatetime" json:"updatetime"`
	Notes       string    `bson:"notes" json:"notes"`
	LastFailed  string    `bson:"lastfailed" json:"lastfailed"`
	Resumed     int       `bson:"resumed" json:"resumed"`     // number of times the job has been resumed from suspension
	ShockHost   string    `bson:"shockhost" json:"shockhost"` // this is a fall-back default if not specified at a lower level
}

type Job struct {
	JobRaw `bson:",inline"`
	Tasks  []*Task `bson:"tasks" json:"tasks"`
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
	Notes      string     `bson:"notes" json:"notes"`
	LastFailed string     `bson:"lastfailed" json:"lastfailed"`
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

	create_new_tasks_array := false
	for _, task := range job.Tasks {

		if task.Id == "" {
			logger.Error("(job.Init) task.Id empty, job %s broken?", job.Id)
			task.Id = job.Id + "_" + uuid.New()
			if job.State != JOB_STAT_SUSPEND {
				job.State = JOB_STAT_SUSPEND
				job.Notes = "task.Id was empty"
				changed = true
			}

		}

		t_changed, xerr := task.Init(job)
		if xerr != nil {
			err = xerr
			return
		}

		if t_changed {
			changed = true
		}

		// set task.info pointer

		if job.Info == nil {
			err = fmt.Errorf("(job.Init) job.Info == nil")
			return
		}

		if task.State != TASK_STAT_COMPLETED {
			job.RemainTasks += 1
		}

		if task.Id == "" {
			create_new_tasks_array = true
		}

	}

	if create_new_tasks_array {
		new_tasks := []*Task{}
		for _, task := range job.Tasks {
			if task.Id != "" {
				new_tasks = append(new_tasks, task)
			}
		}
		job.Tasks = new_tasks
		changed = true
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
				err = errors.New("invalid inputs: task " + task.Id + " contains multiple inputs with filename=" + io.FileName)
				return
			}
			inputFileNames[io.FileName] = true
		}
	}

	return
}

// DEPRECATED in favor of job.Init()
//func (job *Job) InitTasks() (err error) {
//
//return
//}

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

func (job *Job) IncrementResumed(inc int, writelock bool) (err error) {
	if writelock {
		err = job.LockNamed("IncrementResumed")
		if err != nil {
			return
		}
		defer job.Unlock()
	}
	job.Resumed += 1
	err = dbUpdateJobFieldInt(job.Id, "resumed", job.Resumed)
	if err != nil {
		return
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

func (job *Job) Save() (err error) {
	if job.Id == "" {
		err = fmt.Errorf("job id empty")
		return
	}
	logger.Debug(1, "Save() saving job: %s", job.Id)

	job.UpdateTime = time.Now()

	err = job.SaveToDisk()
	if err != nil {
		return
	}

	logger.Debug(1, "Save() dbUpsert next: %s", job.Id)
	err = dbUpsert(job)
	if err != nil {
		err = fmt.Errorf("error in dbUpsert in job.Save(), (job_id=%s) error=%v", job.Id, err)
		return
	}
	logger.Debug(1, "Save() job saved: %s", job.Id)
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

func (job *Job) SetState(newState string, notes string) (err error) {

	err = job.LockNamed("SetState")
	if err != nil {
		return
	}
	defer job.Unlock()

	if job.State == newState {
		return
	}

	if newState == JOB_STAT_COMPLETED && job.State != JOB_STAT_COMPLETED {

		job.Info.CompletedTime = time.Now()

	}

	job.State = newState

	if newState == JOB_STAT_SUSPEND && len(notes) == 0 {
		notes = "unknown"
	}

	if len(notes) > 0 {
		job.Notes = notes
		dbUpdateJobFieldString(job.Id, "notes", notes)
	}

	dbUpdateJobFieldString(job.Id, "state", newState)

	return
}

func (job *Job) GetRemainTasks() (remain_tasks int, err error) {
	read_lock, err := job.RLockNamed("GetRemainTasks")
	if err != nil {
		return
	}
	defer job.RUnlockNamed(read_lock)

	remain_tasks = job.RemainTasks
	return
}

func (job *Job) SetRemainTasks(remain_tasks int) (err error) {

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

func (job *Job) IncrementRemainTasks(inc int, writelock bool) (err error) {
	if writelock {
		err = job.LockNamed("IncrementRemainTasks")
		if err != nil {
			return
		}
		defer job.Unlock()
	}
	job.RemainTasks += 1
	err = dbUpdateJobFieldInt(job.Id, "remaintasks", job.RemainTasks)
	if err != nil {
		return
	}

	return
}

//invoked to modify job info in mongodb when a task in that job changed to the new status
// task is already locked
func (job *Job) UpdateTaskDEPRECATED(task *Task) (remainTasks int, err error) {

	//if this task is complete, count remain tasks for the job
	task_state := task.State
	if task_state == TASK_STAT_COMPLETED ||
		task_state == TASK_STAT_SKIPPED ||
		task_state == TASK_STAT_FAIL_SKIP {
		remain_tasks := len(job.Tasks)
		for _, t := range job.Tasks { //double check all task other than the one with state change
			if t.State == TASK_STAT_COMPLETED ||
				t.State == TASK_STAT_SKIPPED ||
				t.State == TASK_STAT_FAIL_SKIP {
				remain_tasks -= 1
			}
		}

		err = job.SetRemainTasks(remain_tasks)
		if err != nil {
			return
		}

		if remain_tasks == 0 {
			job.SetState(JOB_STAT_COMPLETED, "")

		}
	}
	remainTasks = job.RemainTasks

	return
}

func (job *Job) SetClientgroups(clientgroups string) (err error) {
	job.Info.ClientGroups = clientgroups
	dbUpdateJobFieldString(job.Id, "info.clientgroups", clientgroups)

	//for _, task := range job.Tasks {
	//	if task.Info != nil {
	//		task.Info.ClientGroups = clientgroups
	//	}
	//}
	err = QMgr.UpdateQueueJobInfo(job)

	//err = job.Save()
	return
}

func (job *Job) SetPriority(priority int) (err error) {

	job.Info.Priority = priority
	dbUpdateJobFieldInt(job.Id, "info.priority", priority)

	err = QMgr.UpdateQueueJobInfo(job)

	return
}

func (job *Job) SetPipeline(pipeline string) (err error) {
	job.Info.Pipeline = pipeline
	dbUpdateJobFieldString(job.Id, "info.pipeline", pipeline)

	err = QMgr.UpdateQueueJobInfo(job)

	return
}

func (job *Job) SetDataToken(token string) (err error) {

	job.Info.DataToken = token
	job.Info.Auth = true
	dbUpdateJobFieldString(job.Id, "info.token", token)
	dbUpdateJobFieldBoolean(job.Id, "info.auth", true)

	err = QMgr.UpdateQueueJobInfo(job)

	return
}

func (job *Job) SetExpiration(expire string) (err error) {
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

	job.Expiration = currTime.Add(expireTime)
	//err = job.Save()

	update_value := bson.M{"expiration": job.Expiration}
	err = dbUpdateJobFields(job.Id, update_value)

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

func (job *Job) SetLastFailed(lastfailed string) (err error) {
	err = job.LockNamed("SetLastFailed")
	if err != nil {
		return
	}
	defer job.Unlock()

	err = dbUpdateJobFieldString(job.Id, "lastfailed", lastfailed)
	if err != nil {
		return
	}

	job.LastFailed = lastfailed

	return
}

func (job *Job) GetJobLogs() (jlog *JobLog, err error) {
	jlog = new(JobLog)
	jlog.Id = job.Id
	jlog.State = job.State
	jlog.UpdateTime = job.UpdateTime
	jlog.Notes = job.Notes
	jlog.LastFailed = job.LastFailed
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
