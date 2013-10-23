package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/httpclient"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
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
	WorkId   string
	Status   string
	ClientId string
	Notes    string
}

type coInfo struct {
	workunit *Workunit
	clientid string
}

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

func PostNodeWithToken(io *IO, numParts int, token string) (nodeid string, err error) {
	opts := Opts{}
	node, err := createOrUpdate(opts, io.Host, "", token)
	if err != nil {
		return "", err
	}
	//create "parts" for output splits
	if numParts > 1 {
		opts["upload_type"] = "parts"
		opts["parts"] = strconv.Itoa(numParts)
		if _, err := createOrUpdate(opts, io.Host, node.Id, token); err != nil {
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

//functions for REST API communication
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

func PushOutputData(work *Workunit) (err error) {
	for name, io := range work.Outputs {
		file_path := fmt.Sprintf("%s/%s", work.Path(), name)
		//use full path here, cwd could be changed by Worker (likely in worker-overlapping mode)
		if fi, err := os.Stat(file_path); err != nil {
			if io.Optional {
				continue
			} else {
				return errors.New(fmt.Sprintf("output %s not generated for workunit %s", name, work.Id))
			}
		} else {
			if io.Nonzero && fi.Size() == 0 {
				return errors.New(fmt.Sprintf("workunit %s generated zero-sized output %s while non-zero-sized file required", work.Id, name))
			}
		}
		logger.Debug(2, "deliverer: push output to shock, filename="+name)
		logger.Event(event.FILE_OUT,
			"workid="+work.Id,
			"filename="+name,
			fmt.Sprintf("url=%s/node/%s", io.Host, io.Node))

		if err := putFileToShock(file_path, io.Host, io.Node, work.Rank, work.Info.DataToken); err != nil {
			time.Sleep(3 * time.Second) //wait for 3 seconds and try again
			if err := putFileToShock(file_path, io.Host, io.Node, work.Rank, work.Info.DataToken); err != nil {
				fmt.Errorf("push file error\n")
				logger.Error("op=pushfile,err=" + err.Error())
				return err
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

func putFileToShock(filename string, host string, nodeid string, rank int, token string) (err error) {
	opts := Opts{}
	if rank == 0 {
		opts["upload_type"] = "full"
		opts["full"] = filename
	} else {
		opts["upload_type"] = "part"
		opts["part"] = strconv.Itoa(rank)
		opts["file"] = filename
	}
	_, err = createOrUpdate(opts, host, nodeid, token)
	return
}

func getPerfFilePath(work *Workunit, perfstat *WorkPerf) (reportPath string, err error) {
	perfJsonstream, err := json.Marshal(perfstat)
	if err != nil {
		return reportPath, err
	}
	reportFile := fmt.Sprintf("%s/%s.perf", work.Path(), work.Id)
	if err := ioutil.WriteFile(reportFile, []byte(perfJsonstream), 0644); err != nil {
		return reportPath, err
	}
	return reportFile, nil
}

//shock access functions
func createOrUpdate(opts Opts, host string, nodeid string, token string) (node *ShockNode, err error) {
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
	if opts.HasKey("upload_type") {
		switch opts.Value("upload_type") {
		case "full":
			if opts.HasKey("full") {
				form.AddFile("upload", opts.Value("full"))
			} else {
				return nil, errors.New("missing file parameter: upload")
			}
		case "parts":
			if opts.HasKey("parts") {
				form.AddParam("parts", opts.Value("parts"))
			} else {
				return nil, errors.New("missing partial upload parameter: parts")
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
				url += "?index=" + opts.Value("index_type")
			} else {
				return nil, errors.New("missing index type when creating index")
			}
		}
	}
	err = form.Create()
	if err != nil {
		return
	}
	headers := httpclient.Header{
		"Content-Type":   form.ContentType,
		"Content-Length": strconv.FormatInt(form.Length, 10),
	}
	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}
	if res, err := httpclient.Do(method, url, headers, form.Reader, user); err == nil {
		defer res.Body.Close()
		jsonstream, _ := ioutil.ReadAll(res.Body)
		response := new(ShockResponse)
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

func ShockGet(host string, nodeid string, token string) (node *ShockNode, err error) {
	if host == "" || nodeid == "" {
		return nil, errors.New("empty shock host or node id")
	}

	var res *http.Response
	shockurl := fmt.Sprintf("%s/node/%s", host, nodeid)

	var user *httpclient.Auth
	if token != "" {
		user = httpclient.GetUserByTokenAuth(token)
	}

	c := make(chan int, 1)
	go func() {
		res, err = httpclient.Get(shockurl, httpclient.Header{}, nil, user)
		c <- 1 //we are ending
	}()
	select {
	case <-c:
	//go ahead
	case <-time.After(conf.SHOCK_TIMEOUT):
		return nil, errors.New("timeout when getting node from shock, url=" + shockurl)
	}
	if err != nil {
		return
	}
	defer res.Body.Close()

	jsonstream, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}

	response := new(ShockResponse)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return nil, err
	}
	if len(response.Errs) > 0 {
		return nil, errors.New(strings.Join(response.Errs, ","))
	}
	node = &response.Data
	if node == nil {
		err = errors.New("empty node got from Shock")
	}
	return
}

func ShockPutIndex(host string, nodeid string, indexname string, token string) (err error) {
	opts := Opts{}
	opts["upload_type"] = "index"
	opts["index_type"] = indexname
	createOrUpdate(opts, host, nodeid, token)
	return
}
