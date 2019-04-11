package main

import (
	//"encoding/json"
	"fmt"
	"net/url"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	shock "github.com/MG-RAST/go-shock-client"
	"github.com/davecgh/go-spew/spew"

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

	if len(conf.ARGS) < 2 {
		err = fmt.Errorf("not enough arguments, workflow file and job file are required")
		return
	}

	workflowFile := conf.ARGS[0]
	jobFile := ""

	inputfilePath := ""
	//fmt.Printf("job path: %s\n", inputfile_path) // needed to resolve relative paths

	// ### parse job file
	var jobDoc *cwl.Job_document

	if len(conf.ARGS) >= 2 {
		jobFile = conf.ARGS[1]

		inputfilePath = path.Dir(jobFile)

		jobDoc, err = cwl.ParseJobFile(jobFile)
		if err != nil {
			err = fmt.Errorf("error parsing cwl job: %s", err.Error())
			return
		}
	} else {
		jobDoc = &cwl.Job_document{}
		inputfilePath = path.Dir(workflowFile)
	}
	//fmt.Println("Job input after reading from file:")
	//spew.Dump(*job_doc)

	jobDocMap := jobDoc.GetMap()
	//job_doc_map["test"] = cwl.NewNull()

	//fmt.Println("Job input after reading from file: map !!!!\n")
	//spew.Dump(job_doc_map)

	var data []byte
	data, err = yaml.Marshal(jobDocMap)
	if err != nil {
		return
	}

	jobDocString := string(data[:])
	//fmt.Printf("job_doc_string:\n \"%s\"\n", job_doc_string)
	if jobDocString == "" {
		err = fmt.Errorf("job_doc_string is empty")
		return
	}

	//fmt.Printf("yaml:\n%s\n", job_doc_string)

	// ### upload input files

	shockClient := shock.NewShockClient(conf.SHOCK_URL, shockAuth, false)

	uploadCount := 0

	if jobFile != "" {
		uploadCount, err = cache.ProcessIOData(jobDoc, inputfilePath, inputfilePath, "upload", shockClient)
		if err != nil {
			err = fmt.Errorf("(main_wrapper) ProcessIOData(for upload) returned: %s", err.Error())
			return
		}
	}
	logger.Debug(3, "%d files have been uploaded\n", uploadCount)
	//time.Sleep(2)

	//spew.Dump(*job_doc)
	jobDocMap = jobDoc.GetMap()
	//fmt.Println("------------Job input after parsing:")
	data, err = yaml.Marshal(jobDocMap)
	if err != nil {
		return
	}

	if conf.DEBUG_LEVEL >= 3 {
		fmt.Printf("job input as yaml:\n%s\n", string(data[:]))
	}
	var yamlstream []byte
	// read and pack workfow
	if conf.SUBMITTER_PACK {

		yamlstream, err = exec.Command("cwltool", "--pack", workflowFile).Output()
		if err != nil {
			err = fmt.Errorf("(main_wrapper) exec.Command returned: %s (%s %s %s)", err.Error(), "cwltool", "--pack", workflowFile)
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

	namedObjectArray, schemata, context, _, err = cwl.Parse_cwl_document(yamlStr, inputfilePath)

	if err != nil {
		err = fmt.Errorf("(main_wrapper) error in parsing cwl workflow yaml file: " + err.Error())
		return
	}

	_ = schemata // TODO put into a collection!

	// A) search for File objects in Document, e.g. in CommandLineTools

	subUploadCount := 0
	subUploadCount, err = cache.ProcessIOData(namedObjectArray, inputfilePath, inputfilePath, "upload", shockClient)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) ProcessIOData(for upload) returned: %s", err.Error())
		return
	}
	uploadCount += subUploadCount

	if context.Schemas != nil {
		subUploadCount := 0
		subUploadCount, err = cache.ProcessIOData(context.Schemas, inputfilePath, inputfilePath, "upload", shockClient)
		if err != nil {
			err = fmt.Errorf("(main_wrapper) ProcessIOData(for upload) returned: %s", err.Error())
			return
		}
		uploadCount += subUploadCount

	}

	logger.Debug(3, "%d files have been uploaded\n", uploadCount)

	var shockRequirement cwl.ShockRequirement
	var shockRequirementPtr *cwl.ShockRequirement
	shockRequirementPtr, err = cwl.NewShockRequirement(shockClient.Host)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) NewShockRequirement returned: %s", err.Error())
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
				err = fmt.Errorf("(main_wrapper) AddRequirement returned: %s", err.Error())
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
				err = fmt.Errorf("(main_wrapper) AddRequirement returned: %s", err.Error())
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
				err = fmt.Errorf("(main_wrapper) express_tool==nil")
				return
			}

			expressTool.Requirements, err = cwl.AddRequirement(shockRequirement, expressTool.Requirements)
			if err != nil {
				err = fmt.Errorf("(main_wrapper) AddRequirement returned: %s", err.Error())
			}

		}
	}

	// create temporary workflow document file

	newDocument := context.CWL_document

	//new_document := cwl.CWL_document{}
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
		err = fmt.Errorf("(main_wrapper) len(new_document.Graph) == 0")
		return
	}

	var newDocumentBytes []byte
	newDocumentBytes, err = yaml.Marshal(newDocument)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) yaml.Marshal returned: %s", err.Error())
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
	tempfileName := tmpfile.Name()
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

	// ### Submit job to AWE
	var jobid string

	jobid, err = SubmitCWLJobToAWE(tempfileName, jobFile, &data, aweAuth, shockAuth)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) SubmitCWLJobToAWE returned: %s", err.Error())
		return
	}

	//fmt.Printf("Job id: %s\n", jobid)

	if conf.SUBMITTER_WAIT {
		err = WaitForResults(jobid, aweAuth)
		if err != nil {
			err = fmt.Errorf("(main_wrapper) Wait_for_results returned: %s", err.Error())
			return
		}
	} else {
		fmt.Printf("JobID=%s\n", jobid)
	}
	return
}

// WaitForResults _
func WaitForResults(jobid string, aweAuth string) (err error) {

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
	wi, statusCode, err = GetRootWorkflowInstance(jobid, job, aweAuth)

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

		_, err = cache.ProcessIOData(outputReceipt, outputFilePath, outputFilePath, "download", nil)
		if err != nil {
			spew.Dump(outputReceipt)
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
func SubmitCWLJobToAWE(workflowFile string, jobFile string, data *[]byte, aweAuth string, shockAuth string) (jobid string, err error) {
	multipart := core.NewMultipartWriter()

	err = multipart.AddFile("cwl", workflowFile)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) multipart.AddFile returned: %s", err.Error())
		return
	}

	if jobFile != "" {
		err = multipart.AddDataAsFile("job", jobFile, data)
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
func GetRootWorkflowInstance(jobid string, job *core.Job, aweAuth string) (wi *core.WorkflowInstance, statusCode int, err error) {
	//wi_array := []core.WorkflowInstance{}
	var wiIf interface{}

	statusCode, err = GetAWEObject("workflow_instances", jobid+"_"+url.PathEscape("#main"), aweAuth, &wiIf)
	if err != nil {
		err = fmt.Errorf("(GetRootWorkflowInstance) GetAWEObject returned: %s", err.Error())
		return
	}

	wi, err = core.NewWorkflowInstanceFromInterface(wiIf, job, nil, false)
	if err != nil {
		err = fmt.Errorf("(GetRootWorkflowInstance) NewWorkflowInstanceFromInterface returned: %s", err.Error())
		return
	}

	return
}
