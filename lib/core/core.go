package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/acl"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/httpclient"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/MG-RAST/AWE/lib/user"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

var (
	QMgr          ResourceMgr
	Service       string = "unknown"
	Self          *Client
	ProxyWorkChan chan bool
)

func InitResMgr(service string) {
	if service == "server" {
		QMgr = NewServerMgr()
	} else if service == "proxy" {
		QMgr = NewProxyMgr()
	}
	Service = service
}

func InitClientProfile(profile *Client) {
	Self = profile
}

func InitProxyWorkChan() {
	ProxyWorkChan = make(chan bool, 100)
}

type CoReq struct {
	policy     string
	fromclient string
	count      int
}

type CoAck struct {
	workunits []*Workunit
	err       error
}

type Notice struct {
	WorkId      string
	Status      string
	ClientId    string
	ComputeTime int
	Notes       string
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

type Opts map[string]string

func (o *Opts) HasKey(key string) bool {
	if _, has := (*o)[key]; has {
		return true
	}
	return false
}

func (o *Opts) Value(key string) string {
	val, _ := (*o)[key]
	return val
}

//heartbeat response from awe-server to awe-client
//used for issue operation request to client, e.g. discard suspended workunits
type HBmsg map[string]string //map[op]obj1,obj2 e.g. map[discard]=work1,work2

func CreateJobUpload(u *user.User, params map[string]string, files FormFiles, jid string) (job *Job, err error) {

	if _, has_upload := files["upload"]; has_upload {
		job, err = ParseJobTasks(files["upload"].Path, jid)
	} else {
		job, err = ParseAwf(files["awf"].Path, jid)
	}

	if err != nil {
		err = errors.New("error parsing job, error=" + err.Error())
		return
	}

	// Once, job has been created, set job owner and add owner to all ACL's
	job.Acl.SetOwner(u.Uuid)
	job.Acl.Set(u.Uuid, acl.Rights{"read": true, "write": true, "delete": true})

	err = job.Mkdir()
	if err != nil {
		err = errors.New("error creating job directory, error=" + err.Error())
		return
	}

	// TODO need a way update app-defintions in AWE server...
	if MyAppRegistry == nil && conf.APP_REGISTRY_URL != "" {
		MyAppRegistry, err = MakeAppRegistry()
		if err != nil {
			return job, errors.New("error creating app registry, error=" + err.Error())
		}
		logger.Debug(1, "app defintions read")
	}

	err = MyAppRegistry.createIOnodes(job)
	if err != nil {
		err = errors.New("error in createIOnodes, error=" + err.Error())
		return
	}

	err = job.UpdateFile(params, files)
	if err != nil {
		err = errors.New("error in UpdateFile, error=" + err.Error())
		return
	}

	err = job.Save()
	if err != nil {
		err = errors.New("error in job.Save(), error=" + err.Error())
		return
	}
	return
}

//create a shock node for output  (=deprecated=)
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
	defer res.Body.Close()

	jsonstream, err := ioutil.ReadAll(res.Body)

	response := new(shock.ShockResponse)
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

func PostNodeWithToken(io *IO, numParts int, token string) (nodeid string, err error) {
	opts := Opts{}
	node, err := createOrUpdate(opts, io.Host, "", token, nil)
	if err != nil {
		return "", err
	}
	//create "parts" for output splits
	if numParts > 1 {
		opts["upload_type"] = "parts"
		opts["file_name"] = io.Name
		opts["parts"] = strconv.Itoa(numParts)
		if _, err := createOrUpdate(opts, io.Host, node.Id, token, nil); err != nil {
			return node.Id, err
		}
	}
	return node.Id, nil
}

//create parts (=deprecated=)
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
		return nil, errors.New("error in reading job json file" + err.Error())
	}

	err = json.Unmarshal(jsonstream, job)
	if err != nil {
		return nil, errors.New("error in unmarshaling job json file: " + err.Error())
	}

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
	job.State = JOB_STAT_INIT
	job.Registered = true

	//parse private fields task.Cmd.Environ.Private
	job_p := new(Job_p)
	err = json.Unmarshal(jsonstream, job_p)
	if err == nil {
		for idx, task := range job_p.Tasks {
			if task.Cmd.Environ == nil || task.Cmd.Environ.Private == nil {
				continue
			}
			job.Tasks[idx].Cmd.Environ.Private = make(map[string]string)
			for key, val := range task.Cmd.Environ.Private {
				job.Tasks[idx].Cmd.Environ.Private[key] = val
			}
		}
	}

	for i := 0; i < len(job.Tasks); i++ {
		if err := job.Tasks[i].InitTask(job, i); err != nil {
			return nil, errors.New("error in InitTask: " + err.Error())
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
	}
	return "", errors.New("invalid task id: " + taskid)
}

func GetJobIdByWorkId(workid string) (jobid string, err error) {
	parts := strings.Split(workid, "_")
	if len(parts) == 3 {
		return parts[0], nil
	}
	return "", errors.New("invalid work id: " + workid)
}

func GetTaskIdByWorkId(workid string) (taskid string, err error) {
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

//update job state to "newstate" only if the current state is in one of the "oldstates"
func UpdateJobState(jobid string, newstate string, oldstates []string) (err error) {
	job, err := LoadJob(jobid)
	if err != nil {
		return
	}
	matched := false
	for _, oldstate := range oldstates {
		if oldstate == job.State {
			matched = true
			break
		}
	}
	if !matched {
		return errors.New("old state not matching one of the required ones")
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

//
func contains(list []string, elem string) bool {
	for _, t := range list {
		if t == elem {
			return true
		}
	}
	return false
}

func jidIncr(jid string) (newjid string) {
	if jidint, err := strconv.Atoi(jid); err == nil {
		jidint += 1
		return strconv.Itoa(jidint)
	}
	return jid
}

//functions for REST API communication  (=deprecated=)
//notify AWE server a workunit is finished with status either "failed" or "done", and with perf statistics if "done"
func NotifyWorkunitProcessed(work *Workunit, perf *WorkPerf) (err error) {
	target_url := fmt.Sprintf("%s/work/%s?status=%s&client=%s", conf.SERVER_URL, work.Id, work.State, Self.Id)

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

func NotifyWorkunitProcessedWithLogs(work *Workunit, perf *WorkPerf, sendstdlogs bool) (err error) {
	target_url := fmt.Sprintf("%s/work/%s?status=%s&client=%s&computetime=%d", conf.SERVER_URL, work.Id, work.State, Self.Id, work.ComputeTime)
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
	if hasreport {
		target_url = target_url + "&report"
	}
	if err := form.Create(); err != nil {
		return err
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
	res, err := httpclient.Put(target_url, headers, form.Reader, nil)
	if err == nil {
		defer res.Body.Close()
	}

	return
}

// deprecated, see cache.UploadOutputData
func PushOutputData(work *Workunit) (size int64, err error) {
	for name, io := range work.Outputs {
		var local_filepath string //local file name generated by the cmd
		var file_path string      //file name to be uploaded to shock

		if io.Directory != "" {
			local_filepath = fmt.Sprintf("%s/%s/%s", work.Path(), io.Directory, name)
			//if specified, rename the local file name to the specified shock node file name
			//otherwise use the local name as shock file name
			file_path = local_filepath
			if io.ShockFilename != "" {
				file_path = fmt.Sprintf("%s/%s/%s", work.Path(), io.Directory, io.ShockFilename)
				os.Rename(local_filepath, file_path)
			}
		} else {
			local_filepath = fmt.Sprintf("%s/%s", work.Path(), name)
			file_path = local_filepath
			if io.ShockFilename != "" {
				file_path = fmt.Sprintf("%s/%s", work.Path(), io.ShockFilename)
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
			attrfile_path = fmt.Sprintf("%s/%s", work.Path(), io.AttrFile)
			if fi, err := os.Stat(attrfile_path); err != nil || fi.Size() == 0 {
				attrfile_path = ""
			}
		}

		//set io.FormOptions["parent_node"] if not present and io.FormOptions["parent_name"] exists
		if parent_name, ok := io.FormOptions["parent_name"]; ok {
			for in_name, in_io := range work.Inputs {
				if in_name == parent_name {
					io.FormOptions["parent_node"] = in_io.Node
				}
			}
		}

		if err := PutFileToShock(file_path, io.Host, io.Node, work.Rank, work.Info.DataToken, attrfile_path, io.Type, io.FormOptions, &io.NodeAttr); err != nil {
			time.Sleep(3 * time.Second) //wait for 3 seconds and try again
			if err := PutFileToShock(file_path, io.Host, io.Node, work.Rank, work.Info.DataToken, attrfile_path, io.Type, io.FormOptions, &io.NodeAttr); err != nil {
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

func PutFileToShock(filename string, host string, nodeid string, rank int, token string, attrfile string, ntype string, formopts map[string]string, nodeattr *map[string]interface{}) (err error) {
	opts := Opts{}
	fi, _ := os.Stat(filename)
	if (attrfile != "") && (rank < 2) {
		opts["attributes"] = attrfile
	}
	if filename != "" {
		opts["file"] = filename
	}
	if rank == 0 {
		opts["upload_type"] = "basic"
	} else {
		opts["upload_type"] = "part"
		opts["part"] = strconv.Itoa(rank)
	}
	if (ntype == "subset") && (rank == 0) && (fi.Size() == 0) {
		opts["upload_type"] = "basic"
	} else if ((ntype == "copy") || (ntype == "subset")) && (len(formopts) > 0) {
		opts["upload_type"] = ntype
		for k, v := range formopts {
			opts[k] = v
		}
	}

	_, err = createOrUpdate(opts, host, nodeid, token, nodeattr)
	return
}

func getPerfFilePath(work *Workunit, perfstat *WorkPerf) (reportPath string, err error) {
	perfJsonstream, err := json.Marshal(perfstat)
	if err != nil {
		return reportPath, err
	}
	reportPath = fmt.Sprintf("%s/%s.perf", work.Path(), work.Id)
	err = ioutil.WriteFile(reportPath, []byte(perfJsonstream), 0644)
	return
}

func getStdOutPath(work *Workunit) (stdoutFilePath string, err error) {
	stdoutFilePath = fmt.Sprintf("%s/%s", work.Path(), conf.STDOUT_FILENAME)
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
	stderrFilePath = fmt.Sprintf("%s/%s", work.Path(), conf.STDERR_FILENAME)
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
	worknotesFilePath = fmt.Sprintf("%s/%s", work.Path(), conf.WORKNOTES_FILENAME)
	if work.Notes == "" {
		return worknotesFilePath, errors.New("work notes empty")
	}
	err = ioutil.WriteFile(worknotesFilePath, []byte(work.Notes), 0644)
	return
}

//shock access functions
func createOrUpdate(opts Opts, host string, nodeid string, token string, nodeattr *map[string]interface{}) (node *shock.ShockNode, err error) {
	url := host + "/node"
	method := "POST"
	if nodeid != "" {
		url += "/" + nodeid
		method = "PUT"
	}
	form := httpclient.NewForm()
	if opts.HasKey("attributes") {
		form.AddFile("attributes", opts.Value("attributes"))
	}
	//if opts.HasKey("attributes_str") {
	//	form.AddParam("attributes_str", opts.Value("attributes_str"))
	//}

	if len(nodeattr) != 0 {
		nodeattr_json, err := json.Marshal(nodeattr)
		if err != nil {
			return nil, errors.New("error marshalling NodeAttr")
		}
		form.AddParam("attributes_str", string(nodeattr_json[:]))
	}

	if opts.HasKey("upload_type") {
		switch opts.Value("upload_type") {
		case "basic":
			if opts.HasKey("file") {
				form.AddFile("upload", opts.Value("file"))
			}
		case "parts":
			if opts.HasKey("parts") {
				form.AddParam("parts", opts.Value("parts"))
			} else {
				return nil, errors.New("missing partial upload parameter: parts")
			}
			if opts.HasKey("file_name") {
				form.AddParam("file_name", opts.Value("file_name"))
			}
		case "part":
			if opts.HasKey("part") && opts.HasKey("file") {
				form.AddFile(opts.Value("part"), opts.Value("file"))
			} else {
				return nil, errors.New("missing partial upload parameter: part or file")
			}
		case "remote_path":
			if opts.HasKey("remote_path") {
				form.AddParam("path", opts.Value("remote_path"))
			} else {
				return nil, errors.New("missing remote path parameter: path")
			}
		case "virtual_file":
			if opts.HasKey("virtual_file") {
				form.AddParam("type", "virtual")
				form.AddParam("source", opts.Value("virtual_file"))
			} else {
				return nil, errors.New("missing virtual node parameter: source")
			}
		case "index":
			if opts.HasKey("index_type") {
				url += "/index/" + opts.Value("index_type")
			} else {
				return nil, errors.New("missing index type when creating index")
			}
		case "copy":
			if opts.HasKey("parent_node") {
				form.AddParam("copy_data", opts.Value("parent_node"))
			} else {
				return nil, errors.New("missing copy node parameter: parent_node")
			}
			if opts.HasKey("copy_indexes") {
				form.AddParam("copy_indexes", "1")
			}
		case "subset":
			if opts.HasKey("parent_node") && opts.HasKey("parent_index") && opts.HasKey("file") {
				form.AddParam("parent_node", opts.Value("parent_node"))
				form.AddParam("parent_index", opts.Value("parent_index"))
				form.AddFile("subset_indices", opts.Value("file"))
			} else {
				return nil, errors.New("missing subset node parameter: parent_node or parent_index or file")
			}
		}
	}
	err = form.Create()
	if err != nil {
		return
	}
	headers := httpclient.Header{
		"Content-Type":   []string{form.ContentType},
		"Content-Length": []string{strconv.FormatInt(form.Length, 10)},
	}
	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}
	if res, err := httpclient.Do(method, url, headers, form.Reader, user); err == nil {
		defer res.Body.Close()
		jsonstream, _ := ioutil.ReadAll(res.Body)
		response := new(shock.ShockResponse)
		if err := json.Unmarshal(jsonstream, response); err != nil {
			return nil, errors.New(fmt.Sprintf("failed to marshal response:\"%s\"", jsonstream))
		}
		if len(response.Errs) > 0 {
			return nil, errors.New(strings.Join(response.Errs, ","))
		}
		node = &response.Data
	} else {
		return nil, err
	}
	return
}

func ShockPutIndex(host string, nodeid string, indexname string, token string) (err error) {
	opts := Opts{}
	opts["upload_type"] = "index"
	opts["index_type"] = indexname
	createOrUpdate(opts, host, nodeid, token, nil)
	return
}
