package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/Workflow.html#SchemaDefRequirement
type SchemaDefRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
	Types           []interface{} `yaml:"types,omitempty" json:"types,omitempty" bson:"types,omitempty"` // array<InputRecordSchema | InputEnumSchema | InputArraySchema>
}

func (c SchemaDefRequirement) GetId() string { return "None" }

func NewSchemaDefRequirement(original interface{}) (r *SchemaDefRequirement, schemata []CWLType_Type, err error) {

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	original_map, ok := original.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewSchemaDefRequirement) type error, got: %s", reflect.TypeOf(original))
		return
	}

	types, has_types := original_map["types"]
	if has_types {

		//var array []CWLType_Type
		schemata, err = NewCWLType_TypeArray(types, []CWLType_Type{}, "Input", true)
		if err != nil {
			return
		}
		original_map["types"] = schemata
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

func GetSchemaDefRequirement(original interface{}) (r *SchemaDefRequirement, schemata []CWLType_Type, ok bool, err error) {
	original, err = MakeStringMap(original)
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

			r, schemata, err = NewSchemaDefRequirement(original_array[i])
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
