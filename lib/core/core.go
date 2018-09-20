package core

import (
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"

	"io/ioutil"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"

	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"
	shock "github.com/MG-RAST/go-shock-client"
	"github.com/MG-RAST/golib/httpclient"
)

var (
	QMgr          ResourceMgr
	Service       string = "unknown"
	Self          *Client
	ProxyWorkChan chan bool
	Server_UUID   string
	JM            *JobMap
	Start_time    time.Time
)

type BaseResponse struct {
	Status int      `json:"status"`
	Error  []string `json:"error"`
}

type StandardResponse struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
	Error  []string    `json:"error"`
}

func InitResMgr(service string) {
	if service == "server" {
		QMgr = NewServerMgr()
	} else if service == "proxy" {
		//QMgr = NewProxyMgr()
	}
	Service = service
	Start_time = time.Now()
}

func SetClientProfile(profile *Client) {
	Self = profile
}

func InitProxyWorkChan() {
	ProxyWorkChan = make(chan bool, 100)
}

type CoAck struct {
	workunits []*Workunit
	err       error
}

type CoReq struct {
	policy     string
	fromclient string
	//fromclient *Client
	available int64
	count     int
	response  chan CoAck
}

type coInfo struct {
	workunit *Workunit
	clientid string
}

type FormFiles map[string]FormFile

type FormFile struct {
	Name     string
	Path     string
	Checksum map[string]string
}

//heartbeat response from awe-server to awe-worker
//used for issue operation request to client, e.g. discard suspended workunits
type HeartbeatInstructions map[string]string //map[op]obj1,obj2 e.g. map[discard]=work1,work2

func CreateJobUpload(u *user.User, files FormFiles) (job *Job, err error) {

	upload_file, has_upload := files["upload"]

	if has_upload {
		upload_file_path := upload_file.Path
		job, err = ReadJobFile(upload_file_path)
		if err != nil {
			err = fmt.Errorf("(CreateJobUpload) Parsing: Failed (default and deprecated format) %s", err.Error())
			logger.Debug(3, err.Error())
			return
		} else {
			logger.Debug(3, "Parsing: Success (default or deprecated format)")
		}

	} else {
		err = errors.New("(CreateJobUpload) has_upload is missing")
		return
		// job, err = ParseAwf(files["awf"].Path)
		// if err != nil {
		// 	err = errors.New("(ParseAwf) error parsing job, error=" + err.Error())
		// 	return
		// }
	}

	// Once, job has been created, set job owner and add owner to all ACL's
	if job == nil {
		err = fmt.Errorf("job==nil")
		return
	}

	logger.Debug(3, "OWNER1: %s", u.Uuid)

	job.Acl.SetOwner(u.Uuid)
	logger.Debug(3, "OWNER2: %s", job.Acl.Owner)
	job.Acl.Set(u.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	logger.Debug(3, "OWNER3: %s", job.Acl.Owner)

	err = job.Mkdir()
	if err != nil {
		err = errors.New("(CreateJobUpload) error creating job directory, error=" + err.Error())
		return
	}

	_, err = job.Init()
	if err != nil {
		err = fmt.Errorf("(CreateJobUpload) job.Init returned: %s", err.Error())
		return
	}

	err = job.UpdateFile(files, "upload")
	if err != nil {
		err = errors.New("error in UpdateFile, error=" + err.Error())
		return
	}

	err = job.Save()
	if err != nil {
		err = errors.New("error in job.Save(), error=" + err.Error())
		return
	}

	logger.Debug(3, "OWNER4: %s", job.Acl.Owner)

	return
}

func CreateJobImport(u *user.User, file FormFile) (job *Job, err error) {
	job = NewJob()

	jsonstream, err := ioutil.ReadFile(file.Path)
	if err != nil {
		return nil, errors.New("error in reading job json file" + err.Error())
	}

	err = json.Unmarshal(jsonstream, job)
	if err != nil {
		return nil, errors.New("(CreateJobImport) error in unmarshaling job json file: " + err.Error())
	}

	if len(job.Tasks) == 0 {
		return nil, errors.New("invalid job document: task list empty")
	}
	if job.State != JOB_STAT_COMPLETED {
		return nil, errors.New("invalid job import: must be completed")
	}
	if job.Info == nil {
		return nil, errors.New("invalid job import: missing job info")
	}
	if job.Id == "" {
		return nil, errors.New("invalid job import: missing job id")
	}

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
				return nil, errors.New("invalid inputs: task " + task_str + " contains multiple inputs with filename=" + io.FileName)
			}
			inputFileNames[io.FileName] = true
		}
	}

	// Once, job has been created, set job owner and add owner to all ACL's
	job.Acl.SetOwner(u.Uuid)
	job.Acl.Set(u.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	err = job.Mkdir()
	if err != nil {
		err = errors.New("(CreateJobImport) error creating job directory, error=" + err.Error())
		return
	}

	err = job.Save()
	if err != nil {
		err = errors.New("error in job.Save(), error=" + err.Error())
		return
	}
	return
}

func ReadJobFile(filename string) (job *Job, err error) {
	job = NewJob()

	var jsonstream []byte
	jsonstream, err = ioutil.ReadFile(filename)
	if err != nil {
		err = fmt.Errorf("error in reading job json file: %s", err.Error())
		return
	}

	err = json.Unmarshal(jsonstream, job)
	if err != nil {
		//err = fmt.Errorf("(ReadJobFile) error in unmarshaling job json file: %s ", err.Error())
		logger.Error("(ReadJobFile) error in unmarshaling job json file using normal job struct: %s ", err.Error())
		err = nil

		jobDep := NewJobDep()

		err = json.Unmarshal(jsonstream, jobDep)
		if err != nil {
			err = fmt.Errorf("(ReadJobFile) error in unmarshaling job json file using deprecated job struct: %s ", err.Error())
			return
		} else {
			logger.Debug(3, "(ReadJobFile) Success unmarshaling job json file using deprecated job struct.")
		}

		job, err = JobDepToJob(jobDep)
		if err != nil {
			err = fmt.Errorf("JobDepToJob failed: %s", err.Error())
			return
		}

	} else {
		// jobDep had been initialized already
		_, err = job.Init()
		if err != nil {
			err = fmt.Errorf("(ReadJobFile) job.Init returned error: %s", err.Error())
			return
		}
	}

	//parse private fields task.Cmd.Environ.Private
	job_p := new(Job_p)
	err = json.Unmarshal(jsonstream, job_p)
	if err != nil {
		err = fmt.Errorf("(ReadJobFile) json.Unmarshal (private fields) returned error: %s", err.Error())
		return
	}

	for idx, task_p := range job_p.Tasks {

		task := job.Tasks[idx]
		if task_p.Cmd.Environ == nil || task_p.Cmd.Environ.Private == nil {
			continue
		}
		task.Cmd.Environ.Private = make(map[string]string)
		for key, val := range task_p.Cmd.Environ.Private {
			task.Cmd.Environ.Private[key] = val
		}
	}

	return
}

// Takes the deprecated (version 1) Job struct and returns the version 2 Job struct or an error
func JobDepToJob(jobDep *JobDep) (job *Job, err error) {
	job = NewJob()

	if jobDep.Id != "" {
		job.Id = jobDep.Id
	}

	if job.Id == "" {
		job.setId()
	}

	if len(jobDep.Tasks) == 0 {
		err = fmt.Errorf("(JobDepToJob) jobDep.Tasks empty")
		return
	}

	job.Acl = jobDep.Acl
	job.Info = jobDep.Info
	job.Script = jobDep.Script
	job.State = jobDep.State
	job.Registered = jobDep.Registered
	job.RemainTasks = jobDep.RemainTasks
	job.UpdateTime = jobDep.UpdateTime
	job.Error = jobDep.Error
	job.Resumed = jobDep.Resumed
	job.ShockHost = jobDep.ShockHost

	for _, taskDep := range jobDep.Tasks {
		//task := new(Task)
		//if taskDep.Id == "" {
		//	err = fmt.Errorf("(JobDepToJob) taskDep.Id empty")
		//	return
		//}

		var task *Task
		task, err = NewTask(job, "", taskDep.Id)
		if err != nil {
			err = fmt.Errorf("(JobDepToJob) NewTask returned: %s", err.Error())
			return
		}

		_, err = task.Init(job)
		if err != nil {
			return
		}

		task.Cmd = taskDep.Cmd
		//task.App = taskDep.App
		//task.AppVariablesArray = taskDep.AppVariablesArray
		task.Partition = taskDep.Partition
		task.DependsOn = taskDep.DependsOn
		task.TotalWork = taskDep.TotalWork
		task.MaxWorkSize = taskDep.MaxWorkSize
		task.RemainWork = taskDep.RemainWork
		//task.WorkStatus = taskDep.WorkStatus
		//task.State = taskDep.State
		//task.Skip = taskDep.Skip
		task.CreatedDate = taskDep.CreatedDate
		task.StartedDate = taskDep.StartedDate
		task.CompletedDate = taskDep.CompletedDate
		task.ComputeTime = taskDep.ComputeTime
		task.UserAttr = taskDep.UserAttr
		task.ClientGroups = taskDep.ClientGroups
		for key, val := range taskDep.Inputs {
			io := new(IO)
			io = val
			io.FileName = key
			task.Inputs = append(task.Inputs, io)
		}
		for key, val := range taskDep.Outputs {
			io := new(IO)
			io = val
			io.FileName = key
			task.Outputs = append(task.Outputs, io)
		}
		for key, val := range taskDep.Predata {
			io := new(IO)
			io = val
			io.FileName = key
			task.Predata = append(task.Predata, io)
		}
		job.Tasks = append(job.Tasks, task)
	}

	_, err = job.Init()
	if err != nil {
		err = fmt.Errorf("(JobDepToJob) job.Init() returned: %s", err.Error())
		return
	}

	if len(job.Tasks) == 0 {
		err = fmt.Errorf("(JobDepToJob) job.Tasks empty")
		return
	}

	return
}

//misc

func GetJobIdByTaskId_deprecated(taskid string) (jobid string, err error) { // job_id is embedded in task struct
	parts := strings.Split(taskid, "_")
	if len(parts) == 2 {
		return parts[0], nil
	}
	return "", errors.New("invalid task id: " + taskid)
}

func GetJobIdByWorkId_deprecated(workid string) (jobid string, err error) {
	parts := strings.Split(workid, "_")
	if len(parts) == 3 {
		jobid = parts[0]
		return
	}
	err = errors.New("invalid work id: " + workid)
	return
}

func GetTaskIdByWorkId_deprecated(workid string) (taskid string, err error) {
	parts := strings.Split(workid, "_")
	if len(parts) == 3 {
		return fmt.Sprintf("%s_%s", parts[0], parts[1]), nil
	}
	return "", errors.New("invalid task id: " + workid)
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

//update job state to "newstate" only if the current state is in one of the "oldstates" // TODO make this a job.SetState function
func UpdateJobState_deprecated(jobid string, newstate string, oldstates []string) (err error) {
	job, err := GetJob(jobid)
	if err != nil {
		return
	}

	job_state, err := job.GetState(true)
	if err != nil {
		return
	}

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
	//if err := job.SetState(newstate); err != nil {
	//	return err
	//}
	return
}

func contains(list []string, elem string) bool {
	for _, t := range list {
		if t == elem {
			return true
		}
	}
	return false
}

//functions for REST API communication  (=deprecated=)
//notify AWE server a workunit is finished with status either "failed" or "done", and with perf statistics if "done"
func NotifyWorkunitProcessed(work *Workunit, perf *WorkPerf) (err error) {
	target_url := fmt.Sprintf("%s/work/%s?workid=%s&jobid=%s&status=%s&client=%s", conf.SERVER_URL, work.Id, work.TaskName, work.JobId, work.State, Self.Id)

	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	if work.State == WORK_STAT_DONE && perf != nil {
		reportFile, err := getPerfFilePath(work, perf)
		if err == nil {
			argv = append(argv, "-F")
			argv = append(argv, fmt.Sprintf("perf=@%s", reportFile))
			target_url = target_url + "&report"
		}
	}
	argv = append(argv, target_url)

	cmd := exec.Command("curl", argv...)
	err = cmd.Run()
	if err != nil {
		return
	}
	return
}

func NotifyWorkunitProcessedWithLogs(work *Workunit, perf *WorkPerf, sendstdlogs bool) (response *StandardResponse, err error) {

	var work_str string
	work_str, err = work.String()
	if err != nil {
		err = fmt.Errorf("(NotifyWorkunitProcessedWithLogs) workid.String() returned: %s", err.Error())
		return
	}

	work_id_b64 := "base64:" + base64.StdEncoding.EncodeToString([]byte(work_str))
	target_url := ""
	if work.CWL_workunit != nil {
		target_url = fmt.Sprintf("%s/work/%s?client=%s", conf.SERVER_URL, work_id_b64, Self.Id) // client info is needed for authentication
	} else {
		// old AWE style result reporting (note that nodes had been created by the AWE server)
		target_url = fmt.Sprintf("%s/work/%s?status=%s&client=%s&computetime=%d", conf.SERVER_URL, work_id_b64, work.State, Self.Id, work.ComputeTime)
	}
	form := httpclient.NewForm()
	hasreport := false
	if work.State == WORK_STAT_DONE && perf != nil {
		perflog, err := getPerfFilePath(work, perf)
		if err == nil {
			form.AddFile("perf", perflog)
			hasreport = true
		}
	}
	if sendstdlogs { //send stdout and stderr files if specified and existed
		stdoutFile, err := getStdOutPath(work)
		if err == nil {
			form.AddFile("stdout", stdoutFile)
			hasreport = true
		}
		stderrFile, err := getStdErrPath(work)
		if err == nil {
			form.AddFile("stderr", stderrFile)
			hasreport = true
		}
		worknotesFile, err := getWorkNotesPath(work)
		if err == nil {
			form.AddFile("worknotes", worknotesFile)
			hasreport = true
		}
	}

	if work.CWL_workunit != nil {
		cwl_result := work.CWL_workunit.Notice
		cwl_result.Results = work.CWL_workunit.Outputs
		cwl_result.Status = work.State
		cwl_result.ComputeTime = work.ComputeTime

		var result_bytes []byte
		result_bytes, err = json.Marshal(cwl_result)
		if err != nil {
			err = fmt.Errorf("(NotifyWorkunitProcessedWithLogs) Could not json marshal results: %s", err.Error())
			return
		}

		//fmt.Printf("Notice: %s\n", string(result_bytes[:]))

		form.AddParam("cwl", string(result_bytes[:]))

	}

	if hasreport {
		target_url = target_url + "&report"
	}
	err = form.Create()
	if err != nil {
		return
	}
	var headers httpclient.Header
	if conf.CLIENT_GROUP_TOKEN == "" {
		headers = httpclient.Header{
			"Content-Type":   []string{form.ContentType},
			"Content-Length": []string{strconv.FormatInt(form.Length, 10)},
		}
	} else {
		headers = httpclient.Header{
			"Content-Type":   []string{form.ContentType},
			"Content-Length": []string{strconv.FormatInt(form.Length, 10)},
			"Authorization":  []string{"CG_TOKEN " + conf.CLIENT_GROUP_TOKEN},
		}
	}
	logger.Debug(3, "PUT %s", target_url)
	res, err := httpclient.Put(target_url, headers, form.Reader, nil)
	if err != nil {
		return
	}
	defer res.Body.Close()

	jsonstream, _ := ioutil.ReadAll(res.Body)
	response = new(StandardResponse)
	err = json.Unmarshal(jsonstream, response)
	if err != nil {
		err = fmt.Errorf("(NotifyWorkunitProcessedWithLogs) failed to marshal response:\"%s\"", jsonstream)
		return
	}
	if len(response.Error) > 0 {
		err = errors.New(strings.Join(response.Error, ","))
		return
	}

	return
}

// deprecated, see cache.UploadOutputData
func PushOutputData(work *Workunit) (size int64, err error) {
	for _, io := range work.Outputs {
		name := io.FileName
		var local_filepath string //local file name generated by the cmd
		var file_path string      //file name to be uploaded to shock

		work_path, xerr := work.Path()
		if xerr != nil {
			err = xerr
			return
		}
		if io.Directory != "" {
			local_filepath = fmt.Sprintf("%s/%s/%s", work_path, io.Directory, name)
			//if specified, rename the local file name to the specified shock node file name
			//otherwise use the local name as shock file name
			file_path = local_filepath
			if io.ShockFilename != "" {
				file_path = fmt.Sprintf("%s/%s/%s", work_path, io.Directory, io.ShockFilename)
				os.Rename(local_filepath, file_path)
			}
		} else {
			local_filepath = fmt.Sprintf("%s/%s", work_path, name)
			file_path = local_filepath
			if io.ShockFilename != "" {
				file_path = fmt.Sprintf("%s/%s", work_path, io.ShockFilename)
				os.Rename(local_filepath, file_path)
			}
		}
		//use full path here, cwd could be changed by Worker (likely in worker-overlapping mode)
		if fi, err := os.Stat(file_path); err != nil {
			//ignore missing file if type=copy or type==update or nofile=true
			//skip this output if missing file and optional
			if (io.Type == "copy") || (io.Type == "update") || io.NoFile {
				file_path = ""
			} else if io.Optional {
				continue
			} else {
				return size, errors.New(fmt.Sprintf("output %s not generated for workunit %s", name, work.Id))
			}
		} else {
			if io.Nonzero && fi.Size() == 0 {
				return size, errors.New(fmt.Sprintf("workunit %s generated zero-sized output %s while non-zero-sized file required", work.Id, name))
			}
			size += fi.Size()
		}
		logger.Debug(2, "deliverer: push output to shock, filename="+name)
		logger.Event(event.FILE_OUT,
			"workid="+work.Id,
			"filename="+name,
			fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))

		//upload attribute file to shock IF attribute file is specified in outputs AND it is found in local directory.
		var attrfile_path string = ""
		if io.AttrFile != "" {
			attrfile_path = fmt.Sprintf("%s/%s", work_path, io.AttrFile)
			if fi, err := os.Stat(attrfile_path); err != nil || fi.Size() == 0 {
				attrfile_path = ""
			}
		}

		//set io.FormOptions["parent_node"] if not present and io.FormOptions["parent_name"] exists
		if parent_name, ok := io.FormOptions["parent_name"]; ok {
			for _, in_io := range work.Inputs {
				if in_io.FileName == parent_name {
					io.FormOptions["parent_node"] = in_io.Node
				}
			}
		}
		sc := shock.ShockClient{Host: io.Host, Token: work.Info.DataToken}
		if _, err := sc.PutOrPostFile(file_path, io.Node, work.Rank, attrfile_path, io.Type, io.FormOptions, io.NodeAttr); err != nil {
			time.Sleep(3 * time.Second) //wait for 3 seconds and try again
			if _, err := sc.PutOrPostFile(file_path, io.Node, work.Rank, attrfile_path, io.Type, io.FormOptions, io.NodeAttr); err != nil {
				fmt.Errorf("push file error\n")
				logger.Error("op=pushfile,err=" + err.Error())
				return size, err
			}
		}
		logger.Event(event.FILE_DONE,
			"workid="+work.Id,
			"filename="+name,
			fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))
	}
	return
}

//push file to shock (=deprecated=)
func pushFileByCurl(filename string, host string, node string, rank int) (err error) {
	shockurl := fmt.Sprintf("%s/node/%s", host, node)
	if err := putFileByCurl(filename, shockurl, rank); err != nil {
		return err
	}
	return
}

//(=deprecated=)
func putFileByCurl(filename string, target_url string, rank int) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	argv = append(argv, "-F")

	if rank == 0 {
		argv = append(argv, fmt.Sprintf("upload=@%s", filename))
	} else {
		argv = append(argv, fmt.Sprintf("%d=@%s", rank, filename))
	}
	argv = append(argv, target_url)
	logger.Debug(2, fmt.Sprintf("deliverer: curl argv=%#v", argv))
	cmd := exec.Command("curl", argv...)
	err = cmd.Run()
	if err != nil {
		return
	}
	return
}

func getPerfFilePath(work *Workunit, perfstat *WorkPerf) (reportPath string, err error) {
	perfJsonstream, err := json.Marshal(perfstat)
	if err != nil {
		return reportPath, err
	}
	work_path, err := work.Path()
	if err != nil {
		return
	}
	reportPath = fmt.Sprintf("%s/%s.perf", work_path, work.Id)
	err = ioutil.WriteFile(reportPath, []byte(perfJsonstream), 0644)
	return
}

func getStdOutPath(work *Workunit) (stdoutFilePath string, err error) {
	work_path, err := work.Path()
	if err != nil {
		return
	}
	stdoutFilePath = fmt.Sprintf("%s/%s", work_path, conf.STDOUT_FILENAME)
	fi, err := os.Stat(stdoutFilePath)
	if err != nil {
		return stdoutFilePath, err
	}
	if fi.Size() == 0 {
		return stdoutFilePath, errors.New("stdout file empty")
	}
	return stdoutFilePath, err
}

func getStdErrPath(work *Workunit) (stderrFilePath string, err error) {
	work_path, err := work.Path()
	if err != nil {
		return
	}
	stderrFilePath = fmt.Sprintf("%s/%s", work_path, conf.STDERR_FILENAME)
	fi, err := os.Stat(stderrFilePath)
	if err != nil {
		return stderrFilePath, err
	}
	if fi.Size() == 0 {
		return stderrFilePath, errors.New("stderr file empty")
	}
	return
}

func getWorkNotesPath(work *Workunit) (worknotesFilePath string, err error) {
	work_path, err := work.Path()
	if err != nil {
		return
	}
	worknotesFilePath = fmt.Sprintf("%s/%s", work_path, conf.WORKNOTES_FILENAME)
	if len(work.Notes) == 0 {
		return worknotesFilePath, errors.New("work notes empty")
	}
	err = ioutil.WriteFile(worknotesFilePath, []byte(work.GetNotes()), 0644)
	return
}

func GetJob(id string) (job *Job, err error) {
	job, ok, err := JM.Get(id, true)
	if err != nil {
		err = fmt.Errorf("(GetJob) JM.Get failed: %s", err.Error())
		return
	}
	if !ok {
		// load job if not already in memory
		job, err = LoadJob(id)
		if err != nil {
			err = fmt.Errorf("(GetJob) LoadJob failed: %s", err.Error())
			return
		}
		err = JM.Add(job)
		if err != nil {
			err = fmt.Errorf("(GetJob) JM.Add failed: %s", err.Error())
			return
		}
	}
	return
}
