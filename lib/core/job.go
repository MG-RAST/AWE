package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/golib/mgo/bson"
	"io/ioutil"
	"os"
	"path/filepath"
	"time"
)

const (
	JOB_STAT_INIT       = "init"
	JOB_STAT_QUEUED     = "queued"
	JOB_STAT_INPROGRESS = "in-progress"
	JOB_STAT_COMPLETED  = "completed"
	JOB_STAT_SUSPEND    = "suspend"
	JOB_STAT_DELETED    = "deleted"
)

var JOB_STATS_ACTIVE = []string{JOB_STAT_QUEUED, JOB_STAT_INPROGRESS}
var JOB_STATS_REGISTERED = []string{JOB_STAT_QUEUED, JOB_STAT_INPROGRESS, JOB_STAT_SUSPEND}
var JOB_STATS_TO_RECOVER = []string{JOB_STAT_QUEUED, JOB_STAT_INPROGRESS, JOB_STAT_SUSPEND}

type Job struct {
	Id          string    `bson:"id" json:"id"`
	Jid         string    `bson:"jid" json:"jid"`
	Acl         acl.Acl   `bson:"acl" json:"-"`
	Info        *Info     `bson:"info" json:"info"`
	Tasks       []*Task   `bson:"tasks" json:"tasks"`
	Script      script    `bson:"script" json:"-"`
	State       string    `bson:"state" json:"state"`
	Registered  bool      `bson:"registered" json:"registered"`
	RemainTasks int       `bson:"remaintasks" json:"remaintasks"`
	UpdateTime  time.Time `bson:"updatetime" json:"updatetime"`
	Notes       string    `bson:"notes" json:"notes"`
	LastFailed  string    `bson:"lastfailed" json:"lastfailed"`
	Resumed     int       `bson:"resumed" json:"resumed"`     //number of times the job has been resumed from suspension
	ShockHost   string    `bson:"shockhost" json:"shockhost"` // this is a fall-back default if not specified at a lower level
}

// Deprecated JobDep struct uses deprecated TaskDep struct which uses the deprecated IOmap.  Maintained for backwards compatibility.
// Jobs that cannot be parsed into the Job struct, but can be parsed into the JobDep struct will be translated to the new Job struct.
// (=deprecated=)
type JobDep struct {
	Id          string     `bson:"id" json:"id"`
	Jid         string     `bson:"jid" json:"jid"`
	Acl         acl.Acl    `bson:"acl" json:"-"`
	Info        *Info      `bson:"info" json:"info"`
	Tasks       []*TaskDep `bson:"tasks" json:"tasks"`
	Script      script     `bson:"script" json:"-"`
	State       string     `bson:"state" json:"state"`
	Registered  bool       `bson:"registered" json:"registered"`
	RemainTasks int        `bson:"remaintasks" json:"remaintasks"`
	UpdateTime  time.Time  `bson:"updatetime" json:"updatetime"`
	Notes       string     `bson:"notes" json:"notes"`
	LastFailed  string     `bson:"lastfailed" json:"lastfailed"`
	Resumed     int        `bson:"resumed" json:"resumed"`     //number of times the job has been resumed from suspension
	ShockHost   string     `bson:"shockhost" json:"shockhost"` // this is a fall-back default if not specified at a lower level
}

type JobMin struct {
	Id            string            `bson:"id" json:"id"`
	Name          string            `bson:"name" json:"name"`
	Size          int64             `bson:"size" json:"size"`
	SubmitTime    time.Time         `bson:"submittime" json:"submittime"`
	CompletedTime time.Time         `bson:"completedtime" json:"completedtime"`
	Task          []int             `bson:"task" json:"task"`
	State         []string          `bson:"state" json:"state"`
	UserAttr      map[string]string `bson:"userattr" json:"userattr"`
}

//set job's uuid
func (job *Job) setId() {
	job.Id = uuid.New()
	return
}

//set job's jid
func (job *Job) setJid(jid string) {
	job.Jid = jid
}

func (job *Job) initJob(jid string) {
	if job.Info == nil {
		job.Info = new(Info)
	}
	job.Info.SubmitTime = time.Now()
	job.Info.Priority = conf.BasePriority
	job.setId()     //uuid for the job
	job.setJid(jid) //an incremental id for the jobs within a AWE server domain
	job.State = JOB_STAT_INIT
	job.Registered = true
}

type script struct {
	Name string `bson:"name" json:"name"`
	Type string `bson:"type" json:"type"`
	Path string `bson:"path" json:"-"`
}

//---Script upload
func (job *Job) UpdateFile(params map[string]string, files FormFiles) (err error) {
	_, isRegularUpload := files["upload"]

	if isRegularUpload {
		if err = job.SetFile(files["upload"]); err != nil {
			return err
		}
		delete(files, "upload")
	}

	return
}

func (job *Job) Save() (err error) {
	job.UpdateTime = time.Now()
	bsonPath := fmt.Sprintf("%s/%s.bson", job.Path(), job.Id)
	os.Remove(bsonPath)
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
	err = dbUpsert(job)
	if err != nil {
		err = errors.New("error in dbUpdate in job.Save(), error=" + err.Error())
		return
	}
	return
}

func (job *Job) Mkdir() (err error) {
	err = os.MkdirAll(job.Path(), 0777)
	if err != nil {
		return
	}
	return
}

func (job *Job) SetFile(file FormFile) (err error) {
	if err != nil { //?
		return
	}
	os.Rename(file.Path, job.FilePath())
	job.Script.Name = file.Name
	return
}

//---Path functions
func (job *Job) Path() string {
	return getPathByJobId(job.Id)
}

func (job *Job) FilePath() string {
	if job.Script.Path != "" {
		return job.Script.Path
	}
	return getPathByJobId(job.Id) + "/" + job.Id + ".script"
}

func getPathByJobId(id string) string {
	return fmt.Sprintf("%s/%s/%s/%s/%s", conf.DATA_PATH, id[0:2], id[2:4], id[4:6], id)
}

//---Task functions
func (job *Job) TaskList() []*Task {
	return job.Tasks
}

func (job *Job) NumTask() int {
	return len(job.Tasks)
}

//---Field update functions
func (job *Job) UpdateState(newState string, notes string) (err error) {
	job.State = newState
	if len(notes) > 0 {
		job.Notes = notes
	}
	return job.Save()
}

//invoked to modify job info in mongodb when a task in that job changed to the new status
func (job *Job) UpdateTask(task *Task) (remainTasks int, err error) {
	idx := -1
	for i, t := range job.Tasks {
		if t.Id == task.Id {
			idx = i
			break
		}
	}
	if idx == -1 {
		return job.RemainTasks, errors.New("job.UpdateTask: no task found with id=" + task.Id)
	}
	job.Tasks[idx] = task

	//if this task is complete, count remain tasks for the job
	if task.State == TASK_STAT_COMPLETED ||
		task.State == TASK_STAT_SKIPPED ||
		task.State == TASK_STAT_FAIL_SKIP {
		remain_tasks := len(job.Tasks)
		for _, t := range job.Tasks { //double check all task other than the one with state change
			if t.State == TASK_STAT_COMPLETED ||
				t.State == TASK_STAT_SKIPPED ||
				t.State == TASK_STAT_FAIL_SKIP {
				remain_tasks -= 1
			}
		}
		job.RemainTasks = remain_tasks
		if job.RemainTasks == 0 {
			job.State = JOB_STAT_COMPLETED
			job.Info.CompletedTime = time.Now()
		}
	}
	return job.RemainTasks, job.Save()
}

//set token
func (job *Job) SetDataToken(token string) {
	job.Info.DataToken = token
	job.Info.Auth = true
	for _, task := range job.Tasks {
		task.Info.DataToken = token
		task.setTokenForIO()
	}
	job.Save()
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

func ReloadFromDisk(path string) (err error) {
	id := filepath.Base(path)
	jobbson, err := ioutil.ReadFile(path + "/" + id + ".bson")
	if err != nil {
		return
	}
	job := new(Job)
	err = bson.Unmarshal(jobbson, &job)
	if err == nil {
		if err = dbUpsert(job); err != nil {
			return err
		}
	}
	return
}
