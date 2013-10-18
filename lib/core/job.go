package core

import (
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/uuid"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"os"
	"path/filepath"
	"time"
)

const (
	JOB_STAT_SUBMITTED  = "submitted"
	JOB_STAT_INPROGRESS = "in-progress"
	JOB_STAT_COMPLETED  = "completed"
	JOB_STAT_SUSPEND    = "suspend"
	JOB_STAT_DELETED    = "deleted"
)

type Job struct {
	Id          string    `bson:"id" json:"id"`
	Jid         string    `bson:"jid" json:"jid"`
	Info        *Info     `bson:"info" json:"info"`
	Tasks       []*Task   `bson:"tasks" json:"tasks"`
	Script      script    `bson:"script" json:"-"`
	State       string    `bson:"state" json:"state"`
	RemainTasks int       `bson:"remaintasks" json:"remaintasks"`
	UpdateTime  time.Time `bson:"updatetime" json:"updatetime"`
	Notes       string    `bson:"notes" json:"notes"`
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
	job.State = JOB_STAT_SUBMITTED
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
		return
	}
	err = ioutil.WriteFile(bsonPath, nbson, 0644)
	if err != nil {
		return
	}
	err = dbUpsert(job)
	if err != nil {
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
	if err != nil {
		return
	}
	os.Rename(file.Path, job.FilePath())
	job.Script.Name = file.Name
	return
}

//---Path functions
func (job *Job) Path() string {
	return getPath(job.Id)
}

func (job *Job) FilePath() string {
	if job.Script.Path != "" {
		return job.Script.Path
	}
	return getPath(job.Id) + "/" + job.Id + ".script"
}

func getPath(id string) string {
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

	//if task state changed to "completed", update remaining task in job
	if task.State != job.Tasks[idx].State {
		if task.State == TASK_STAT_COMPLETED ||
			task.State == TASK_STAT_SKIPPED ||
			task.State == TASK_STAT_FAIL_SKIP {
			job.RemainTasks -= 1
			if job.RemainTasks == 0 {
				job.State = JOB_STAT_COMPLETED
			}
		}
	}
	job.Tasks[idx] = task
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
