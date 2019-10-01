package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// EnvironmentDef _
type EnvironmentDef struct {
	EnvName  string      `yaml:"envName,omitempty" bson:"envName,omitempty" json:"envName,omitempty" mapstructure:"envName,omitempty"`
	EnvValue interface{} `yaml:"envValue,omitempty" bson:"envValue,omitempty" json:"envValue,omitempty" mapstructure:"envValue,omitempty"`
}

// NewEnvironmentDefFromInterface _
func NewEnvironmentDefFromInterface(original interface{}, name string) (enfDev EnvironmentDef, err error) {

	err = mapstructure.Decode(original, &enfDev)
	if err != nil {
		err = fmt.Errorf("(NewEnvironmentDefFromInterface) mapstructure.Decode returned: %s", err.Error())
		return
	}

	if enfDev.EnvName == "" {
		enfDev.EnvName = name
	}

	return
}

// GetEnfDefArray _
func GetEnfDefArray(original interface{}, context *WorkflowContext) (array []EnvironmentDef, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	array = []EnvironmentDef{}

	switch original.(type) {
	case []interface{}:
		originalArray := original.([]interface{})
		for i := range originalArray {
			var enfDev EnvironmentDef
			enfDev, err = NewEnvironmentDefFromInterface(originalArray[i], "")
			if err != nil {
				err = fmt.Errorf("(GetEnfDefArray) NewEnvironmentDefFromInterface returned: %s", err.Error())
				return
			}

			array = append(array, enfDev)

		}
		return
	case map[string]interface{}:
		originalMap := original.(map[string]interface{})
		for key, value := range originalMap {
			var enfDev EnvironmentDef

			enfDev.EnvName = key
			enfDev.EnvValue = value

			array = append(array, enfDev)

		}
		return

	default:
		err = fmt.Errorf("(GetEnfDefArray) type %s not supported", reflect.TypeOf(original))
	}

	return
}

func (d *EnvironmentDef) Evaluate(inputs interface{}, context *WorkflowContext) (err error) {

	if inputs == nil {
		err = fmt.Errorf("(EnvironmentDef/Evaluate) no inputs")
		return
	}

	//*** d.EnvValue
	var ok bool
	var entry_expr Expression
	entry_expr, ok = d.EnvValue.(Expression)
	if ok {
		var new_value interface{}
		new_value, err = entry_expr.EvaluateExpression(nil, inputs, context)
		if err != nil {
			err = fmt.Errorf("(EnvironmentDef/Evaluate) EvaluateExpression returned: %s", err.Error())
			return
		}

		// verify return type:
		switch new_value.(type) {
		case String, string:
			// valid returns
			d.EnvValue = new_value
		default:
			err = fmt.Errorf("(EnvironmentDef/Evaluate) EvaluateExpression returned type %s, this is not expected", reflect.TypeOf(new_value))
			return

		}
	}

	return

}
