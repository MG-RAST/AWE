package main

import (
	//"encoding/json"
	"fmt"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	shock "github.com/MG-RAST/go-shock-client"
	"github.com/davecgh/go-spew/spew"
	//"github.com/MG-RAST/AWE/lib/logger/event"
	"bytes"
	"encoding/json"

	"github.com/MG-RAST/AWE/lib/cache"
	//"github.com/davecgh/go-spew/spew"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
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
	err := main_wrapper()
	if err != nil {
		fmt.Fprintf(os.Stderr, "\nerror: %s\n\n", err.Error())
		time.Sleep(time.Second)
		os.Exit(1)
	}
	time.Sleep(time.Second)
	os.Exit(0)
}

func main_wrapper() (err error) {

	conf.LOG_OUTPUT = "console"

	err = conf.Init_conf("submitter")

	if err != nil {
		err = fmt.Errorf("error reading conf file: %s", err.Error())
		return
	}

	logger.Initialize("client")

	awe_auth := os.Getenv("AWE_AUTH")
	shock_auth := os.Getenv("SHOCK_AUTH")

	if awe_auth != "" {
		awe_auth_array := strings.SplitN(awe_auth, " ", 2)
		if len(awe_auth_array) != 2 {
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

	workflow_file := conf.ARGS[0]
	job_file := conf.ARGS[1]

	inputfile_path := path.Dir(job_file)
	//fmt.Printf("job path: %s\n", inputfile_path) // needed to resolve relative paths

	// ### parse job file
	var job_doc *cwl.Job_document
	job_doc, err = cwl.ParseJobFile(job_file)
	if err != nil {
		err = fmt.Errorf("error parsing cwl job: %s", err.Error())
		return
	}

	//fmt.Println("Job input after reading from file:")
	//spew.Dump(*job_doc)

	job_doc_map := job_doc.GetMap()
	//job_doc_map["test"] = cwl.NewNull()

	//fmt.Println("Job input after reading from file: map !!!!\n")
	//spew.Dump(job_doc_map)

	var data []byte
	data, err = yaml.Marshal(job_doc_map)
	if err != nil {
		return
	}

	job_doc_string := string(data[:])
	//fmt.Printf("job_doc_string:\n \"%s\"\n", job_doc_string)
	if job_doc_string == "" {
		err = fmt.Errorf("job_doc_string is empty")
		return
	}

	//fmt.Printf("yaml:\n%s\n", job_doc_string)

	// ### upload input files

	shock_client := shock.NewShockClient(conf.SHOCK_URL, shock_auth, false)

	var upload_count int
	upload_count, err = cache.ProcessIOData(job_doc, inputfile_path, inputfile_path, "upload", shock_client)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) ProcessIOData(for upload) returned: %s", err.Error())
		return
	}
	logger.Debug(3, "%d files have been uploaded\n", upload_count)
	//time.Sleep(2)

	//spew.Dump(*job_doc)
	job_doc_map = job_doc.GetMap()
	//fmt.Println("------------Job input after parsing:")
	data, err = yaml.Marshal(job_doc_map)
	if err != nil {
		return
	}

	if conf.DEBUG_LEVEL >= 3 {
		fmt.Printf("job input as yaml:\n%s\n", string(data[:]))
	}
	var yamlstream []byte
	// read and pack workfow
	if conf.SUBMITTER_PACK {

		yamlstream, err = exec.Command("cwltool", "--pack", workflow_file).Output()
		if err != nil {
			err = fmt.Errorf("(main_wrapper) exec.Command returned: %s (%s %s %s)", err.Error(), "cwltool", "--pack", workflow_file)
			return
		}

	} else {

		yamlstream, err = ioutil.ReadFile(workflow_file)
		if err != nil {
			err = fmt.Errorf("error in reading workflow file: " + err.Error())
			return
		}
	}
	// ### PARSE (maybe PACKED) WORKFLOW DOCUMENT, in case default files have to be uploaded

	// convert CWL to string
	yaml_str := string(yamlstream[:])
	//fmt.Printf("after cwltool --pack: \n%s\n", yaml_str)
	var named_object_array []cwl.Named_CWL_object
	var cwl_version cwl.CWLVersion
	var schemata []cwl.CWLType_Type
	var namespaces map[string]string
	var schemas []interface{}
	named_object_array, cwl_version, schemata, namespaces, schemas, err = cwl.Parse_cwl_document(yaml_str)

	if err != nil {
		err = fmt.Errorf("(main_wrapper) error in parsing cwl workflow yaml file: " + err.Error())
		return
	}

	_ = schemata // TODO put into a collection!

	// A) search for File objects in Document, e.g. in CommandLineTools

	sub_upload_count := 0
	sub_upload_count, err = cache.ProcessIOData(named_object_array, inputfile_path, inputfile_path, "upload", shock_client)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) ProcessIOData(for upload) returned: %s", err.Error())
		return
	}
	upload_count += sub_upload_count

	if schemas != nil {
		sub_upload_count := 0
		sub_upload_count, err = cache.ProcessIOData(schemas, inputfile_path, inputfile_path, "upload", shock_client)
		if err != nil {
			err = fmt.Errorf("(main_wrapper) ProcessIOData(for upload) returned: %s", err.Error())
			return
		}
		upload_count += sub_upload_count
	}
	// for j, _ := range named_object_array {

	// 	pair := named_object_array[j]
	// 	object := pair.Value

	// 	var ok bool

	// 	switch object.(type) {
	// 	case *cwl.Workflow:
	// 		workflow := object.(*cwl.Workflow)
	// 		//if cwl_version != "" {
	// 		//	workflow.CwlVersion = cwl_version
	// 		//}
	// 		sub_upload_count := 0
	// 		sub_upload_count, err = cache.ProcessIOData(workflow, inputfile_path, inputfile_path, "upload", shock_client)
	// 		if err != nil {
	// 			err = fmt.Errorf("(main_wrapper) ProcessIOData(for upload) returned: %s", err.Error())
	// 			return
	// 		}
	// 		upload_count += sub_upload_count

	// 	case *cwl.CommandLineTool:
	// 		var cmd_line_tool *cwl.CommandLineTool
	// 		cmd_line_tool = object.(*cwl.CommandLineTool)

	// 		//if cwl_version != "" {
	// 		//	cmd_line_tool.CwlVersion = cwl_version
	// 		//}
	// 		if cmd_line_tool == nil {
	// 			err = fmt.Errorf("(main_wrapper) cmd_line_tool==nil")
	// 			return
	// 		}
	// 		sub_upload_count := 0
	// 		sub_upload_count, err = cache.ProcessIOData(cmd_line_tool, inputfile_path, inputfile_path, "upload", shock_client)
	// 		if err != nil {
	// 			err = fmt.Errorf("(main_wrapper) ProcessIOData(for upload) returned: %s", err.Error())
	// 			return
	// 		}
	// 		upload_count += sub_upload_count

	// 	case *cwl.ExpressionTool:
	// 		var express_tool *cwl.ExpressionTool
	// 		express_tool, ok = object.(*cwl.ExpressionTool) // TODO this misses embedded ExpressionTools !
	// 		if !ok {
	// 			//fmt.Println("nope.")
	// 			err = nil
	// 			continue
	// 		}

	// 		if express_tool == nil {
	// 			err = fmt.Errorf("(main_wrapper) express_tool==nil")
	// 			return
	// 		}
	// 		//if cwl_version != "" {
	// 		//	express_tool.CwlVersion = cwl_version
	// 		//}

	// 		sub_upload_count := 0
	// 		sub_upload_count, err = cache.ProcessIOData(express_tool, inputfile_path, inputfile_path, "upload", shock_client)
	// 		if err != nil {
	// 			err = fmt.Errorf("(main_wrapper) ProcessIOData(for upload) returned: %s", err.Error())
	// 			return
	// 		}
	// 		upload_count += sub_upload_count

	// 	}
	// }

	logger.Debug(3, "%d files have been uploaded\n", upload_count)

	var shock_requirement cwl.ShockRequirement
	var shock_requirement_ptr *cwl.ShockRequirement
	shock_requirement_ptr, err = cwl.NewShockRequirement(shock_client.Host)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) NewShockRequirement returned: %s", err.Error())
		return
	}

	shock_requirement = *shock_requirement_ptr

	// B) inject ShockRequirement into CommandLineTools, ExpressionTools and Workflow
	for j, _ := range named_object_array {

		pair := named_object_array[j]
		object := pair.Value

		var ok bool

		switch object.(type) {
		case *cwl.Workflow:
			workflow := object.(*cwl.Workflow)

			workflow.Requirements, err = cwl.AddRequirement(shock_requirement, workflow.Requirements)
			if err != nil {
				err = fmt.Errorf("(main_wrapper) AddRequirement returned: %s", err.Error())
				return
			}

		case *cwl.CommandLineTool:
			var cmd_line_tool *cwl.CommandLineTool
			cmd_line_tool, ok = object.(*cwl.CommandLineTool) // TODO this misses embedded CommandLineTools !
			if !ok {
				//fmt.Println("nope.")
				err = nil
				continue
			}

			cmd_line_tool.Requirements, err = cwl.AddRequirement(shock_requirement, cmd_line_tool.Requirements)
			if err != nil {
				err = fmt.Errorf("(main_wrapper) AddRequirement returned: %s", err.Error())
			}

		case *cwl.ExpressionTool:
			var express_tool *cwl.ExpressionTool
			express_tool, ok = object.(*cwl.ExpressionTool) // TODO this misses embedded ExpressionTools !
			if !ok {
				//fmt.Println("nope.")
				err = nil
				continue
			}

			if express_tool == nil {
				err = fmt.Errorf("(main_wrapper) express_tool==nil")
				return
			}

			express_tool.Requirements, err = cwl.AddRequirement(shock_requirement, express_tool.Requirements)
			if err != nil {
				err = fmt.Errorf("(main_wrapper) AddRequirement returned: %s", err.Error())
			}

		}
	}

	// create temporary workflow document file

	new_document := cwl.CWL_document_generic{}
	new_document.CwlVersion = cwl_version
	new_document.Namespaces = namespaces
	new_document.Schemas = schemas
	for i, _ := range named_object_array {
		pair := named_object_array[i]
		object := pair.Value
		new_document.Graph = append(new_document.Graph, object)
	}

	var new_document_bytes []byte
	new_document_bytes, err = yaml.Marshal(new_document)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) yaml.Marshal returned: %s", err.Error())
		return
	}
	new_document_str := string(new_document_bytes[:])
	graph_pos := strings.Index(new_document_str, "\ngraph:")

	//new_document_str = strings.Replace(new_document_str, "\nnamespaces", "\n$namespaces", -1) // remove dollar sign

	if graph_pos != -1 {
		new_document_str = strings.Replace(new_document_str, "\ngraph", "\n$graph", -1) // remove dollar sign
	} else {
		err = fmt.Errorf("(main_wrapper) keyword graph not found")
		return
	}

	if conf.DEBUG_LEVEL >= 3 {
		fmt.Println("------------ new_document_str:")
		fmt.Println(new_document_str)
		fmt.Println("------------")
		//panic("hhhh")
	}
	new_document_bytes = []byte(new_document_str)

	// ### Write workflow to file, so we can run "cwltool --pack""
	var tmpfile *os.File
	tmpfile, err = ioutil.TempFile(os.TempDir(), "awe-submitter_")
	if err != nil {
		err = fmt.Errorf("(main_wrapper) ioutil.TempFile returned: %s", err.Error())
		return
	}
	tempfile_name := tmpfile.Name()
	//defer os.Remove(tempfile_name)

	_, err = tmpfile.Write(new_document_bytes)
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
	jobid, err = SubmitCWLJobToAWE(tempfile_name, job_file, &data, awe_auth, shock_auth)
	if err != nil {
		err = fmt.Errorf("(main_wrapper) SubmitCWLJobToAWE returned: %s", err.Error())
		return
	}

	//fmt.Printf("Job id: %s\n", jobid)

	if conf.SUBMITTER_WAIT {
		var job *core.Job

		// ***** Wait for job to complete

	FORLOOP:
		for true {
			time.Sleep(5 * time.Second)
			job = nil

			job, err = GetAWEJob(jobid, awe_auth)
			if err != nil {
				return
			}

			//fmt.Printf("job state: %s\n", job.State)

			switch job.State {
			case core.JOB_STAT_COMPLETED:
				break FORLOOP
			case core.JOB_STAT_SUSPEND:
				error_msg_str := ""

				if job.Error != nil {

					error_msg, _ := json.Marshal(job.Error)
					error_msg_str = string(error_msg[:])
				}

				err = fmt.Errorf("(main_wrapper) job is in state \"%s\" (error: %s)", job.State, error_msg_str)
				return
			case core.JOB_STAT_FAILED_PERMANENT:
				err = fmt.Errorf("(main_wrapper) job is in state \"%s\"", job.State)
				return
			case core.JOB_STAT_DELETED:
				err = fmt.Errorf("(main_wrapper) job is in state \"%s\"", job.State)
				return
			}
		}
		//spew.Dump(job)

		_, err = job.Init(cwl_version, namespaces)
		if err != nil {
			return
		}
		var wi *core.WorkflowInstance
		wi, err = job.GetWorkflowInstance("", false)
		if err != nil {
			err = fmt.Errorf("(main_wrapper) GetWorkflowInstance returned: %s", err.Error())
			return
		}
		//spew.Dump(wi.Outputs)

		output_receipt := map[string]interface{}{}
		for _, out := range wi.Outputs {

			out_id := strings.TrimPrefix(out.Id, job.Entrypoint+"/")

			output_receipt[out_id] = out.Value
		}

		if conf.SUBMITTER_DOWNLOAD_FILES { // TODO
			var output_file_path string
			output_file_path, err = os.Getwd()

			_, err = cache.ProcessIOData(output_receipt, output_file_path, output_file_path, "download", nil)
			if err != nil {
				spew.Dump(output_receipt)
				err = fmt.Errorf("(main_wrapper) ProcessIOData(for download) returned: %s", err.Error())
				return
			}
		}

		var output_receipt_bytes []byte
		output_receipt_bytes, err = json.MarshalIndent(output_receipt, "", "    ")
		if err != nil {
			if err != nil {
				err = fmt.Errorf("(main_wrapper) json.MarshalIndent returned: %s", err.Error())
				return
			}
		}
		logger.Debug(3, string(output_receipt_bytes[:]))

		if conf.SUBMITTER_OUTPUT != "" {
			err = ioutil.WriteFile(conf.SUBMITTER_OUTPUT, output_receipt_bytes, 0644)
			if err != nil {
				err = fmt.Errorf("(main_wrapper) ioutil.WriteFile returned: %s", err.Error())
				return
			}
		} else {
			fmt.Println(string(output_receipt_bytes[:]))
		}

	} else {
		fmt.Printf("JobID=%s\n", jobid)
	}
	return
}

func SubmitCWLJobToAWE(workflow_file string, job_file string, data *[]byte, awe_auth string, shock_auth string) (jobid string, err error) {
	multipart := NewMultipartWriter()

	err = multipart.AddFile("cwl", workflow_file)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) multipart.AddFile returned: %s", err.Error())
		return
	}

	err = multipart.AddDataAsFile("job", job_file, data)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) AddDataAsFile returned: %s", err.Error())
		return
	}

	header := make(map[string][]string)
	if awe_auth != "" {
		header["Authorization"] = []string{awe_auth}
	}
	if shock_auth != "" {
		header["Datatoken"] = []string{shock_auth}
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

	var job_bytes []byte
	job_bytes, err = json.Marshal(sr.Data)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) json.Marshal returned: %s", err.Error())
		return
	}

	var job core.Job
	err = json.Unmarshal(job_bytes, &job)
	if err != nil {
		err = fmt.Errorf("(SubmitCWLJobToAWE) json.Unmarshal returned: %s (%s) (job_bytes: %s)", err.Error(), conf.SERVER_URL+"/job", job_bytes)
		return
	}
	jobid = job.Id

	return

}

func GetAWEJob(jobid string, awe_auth string) (job *core.Job, err error) {

	if jobid == "" {
		err = fmt.Errorf("(GetAWEJob) jobid empty")
		return
	}

	multipart := NewMultipartWriter()

	header := make(map[string][]string)
	if awe_auth != "" {
		header["Authorization"] = []string{awe_auth}
	}

	response, err := multipart.Send("GET", conf.SERVER_URL+"/job/"+jobid, header)
	if err != nil {
		err = fmt.Errorf("(GetAWEJob) multipart.Send returned: %s", err.Error())
		return
	}

	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		err = fmt.Errorf("(GetAWEJob) ioutil.ReadAll returned: %s", err.Error())
		return
	}

	//responseString := string(responseData)

	//fmt.Println(responseString)

	var sr standardResponse
	err = json.Unmarshal(responseData, &sr)
	if err != nil {
		err = fmt.Errorf("(GetAWEJob) json.Unmarshal returned: %s (%s) (response.StatusCode: %d)", err.Error(), conf.SERVER_URL+"/job/"+jobid, response.StatusCode)
		return
	}

	if len(sr.Error) > 0 {
		err = fmt.Errorf("%s", sr.Error[0])
		return
	}

	if response.StatusCode != 200 {
		err = fmt.Errorf("(GetAWEJob) response.StatusCode: %d", response.StatusCode)
		return
	}

	var job_bytes []byte
	job_bytes, err = json.Marshal(sr.Data)
	if err != nil {
		return
	}

	//var job core.Job
	job = &core.Job{}
	err = json.Unmarshal(job_bytes, job)
	if err != nil {
		fmt.Printf("job_bytes: %s\n", job_bytes)
		err = fmt.Errorf("(GetAWEJob) (second call) json.Unmarshal returned: %s (%s)", err.Error(), conf.SERVER_URL+"/job/"+jobid)
		return
	}

	return
}

type MultipartWriter struct {
	b bytes.Buffer
	w *multipart.Writer
}

func NewMultipartWriter() *MultipartWriter {
	m := &MultipartWriter{}
	m.w = multipart.NewWriter(&m.b)
	return m
}

func (m *MultipartWriter) Send(method string, url string, header map[string][]string) (response *http.Response, err error) {
	m.w.Close()
	//fmt.Println("------------")
	//spew.Dump(m.w)
	//fmt.Println("------------")

	req, err := http.NewRequest(method, url, &m.b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", m.w.FormDataContentType())

	for key := range header {
		header_array := header[key]
		for _, value := range header_array {
			req.Header.Add(key, value)
		}

	}

	// Submit the request
	client := &http.Client{}
	//fmt.Printf("%s %s\n\n", method, url)
	response, err = client.Do(req)
	if err != nil {
		return
	}

	// Check the response
	//if response.StatusCode != http.StatusOK {
	//	err = fmt.Errorf("bad status: %s", response.Status)
	//}
	return

}

func (m *MultipartWriter) AddDataAsFile(fieldname string, filepath string, data *[]byte) (err error) {

	fw, err := m.w.CreateFormFile(fieldname, filepath)
	if err != nil {
		return
	}
	_, err = fw.Write(*data)
	if err != nil {
		return
	}
	return
}

func (m *MultipartWriter) AddFile(fieldname string, filepath string) (err error) {

	f, err := os.Open(filepath)
	if err != nil {
		return
	}
	defer f.Close()
	fw, err := m.w.CreateFormFile(fieldname, filepath)
	if err != nil {
		return
	}
	if _, err = io.Copy(fw, f); err != nil {
		return
	}

	return
}
