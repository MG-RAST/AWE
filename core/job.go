package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"github.com/MG-RAST/Shock/store/uuid"
	"github.com/kless/goconfig/config"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"os"
	"strings"
	"time"
)

type Job struct {
	Id     string `bson:"id" json:"id"`
	Info   *Info  `bson:"info" json:"info"`
	Tasks  []Task `bson:"tasks" json:"tasks"`
	Script script `bson:"script" json:"script"`
	State  string `bson:"state" json:"state"`
}

func NewJob() (job *Job) {
	job = new(Job)
	job.Info = NewInfo()
	job.Tasks = []Task{}
	job.setId()
	job.State = "submitted"
	return
}

func (job *Job) setId() {
	job.Id = uuid.New()
	return
}

type script struct {
	Name string `bson:"name" json:"name"`
	Type string `bson:"type" json:"type"`
	Path string `bson:"path" json:"-"`
}

type FormFiles map[string]FormFile

type FormFile struct {
	Name     string
	Path     string
	Checksum map[string]string
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
	db, err := DBConnect()
	if err != nil {
		return
	}
	defer db.Close()

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
	err = db.Upsert(job)
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
func (job *Job) TaskList() []Task {
	return job.Tasks
}

func (job *Job) NumTask() int {
	return len(job.Tasks)
}

func (job *Job) parseTasksFromScript() (tasks []Task, err error) {
	return
}

func (job *Job) TestSetTasks() (err error) {
	var lastId string
	for i := 0; i < 5; i++ {
		task := NewTask(job, i)
		if lastId != "" {
			task.DependsOn = []string{lastId}
		}
		lastId = task.Id
		job.Tasks = append(job.Tasks, *task)
	}
	return
}

func (job *Job) ParseTasks() (err error) {
	c, err := config.ReadDefault(job.FilePath())
	if err != nil {
		return errors.New("could not parse job script file")
	}

	jobname, _ := c.String("job", "name")
	job.Info.Name = jobname

	owner, _ := c.String("job", "owner")
	job.Info.Owner = owner

	totaltask, _ := c.Int("job", "totaltask")
	for i := 0; i < totaltask; i++ {
		task := NewTask(job, i)

		section := fmt.Sprintf("task-%d", i)

		cmdName, _ := c.String(section, "cmd_name")
		cmdDescription, _ := c.String(section, "cmd_description")
		cmdArgs, _ := c.String(section, "cmd_args")

		cmd := NewCommand(cmdName)
		cmd.Description = cmdDescription
		cmd.Args = cmdArgs

		task.Cmd = cmd

		inputName, _ := c.String(section, "input_name")
		inputUrl, _ := c.String(section, "input_url")

		io := &IO{Name: inputName,
			Url:   inputUrl,
			MD5:   "",
			Cache: false,
		}

		task.Inputs[inputName] = io

		dependon, _ := c.String(section, "dependOn")
		if dependon != "" {
			for _, depend := range strings.Split(dependon, ",") {
				depend_id := fmt.Sprintf("%s_%s", job.Id, depend)
				task.DependsOn = append(task.DependsOn, depend_id)
			}
		}

		job.Tasks = append(job.Tasks, *task)
	}
	return
}

func ParseJobTasksByJson(filename string) (job *Job, err error) {
	job = new(Job)

	jsonstream, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, errors.New("error in reading job json file")
	}

	json.Unmarshal(jsonstream, job)

	if job.Info == nil {
		job.Info = NewInfo()
	}

	job.Info.SubmitTime = time.Now()
	job.Info.Priority = conf.BasePriority

	job.setId()
	job.State = "submitted"

	for i := 0; i < len(job.Tasks); i++ {
		taskid := fmt.Sprintf("%s_%d", job.Id, i)
		job.Tasks[i].Id = taskid
		job.Tasks[i].Info = job.Info
		job.Tasks[i].State = "init"
		job.Tasks[i].RemainWork = job.Tasks[i].TotalWork

		for j := 0; j < len(job.Tasks[i].DependsOn); j++ {
			depend := job.Tasks[i].DependsOn[j]
			job.Tasks[i].DependsOn[j] = fmt.Sprintf("%s_%s", job.Id, depend)
		}
	}
	return
}

//---Field update functions
func (job *Job) UpdateState(newState string) string {
	job.State = newState
	return newState
}
