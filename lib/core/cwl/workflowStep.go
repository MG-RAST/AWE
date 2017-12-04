package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"gopkg.in/mgo.v2/bson"
	"reflect"
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

func NewWorkflowStep(original interface{}) (w *WorkflowStep, err error) {
	var step WorkflowStep

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
			v_map["run"], err = NewProcess(run)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) run %s", err.Error())
				return
			}
		}

		hints, ok := v_map["hints"]
		if ok {
			v_map["hints"], err = CreateRequirementArray(hints)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) CreateRequirementArray %s", err.Error())
				return
			}
		}

		requirements, ok := v_map["requirements"]
		if ok {
			v_map["requirements"], err = CreateRequirementArray(requirements)
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
		return
	default:
		err = fmt.Errorf("(NewWorkflowStep) type %s unknown", reflect.TypeOf(original))
		return
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
func CreateWorkflowStepsArray(original interface{}) (err error, array_ptr *[]WorkflowStep) {

	array := []WorkflowStep{}

	switch original.(type) {

	case map[interface{}]interface{}:

		// iterate over workflow steps
		for k, v := range original.(map[interface{}]interface{}) {
			//fmt.Printf("A step\n")
			//spew.Dump(v)

			//fmt.Println("type: ")
			//fmt.Println(reflect.TypeOf(v))

			step, xerr := NewWorkflowStep(v)
			if xerr != nil {
				err = fmt.Errorf("(CreateWorkflowStepsArray) NewWorkflowStep failed: %s", xerr.Error())
				return
			}

			step.Id = k.(string)

			//fmt.Printf("Last step\n")
			//spew.Dump(step)
			//fmt.Printf("C")
			array = append(array, *step)
			//fmt.Printf("D")

		}

		array_ptr = &array
		return
	case []interface{}:

		// iterate over workflow steps
		for _, v := range original.([]interface{}) {
			//fmt.Printf("A step\n")
			//spew.Dump(v)

			//fmt.Println("type: ")
			//fmt.Println(reflect.TypeOf(v))

			step, xerr := NewWorkflowStep(v)
			if xerr != nil {
				err = fmt.Errorf("(CreateWorkflowStepsArray) NewWorkflowStep failed: %s", xerr.Error())
				return
			}

			//step.Id = k.(string)

			//fmt.Printf("Last step\n")
			//spew.Dump(step)
			//fmt.Printf("C")
			array = append(array, *step)
			//fmt.Printf("D")

		}

		array_ptr = &array
		return
	default:
		err = fmt.Errorf("(CreateWorkflowStepsArray) Type unknown")
		return
	}
	//spew.Dump(new_array)
	return
}

func GetProcessName(p interface{}) (process_name string, err error) {

	switch p.(type) {
	case string:

		p_str := p.(string)

		process_name = p_str

	case bson.M: // (because of mongo this is bson.M)

		p_bson := p.(bson.M)

		process_name_interface, ok := p_bson["value"]
		if !ok {
			err = fmt.Errorf("(GetProcessName) bson.M did not hold a field named value")
			return
		}

		process_name, ok = process_name_interface.(string)
		if !ok {
			err = fmt.Errorf("(GetProcessName) bson.M value field is not a string")
			return
		}

	default:
		err = fmt.Errorf("(GetProcessName) Process type %s unknown, cannot create Workunit", reflect.TypeOf(p))
		return

	}

	return
}
