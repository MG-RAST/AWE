package cwl

import (
	"fmt"
	"path"
	"strings"

	//"github.com/davecgh/go-spew/spew"
	"reflect"

	"github.com/MG-RAST/AWE/lib/core/uuid"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// WorkflowStep _
type WorkflowStep struct {
	CWLObjectImpl `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"`
	ID            string               `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	In            []WorkflowStepInput  `yaml:"in,omitempty" bson:"in,omitempty" json:"in,omitempty" mapstructure:"in,omitempty"` // array<WorkflowStepInput> | map<WorkflowStepInput.id, WorkflowStepInput.source> | map<WorkflowStepInput.id, WorkflowStepInput>
	Out           []WorkflowStepOutput `yaml:"out,omitempty" bson:"out,omitempty" json:"out,omitempty" mapstructure:"out,omitempty"`
	Run           interface{}          `yaml:"run,omitempty" bson:"run,omitempty" json:"run,omitempty" mapstructure:"run,omitempty"`                                     //  string | CommandLineTool | ExpressionTool | Workflow
	Requirements  []interface{}        `yaml:"requirements,omitempty" bson:"requirements,omitempty" json:"requirements,omitempty" mapstructure:"requirements,omitempty"` //[]Requirement
	Hints         []interface{}        `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty" mapstructure:"hints,omitempty"`                             //[]Requirement
	Label         string               `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Doc           string               `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	Scatter       []string             `yaml:"scatter,omitempty" bson:"scatter,omitempty" json:"scatter,omitempty" mapstructure:"scatter,omitempty"`                         // ScatterFeatureRequirement
	ScatterMethod string               `yaml:"scatterMethod,omitempty" bson:"scatterMethod,omitempty" json:"scatterMethod,omitempty" mapstructure:"scatterMethod,omitempty"` // ScatterFeatureRequirement
	//CwlVersion    CWLVersion           `bson:"cwlVersion,omitempty"  mapstructure:"cwlVersion,omitempty"`
	//Namespaces    map[string]string    `yaml:"$namespaces,omitempty" bson:"_DOLLAR_namespaces,omitempty" json:"$namespaces,omitempty" mapstructure:"$namespaces,omitempty"`
}

// NewWorkflowStep _
func NewWorkflowStep() (w *WorkflowStep) {

	w = &WorkflowStep{}

	return
}

// Init _
func (ws *WorkflowStep) Init(context *WorkflowContext) (err error) {
	if ws.Run == nil {
		return
	}
	p := ws.Run
	switch p.(type) {
	case *CommandLineTool:
		return
	case *ExpressionTool:
		return
	case *Workflow:
		return
	}

	baseIdentifier := path.Dir(ws.ID)

	//ws.CwlVersion = context.CwlVersion
	ws.Run, _, err = NewProcess(p, baseIdentifier, nil, context) // requirements should already be injected
	if err != nil {
		err = fmt.Errorf("(WorkflowStep/Init) NewProcess() returned %s", err.Error())
		return
	}

	if ws.Run == nil {
		err = fmt.Errorf("(WorkflowStep/Init) ws.Run == nil")
		return
	}

	return
}

// NewWorkflowStepFromInterface _
// stepID is different for embedded vs referenced workflows
// stepID argument should  be relative only?
func NewWorkflowStepFromInterface(original interface{}, stepID string, workflowID string, injectedRequirements []Requirement, context *WorkflowContext) (step *WorkflowStep, schemata []CWLType_Type, err error) {
	//var step WorkflowStep

	step = &WorkflowStep{}

	logger.Debug(3, "NewWorkflowStep starting")
	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:
		originalMap := original.(map[string]interface{})
		//spew.Dump(v_map)

		idIf, ok := originalMap["id"]
		if !ok {
			if stepID == "" {
				err = fmt.Errorf("(NewWorkflowStep) object has no id and argument stepID is empty")
				return
			}

			if !strings.HasPrefix(stepID, "#") {

				if workflowID == "" {
					err = fmt.Errorf("(NewWorkflowStep) workflowID is empty")
					return
				}

				stepID = path.Join(workflowID, stepID)
			}

			originalMap["id"] = stepID
		} else {

			idStr := idIf.(string)

			if !strings.HasPrefix(idStr, "#") {

				if stepID == "" {

					if workflowID == "" {
						err = fmt.Errorf("(NewWorkflowStep) workflowID is empty")
						return
					}

					stepID = path.Join(workflowID, idStr)
				}

				if !strings.HasPrefix(stepID, "#") {
					err = fmt.Errorf("(NewWorkflowStep) stepID is not absoule")
					return
				}

				originalMap["id"] = stepID
			}
		}

		requirements, ok := originalMap["requirements"]
		if !ok {
			requirements = nil
		}

		var requirementsArray []Requirement
		//var requirements_array_temp *[]Requirement
		//var schemataNew []CWLType_Type
		//fmt.Printf("(NewWorkflowStep) Injecting %d \n", len(injectedRequirements))
		//spew.Dump(injectedRequirements)
		requirementsArray, err = CreateRequirementArrayAndInject(requirements, injectedRequirements, nil, context) // not sure what input to use
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStep) error in CreateRequirementArray (requirements): %s", err.Error())
			return
		}

		//for i, _ := range schemataNew {
		//	schemata = append(schemata, schemataNew[i])
		//}

		originalMap["requirements"] = requirementsArray

		stepIn, ok := originalMap["in"]
		if ok {
			originalMap["in"], err = CreateWorkflowStepInputArray(stepIn, context)
			if err != nil {
				return
			}
		}

		stepOut, ok := originalMap["out"]
		if ok {
			originalMap["out"], err = NewWorkflowStepOutputArray(stepOut, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) CreateWorkflowStepOutputArray %s", err.Error())
				return
			}
		}

		run, ok := originalMap["run"]
		if ok {
			var schemataNew []CWLType_Type
			//fmt.Printf("(NewWorkflowStep) Injecting %d\n", len(requirements_array))
			//spew.Dump(requirements_array)

			referenceStr, isReference := run.(string)

			// if run is a reference we do not need to add the process to the step, we just have to load it into the context !

			processID := ""
			if !isReference {
				// process is embedded !
				processID = path.Join(stepID, uuid.New())
			} else {

				if !strings.HasPrefix(referenceStr, "#") {
					referenceStr = "#" + referenceStr
					run = referenceStr
					originalMap["run"] = run
				}

			}

			var process interface{}
			process, schemataNew, err = NewProcess(run, processID, requirementsArray, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) NewProcess returned: %s (processID: %s)", err.Error(), processID)
				return
			}
			if process == nil {
				//spew.Dump(originalMap)
				//panic("(NewWorkflowStep) process == nil")
				err = fmt.Errorf("(NewWorkflowStep) process == nil")
				return
			}
			if !isReference {
				originalMap["run"] = process
				for i := range schemataNew {
					schemata = append(schemata, schemataNew[i])
				}
			}
		}

		scatter, ok := originalMap["scatter"]
		if ok {
			switch scatter.(type) {
			case string:
				var scatterStr string

				scatterStr, ok = scatter.(string)
				if !ok {
					err = fmt.Errorf("(NewWorkflowStep) expected string")
					return
				}
				originalMap["scatter"] = []string{scatterStr}

			case []string:
				// all ok
			case []interface{}:
				scatterArray := scatter.([]interface{})
				scatterStringArray := []string{}
				for _, element := range scatterArray {
					var elementStr string
					elementStr, ok = element.(string)
					if !ok {
						err = fmt.Errorf("(NewWorkflowStep) Element of scatter array is not string (%s)", reflect.TypeOf(element))
						return
					}
					scatterStringArray = append(scatterStringArray, elementStr)
				}
				originalMap["scatter"] = scatterStringArray

			default:
				err = fmt.Errorf("(NewWorkflowStep) scatter has unsopported type: %s", reflect.TypeOf(scatter))
				return
			}
		}

		scatter, ok = originalMap["scatter"]
		if ok {
			switch scatter.(type) {
			case []string:

			default:
				err = fmt.Errorf("(NewWorkflowStep) scatter is not []string: (type: %s)", reflect.TypeOf(scatter))
				return
			}
		}

		hints, ok := originalMap["hints"]
		if ok && (hints != nil) {
			//var schemataNew []CWLType_Type

			var hintsArray []Requirement
			hintsArray, err = CreateHintsArray(hints, injectedRequirements, nil, context)
			if err != nil {
				err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (hints): %s", err.Error())
				return
			}
			//for i, _ := range schemataNew {
			//	schemata = append(schemata, schemataNew[i])
			//}
			originalMap["hints"] = hintsArray
		}

		//spew.Dump(v_map["run"])
		err = mapstructure.Decode(original, step)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStep) mapstructure.Decode returned: %s", err.Error())
			return
		}
		//w = &step

		if step.Run == nil {
			spew.Dump(original)
			err = fmt.Errorf("(NewWorkflowStep) ws.Run == nil")
			return
		}

		if step.ID == "" {

			err = fmt.Errorf("(NewWorkflowStep) step.Id empty")
			return
		}

		if context != nil && context.Initialzing && err == nil {
			// err = context.Add(step.ID, step, "NewWorkflowStepFromInterface")
			// if err != nil {
			// 	err = fmt.Errorf("(NewWorkflowStep) context.Add returned: %s", err.Error())
			// 	return
			// }
		}

		// this happens in handleNoticeWorkDelivered !

		//for i, _ := range step.Out {
		//	out := &step.Out[i]
		//	context.Add(out.Id, out) // adding WorkflowStepOutput is not helpful, it needs to add the reference Tool output
		//
		//		}

		//spew.Dump(w.Run)

		//fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")

	default:
		err = fmt.Errorf("(NewWorkflowStep) type %s unknown", reflect.TypeOf(original))

	}

	return
}

// GetOutput _
func (ws *WorkflowStep) GetOutput(id string) (output *WorkflowStepOutput, err error) {
	for _, o := range ws.Out {
		// o is a WorkflowStepOutput
		if o.Id == id {
			output = &o
			return
		}
	}
	err = fmt.Errorf("WorkflowStepOutput %s not found in WorkflowStep", id)
	return
}

// CreateWorkflowStepsArray _
func CreateWorkflowStepsArray(original interface{}, workflowID string, injectedRequirements []Requirement, context *WorkflowContext) (schemata []CWLType_Type, arrayPtr *[]WorkflowStep, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	array := []WorkflowStep{}

	if context.CwlVersion == "" {
		err = fmt.Errorf("(CreateWorkflowStepsArray) CwlVersion empty")
		return
	}
	switch original.(type) {

	case map[string]interface{}:

		// iterate over workflow steps
		for stepID, v := range original.(map[string]interface{}) {
			//fmt.Printf("A step\n")
			//spew.Dump(v)

			//fmt.Println("type: ")
			//fmt.Println(reflect.TypeOf(v))
			//if !strings.HasPrefix(stepID, "#") {
			//	stepID = path.Join(workflowID, stepID)
			//}

			var schemataNew []CWLType_Type
			var step *WorkflowStep
			//fmt.Printf("(CreateWorkflowStepsArray) Injecting %d \n", len(injectedRequirements))
			//spew.Dump(injectedRequirements)
			step, schemataNew, err = NewWorkflowStepFromInterface(v, stepID, workflowID, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(CreateWorkflowStepsArray) NewWorkflowStep failed: %s", err.Error())
				return
			}

			//fmt.Printf("Last step\n")
			//spew.Dump(step)
			//fmt.Printf("C")
			array = append(array, *step)
			for i := range schemataNew {
				schemata = append(schemata, schemataNew[i])
			}
			//fmt.Printf("D")

		}

		arrayPtr = &array
		return
	case []interface{}:

		// iterate over workflow steps
		for _, v := range original.([]interface{}) {
			//fmt.Printf("A(2) step\n")
			//spew.Dump(v)

			//fmt.Println("type: ")
			//fmt.Println(reflect.TypeOf(v))
			var schemataNew []CWLType_Type
			var step *WorkflowStep
			//fmt.Printf("(CreateWorkflowStepsArray) Injecting %d \n", len(injectedRequirements))
			//spew.Dump(injectedRequirements)
			step, schemataNew, err = NewWorkflowStepFromInterface(v, "", workflowID, injectedRequirements, context)
			if err != nil {
				err = fmt.Errorf("(CreateWorkflowStepsArray) NewWorkflowStep failed: %s", err.Error())
				return
			}
			for i := range schemataNew {
				schemata = append(schemata, schemataNew[i])
			}
			//step.Id = k.(string)

			//fmt.Printf("Last step\n")
			//spew.Dump(step)
			//fmt.Printf("C")
			array = append(array, *step)
			//fmt.Printf("D")

		}

		arrayPtr = &array

	default:
		err = fmt.Errorf("(CreateWorkflowStepsArray) Type unknown: %s", reflect.TypeOf(original))

	}
	//spew.Dump(new_array)
	return
}

// GetProcess _
func GetProcess(original interface{}, context *WorkflowContext) (process interface{}, schemata []CWLType_Type, err error) {

	var p interface{}
	p, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	var clt *CommandLineTool
	var et *ExpressionTool
	var wfl *Workflow

	switch p.(type) {

	case *CommandLineTool:
		process = p

	case *ExpressionTool:
		process = p

	case *Workflow:
		process = p

	case string:

		processName := p.(string)

		clt, err = context.GetCommandLineTool(processName)
		if err == nil {
			process = clt
			return
		}
		err = nil

		et, err = context.GetExpressionTool(processName)
		if err == nil {
			process = et
			return
		}
		err = nil

		wfl, err = context.GetWorkflow(processName)
		if err == nil {
			process = wfl
			return
		}
		err = nil
		spew.Dump(context)
		err = fmt.Errorf("(GetProcess) Process %s not found ", processName)

	default:
		err = fmt.Errorf("(GetProcess) Process type %s unknown", reflect.TypeOf(p))

	}

	return
}

// GetProcessType _
func (ws *WorkflowStep) GetProcessType(context *WorkflowContext) (processType string, err error) {

	p := ws.Run
	switch p.(type) {

	case *CommandLineTool:
		processType = "CommandLineTool"

	case *ExpressionTool:
		processType = "ExpressionTool"

	case *Workflow:
		processType = "Workflow"

	case string:

		processName := p.(string)

		_, err = context.GetCommandLineTool(processName)
		if err == nil {
			processType = "CommandLineTool"
			return
		}
		err = nil

		_, err = context.GetExpressionTool(processName)
		if err == nil {
			processType = "ExpressionTool"
			return
		}
		err = nil

		_, err = context.GetWorkflow(processName)
		if err == nil {
			processType = "Workflow"
			return
		}
		err = nil
		spew.Dump(context)
		err = fmt.Errorf("(GetProcessType) Process %s not found ", processName)

	default:
		err = fmt.Errorf("(GetProcessType) Process type %s unknown", reflect.TypeOf(p))

	}

	return

}

// GetProcess _
func (ws *WorkflowStep) GetProcess(context *WorkflowContext) (process interface{}, schemata []CWLType_Type, err error) {

	//var p interface{}
	//p, err = MakeStringMap(original, context)
	//if err != nil {
	//	return
	//}

	if ws.Run == nil {
		err = fmt.Errorf("(WorkflowStep/GetProcess) ws.Run == nil ")
		return
	}

	p := ws.Run

	var clt *CommandLineTool
	var et *ExpressionTool
	var wfl *Workflow

	switch p.(type) {

	case *CommandLineTool:
		process = p

	case *ExpressionTool:
		process = p

	case *Workflow:
		process = p

	case string:

		processName := p.(string)

		clt, err = context.GetCommandLineTool(processName)
		if err == nil {
			process = clt
			return
		}
		err = nil

		et, err = context.GetExpressionTool(processName)
		if err == nil {
			process = et
			return
		}
		err = nil

		wfl, err = context.GetWorkflow(processName)
		if err == nil {
			process = wfl
			return
		}
		err = nil
		spew.Dump(context)
		err = fmt.Errorf("(WorkflowStep/GetProcess) Process %s not found ", processName)

	default:
		err = fmt.Errorf("(WorkflowStep/GetProcess) Process type %s unknown", reflect.TypeOf(p))

	}

	return
}
