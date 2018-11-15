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
	CWL_object_Impl `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"`
	Id              string               `yaml:"id,omitempty" bson:"id,omitempty" json:"id,omitempty" mapstructure:"id,omitempty"`
	In              []WorkflowStepInput  `yaml:"in,omitempty" bson:"in,omitempty" json:"in,omitempty" mapstructure:"in,omitempty"` // array<WorkflowStepInput> | map<WorkflowStepInput.id, WorkflowStepInput.source> | map<WorkflowStepInput.id, WorkflowStepInput>
	Out             []WorkflowStepOutput `yaml:"out,omitempty" bson:"out,omitempty" json:"out,omitempty" mapstructure:"out,omitempty"`
	Run             interface{}          `yaml:"run,omitempty" bson:"run,omitempty" json:"run,omitempty" mapstructure:"run,omitempty"`                                     //  string | CommandLineTool | ExpressionTool | Workflow
	Requirements    []interface{}        `yaml:"requirements,omitempty" bson:"requirements,omitempty" json:"requirements,omitempty" mapstructure:"requirements,omitempty"` //[]Requirement
	Hints           []interface{}        `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty" mapstructure:"hints,omitempty"`                             //[]Requirement
	Label           string               `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Doc             string               `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	Scatter         []string             `yaml:"scatter,omitempty" bson:"scatter,omitempty" json:"scatter,omitempty" mapstructure:"scatter,omitempty"`                         // ScatterFeatureRequirement
	ScatterMethod   string               `yaml:"scatterMethod,omitempty" bson:"scatterMethod,omitempty" json:"scatterMethod,omitempty" mapstructure:"scatterMethod,omitempty"` // ScatterFeatureRequirement
	//CwlVersion    CWLVersion           `bson:"cwlVersion,omitempty"  mapstructure:"cwlVersion,omitempty"`
	//Namespaces    map[string]string    `yaml:"$namespaces,omitempty" bson:"_DOLLAR_namespaces,omitempty" json:"$namespaces,omitempty" mapstructure:"$namespaces,omitempty"`
}

func NewWorkflowStep() (w *WorkflowStep) {

	w = &WorkflowStep{}

	return
}

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
	//ws.CwlVersion = context.CwlVersion
	ws.Run, _, err = NewProcess(p, nil, context) // requirements should already be injected
	if err != nil {
		err = fmt.Errorf("(WorkflowStep/Init) NewProcess() returned %s", err.Error())
		return
	}

	return
}

func NewWorkflowStepFromInterface(original interface{}, injectedRequirements []Requirement, context *WorkflowContext) (w *WorkflowStep, schemata []CWLType_Type, err error) {
	var step WorkflowStep

	logger.Debug(3, "NewWorkflowStep starting")
	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	defer func() {
		if context != nil && context.Initialzing && err == nil {
			context.Add(w.Id, w)
		}
	}()

	switch original.(type) {

	case map[string]interface{}:
		v_map := original.(map[string]interface{})
		//spew.Dump(v_map)

		requirements, ok := v_map["requirements"]
		if !ok {
			requirements = nil
		}

		var requirements_array []Requirement
		//var requirements_array_temp *[]Requirement
		//var schemata_new []CWLType_Type
		//fmt.Printf("(NewWorkflowStep) Injecting %d \n", len(injectedRequirements))
		//spew.Dump(injectedRequirements)
		requirements_array, err = CreateRequirementArrayAndInject(requirements, injectedRequirements, nil, context) // not sure what input to use
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStep) error in CreateRequirementArray (requirements): %s", err.Error())
			return
		}

		//for i, _ := range schemata_new {
		//	schemata = append(schemata, schemata_new[i])
		//}

		v_map["requirements"] = requirements_array

		step_in, ok := v_map["in"]
		if ok {
			v_map["in"], err = CreateWorkflowStepInputArray(step_in, context)
			if err != nil {
				return
			}
		} else {
			err = fmt.Errorf("(NewWorkflowStep) no inputs ????")
			return
		}

		step_out, ok := v_map["out"]
		if ok {
			v_map["out"], err = NewWorkflowStepOutputArray(step_out, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) CreateWorkflowStepOutputArray %s", err.Error())
				return
			}
		}

		run, ok := v_map["run"]
		if ok {
			var schemata_new []CWLType_Type
			//fmt.Printf("(NewWorkflowStep) Injecting %d\n", len(requirements_array))
			//spew.Dump(requirements_array)

			v_map["run"], schemata_new, err = NewProcess(run, requirements_array, context)
			if err != nil {
				err = fmt.Errorf("(NewWorkflowStep) run %s", err.Error())
				return
			}
			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}
		}

		scatter, ok := v_map["scatter"]
		if ok {
			switch scatter.(type) {
			case string:
				var scatter_str string

				scatter_str, ok = scatter.(string)
				if !ok {
					err = fmt.Errorf("(NewWorkflowStep) expected string")
					return
				}
				v_map["scatter"] = []string{scatter_str}

			case []string:
				// all ok
			case []interface{}:
				scatter_array := scatter.([]interface{})
				scatter_string_array := []string{}
				for _, element := range scatter_array {
					var element_str string
					element_str, ok = element.(string)
					if !ok {
						err = fmt.Errorf("(NewWorkflowStep) Element of scatter array is not string (%s)", reflect.TypeOf(element))
						return
					}
					scatter_string_array = append(scatter_string_array, element_str)
				}
				v_map["scatter"] = scatter_string_array

			default:
				err = fmt.Errorf("(NewWorkflowStep) scatter has unsopported type: %s", reflect.TypeOf(scatter))
				return
			}
		}

		scatter, ok = v_map["scatter"]
		if ok {
			switch scatter.(type) {
			case []string:

			default:
				err = fmt.Errorf("(NewWorkflowStep) scatter is not []string: (type: %s)", reflect.TypeOf(scatter))
				return
			}
		}

		hints, ok := v_map["hints"]
		if ok && (hints != nil) {
			//var schemata_new []CWLType_Type

			var hints_array []Requirement
			hints_array, err = CreateHintsArray(hints, injectedRequirements, nil, context)
			if err != nil {
				err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (hints): %s", err.Error())
				return
			}
			//for i, _ := range schemata_new {
			//	schemata = append(schemata, schemata_new[i])
			//}
			v_map["hints"] = hints_array
		}

		//spew.Dump(v_map["run"])
		err = mapstructure.Decode(original, &step)
		if err != nil {
			err = fmt.Errorf("(NewWorkflowStep) %s", err.Error())
			return
		}
		w = &step

		if step.Id == "" {
			err = fmt.Errorf("(NewWorkflowStep) step.Id empty")
			return
		}
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
func CreateWorkflowStepsArray(original interface{}, injectedRequirements []Requirement, context *WorkflowContext) (schemata []CWLType_Type, array_ptr *[]WorkflowStep, err error) {

	array := []WorkflowStep{}

	if context.CwlVersion == "" {
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
			//fmt.Printf("(CreateWorkflowStepsArray) Injecting %d \n", len(injectedRequirements))
			//spew.Dump(injectedRequirements)
			step, schemata_new, err = NewWorkflowStepFromInterface(v, injectedRequirements, context)
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
			//fmt.Printf("(CreateWorkflowStepsArray) Injecting %d \n", len(injectedRequirements))
			//spew.Dump(injectedRequirements)
			step, schemata_new, err = NewWorkflowStepFromInterface(v, injectedRequirements, context)
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

// func (ws *WorkflowStep) GetInputType(name string) (result CWLType_Type, err error) {

// 	if ws.Run == nil {
// 		err = fmt.Errorf("(WorkflowStep/GetInputType) ws.Run == nil ")
// 		return
// 	}

// 	switch ws.Run.(type) {
// 	case *CommandLineTool:

// 		clt, ok := ws.Run.(*CommandLineTool)
// 		if !ok {
// 			err = fmt.Errorf("(WorkflowStep/GetInputType) type assertion error (%s)", reflect.TypeOf(ws.Run))
// 			return
// 		}
// 		_ = clt

// 		for _, input := range clt.Inputs {
// 			if input.Id == name {
// 				result = input.Type
// 			}

// 		}

// 	case *ExpressionTool:
// 		et, ok := ws.Run.(*ExpressionTool)
// 		if !ok {
// 			err = fmt.Errorf("(WorkflowStep/GetInputType) type assertion error (%s)", reflect.TypeOf(ws.Run))
// 			return
// 		}
// 		_ = et
// 	case *Workflow:
// 		wf, ok := ws.Run.(*Workflow)
// 		if !ok {
// 			err = fmt.Errorf("(WorkflowStep/GetInputType) type assertion error (%s)", reflect.TypeOf(ws.Run))
// 			return
// 		}
// 		_ = wf
// 	default:
// 		err = fmt.Errorf("(WorkflowStep/GetInputType) process type not supported (%s)", reflect.TypeOf(ws.Run))
// 		return
// 	}
// 	return
// }

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

		process_name := p.(string)

		clt, err = context.GetCommandLineTool(process_name)
		if err == nil {
			process = clt
			return
		}
		err = nil

		et, err = context.GetExpressionTool(process_name)
		if err == nil {
			process = et
			return
		}
		err = nil

		wfl, err = context.GetWorkflow(process_name)
		if err == nil {
			process = wfl
			return
		}
		err = nil
		spew.Dump(context)
		err = fmt.Errorf("(GetProcess) Process %s not found ", process_name)

	// case map[string]interface{}:

	// 	err = fmt.Errorf("(GetProcess) Process should have been parsed by now !?") // otherwise we to inject Requirements
	// 	return

	// 	//fmt.Println("GetProcess got:")
	// 	//spew.Dump(p)

	// 	p_map := p.(map[string]interface{})

	// 	class_name_if, ok := p_map["class"]
	// 	if ok {
	// 		var class_name string
	// 		class_name, ok = class_name_if.(string)
	// 		if ok {
	// 			switch class_name {
	// 			case "CommandLineTool":

	// 				clt, schemata, err = NewCommandLineTool(p, CwlVersion, nil)
	// 				process = clt
	// 				return
	// 			case "Workflow":
	// 				wfl, schemata, err = NewWorkflow(p, CwlVersion, nil)
	// 				process = wfl
	// 				return
	// 			case "ExpressionTool":
	// 				et, err = NewExpressionTool(p, "", input_schemata, nil)
	// 				process = et
	// 				return
	// 			default:
	// 				err = fmt.Errorf("(GetProcess) class \"%s\" not a supported process", class_name)
	// 				return
	// 			}

	// 		}
	// 	}

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
