package cwl

import (
	"fmt"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	//"os"
	"reflect"
	//"strings"
	//"gopkg.in/mgo.v2/bson"
)

type Workflow struct {
	CWL_object_Impl `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"`
	CWL_class_Impl  `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"` // provides Id and Class fields
	CWL_id_Impl     `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"`
	Inputs          []InputParameter          `yaml:"inputs,omitempty" bson:"inputs,omitempty" json:"inputs,omitempty" mapstructure:"inputs,omitempty"`
	Outputs         []WorkflowOutputParameter `yaml:"outputs,omitempty" bson:"outputs,omitempty" json:"outputs,omitempty" mapstructure:"outputs,omitempty"`
	Steps           []WorkflowStep            `yaml:"steps,omitempty" bson:"steps,omitempty" json:"steps,omitempty" mapstructure:"steps,omitempty"`
	Requirements    *[]Requirement            `yaml:"requirements,omitempty" bson:"requirements,omitempty" json:"requirements,omitempty" mapstructure:"requirements,omitempty"` //[]Requirement
	Hints           []interface{}             `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty" mapstructure:"hints,omitempty"`                             // []Requirement TODO Hints may contain non-requirement objects. Give warning in those cases.
	Label           string                    `yaml:"label,omitempty" bson:"label,omitempty" json:"label,omitempty" mapstructure:"label,omitempty"`
	Doc             string                    `yaml:"doc,omitempty" bson:"doc,omitempty" json:"doc,omitempty" mapstructure:"doc,omitempty"`
	CwlVersion      CWLVersion                `yaml:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" json:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
	Metadata        map[string]interface{}    `yaml:"metadata,omitempty" bson:"metadata,omitempty" json:"metadata,omitempty" mapstructure:"metadata,omitempty"`
}

func (w *Workflow) GetClass() string { return string(CWL_Workflow) }

//func (w *Workflow) GetId() string    { return w.Id }
//func (w *Workflow) SetId(id string)  { w.Id = id }
//func (w *Workflow) Is_CWL_minimal()  {}
//func (w *Workflow) Is_Any()          {}
func (w *Workflow) Is_process() {}

func GetMapElement(m map[interface{}]interface{}, key string) (value interface{}, err error) {

	for k, v := range m {
		k_str, ok := k.(string)
		if ok {
			if k_str == key {
				value = v
				return
			}
		}
	}
	err = fmt.Errorf("Element \"%s\" not found in map", key)
	return
}

func NewWorkflowEmpty() (w Workflow) {
	w = Workflow{}
	w.Class = string(CWL_Workflow)
	return w
}

func NewWorkflow(original interface{}, cwl_version CWLVersion) (workflow_ptr *Workflow, schemata []CWLType_Type, err error) {

	// convert input map into input array

	original, err = MakeStringMap(original)
	if err != nil {
		err = fmt.Errorf("(NewWorkflow) MakeStringMap returned: %s", err.Error())
		return
	}

	workflow := NewWorkflowEmpty()
	workflow_ptr = &workflow

	switch original.(type) {
	case map[string]interface{}:
		object := original.(map[string]interface{})

		var CwlVersion CWLVersion

		cwl_version_if, ok := object["cwlVersion"]
		if ok {
			//CwlVersion = cwl_version_if.(string)
			var cwl_version_str string
			cwl_version_str, ok = cwl_version_if.(string)
			if !ok {
				err = fmt.Errorf("(NewWorkflow) Could not read CWLVersion (%s)", reflect.TypeOf(cwl_version_if))
				return
			}
			CwlVersion = CWLVersion(cwl_version_str)
		} else {
			CwlVersion = cwl_version
		}

		if CwlVersion == "" {
			fmt.Println("workflow without version:")
			spew.Dump(object)
			err = fmt.Errorf("(NewWorkflow) CwlVersion empty")
			return
		}

		inputs, ok := object["inputs"]
		if ok {
			object["inputs"], err = NewInputParameterArray(inputs, schemata)
			if err != nil {
				err = fmt.Errorf("(NewWorkflow) NewInputParameterArray returned: %s", err.Error())
				return
			}
		}

		outputs, ok := object["outputs"]
		if ok {
			object["outputs"], err = NewWorkflowOutputParameterArray(outputs, schemata)
			if err != nil {
				err = fmt.Errorf("(NewWorkflow) NewWorkflowOutputParameterArray returned: %s", err.Error())
				return
			}
		}

		// convert steps to array if it is a map
		steps, ok := object["steps"]
		if ok {
			logger.Debug(3, "(NewWorkflow) Parsing steps in Workflow")
			var schemata_new []CWLType_Type
			schemata_new, object["steps"], err = CreateWorkflowStepsArray(steps, CwlVersion)
			if err != nil {
				err = fmt.Errorf("(NewWorkflow) CreateWorkflowStepsArray returned: %s", err.Error())
				return
			}
			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}
		} else {
			err = fmt.Errorf("(NewWorkflow) Workflow has no steps ")
			spew.Dump(object)
			return
		}

		requirements, ok := object["requirements"]
		if ok {
			var schemata_new []CWLType_Type
			//fmt.Println("---- Workflow (before CreateRequirementArray) ----")
			//spew.Dump(object)
			object["requirements"], schemata_new, err = CreateRequirementArray(requirements)
			if err != nil {
				fmt.Println("---- Workflow ----")
				spew.Dump(object)
				fmt.Println("---- requirements ----")
				spew.Dump(requirements)
				err = fmt.Errorf("(NewWorkflow) CreateRequirementArray returned: %s", err.Error())
				return
			}
			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}
		}

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
		//fmt.Printf(".....WORKFLOW")
		//spew.Dump(workflow)
		return

	default:

		err = fmt.Errorf("(NewWorkflow) Input type %s can not be parsed", reflect.TypeOf(original))

	}

	return
}
