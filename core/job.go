package core

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"github.com/MG-RAST/Shock/store/uuid"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"os"
)

type Job struct {
	Id     string    `bson:"id" json:"id"`
	Info   *Info     `bson:"info" json:"info"`
	Tasks  *TaskList `bson:"tasks" json:"tasks"`
	Script script    `bson:"script" json:"script"`
}

func NewJob() (job *Job) {
	job = new(Job)
	job.Info = NewInfo()
	job.Tasks = NewTaskList()
	job.setId()
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

//Modification functions
func (job *Job) Update(params map[string]string, files FormFiles) (err error) {
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

// Path functions
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
