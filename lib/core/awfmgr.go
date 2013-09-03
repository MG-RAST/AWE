package core

import (
	"encoding/json"
	"errors"
	"github.com/MG-RAST/AWE/lib/conf"
	"io/ioutil"
	"strings"
)

var (
	AwfMgr *WorkflowMgr
)

type WorkflowMgr struct {
	awfMap     map[string]*Workflow
	awfDirPath string
}

func NewWorkflowMgr() *WorkflowMgr {
	wfm := new(WorkflowMgr)
	wfm.awfMap = map[string]*Workflow{}
	wfm.awfDirPath = conf.AWF_PATH
	return wfm
}

func InitAwfMgr() {
	AwfMgr = NewWorkflowMgr()
}

func (wfm *WorkflowMgr) GetWorkflow(name string) (awf *Workflow, err error) {
	if _, ok := wfm.awfMap[name]; ok {
		return wfm.awfMap[name], nil
	}
	return nil, errors.New("workflow not found: " + name)
}

func (wfm *WorkflowMgr) GetAllWorkflows() (workflows []*Workflow) {
	for _, wf := range wfm.awfMap {
		workflows = append(workflows, wf)
	}
	return
}

func (wfm *WorkflowMgr) AddWorkflow(name string, awf *Workflow) {
	if _, ok := wfm.awfMap[name]; !ok {
		wfm.awfMap[name] = awf
	}
}

func (wfm *WorkflowMgr) LoadWorkflows() (err error) {
	if wfm.awfDirPath == "" {
		return errors.New("LoadWorkflows: awfPath not set")
	}
	files, err := ioutil.ReadDir(wfm.awfDirPath)
	if err != nil {
		return errors.New("LoadWorkflows: list dir error awfPath:" + wfm.awfDirPath)
	}
	for _, fileinfo := range files {
		filename := fileinfo.Name()
		if strings.HasSuffix(filename, ".awf") {
			awfpath := wfm.awfDirPath + "/" + filename
			awfjson, err := ioutil.ReadFile(awfpath)
			if err != nil {
				return err
			}
			wf := new(Workflow)
			if err := json.Unmarshal(awfjson, &wf); err != nil {
				return err
			}
			wfm.AddWorkflow(wf.WfInfo.Name, wf)
		}
	}
	return
}
