package main

import (
	//"encoding/json"
	"fmt"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	shock "github.com/MG-RAST/go-shock-client"

	//"github.com/MG-RAST/AWE/lib/logger/event"

	"encoding/json"

	"github.com/MG-RAST/AWE/lib/cache"
	//"github.com/davecgh/go-spew/spew"

	"io/ioutil"
	"os"
	"os/exec"
	"path"
	"strings"
	"time"

	"gopkg.in/yaml.v2"
)

type standardResponse struct {
	Status int         `json:"status"`
	Data   interface{} `json:"data"`
	Error  []string    `json:"error"`
}

func main() {
	err := mainWrapper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %s\n\n", err.Error())
		time.Sleep(time.Second)
		os.Exit(1)
	}
	time.Sleep(time.Second)
	os.Exit(0)
}

func mainWrapper() (err error) {

	conf.LOG_OUTPUT = "console"

	err = conf.Init_conf("submitter")

	if err != nil {
		err = fmt.Errorf("error reading conf file: %s", err.Error())
		return
	}

	logger.Initialize("client")

	aweAuth := conf.SUBMITTER_AWE_AUTH
	shockAuth := conf.SUBMITTER_SHOCK_AUTH

	if aweAuth != "" {
		aweAuthArray := strings.SplitN(aweAuth, " ", 2)
		if len(aweAuthArray) != 2 {
			err = fmt.Errorf("error parsing AWE_AUTH (expected format \"bearer token\")")
			return
		}
		fmt.Fprintf(os.Stderr, "Using AWE authentication\n")
	} else {
		fmt.Fprintf(os.Stderr, "No AWE authentication. (Example: AWE_AUTH=\"bearer token\")\n")
	}

	//fmt.Printf("AWE_AUTH=%s\n", awe_auth) // TODO needs to have bearer embedded
	//fmt.Printf("SHOCK_AUTH=%s\n", shock_auth)

	//for _, value := range conf.ARGS {
	//	println(value)
	//}
	jobFile := ""

	if conf.SUBMITTER_UPLOAD_INPUT {
		jobFile = conf.ARGS[0]

		inputfilePath := path.Dir(jobFile)

		var jobDoc *cwl.Job_document

		jobDoc, err = cwl.ParseJobFile(jobFile)
		if err != nil {
			err = fmt.Errorf("error parsing cwl job: %s", err.Error())
			return
		}

		shockClient := shock.NewShockClient(conf.SHOCK_URL, shockAuth, false)
		shockClient.Debug = true
		var uploadCount int
		//var jobData []byte

		uploadCount, err = cache.ProcessIOData(jobDoc, inputfilePath, inputfilePath, "upload", shockClient, true)
		if err != nil {
			err = fmt.Errorf("(mainWrapper) ProcessIOData(for upload) returned: %s", err.Error())
			return
		}

		logger.Debug(3, "%d files have been uploaded\n", uploadCount)
		//time.Sleep(2)

		//spew.Dump(*job_doc)
		jobDocMap := jobDoc.GetMap()

		var jobData []byte
		jobData, err = yaml.Marshal(jobDocMap)
		if err != nil {
			return
		}

		fmt.Printf("%s", jobData)

		return
	}

	var workflowTemporaryFile string
	var entrypoint string
	//fmt.Printf("conf.ARGS: %d\n", conf.ARGS)
	if len(conf.ARGS) >= 2 {

		args1Array := strings.Split(conf.ARGS[1], "#")
		jobFile = args1Array[0]
		if len(args1Array) > 1 {
			entrypoint = args1Array[1]
		}

		//fmt.Printf("jobFile: %s\n", jobFile)
	}
	logger.Debug(3, "(main_wrapper) A entrypoint: %s", entrypoint)
	//fmt.Printf("---------- A\n")

	var jobData []byte
	var newEntrypoint string // name of Workflow in single document for example
	workflowTemporaryFile, jobData, newEntrypoint, err = createNormalizedSubmisson(aweAuth, shockAuth, conf.ARGS[0], jobFile, entrypoint)
	if err != nil {
		err = fmt.Errorf("(mainWrapper) createNormalizedSubmisson returned: %s", err.Error())
		return
	}

	if newEntrypoint != "" {
		entrypoint = newEntrypoint
	}
	logger.Debug(3, "(main_wrapper) B entrypoint: %s", entrypoint)
	//fmt.Printf("---------- B\n")
	// ### Submit job to AWE
	var jobid string

	// single tools are wrapped into a workflow, thus returns new entrypoint
	jobid, newEntrypoint, err = SubmitCWLJobToAWE(workflowTemporaryFile, jobFile, entrypoint, &jobData, aweAuth, shockAuth)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) SubmitCWLJobToAWE returned: %s", err.Error())
		return
	}

	if newEntrypoint != "" {
		entrypoint = newEntrypoint
	}
	logger.Debug(3, "(main_wrapper) C entrypoint: %s", entrypoint)
	// if strings.HasPrefix(entrypoint, "#") {
	// 	entrypoint = strings.TrimPrefix(entrypoint, "#")
	// }
	//fmt.Printf("---------- C\n")
	//fmt.Printf("Job id: %s\n", jobid)

	if conf.SUBMITTER_WAIT {
		err = WaitForResults(jobid, entrypoint, aweAuth)
		if err != nil {
			err = fmt.Errorf("(main_wrapper) Wait_for_results returned: %s", err.Error())
			return
		}
	} else {
		fmt.Printf("JobID=%s\n", jobid)
	}

	return
}

func createNormalizedSubmisson(aweAuth string, shockAuth string, workflowFile string, jobFile string, entrypoint string) (workflowTemporaryFile string, jobData []byte, newEntrypoint string, err error) {

	//workflowFile := conf.ARGS[0]

	inputfilePath := ""
	inputFileBase := ""
	//fmt.Printf("job path: %s\n", inputfile_path) // needed to resolve relative paths
	//fmt.Printf("createNormalizedSubmisson A\n")
	// ### parse job file
	var jobDoc *cwl.Job_document

	if jobFile != "" {

		inputfilePath = path.Dir(jobFile)
		inputFileBase = path.Base(jobFile)
		jobDoc, err = cwl.ParseJobFile(jobFile)
		if err != nil {
			err = fmt.Errorf("error parsing cwl job: %s", err.Error())
			return
		}
	} else {
		jobDoc = &cwl.Job_document{}
		inputfilePath = path.Dir(workflowFile)
		inputFileBase = path.Base(workflowFile)
	}
	//fmt.Println("Job input after reading from file:")
	//spew.Dump(*job_doc)

	var jobDocMap cwl.JobDocMap
	//fmt.Printf("createNormalizedSubmisson B\n")
	//job_doc_map["test"] = cwl.NewNull()

	// if conf.DEBUG_LEVEL >= 3 {
	// 	fmt.Println("Job input after reading from file: map !!!!\n")
	// 	spew.Dump(jobDocMap)
	// }
	//var jobData []byte

	if conf.DEBUG_LEVEL >= 3 {
		fmt.Printf("jobFile: \"%s\"\n", jobFile)

		jobDocMap = jobDoc.GetMap()
		jobData, err = yaml.Marshal(jobDocMap)
		if err != nil {
			return
		}

		jobDocString := string(jobData[:])
		fmt.Printf("job_doc_string:\n \"%s\"\n", jobDocString)

		fmt.Printf("yaml:\n%s\n", jobDocString)
	}

	// ### upload input files

	shockClient := shock.NewShockClient(conf.SHOCK_URL, shockAuth, false)
	//fmt.Printf("createNormalizedSubmisson C\n")
	uploadCount := 0

	if jobFile != "" {
		uploadCount, err = cache.ProcessIOData(jobDoc, inputfilePath, inputfilePath, "upload", shockClient, true)
		if err != nil {
			err = fmt.Errorf("(createNormalizedSubmisson) A) ProcessIOData(for upload) returned: %s", err.Error())
			return
		}
	}
	logger.Debug(3, "%d files have been uploaded\n", uploadCount)
	//time.Sleep(2)

	//spew.Dump(*job_doc)
	jobDocMap = jobDoc.GetMap()
	//fmt.Println("------------Job input after parsing:")
	jobData, err = yaml.Marshal(jobDocMap)
	if err != nil {
		return
	}
	//fmt.Printf("createNormalizedSubmisson D\n")
	if conf.DEBUG_LEVEL >= 3 {
		fmt.Printf("job input as yaml:\n%s\n", string(jobData[:]))
	}
	var yamlstream []byte
	// read and pack workfow
	if conf.SUBMITTER_PACK {

		yamlstream, err = exec.Command("cwltool", "--pack", workflowFile).Output()
		if err != nil {
			err = fmt.Errorf("(createNormalizedSubmisson) exec.Command returned: %s (%s %s %s)", err.Error(), "cwltool", "--pack", workflowFile)
			return
		}

	} else {

		yamlstream, err = ioutil.ReadFile(workflowFile)
		if err != nil {
			err = fmt.Errorf("error in reading workflow file: " + err.Error())
			return
		}
	}
	// ### PARSE (maybe PACKED) WORKFLOW DOCUMENT, in case default files have to be uploaded

	// convert CWL to string
	yamlStr := string(yamlstream[:])
	//fmt.Printf("after cwltool --pack: \n%s\n", yaml_str)
	var namedObjectArray []cwl.NamedCWLObject
	//var cwl_version cwl.CWLVersion
	var schemata []cwl.CWLType_Type
	//var namespaces map[string]string
	//var schemas []interface{}
	var context *cwl.WorkflowContext
	//fmt.Printf("createNormalizedSubmisson E\n")
	//var newEntrypoint string
	namedObjectArray, schemata, context, _, newEntrypoint, err = cwl.ParseCWLDocument(yamlStr, entrypoint, inputfilePath, inputFileBase)

	// if newEntrypoint != "" {
	// 	entrypoint = newEntrypoint
	// }

	//fmt.Printf("createNormalizedSubmisson F\n")
	if err != nil {
		if conf.SUBMITTER_PACK {
			err = fmt.Errorf("(createNormalizedSubmisson) error in parsing output of \"cwltool --pack %s\" (%s)", workflowFile, err.Error())
		} else {
			err = fmt.Errorf("(createNormalizedSubmisson) error in parsing file %s: %s", workflowFile, err.Error())
		}
		return
	}

	_ = schemata // TODO put into a collection!

	// A) search for File objects in Document, e.g. in CommandLineTools

	subUploadCount := 0
	subUploadCount, err = cache.ProcessIOData(namedObjectArray, inputfilePath, inputfilePath, "upload", shockClient, true)
	if err != nil {
		err = fmt.Errorf("(createNormalizedSubmisson) B) ProcessIOData(for upload) returned: %s", err.Error())
		return
	}
	uploadCount += subUploadCount
	//fmt.Printf("createNormalizedSubmisson G\n")
	if context.Schemas != nil {
		subUploadCount := 0
		subUploadCount, err = cache.ProcessIOData(context.Schemas, inputfilePath, inputfilePath, "upload", shockClient, true)
		if err != nil {
			err = fmt.Errorf("(createNormalizedSubmisson) C) ProcessIOData(for upload) returned: %s", err.Error())
			return
		}
		uploadCount += subUploadCount

	}

	logger.Debug(3, "%d files have been uploaded\n", uploadCount)

	var shockRequirement cwl.ShockRequirement
	var shockRequirementPtr *cwl.ShockRequirement
	shockRequirementPtr, err = cwl.NewShockRequirement(shockClient.Host)
	if err != nil {
		err = fmt.Errorf("(createNormalizedSubmisson) NewShockRequirement returned: %s", err.Error())
		return
	}

	shockRequirement = *shockRequirementPtr

	// B) inject ShockRequirement into CommandLineTools, ExpressionTools and Workflow
	for j := range namedObjectArray {

		pair := namedObjectArray[j]
		object := pair.Value

		var ok bool

		switch object.(type) {
		case *cwl.Workflow:
			workflow := object.(*cwl.Workflow)

			workflow.Requirements, err = cwl.AddRequirement(shockRequirement, workflow.Requirements)
			if err != nil {
				err = fmt.Errorf("(createNormalizedSubmisson) AddRequirement returned: %s", err.Error())
				return
			}

		case *cwl.CommandLineTool:
			var cmdLineTool *cwl.CommandLineTool
			cmdLineTool, ok = object.(*cwl.CommandLineTool) // TODO this misses embedded CommandLineTools !
			if !ok {
				//fmt.Println("nope.")
				err = nil
				continue
			}

			cmdLineTool.Requirements, err = cwl.AddRequirement(shockRequirement, cmdLineTool.Requirements)
			if err != nil {
				err = fmt.Errorf("(createNormalizedSubmisson) AddRequirement returned: %s", err.Error())
			}

		case *cwl.ExpressionTool:
			var expressTool *cwl.ExpressionTool
			expressTool, ok = object.(*cwl.ExpressionTool) // TODO this misses embedded ExpressionTools !
			if !ok {
				//fmt.Println("nope.")
				err = nil
				continue
			}

			if expressTool == nil {
				err = fmt.Errorf("(createNormalizedSubmisson) express_tool==nil")
				return
			}

			expressTool.Requirements, err = cwl.AddRequirement(shockRequirement, expressTool.Requirements)
			if err != nil {
				err = fmt.Errorf("(createNormalizedSubmisson) AddRequirement returned: %s", err.Error())
			}

		}
	}

	// create temporary workflow document file

	newDocument := context.GraphDocument

	//new_document := cwl.GraphDocument{}
	//new_document.CwlVersion = context.CwlVersion
	//new_document.Namespaces = namespaces
	//new_document.Schemas = schemas

	// replace graph
	newDocument.Graph = []interface{}{}
	for i := range namedObjectArray {
		pair := namedObjectArray[i]
		object := pair.Value
		newDocument.Graph = append(newDocument.Graph, object)
	}

	if len(newDocument.Graph) == 0 {
		err = fmt.Errorf("(createNormalizedSubmisson) len(new_document.Graph) == 0")
		return
	}

	var newDocumentBytes []byte
	newDocumentBytes, err = yaml.Marshal(newDocument)
	if err != nil {
		err = fmt.Errorf("(createNormalizedSubmisson) yaml.Marshal returned: %s", err.Error())
		return
	}
	newDocumentStr := string(newDocumentBytes[:])
	//graph_pos := strings.Index(new_document_str, "\ngraph:")

	//new_document_str = strings.Replace(new_document_str, "\nnamespaces", "\n$namespaces", -1) // remove dollar sign

	//if graph_pos != -1 {
	//new_document_str = strings.Replace(new_document_str, "\ngraph", "\n$graph", -1) // remove dollar sign
	//} else {

	//	err = fmt.Errorf("(main_wrapper) keyword graph not found")
	//	return
	//}

	if conf.DEBUG_LEVEL >= 3 {
		fmt.Println("------------ new_document_str:")
		fmt.Println(newDocumentStr)
		fmt.Println("------------")
		//panic("hhhh")
	}
	newDocumentBytes = []byte(newDocumentStr)

	// ### Write workflow to file, so we can run "cwltool --pack""
	var tmpfile *os.File
	tmpfile, err = ioutil.TempFile(os.TempDir(), "awe-submitter_")
	if err != nil {
		err = fmt.Errorf("(main_wrapper) ioutil.TempFile returned: %s", err.Error())
		return
	}
	workflowTemporaryFile = tmpfile.Name()
	//defer os.Remove(tempfile_name)

	_, err = tmpfile.Write(newDocumentBytes)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) tmpfile.Write returned: %s", err.Error())
		return
	}

	err = tmpfile.Close()
	if err != nil {
		err = fmt.Errorf("(main_wrapper) tmpfile.Close returned: %s", err.Error())
		return
	}

	// job submission example:
	// curl -X POST -F job=@test.yaml -F cwl=@/Users/wolfganggerlach/awe_data/pipeline/CWL/PackedWorkflow/preprocess-fasta.workflow.cwl http://localhost:8001/job

	//var b bytes.Buffer
	//w := multipart.NewWriter(&b)

	return
}

// WaitForResults _
func WaitForResults(jobid string, entrypoint string, aweAuth string) (err error) {

	var job *core.Job
	var statusCode int
	// ***** Wait for job to complete

FORLOOP:
	for true {
		time.Sleep(5 * time.Second)
		job = nil

		job, statusCode, err = GetAWEJob(jobid, aweAuth)
		if err != nil {
			fmt.Fprintf(os.Stderr, "(Wait_for_results) GetAWEJob returned: (status_code: %d) error: %s  \n", statusCode, err.Error())
			if statusCode == -1 {
				err = nil
				continue
			}
			return
		}

		//fmt.Printf("job state: %s\n", job.State)

		switch job.State {
		case core.JOB_STAT_COMPLETED:
			logger.Debug(1, "(Wait_for_results) job state: %s", core.JOB_STAT_COMPLETED)

			//root = job.Root

			break FORLOOP
		case core.JOB_STAT_SUSPEND:
			errorMessageStr := ""

			if job.Error != nil {

				errorMessage, _ := json.Marshal(job.Error)
				errorMessageStr = string(errorMessage[:])
			}

			err = fmt.Errorf("(Wait_for_results) job is in state \"%s\" (error: %s)", job.State, errorMessageStr)
			return
		case core.JOB_STAT_FAILED_PERMANENT:
			err = fmt.Errorf("(Wait_for_results) job is in state \"%s\"", job.State)
			return
		case core.JOB_STAT_DELETED:
			err = fmt.Errorf("(Wait_for_results) job is in state \"%s\"", job.State)
			return
		}
	}
	//spew.Dump(job)

	//_, err = job.Init()
	//if err != nil {
	//return
	//}

	// example: curl http://skyport.local:8001/awe/api/workflow_instances/c1cad21a-5ab5-4015-8aae-1dfda844e559_root

	var wi *core.WorkflowInstance
	wi, statusCode, err = GetRootWorkflowInstance(job, aweAuth)

	//panic("Implement getting for output")
	//var wi *core.WorkflowInstance
	// var ok bool
	// wi, ok, err = job.GetWorkflowInstance("_root", false)
	if err != nil {
		err = fmt.Errorf("(Wait_for_results) GetWorkflowInstance returned: %s", err.Error())
		return
	}

	// if !ok {
	// 	err = fmt.Errorf("(Wait_for_results) WorkflowInstance not found")
	// 	return
	// }

	//spew.Dump(wi.Outputs)

	outputReceipt := map[string]interface{}{}
	for _, out := range wi.Outputs {

		outID := strings.TrimPrefix(out.Id, job.Entrypoint+"/")

		outputReceipt[outID] = out.Value
	}

	if conf.SUBMITTER_DOWNLOAD_FILES { // TODO
		var outputFilePath string
		outputFilePath, err = os.Getwd()

		_, err = cache.ProcessIOData(outputReceipt, outputFilePath, outputFilePath, "download", nil, true)
		if err != nil {
			//spew.Dump(outputReceipt)
			err = fmt.Errorf("(Wait_for_results) ProcessIOData(for download) returned: %s", err.Error())
			return
		}
	}

	var outputReceiptBytes []byte
	outputReceiptBytes, err = json.MarshalIndent(outputReceipt, "", "    ")
	if err != nil {
		if err != nil {
			err = fmt.Errorf("(Wait_for_results) json.MarshalIndent returned: %s", err.Error())
			return
		}
	}
	logger.Debug(3, string(outputReceiptBytes[:]))

	if conf.SUBMITTER_OUTPUT != "" {
		err = ioutil.WriteFile(conf.SUBMITTER_OUTPUT, outputReceiptBytes, 0644)
		if err != nil {
			err = fmt.Errorf("(Wait_for_results) ioutil.WriteFile returned: %s", err.Error())
			return
		}
	} else {
		fmt.Println(string(outputReceiptBytes[:]))
	}
	return
}

// SubmitCWLJobToAWE _
func SubmitCWLJobToAWE(workflowFile string, jobFile string, entrypoint string, jobData *[]byte, aweAuth string, shockAuth string) (jobid string, newEntrypoint string, err error) {
	multipart := core.NewMultipartWriter()

	err = multipart.AddFile("cwl", workflowFile)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) multipart.AddFile returned: %s (workflowFile=%s)", err.Error(), workflowFile)
		return
	}

	if jobFile != "" {
		err = multipart.AddDataAsFile("job", jobFile, jobData)
		if err != nil {
			err = fmt.Errorf("(SubmitCWLJobToAWE) AddDataAsFile returned: %s", err.Error())
			return
		}
	}
	//fmt.Fprintf(os.Stderr, "CLIENT_GROUP: %s\n", conf.CLIENT_GROUP)
	err = multipart.AddForm("CLIENT_GROUP", conf.CLIENT_GROUP)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) AddForm returned: %s", err.Error())
		return
	}

	logger.Debug(3, "(SubmitCWLJobToAWE) entrypoint: %s", entrypoint)

	err = multipart.AddForm("entrypoint", entrypoint)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) AddForm returned: %s", err.Error())
		return
	}

	header := make(map[string][]string)
	if aweAuth != "" {
		header["Authorization"] = []string{aweAuth}
	}
	if shockAuth != "" {
		header["Datatoken"] = []string{shockAuth}
	}

	response, err := multipart.Send("POST", conf.SERVER_URL+"/job", header)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) multipart.Send returned: %s", err.Error())
		return
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) ioutil.ReadAll returned: %s", err.Error())
		return
	}

	//responseString := string(responseData)
	//fmt.Println(responseString)

	var sr standardResponse
	err = json.Unmarshal(responseData, &sr)
	if err != nil {
		//fmt.Println(string(responseData[:]))
		err = fmt.Errorf("(SubmitCWLJobToAWE) json.Unmarshal returned: %s (%s) response: %s (response.StatusCode: %d)", err.Error(), conf.SERVER_URL+"/job", responseData, response.StatusCode)
		return
	}

	if len(sr.Error) > 0 {
		err = fmt.Errorf("(SubmitCWLJobToAWE) Response from AWE server contained error: %s", sr.Error[0])
		return
	}

	var jobBytes []byte
	jobBytes, err = json.Marshal(sr.Data)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) json.Marshal returned: %s", err.Error())
		return
	}

	var job core.Job
	err = json.Unmarshal(jobBytes, &job)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) json.Unmarshal returned: %s (%s) (job_bytes: %s)", err.Error(), conf.SERVER_URL+"/job", jobBytes)
		return
	}
	jobid = job.ID

	newEntrypoint = job.Entrypoint

	return

}

// GetAWEObject _
func GetAWEObject(resource string, objectid string, aweAuth string, result interface{}) (statusCode int, err error) {
	statusCode = -1
	if objectid == "" {
		err = fmt.Errorf("(GetAWEObject) objectid empty")
		return
	}

	multipart := core.NewMultipartWriter()

	header := make(map[string][]string)
	if aweAuth != "" {
		header["Authorization"] = []string{aweAuth}
	}

	getURL := fmt.Sprintf("%s/%s/%s", conf.SERVER_URL, resource, objectid)
	response, err := multipart.Send("GET", getURL, header)
	if err != nil {
		err = fmt.Errorf("(GetAWEObject) multipart.Send returned: %s (url was %s)", err.Error(), getURL)
		return
	}

	statusCode = response.StatusCode

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("(GetAWEObject) ioutil.ReadAll returned: %s", err.Error())
		return
	}

	//responseString := string(responseData)

	//fmt.Println(responseString)

	var sr standardResponse
	err = json.Unmarshal(responseData, &sr)
	if err != nil {

		r := strings.NewReplacer("\n", " ", "\r", " ") // remove newlines

		extension := ""
		prefixlen := 100
		if len(responseData) < 100 {
			prefixlen = len(responseData)
			extension = "..."
		}

		bodyPrefix := r.Replace(string(responseData[0:prefixlen])) + extension // this helps debugging

		err = fmt.Errorf("(GetAWEObject) json.Unmarshal returned: %s (%s) (response.StatusCode: %d) (body: \"%s\")", err.Error(), getURL, statusCode, bodyPrefix)
		return
	}

	if len(sr.Error) > 0 {
		err = fmt.Errorf("(GetAWEObject) AWE server returned error: %s (url was: %s)", sr.Error[0], getURL)
		return
	}

	if statusCode != 200 {
		err = fmt.Errorf("(GetAWEObject) response.StatusCode: %d", statusCode)
		return
	}

	var resultBytes []byte
	resultBytes, err = json.Marshal(sr.Data)
	if err != nil {
		err = fmt.Errorf("(GetAWEObject) json.Marshal returned: %s", err.Error())
		return
	}

	//var job core.Job
	//job = &core.Job{}
	err = json.Unmarshal(resultBytes, result)
	if err != nil {
		fmt.Printf("result_bytes: %s\n", resultBytes)
		err = fmt.Errorf("(GetAWEObject) (second call) json.Unmarshal returned: %s (%s)", err.Error(), getURL)
		return
	}

	return
}

// GetAWEJob _
func GetAWEJob(jobid string, aweAuth string) (job *core.Job, statusCode int, err error) {

	job = &core.Job{}
	statusCode, err = GetAWEObject("job", jobid, aweAuth, job)
	if err != nil {
		err = fmt.Errorf("(GetAWEJob) GetAWEObject returned: %s", err.Error())
	}
	return
}

// GetRootWorkflowInstance _
func GetRootWorkflowInstance(job *core.Job, aweAuth string) (wi *core.WorkflowInstance, statusCode int, err error) {
	//wi_array := []core.WorkflowInstance{}
	var wiIf interface{}

	queryStr := job.Root

	statusCode, err = GetAWEObject("workflow_instances", queryStr, aweAuth, &wiIf)
	if err != nil {
		err = fmt.Errorf("(GetRootWorkflowInstance) GetAWEObject returned: %s (queryStr: %s)", err.Error(), queryStr)
		return
	}

	wi, err = core.NewWorkflowInstanceFromInterface(wiIf, job, nil, false)
	if err != nil {
		err = fmt.Errorf("(GetRootWorkflowInstance) NewWorkflowInstanceFromInterface returned: %s", err.Error())
		return
	}

	return
}
