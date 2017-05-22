package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"reflect"
)

type WorkflowStep struct {
	Id            string               `yaml:"id"`
	In            []WorkflowStepInput  `yaml:"in"` // array<WorkflowStepInput> | map<WorkflowStepInput.id, WorkflowStepInput.source> | map<WorkflowStepInput.id, WorkflowStepInput>
	Out           []WorkflowStepOutput `yaml:"out"`
	Run           *Process             `yaml:"run"` // Specification unclear: string | CommandLineTool | ExpressionTool | Workflow
	Requirements  []Requirement        `yaml:"requirements"`
	Hints         []Requirement        `yaml:"hints"`
	Label         string               `yaml:"label"`
	Doc           string               `yaml:"doc"`
	Scatter       string               `yaml:"scatter"`       // ScatterFeatureRequirement
	ScatterMethod string               `yaml:"scatterMethod"` // ScatterFeatureRequirement
}

func NewWorkflowStep(original interface{}, collection *CWL_collection) (w *WorkflowStep, err error) {
	var step WorkflowStep

	switch original.(type) {
	case map[interface{}]interface{}:
		v_map := original.(map[interface{}]interface{})
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
			v_map["out"], err = CreateWorkflowStepOutputArray(step_out)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) CreateWorkflowStepOutputArray %s", err.Error())
				return
			}
		}

		run, ok := v_map["run"]
		if ok {
			v_map["run"], err = NewProcess(run, collection)
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

		err = mapstructure.Decode(original, &step)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStep) %s", err.Error())
			return
		}
		w = &step
		return
	default:
		err = fmt.Errorf("(NewWorkflowStep) type unknown")
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

//http://www.commonwl.org/v1.0/Workflow.html#CommandLineBinding
type CommandLineBinding struct {
	LoadContents  bool   `yaml:"loadContents"`
	Position      int    `yaml:"position"`
	Prefix        string `yaml:"prefix"`
	Separate      string `yaml:"separate"`
	ItemSeparator string `yaml:"itemSeparator"`
	ValueFrom     string `yaml:"valueFrom"`
	ShellQuote    bool   `yaml:"shellQuote"`
}

// CreateWorkflowStepsArray
func CreateWorkflowStepsArray(original interface{}, collection *CWL_collection) (err error, array_ptr *[]WorkflowStep) {

	array := []WorkflowStep{}

	switch original.(type) {

	case map[interface{}]interface{}:

		// iterate over workflow steps
		for k, v := range original.(map[interface{}]interface{}) {
			fmt.Printf("A step\n")
			spew.Dump(v)

			fmt.Println("type: ")
			fmt.Println(reflect.TypeOf(v))

			step, xerr := NewWorkflowStep(v, collection)
			if xerr != nil {
				err = fmt.Errorf("(CreateWorkflowStepsArray) NewWorkflowStep failed: %s", xerr.Error())
				return
			}

			step.Id = k.(string)

			fmt.Printf("Last step\n")
			spew.Dump(step)
			//fmt.Printf("C")
			array = append(array, *step)
			//fmt.Printf("D")

		}

		array_ptr = &array
		return
	case []interface{}:

		// iterate over workflow steps
		for _, v := range original.([]interface{}) {
			fmt.Printf("A step\n")
			spew.Dump(v)

			fmt.Println("type: ")
			fmt.Println(reflect.TypeOf(v))

			step, xerr := NewWorkflowStep(v, collection)
			if xerr != nil {
				err = fmt.Errorf("(CreateWorkflowStepsArray) NewWorkflowStep failed: %s", xerr.Error())
				return
			}

			//step.Id = k.(string)

			fmt.Printf("Last step\n")
			spew.Dump(step)
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
