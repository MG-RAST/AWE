package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type EnvironmentDef struct {
	EnvName  string      `yaml:"envName,omitempty" bson:"envName,omitempty" json:"envName,omitempty" mapstructure:"envName,omitempty"`
	EnvValue interface{} `yaml:"envValue,omitempty" bson:"envValue,omitempty" json:"envValue,omitempty" mapstructure:"envValue,omitempty"`
}

func NewEnvironmentDefFromInterface(original interface{}) (enfDev EnvironmentDef, err error) {

	err = mapstructure.Decode(original, &enfDev)
	if err != nil {
		err = fmt.Errorf("(NewEnvironmentDefFromInterface) mapstructure.Decode returned: %s", err.Error())
		return
	}
	return
}

func GetEnfDefArray(original interface{}) (array []EnvironmentDef, err error) {

	array = []EnvironmentDef{}

	switch original.(type) {
	case []interface{}:
		original_array := original.([]interface{})
		for i, _ := range original_array {
			var enfDev EnvironmentDef
			enfDev, err = NewEnvironmentDefFromInterface(original_array[i])
			if err != nil {
				err = fmt.Errorf("(GetEnfDefArray) NewEnvironmentDefFromInterface returned: %s", err.Error())
				return
			}

			array = append(array, enfDev)

		}
		return
	}

	err = fmt.Errorf("(GetEnfDefArray) type %s not supported", reflect.TypeOf(original))

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
