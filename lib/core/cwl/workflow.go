package cwl

import (
	"fmt"
	"path"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/mitchellh/mapstructure"

	//"os"
	"reflect"
	//"strings"
	//"gopkg.in/mgo.v2/bson"
)

// Workflow https://www.commonwl.org/v1.0/Workflow.html#Workflow
type Workflow struct {
	ProcessImpl `yaml:",inline" bson:",inline" json:",inline" mapstructure:"-"` // provides Class, ID, Requirements, Hints

	Inputs     []InputParameter          `yaml:"inputs,omitempty" bson:"inputs,omitempty" json:"inputs,omitempty" mapstructure:"inputs,omitempty"`
	Outputs    []WorkflowOutputParameter `yaml:"outputs,omitempty" bson:"outputs,omitempty" json:"outputs,omitempty" mapstructure:"outputs,omitempty"`
	Steps      []WorkflowStep            `yaml:"steps,omitempty" bson:"steps,omitempty" json:"steps,omitempty" mapstructure:"steps,omitempty"`
	Label      string                    `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Doc        string                    `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	CwlVersion CWLVersion                `yaml:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" json:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
	Metadata   map[string]interface{}    `yaml:"metadata,omitempty" bson:"metadata,omitempty" json:"metadata,omitempty" mapstructure:"metadata,omitempty"`
	Namespaces map[string]string         `yaml:"$namespaces,omitempty" bson:"_DOLLAR_namespaces,omitempty" json:"$namespaces,omitempty" mapstructure:"$namespaces,omitempty"`
}

// GetClass _
func (wf *Workflow) GetClass() string { return string(CWLWorkflow) }

// func (w *Workflow) GetID() string    { return w.Id }
// func (w *Workflow) SetID(id string)  { w.Id = id }
// func (w *Workflow) IsCWLMinimal()  {}
// func (w *Workflow) Is_Any()          {}

// IsProcess _
func (wf *Workflow) IsProcess() {}

// GetMapElement _
func GetMapElement(m map[interface{}]interface{}, key string) (value interface{}, err error) {

	for k, v := range m {
		kStr, ok := k.(string)
		if ok {
			if kStr == key {
				value = v
				return
			}
		}
	}
	err = fmt.Errorf("Element \"%s\" not found in map", key)
	return
}

// NewWorkflowEmpty _
func NewWorkflowEmpty() (w Workflow) {
	w = Workflow{}
	w.Class = string(CWLWorkflow)
	return w
}

// NewWorkflow _
func NewWorkflow(original interface{}, parentIdentifier string, objectIdentifier string, injectedRequirements []Requirement, context *WorkflowContext) (workflowPtr *Workflow, schemata []CWLType_Type, err error) {

	// convert input map into input array
	original, err = MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewWorkflow) MakeStringMap returned: %s", err.Error())
		return
	}

	//fmt.Println("original:")
	//spew.Dump(original)

	workflow := NewWorkflowEmpty()
	workflowPtr = &workflow

	workflow.ProcessImpl = ProcessImpl{}
	var process *ProcessImpl
	process = &workflow.ProcessImpl
	process.Class = "Workflow"
	err = ProcessImplInit(original, process, parentIdentifier, objectIdentifier, context)
	if err != nil {
		err = fmt.Errorf("(NewWorkflow) NewProcessImpl returned: %s", err.Error())
		return
	}

	workflow.ProcessImpl = *process

	switch original.(type) {
	case map[string]interface{}:
		object := original.(map[string]interface{})

		var CwlVersion CWLVersion
		var ok bool
		cwlVersionIf, hasCWLVersion := object["cwlVersion"]
		if hasCWLVersion {
			//CwlVersion = cwl_version_if.(string)
			var cwlVersionStr string
			cwlVersionStr, ok = cwlVersionIf.(string)
			if !ok {
				err = fmt.Errorf("(NewWorkflow) Could not read CWLVersion (%s)", reflect.TypeOf(cwlVersionIf))
				return
			}
			CwlVersion = CWLVersion(cwlVersionStr)
		} else {
			CwlVersion = context.CwlVersion
		}

		if CwlVersion == "" {
			fmt.Println("workflow without version:")
			//spew.Dump(object)
			err = fmt.Errorf("(NewWorkflow) CwlVersion empty (has_cwl_version: %t, context.CwlVersion: %s)", hasCWLVersion, context.CwlVersion)
			return
		}

		inputs, ok := object["inputs"]
		if ok {
			object["inputs"], err = NewInputParameterArray(inputs, schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflow) NewInputParameterArray returned: %s", err.Error())
				return
			}
		}

		outputs, ok := object["outputs"]
		if ok {
			object["outputs"], err = NewWorkflowOutputParameterArray(outputs, schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflow) NewWorkflowOutputParameterArray returned: %s", err.Error())
				return
			}
		}

		err = CreateRequirementAndHints(object, process, injectedRequirements, inputs, context)
		if err != nil {
			err = fmt.Errorf("(NewWorkflow) CreateRequirementArrayAndInject returned: %s", err.Error())
		}

		// convert steps to array if it is a map
		steps, ok := object["steps"]
		if ok {
			logger.Debug(3, "(NewWorkflow) Parsing steps in Workflow")
			var schemataNew []CWLType_Type
			workflowID := process.ID
			requirementsArray := process.Requirements
			//fmt.Printf("(NewWorkflow) Injecting %d\n", len(requirements_array))
			//spew.Dump(requirements_array)
			schemataNew, object["steps"], err = CreateWorkflowStepsArray(steps, workflowID, requirementsArray, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflow) CreateWorkflowStepsArray returned: %s", err.Error())
				return
			}
			for i := range schemataNew {
				schemata = append(schemata, schemataNew[i])
			}
		}
		// } else {
		// 	err = fmt.Errorf("(NewWorkflow) Workflow has no steps ")
		// 	//spew.Dump(object)
		// 	return
		// }

		//fmt.Printf("......WORKFLOW raw")
		//spew.Dump(object)
		//fmt.Printf("-- Steps found ------------") // WorkflowStep
		//for _, step := range elem["steps"].([]interface{}) {

		//	spew.Dump(step)

		//}

		err = mapstructure.Decode(object, &workflow)
		if err != nil {
			err = fmt.Errorf("(NewWorkflow) error parsing workflow class: %s", err.Error())
			return
		}

		//fmt.Println("workflow:")
		//spew.Dump(workflow)

		if context.Namespaces != nil {
			workflow.Namespaces = context.Namespaces
		}

		if context != nil {
			//if context.Initialzing {
			// err = context.Add(workflow_ptr.Id, workflow_ptr, "NewWorkflow")
			// if err != nil {
			// 	err = fmt.Errorf("(NewWorkflow) context.Add returned: %s", err.Error())
			// 	return
			// }
			//}

			//for i, _ := range workflow.Inputs { // this can conflict with workflow_instance fields
			//	inp := &workflow.Inputs[i]
			//	err = context.Add(inp.Id, inp, "NewWorkflow")
			//	if err != nil {
			//		err = fmt.Errorf("(NewWorkflow) context.Add returned: %s", err.Error())
			//		return
			//	}
			//}
		} else {
			err = fmt.Errorf("(NewWorkflow) context empty")
			return
		}
		//fmt.Printf(".....WORKFLOW")
		//spew.Dump(workflow)
		return

	default:

		err = fmt.Errorf("(NewWorkflow) Input type %s can not be parsed", reflect.TypeOf(original))

	}

	return
}

// GetStep _
func (wf *Workflow) GetStep(name string) (step *WorkflowStep, err error) {

	foundSteps := ""

	if wf == nil {
		err = fmt.Errorf("(Workflow/GetStep) wf == nil")
		return
	}

	if wf.Steps == nil {
		err = fmt.Errorf("(Workflow/GetStep) wf.Steps == nil")
		return
	}

	for i := range wf.Steps {

		s := &wf.Steps[i]

		sBase := path.Base(s.ID)

		if sBase == name {
			step = s
			return
		}
		foundSteps += "," + sBase

	}
	err = fmt.Errorf("(Workflow/GetStep) step %s not found (found_steps: %s)", name, foundSteps)
	return
}

// Evaluate _
func (wf *Workflow) Evaluate(inputs interface{}, context *WorkflowContext) (err error) {

	for i := range wf.Requirements {

		r := wf.Requirements[i]

		err = r.Evaluate(inputs, context)
		if err != nil {
			err = fmt.Errorf("(Workflow/Evaluate) Requirements r.Evaluate returned: %s", err.Error())
			return
		}

	}

	for i := range wf.Hints {

		r := wf.Hints[i]

		err = r.Evaluate(inputs, context)
		if err != nil {
			err = fmt.Errorf("(Workflow/Evaluate) Hints r.Evaluate returned: %s", err.Error())
			return
		}

	}

	return
}
