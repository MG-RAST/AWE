package cwl

import (
	"fmt"

	//"github.com/davecgh/go-spew/spew"
	"reflect"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type WorkflowStep struct {
	Id            string               `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	In            []WorkflowStepInput  `yaml:"in,omitempty" bson:"in,omitempty" json:"in,omitempty" mapstructure:"in,omitempty"` // array<WorkflowStepInput> | map<WorkflowStepInput.id, WorkflowStepInput.source> | map<WorkflowStepInput.id, WorkflowStepInput>
	Out           []WorkflowStepOutput `yaml:"out,omitempty" bson:"out,omitempty" json:"out,omitempty" mapstructure:"out,omitempty"`
	Run           interface{}          `yaml:"run,omitempty" bson:"run,omitempty" json:"run,omitempty" mapstructure:"run,omitempty"`                                     // (*Process) Specification unclear: string | CommandLineTool | ExpressionTool | Workflow
	Requirements  []interface{}        `yaml:"requirements,omitempty" bson:"requirements,omitempty" json:"requirements,omitempty" mapstructure:"requirements,omitempty"` //[]Requirement
	Hints         []interface{}        `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty" mapstructure:"hints,omitempty"`                             //[]Requirement
	Label         string               `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Doc           string               `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	Scatter       []string             `yaml:"scatter,omitempty" bson:"scatter,omitempty" json:"scatter,omitempty" mapstructure:"scatter,omitempty"`                         // ScatterFeatureRequirement
	ScatterMethod string               `yaml:"scatterMethod,omitempty" bson:"scatterMethod,omitempty" json:"scatterMethod,omitempty" mapstructure:"scatterMethod,omitempty"` // ScatterFeatureRequirement
}

func NewWorkflowStep(original interface{}, CwlVersion CWLVersion) (w *WorkflowStep, schemata []CWLType_Type, err error) {
	var step WorkflowStep

	logger.Debug(3, "NewWorkflowStep starting")
	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:
		v_map := original.(map[string]interface{})
		//spew.Dump(v_map)

		step_in, ok := v_map["in"]
		if ok {
			v_map["in"], err = CreateWorkflowStepInputArray(step_in)
			if err != nil {
				return
			}
		}

		step_out, ok := v_map["out"]
		if ok {
			v_map["out"], err = NewWorkflowStepOutputArray(step_out)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) CreateWorkflowStepOutputArray %s", err.Error())
				return
			}
		}

		run, ok := v_map["run"]
		if ok {
			var schemata_new []CWLType_Type
			v_map["run"], schemata_new, err = NewProcess(run, CwlVersion)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) run %s", err.Error())
				return
			}
			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}
		}

		hints, ok := v_map["hints"]
		if ok {
			v_map["hints"], schemata, err = CreateRequirementArray(hints)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) CreateRequirementArray %s", err.Error())
				return
			}
		}

		requirements, ok := v_map["requirements"]
		if ok {
			v_map["requirements"], schemata, err = CreateRequirementArray(requirements)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) CreateRequirementArray %s", err.Error())
				return
			}
		}
		//spew.Dump(v_map["run"])
		err = mapstructure.Decode(original, &step)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStep) %s", err.Error())
			return
		}
		w = &step
		//spew.Dump(w.Run)

		//fmt.Println("xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx")

	default:
		err = fmt.Errorf("(NewWorkflowStep) type %s unknown", reflect.TypeOf(original))

	}

	return
}

func (w WorkflowStep) GetOutput(id string) (output *WorkflowStepOutput, err error) {
	for _, o := range w.Out {
		// o is a WorkflowStepOutput
		if o.Id == id {
			output = &o
			return
		}
	}
	err = fmt.Errorf("WorkflowStepOutput %s not found in WorkflowStep", id)
	return
}

// CreateWorkflowStepsArray
func CreateWorkflowStepsArray(original interface{}, CwlVersion CWLVersion) (schemata []CWLType_Type, array_ptr *[]WorkflowStep, err error) {

	array := []WorkflowStep{}

	if CwlVersion == "" {
		err = fmt.Errorf("(CreateWorkflowStepsArray) CwlVersion empty")
		return
	}
	switch original.(type) {

	case map[interface{}]interface{}:

		// iterate over workflow steps
		for k, v := range original.(map[interface{}]interface{}) {
			//fmt.Printf("A step\n")
			//spew.Dump(v)

			//fmt.Println("type: ")
			//fmt.Println(reflect.TypeOf(v))

			var schemata_new []CWLType_Type
			var step *WorkflowStep
			step, schemata_new, err = NewWorkflowStep(v, CwlVersion)
			if err != nil {
				err = fmt.Errorf("(CreateWorkflowStepsArray) NewWorkflowStep failed: %s", err.Error())
				return
			}

			step.Id = k.(string)

			//fmt.Printf("Last step\n")
			//spew.Dump(step)
			//fmt.Printf("C")
			array = append(array, *step)
			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}
			//fmt.Printf("D")

		}

		array_ptr = &array
		return
	case []interface{}:

		// iterate over workflow steps
		for _, v := range original.([]interface{}) {
			//fmt.Printf("A(2) step\n")
			//spew.Dump(v)

			//fmt.Println("type: ")
			//fmt.Println(reflect.TypeOf(v))
			var schemata_new []CWLType_Type
			var step *WorkflowStep
			step, schemata_new, err = NewWorkflowStep(v, CwlVersion)
			if err != nil {
				err = fmt.Errorf("(CreateWorkflowStepsArray) NewWorkflowStep failed: %s", err.Error())
				return
			}
			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}
			//step.Id = k.(string)

			//fmt.Printf("Last step\n")
			//spew.Dump(step)
			//fmt.Printf("C")
			array = append(array, *step)
			//fmt.Printf("D")

		}

		array_ptr = &array

	default:
		err = fmt.Errorf("(CreateWorkflowStepsArray) Type unknown")

	}
	//spew.Dump(new_array)
	return
}

func GetProcess(original interface{}, collection *CWL_collection, CwlVersion CWLVersion, input_schemata []CWLType_Type) (process interface{}, schemata []CWLType_Type, err error) {

	var p interface{}
	p, err = MakeStringMap(original)
	if err != nil {
		return
	}

	var clt *CommandLineTool
	var et *ExpressionTool
	var wfl *Workflow

	switch p.(type) {
	case string:

		process_name := p.(string)

		clt, err = collection.GetCommandLineTool(process_name)
		if err == nil {
			process = clt
			return
		}
		err = nil

		et, err = collection.GetExpressionTool(process_name)
		if err == nil {
			process = et
			return
		}
		err = nil

		wfl, err = collection.GetWorkflow(process_name)
		if err == nil {
			process = wfl
			return
		}
		err = nil
		spew.Dump(collection)
		err = fmt.Errorf("(GetProcess) Process %s not found ", process_name)

	case map[string]interface{}:

		//fmt.Println("GetProcess got:")
		//spew.Dump(p)

		p_map := p.(map[string]interface{})

		class_name_if, ok := p_map["class"]
		if ok {
			var class_name string
			class_name, ok = class_name_if.(string)
			if ok {
				switch class_name {
				case "CommandLineTool":

					clt, schemata, err = NewCommandLineTool(p, CwlVersion)
					process = clt
					return
				case "Workflow":
					wfl, schemata, err = NewWorkflow(p, CwlVersion)
					process = wfl
					return
				case "ExpressionTool":
					et, err = NewExpressionTool(p, "", input_schemata)
					process = et
					return
				default:
					err = fmt.Errorf("(GetProcess) class \"%s\" not a supported process", class_name)
					return
				}

			}
		}

		// in case of bson, check field "value"
		//process_name_interface, ok := p_map["value"]
		//if !ok {
		//	err = fmt.Errorf("(GetProcess) map did not hold a field named value")
		//	return
		//}
		//
		//process_name, ok = process_name_interface.(string)
		//if !ok {
		//	err = fmt.Errorf("(GetProcess) map value field is not a string")
		//	return
		//}

	default:
		err = fmt.Errorf("(GetProcess) Process type %s unknown", reflect.TypeOf(p))

	}

	return
}
