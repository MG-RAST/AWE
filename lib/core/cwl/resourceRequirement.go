package cwl

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/draft-3/CommandLineTool.html#ResourceRequirement

type ResourceRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
	CoresMin        interface{} `yaml:"coresMin,omitempty" bson:"coresMin,omitempty" json:"coresMin,omitempty"`    //long | string | Expression
	CoresMax        interface{} `yaml:"coresMax,omitempty" bson:"coresMax,omitempty" json:"coresMax,omitempty"`    // int | string | Expression
	RamMin          interface{} `yaml:"ramMin,omitempty" bson:"ramMin,omitempty" json:"ramMin,omitempty"`          //long | string | Expression
	RamMax          interface{} `yaml:"ramMax,omitempty" bson:"ramMax,omitempty" json:"ramMax,omitempty"`          //long | string | Expression
	TmpdirMin       interface{} `yaml:"tmpdirMin,omitempty" bson:"tmpdirMin,omitempty" json:"tmpdirMin,omitempty"` //long | string | Expression
	TmpdirMax       interface{} `yaml:"tmpdirMax,omitempty" bson:"tmpdirMax,omitempty" json:"tmpdirMax,omitempty"` //long | string | Expression
	OutdirMin       interface{} `yaml:"outdirMin,omitempty" bson:"outdirMin,omitempty" json:"outdirMin,omitempty"` //long | string | Expression
	OutdirMax       interface{} `yaml:"outdirMax,omitempty" bson:"outdirMax,omitempty" json:"outdirMax,omitempty"` //long | string | Expression
}

func (r ResourceRequirement) GetId() string { return "None" }

func (r *ResourceRequirement) Evaluate(inputs interface{}, context *WorkflowContext) (err error) {

	if inputs == nil {
		err = fmt.Errorf("(ResourceRequirement/Evaluate) no inputs")
		return
	}
	spew.Dump(*r)
	fields := []string{"coresMin", "coresMax", "ramMin", "ramMax", "tmpdirMin", "tmpdirMax", "outdirMin", "outdirMax"}

	for _, field := range fields {
		var value reflect.Value
		value = reflect.ValueOf(r).Elem().FieldByName(strings.Title(field))
		value_if := value.Interface()
		if value_if != nil {
			switch value_if.(type) {
			case string:
				original_str := value_if.(string)

				var new_value interface{}

				var original_expr *Expression
				original_expr = NewExpressionFromString(original_str)

				new_value, err = original_expr.EvaluateExpression(nil, inputs, context)
				//value_if = new_value
				value.Set(reflect.ValueOf(new_value).Elem())
				//fmt.Printf("(ResourceRequirement/Evaluate)EvaluateExpression returned: %v\n", new_value)

			case int:

			case int64:

			default:
				err = fmt.Errorf("(ResourceRequirement/Evaluate) type invalid for field %s", reflect.TypeOf(value_if))
				return
			}
			//fmt.Println(reflect.TypeOf(value_if))
			//fmt.Println(value_if)
		}

	}
	//spew.Dump(*r)

	//panic("done")
	// value.SetString( )

	// original_str := original.(string)

	//
	return

}

func NewResourceRequirement(original interface{}, inputs interface{}, context *WorkflowContext) (r *ResourceRequirement, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	fields := []string{"coresMin", "coresMax", "ramMin", "ramMax", "tmpdirMin", "tmpdirMax", "outdirMin", "outdirMax"}
	_ = fields
	//var requirement ResourceRequirement

	r = &ResourceRequirement{}
	err = mapstructure.Decode(original, r)

	// this is just type checking
	for _, field := range fields {
		var value reflect.Value
		value = reflect.ValueOf(r).Elem().FieldByName(strings.Title(field))
		value_if := value.Interface()
		if value_if != nil {
			switch value_if.(type) {
			case string:

			case int:

			case int64:

			case float64:

			default:
				err = fmt.Errorf("(NewResourceRequirement) type invalid for field %s", reflect.TypeOf(value_if))
				return
			}
			//fmt.Println(reflect.TypeOf(value_if))
			//fmt.Println(value_if)
		}

	}

	r.Class = "ResourceRequirement"
	return
}
