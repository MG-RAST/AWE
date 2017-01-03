package taverna

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	"time"
)

type WorkflowRun struct {
	State            string        `bson:"state" json:"state"`
	Invocations      []*Invocation `bson:"invocations" json:"invocations"`
	CreatedDate      time.Time     `bson:"createdDate" json:"createdDate"`
	StartedDate      time.Time     `bson:"startedDate" json:"startedDate"`
	CompletedDate    time.Time     `bson:"completedDate" json:"completedDate"`
	ProcessorReports []*ProcReport `bson:"processorReports" json:"processorReports"`
	Subject          string        `bson:"subject" json:"subject"`
}

type Invocation struct {
	Inputs  map[string]string `bson:"inputs" json:"inputs"`
	Outputs map[string]string `bson:"outputs" json:"outputs"`
	Name    string            `bson:"name" json:"name"`
	Id      string            `bson:"id" json:"id"`
}

type ProcReport struct {
	State         string        `bson:"state" json:"state"`
	Invocations   []*Invocation `bson:"invocations" json:"invocations"`
	CreatedDate   time.Time     `bson:"createdDate" json:"createdDate"`
	StartedDate   time.Time     `bson:"startedDate" json:"startedDate"`
	CompletedDate time.Time     `bson:"completedDate" json:"completedDate"`
	Subject       string        `bson:"subject" json:"subject"`
}

func ExportWorkflowRun(job *core.Job) (wfrun *WorkflowRun, err error) {
	wfrun = new(WorkflowRun)
	wfrun.State = job.State
	wfrun.CreatedDate = job.Info.SubmitTime
	wfrun.StartedDate = job.Info.SubmitTime
	wfrun.CompletedDate = job.UpdateTime
	wfrun.Subject = fmt.Sprintf("%s/job/%s/?export=taverna", conf.API_URL, job.Id)
	job_invocation := new(Invocation)
	job_invocation.Id = job.Id
	job_invocation.Name = job.Info.Pipeline
	job_invocation.Inputs = make(map[string]string)
	job_invocation.Outputs = make(map[string]string)

	for _, task := range job.Tasks {
		report := new(ProcReport)
		report.State = task.GetState()
		invocation := new(Invocation)
		invocation.Id = task.Id
		invocation.Name = task.Cmd.Name
		invocation.Inputs = make(map[string]string)
		invocation.Outputs = make(map[string]string)
		for _, io := range task.Inputs {
			invocation.Inputs[io.FileName] = io.Url
			if !io.Intermediate {
				job_invocation.Inputs[io.FileName] = io.Url
			}
		}
		for _, io := range task.Predata {
			invocation.Inputs[io.FileName] = io.Url
		}
		for _, io := range task.Outputs {
			invocation.Outputs[io.FileName] = io.Url
			if !io.Intermediate {
				job_invocation.Outputs[io.FileName] = io.Url
			}
		}
		report.Invocations = append(report.Invocations, invocation)
		report.CreatedDate = task.CreatedDate
		report.StartedDate = task.StartedDate
		report.CompletedDate = task.CompletedDate
		report.Subject = task.Cmd.Name
		wfrun.ProcessorReports = append(wfrun.ProcessorReports, report)
	}
	wfrun.Invocations = append(wfrun.Invocations, job_invocation)
	return
}
