package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// CommandOutputParameter https://www.commonwl.org/v1.0/CommandLineTool.html#CommandOutputParameter
type CommandOutputParameter struct {
	OutputParameter `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Id, Label, SecondaryFiles, Format, Streamable, OutputBinding, Type

	Description string `yaml:"description,omitempty" bson:"description,omitempty" json:"description,omitempty" mapstructure:"description,omitempty"`
}

// NewCommandOutputParameter _
func NewCommandOutputParameter(original interface{}, thisID string, schemata []CWLType_Type, context *WorkflowContext) (cmdOutputParameter *CommandOutputParameter, err error) {

	cmdOutputParameter = &CommandOutputParameter{}

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	var op *OutputParameter
	op, err = NewOutputParameterFromInterface(original, thisID, schemata, "CommandOutput", context)
	if err != nil {
		err = fmt.Errorf("(NewCommandOutputParameter) NewOutputParameterFromInterface returns %s", err.Error())
		return
	}

	switch original.(type) {

	case string:
		//fmt.Println("(NewCommandOutputParameter) string")
		cmdOutputParameter.OutputParameter = *op

	case map[string]interface{}:
		//fmt.Println("(NewCommandOutputParameter) map")
		err = mapstructure.Decode(original, cmdOutputParameter)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputParameter) mapstructure returned: %s", err.Error())
			return
		}

		cmdOutputParameter.OutputParameter = *op
	default:
		spew.Dump(original)
		err = fmt.Errorf("NewCommandOutputParameter, unknown type %s", reflect.TypeOf(original))
	}
	//spew.Dump(new_array)
	return
}

// NewCommandOutputParameterArray _
func NewCommandOutputParameterArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (copa []interface{}, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case map[string]interface{}:
		//fmt.Println("(NewCommandOutputParameterArray) map")
		originalMap := original.(map[string]interface{})

		copa = []interface{}{}
		for key, element := range originalMap {
			//fmt.Printf("(NewCommandOutputParameterArray) map element %s\n", key)
			//spew.Dump(element)
			var cop *CommandOutputParameter
			cop, err = NewCommandOutputParameter(element, key, schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewCommandOutputParameterArray) c NewCommandOutputParameter returns: %s", err.Error())
				return
			}
			copa = append(copa, *cop)
		}

	case []interface{}:
		//fmt.Println("(NewCommandOutputParameterArray) array")
		copa = []interface{}{}

		originalArray := original.([]interface{})

		for _, element := range originalArray {
			//fmt.Println("(NewCommandOutputParameterArray) array element")
			var elementStr string
			var ok bool
			elementStr, ok = element.(string)

			if ok {
				//fmt.Println("(NewCommandOutputParameterArray) array element is a string")
				var resultArray []CWLType_Type
				resultArray, err = NewCWLType_TypeFromString(schemata, elementStr, "CommandOutput", context)
				if err != nil {
					err = fmt.Errorf("(NewCommandOutputParameterArray) NewCWLType_TypeFromString returns: %s", err.Error())
					return
				}
				for result := range resultArray {
					copa = append(copa, result)
				}

				continue
			}
			//fmt.Println("(NewCommandOutputParameterArray) array element is NOT a string")
			var cop *CommandOutputParameter
			cop, err = NewCommandOutputParameter(element, "", schemata, context)
			if err != nil {
				err = fmt.Errorf("(NewCommandOutputParameterArray) b NewCommandOutputParameter returns: %s", err.Error())
				return
			}
			copa = append(copa, *cop)
		}

	default:
		err = fmt.Errorf("NewCommandOutputParameterArray, unknown type %s", reflect.TypeOf(original))
	}
	return

}
