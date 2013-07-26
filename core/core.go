package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"net/http"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type ShockResponse struct {
	Code int       `bson:"status" json:"status"`
	Data ShockNode `bson:"data" json:"data"`
	Errs []string  `bson:"error" json:"error"`
}

type ShockNode struct {
	Id         string             `bson:"id" json:"id"`
	Version    string             `bson:"version" json:"version"`
	File       shockfile          `bson:"file" json:"file"`
	Attributes interface{}        `bson:"attributes" json:"attributes"`
	Indexes    map[string]IdxInfo `bson:"indexes" json:"indexes"`
	//Acl          Acl                `bson:"acl" json:"-"`
	VersionParts map[string]string `bson:"version_parts" json:"-"`
	Tags         []string          `bson:"tags" json:"tags"`
	//	Revisions    []ShockNode       `bson:"revisions" json:"-"`
	Linkages []linkage `bson:"linkage" json:"linkages"`
}

type shockfile struct {
	Name         string            `bson:"name" json:"name"`
	Size         int64             `bson:"size" json:"size"`
	Checksum     map[string]string `bson:"checksum" json:"checksum"`
	Format       string            `bson:"format" json:"format"`
	Path         string            `bson:"path" json:"-"`
	Virtual      bool              `bson:"virtual" json:"virtual"`
	VirtualParts []string          `bson:"virtual_parts" json:"virtual_parts"`
}

type IdxInfo struct {
	Type        string `bson:"index_type" json:"-"`
	TotalUnits  int64  `bson:"total_units" json:"total_units"`
	AvgUnitSize int64  `bson:"average_unit_size" json:"average_unit_size"`
}

type FormFiles map[string]FormFile

type FormFile struct {
	Name     string
	Path     string
	Checksum map[string]string
}

type linkage struct {
	Type      string   `bson: "relation" json:"relation"`
	Ids       []string `bson:"ids" json:"ids"`
	Operation string   `bson:"operation" json:"operation"`
}

func CreateJobUpload(params map[string]string, files FormFiles, jid string) (job *Job, err error) {

	if _, has_upload := files["upload"]; has_upload {
		job, err = ParseJobTasks(files["upload"].Path, jid)
	} else {
		job, err = ParseAwf(files["awf"].Path, jid)
	}

	if err != nil {
		return
	}
	err = job.Mkdir()
	if err != nil {
		return
	}
	err = job.UpdateFile(params, files)
	if err != nil {
		return
	}

	err = job.Save()
	return
}

func LoadJob(id string) (job *Job, err error) {
	if db, err := DBConnect(); err == nil {
		defer db.Close()
		job = new(Job)
		if err = db.FindById(id, job); err == nil {
			return job, nil
		} else {
			return nil, err
		}
	}
	return nil, err
}

func LoadJobs(ids []string) (jobs []*Job, err error) {
	if db, err := DBConnect(); err == nil {
		defer db.Close()
		if err = db.FindJobs(ids, &jobs); err == nil {
			return jobs, err
		} else {
			return nil, err
		}
	}
	return nil, err
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
		db, er := DBConnect()
		if er != nil {
			err = er
		}
		defer db.Close()
		err = db.Upsert(job)
		if err != nil {
			err = er
		}
	}
	return
}

//create a shock node for output
func PostNode(io *IO, numParts int) (nodeid string, err error) {
	var res *http.Response
	shockurl := fmt.Sprintf("%s/node", io.Host)

	c := make(chan int, 1)

	go func() {
		res, err = http.Post(shockurl, "", strings.NewReader(""))
		c <- 1 //we are ending
	}()

	select {
	case <-c:
		//go ahead
	case <-time.After(conf.SHOCK_TIMEOUT):
		fmt.Printf("timeout when creating node in shock, url=" + shockurl)
		return "", errors.New("timeout when creating node in shock, url=" + shockurl)
	}

	//fmt.Printf("shockurl=%s\n", shockurl)
	if err != nil {
		return "", err
	}

	jsonstream, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	response := new(ShockResponse)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return "", errors.New(fmt.Sprintf("failed to marshal post response:\"%s\"", jsonstream))
	}
	if len(response.Errs) > 0 {
		return "", errors.New(strings.Join(response.Errs, ","))
	}

	shocknode := &response.Data
	nodeid = shocknode.Id

	if numParts > 1 {
		putParts(io.Host, nodeid, numParts)
	}
	return
}

//create parts
func putParts(host string, nodeid string, numParts int) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	argv = append(argv, "-F")
	argv = append(argv, fmt.Sprintf("parts=%d", numParts))
	target_url := fmt.Sprintf("%s/node/%s", host, nodeid)
	argv = append(argv, target_url)

	cmd := exec.Command("curl", argv...)
	err = cmd.Run()
	if err != nil {
		return
	}
	return
}

//get jobid from task id or workunit id
func getParentJobId(id string) (jobid string) {
	parts := strings.Split(id, "_")
	return parts[0]
}

//parse job by job script
func ParseJobTasks(filename string, jid string) (job *Job, err error) {
	job = new(Job)

	jsonstream, err := ioutil.ReadFile(filename)

	if err != nil {
		return nil, errors.New("error in reading job json file")
	}

	json.Unmarshal(jsonstream, job)

	if len(job.Tasks) == 0 {
		return nil, errors.New("invalid job script: task list empty")
	}

	if job.Info == nil {
		job.Info = NewInfo()
	}

	job.Info.SubmitTime = time.Now()
	job.Info.Priority = conf.BasePriority

	job.setId()     //uuid for the job
	job.setJid(jid) //an incremental id for the jobs within a AWE server domain
	job.State = JOB_STAT_SUBMITTED

	for i := 0; i < len(job.Tasks); i++ {
		if err := job.Tasks[i].InitTask(job, i); err != nil {
			return nil, err
		}
	}

	job.RemainTasks = len(job.Tasks)

	return
}

//parse .awf.json - sudo-function only, to be finished
func ParseAwf(filename string, jid string) (job *Job, err error) {
	workflow := new(Workflow)
	jsonstream, err := ioutil.ReadFile(filename)
	if err != nil {
		return nil, errors.New("error in reading job json file")
	}
	json.Unmarshal(jsonstream, workflow)
	job, err = AwfToJob(workflow, jid)
	if err != nil {
		return
	}
	return
}

func AwfToJob(awf *Workflow, jid string) (job *Job, err error) {
	job = new(Job)
	job.initJob(jid)

	//mapping info
	job.Info.Pipeline = awf.WfInfo.Name
	job.Info.Name = awf.JobInfo.Name
	job.Info.Project = awf.JobInfo.Project
	job.Info.User = awf.JobInfo.User
	job.Info.ClientGroups = awf.JobInfo.Queue

	//create task 0: pseudo-task representing the success of job submission
	//to-do: in the future this task can serve as raw input data validation
	task := NewTask(job, 0)
	task.Cmd.Description = "job submission"
	task.State = TASK_STAT_PASSED
	task.RemainWork = 0
	task.TotalWork = 0
	job.Tasks = append(job.Tasks, task)

	//mapping tasks
	for _, awf_task := range awf.Tasks {
		task := NewTask(job, awf_task.TaskId)
		for name, origin := range awf_task.Inputs {
			io := new(IO)
			io.Name = name
			io.Host = awf.DataServer
			io.Node = "-"
			io.Origin = strconv.Itoa(origin)
			task.Inputs[name] = io
			if origin == 0 {
				if dataurl, ok := awf.RawInputs[io.Name]; ok {
					io.Url = dataurl
				}
			}
		}

		for _, name := range awf_task.Outputs {
			io := new(IO)
			io.Name = name
			io.Host = awf.DataServer
			io.Node = "-"
			task.Outputs[name] = io
		}
		if awf_task.Splits == 0 {
			task.TotalWork = 1
		} else {
			task.TotalWork = awf_task.Splits
		}

		task.Cmd.Name = awf_task.Cmd.Name
		arg_str := awf_task.Cmd.Args
		if strings.Contains(arg_str, "$") { //contains variables, parse them
			for name, value := range awf.Variables {
				var_name := "$" + name
				arg_str = strings.Replace(arg_str, var_name, value, -1)
			}
		}
		task.Cmd.Args = arg_str

		for _, parent := range awf_task.DependsOn {
			parent_id := getParentTask(task.Id, parent)
			task.DependsOn = append(task.DependsOn, parent_id)
		}
		task.InitTask(job, awf_task.TaskId)
		job.Tasks = append(job.Tasks, task)
	}
	job.RemainTasks = len(job.Tasks) - 1
	return
}

//misc
func GetJobIdByTaskId(taskid string) (jobid string, err error) {
	parts := strings.Split(taskid, "_")
	if len(parts) == 2 {
		return parts[0], nil
	} else {
		return "", errors.New("invalid task id: " + taskid)
	}
	return
}

func IsFirstTask(taskid string) bool {
	parts := strings.Split(taskid, "_")
	if len(parts) == 2 {
		if parts[1] == "0" || parts[1] == "1" {
			return true
		}
	}
	return false
}

func UpdateJobState(jobid string, newstate string) (err error) {
	job, err := LoadJob(jobid)
	if err != nil {
		return
	}
	if err := job.UpdateState(newstate, ""); err != nil {
		return err
	}
	return
}

func getParentTask(taskid string, origin int) string {
	parts := strings.Split(taskid, "_")
	if len(parts) == 2 {
		return fmt.Sprintf("%s_%d", parts[0], origin)
	}
	return taskid
}
