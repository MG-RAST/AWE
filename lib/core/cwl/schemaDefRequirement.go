package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// SchemaDefRequirement http://www.commonwl.org/v1.0/Workflow.html#SchemaDefRequirement
type SchemaDefRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"` // provides class
	Types           []interface{}                                                         `yaml:"types,omitempty" json:"types,omitempty" bson:"types,omitempty"` // array<InputRecordSchema | InputEnumSchema | InputArraySchema>
}

// GetID _
func (c SchemaDefRequirement) GetID() string { return "None" }

// NewSchemaDefRequirement _
func NewSchemaDefRequirement(original interface{}, context *WorkflowContext) (r *SchemaDefRequirement, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	originalMap, ok := original.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewSchemaDefRequirement) type error, got: %s", reflect.TypeOf(original))
		return
	}

	var schemata []CWLType_Type

	types, hasTypes := originalMap["types"]
	if hasTypes {

		//var array []CWLType_Type
		schemata, err = NewCWLType_TypeArray(types, []CWLType_Type{}, "Input", true, context)
		if err != nil {
			return
		}
		originalMap["types"] = schemata
		if context != nil {
			err = context.AddSchemata(schemata, true)
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

// GetSchemaDefRequirement _
func GetSchemaDefRequirement(original interface{}, context *WorkflowContext) (r *SchemaDefRequirement, ok bool, err error) {
	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case []interface{}:

		originalArray := original.([]interface{})
		ok = false
		for i := range originalArray {

			var class string
			class, err = GetClass(originalArray[i])
			if err != nil {
				err = fmt.Errorf("(GetSchemaDefRequirement) GetClass returned: %s", err.Error())
				return
			}

			if class != "SchemaDefRequirement" {
				continue
			}

			r, err = NewSchemaDefRequirement(originalArray[i], context)
			if err != nil {
				err = fmt.Errorf("(GetSchemaDefRequirement) NewSchemaDefRequirement returned: %s", err.Error())
				return
			}
			ok = true
			return
		}

	case map[string]interface{}:

		var class string
		class, err = GetClass(original)
		if err != nil {
			err = fmt.Errorf("(GetSchemaDefRequirement) GetClass returned: %s", err.Error())
			return
		}

		if class != "SchemaDefRequirement" {
			return
		}

		r, err = NewSchemaDefRequirement(original, context)
		if err != nil {
			err = fmt.Errorf("(GetSchemaDefRequirement) NewSchemaDefRequirement returned: %s", err.Error())
			return
		}
		ok = true

	default:
		err = fmt.Errorf("(GetSchemaDefRequirement) type %s unknown", reflect.TypeOf(original))

	}

	return

}
