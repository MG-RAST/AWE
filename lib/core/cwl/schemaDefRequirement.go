package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/Workflow.html#SchemaDefRequirement
type SchemaDefRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"` // provides class
	Types           []interface{}                                                         `yaml:"types,omitempty" json:"types,omitempty" bson:"types,omitempty"` // array<InputRecordSchema | InputEnumSchema | InputArraySchema>
}

func (c SchemaDefRequirement) GetId() string { return "None" }

func NewSchemaDefRequirement(original interface{}, context *WorkflowContext) (r *SchemaDefRequirement, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	original_map, ok := original.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewSchemaDefRequirement) type error, got: %s", reflect.TypeOf(original))
		return
	}

	var schemata []CWLType_Type

	types, has_types := original_map["types"]
	if has_types {

		//var array []CWLType_Type
		schemata, err = NewCWLType_TypeArray(types, []CWLType_Type{}, "Input", true, context)
		if err != nil {
			return
		}
		original_map["types"] = schemata
		if context != nil {
			err = context.AddSchemata(schemata)
			if err != nil {
				err = fmt.Errorf("(NewSchemaDefRequirement) context.AddSchemata returned: %s", err.Error())
			}
		}
	}

	var requirement SchemaDefRequirement
	r = &requirement

	err = mapstructure.Decode(original, &requirement)
	if err != nil {
		return
	}

	requirement.Class = "SchemaDefRequirement"

	//spew.Dump(requirement)

	return
}

func GetSchemaDefRequirement(original interface{}, context *WorkflowContext) (r *SchemaDefRequirement, ok bool, err error) {
	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case []interface{}:

		original_array := original.([]interface{})

		for i, _ := range original_array {

			var class string
			class, err = GetClass(original_array[i])
			if err != nil {
				err = fmt.Errorf("(GetSchemaDefRequirement) GetClass returned: %s", err.Error())
				return
			}

			if class != "SchemaDefRequirement" {
				continue
			}

			r, err = NewSchemaDefRequirement(original_array[i], context)
			if err != nil {
				err = fmt.Errorf("(GetSchemaDefRequirement) NewSchemaDefRequirement returned: %s", err.Error())
				return
			}

		}

	default:
		err = fmt.Errorf("(GetSchemaDefRequirement) type %s unknown", reflect.TypeOf(original))

	}

	return

}
