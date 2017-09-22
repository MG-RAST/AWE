package main

import (
	//"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core/cwl"

	"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/MG-RAST/AWE/lib/logger/event"
	"bytes"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/davecgh/go-spew/spew"
	"gopkg.in/yaml.v2"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	"net/url"
	"os"
	"path"
	"reflect"
	"time"
)

func uploadFile(file *cwl.File, inputfile_path string) (err error) {
	fmt.Printf("(uploadFile) start\n")
	defer fmt.Printf("(uploadFile) end\n")
	//if err := core.PutFileToShock(file_path, io.Host, io.Node, work.Rank, work.Info.DataToken, attrfile_path, io.Type, io.FormOptions, io.NodeAttr); err != nil {

	//	time.Sleep(3 * time.Second) //wait for 3 seconds and try again
	//	if err := core.PutFileToShock(file_path, io.Host, io.Node, work.Rank, work.Info.DataToken, attrfile_path, io.Type, io.FormOptions, io.NodeAttr); err != nil {
	//		fmt.Errorf("push file error\n")
	//		logger.Error("op=pushfile,err=" + err.Error())
	//		return size, err
	//	}
	//}

	scheme := ""
	if file.Location_url != nil {
		scheme = file.Location_url.Scheme
		host := file.Location_url.Host
		path := file.Location_url.Path
		fmt.Printf("scheme: %s", scheme)

		if scheme == "file" {
			if host == "" || host == "localhost" {
				file.Path = path
			}
		} else {
			return
		}

	}

	if file.Location_url == nil && file.Location != "" {
		err = fmt.Errorf("URL has not been parsed correctly")
		return
	}

	file_path := file.Path

	basename := path.Base(file_path)

	if file_path == "" {
		return
	}

	fmt.Printf("file.Path: %s\n", file_path)

	if !path.IsAbs(file_path) {
		file_path = path.Join(inputfile_path, file_path)
	}

	fmt.Printf("Using path %s\n", file_path)

	sc := shock.ShockClient{Host: conf.SHOCK_URL, Token: "", Debug: true} // "shock:7445"

	opts := shock.Opts{"upload_type": "basic", "file": file_path}
	node, err := sc.CreateOrUpdate(opts, "", nil)
	if err != nil {
		return
	}
	spew.Dump(node)

	file.Location_url, err = url.Parse(conf.SHOCK_URL + "/node/" + node.Id + "?download")
	if err != nil {
		return
	}

	file.Location = file.Location_url.String()
	file.Path = ""
	file.Basename = basename

	return
}

func processInputData(native interface{}, inputfile_path string) (count int, err error) {

	fmt.Printf("(processInputData) start\n")
	defer fmt.Printf("(processInputData) end\n")
	switch native.(type) {
	case *cwl.Job_document:
		fmt.Printf("found Job_document\n")
		job_doc_ptr := native.(*cwl.Job_document)

		job_doc := *job_doc_ptr

		for _, value := range job_doc {

			id := value.Id
			fmt.Printf("recurse into key: %s\n", id)
			var sub_count int
			sub_count, err = processInputData(value, inputfile_path)
			if err != nil {
				return
			}
			count += sub_count
		}

		return
	case cwl.NamedCWLType:
		named := native.(cwl.NamedCWLType)
		var sub_count int
		sub_count, err = processInputData(named.Value, inputfile_path)
		if err != nil {
			return
		}
		count += sub_count

	case *cwl.String:
		fmt.Printf("found string\n")
		return
	case *cwl.File:

		fmt.Printf("found File\n")
		file, ok := native.(*cwl.File)
		if !ok {
			err = fmt.Errorf("could not cast to *cwl.File")
			return
		}
		err = uploadFile(file, inputfile_path)
		if err != nil {
			return
		}
		count += 1
		return
	case *cwl.Array:

		array, ok := native.(*cwl.Array)
		if !ok {
			err = fmt.Errorf("could not cast to *cwl.Array")
			return
		}

		for _, value := range *array {

			id := value.GetId()
			fmt.Printf("recurse into key: %s\n", id)
			var sub_count int
			sub_count, err = processInputData(value, inputfile_path)
			if err != nil {
				return
			}
			count += sub_count

		}
		return
	default:
		spew.Dump(native)
		err = fmt.Errorf("(processInputData) No handler for type \"%s\"\n", reflect.TypeOf(native))
		return
	}

	return
}

func main() {
	err := main_wrapper()
	if err != nil {
		println(err.Error())
		os.Exit(1)
	}
	os.Exit(0)
}

func main_wrapper() (err error) {

	conf.LOG_OUTPUT = "console"

	err = conf.Init_conf("submitter")

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: error reading conf file: "+err.Error())
		os.Exit(1)
	}

	logger.Initialize("client")

	for _, value := range conf.ARGS {
		println(value)
	}

	job_file := conf.ARGS[0]
	workflow_file := conf.ARGS[1]

	inputfile_path := path.Dir(job_file)
	fmt.Printf("job path: %s\n", inputfile_path) // needed to resolve relative paths

	job_doc, err := cwl.ParseJobFile(job_file)
	if err != nil {
		logger.Error("error parsing cwl job: %v", err)
		time.Sleep(time.Second)
		os.Exit(1)
	}

	fmt.Println("Job input after reading from file:")
	spew.Dump(*job_doc)

	job_doc_map := job_doc.GetMap()
	fmt.Println("Job input after reading from file: map !!!!\n")
	spew.Dump(job_doc_map)

	data, err := yaml.Marshal(job_doc_map)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}

	job_doc_string := string(data[:])
	fmt.Printf("job_doc_string:\n \"%s\"\n", job_doc_string)
	if job_doc_string == "" {
		fmt.Println("job_doc_string is empty")
		os.Exit(1)
	}

	fmt.Printf("yaml:\n%s\n", job_doc_string)

	// process input files

	upload_count, err := processInputData(job_doc, inputfile_path)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}
	fmt.Printf("%d files have been uploaded\n", upload_count)
	time.Sleep(2)

	spew.Dump(*job_doc)
	job_doc_map = job_doc.GetMap()
	fmt.Println("------------Job input after parsing:")
	data, err = yaml.Marshal(job_doc_map)
	if err != nil {
		fmt.Printf("error: %s", err.Error())
		os.Exit(1)
	}

	fmt.Printf("yaml:\n%s\n", string(data[:]))

	// job submission example:
	// curl -X POST -F job=@test.yaml -F cwl=@/Users/wolfganggerlach/awe_data/pipeline/CWL/PackedWorkflow/preprocess-fasta.workflow.cwl http://localhost:8001/job

	//var b bytes.Buffer
	//w := multipart.NewWriter(&b)
	err = SubmitCWLJobToAWE(workflow_file, job_file, &data)
	if err != nil {
		return
	}

	return
}

func SubmitCWLJobToAWE(workflow_file string, job_file string, data *[]byte) (err error) {
	multipart := NewMultipartWriter()
	err = multipart.AddFile("cwl", workflow_file)
	if err != nil {
		return
	}
	err = multipart.AddDataAsFile("job", job_file, data)
	if err != nil {
		return
	}
	response, err := multipart.Send("POST", conf.SERVER_URL+"/job")
	if err != nil {
		return
	}
	responseData, err := ioutil.ReadAll(response.Body)
	if err != nil {
		return
	}
	responseString := string(responseData)

	fmt.Println(responseString)
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

func (m *MultipartWriter) Send(method string, url string) (response *http.Response, err error) {
	m.w.Close()
	fmt.Println("------------")
	spew.Dump(m.w)
	fmt.Println("------------")

	req, err := http.NewRequest(method, url, &m.b)
	if err != nil {
		return
	}
	// Don't forget to set the content type, this will contain the boundary.
	req.Header.Set("Content-Type", m.w.FormDataContentType())

	// Submit the request
	client := &http.Client{}
	fmt.Printf("%s %s\n\n", method, url)
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
