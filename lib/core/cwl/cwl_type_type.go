package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"reflect"
	"strings"
)

type CWLType_Type interface {
	Is_Type()
	Type2String() string
	GetId() string
}

type CWLType_Type_Basic string

func (s CWLType_Type_Basic) Is_Type()            {}
func (s CWLType_Type_Basic) Type2String() string { return string(s) }
func (s CWLType_Type_Basic) GetId() string       { return "" }

func NewCWLType_TypeFromString(schemata []CWLType_Type, native string, context string) (result CWLType_Type, err error) {

	if native == "" {
		err = fmt.Errorf("(NewCWLType_TypeFromString) string is empty")
		return
	}

	if strings.HasSuffix(native, "[]") {

		// is array

		an_array_schema := make(map[string]interface{})
		an_array_schema["type"] = "array"

		base_type := strings.TrimSuffix(native, "[]")

		var ok bool
		result, ok = IsValidType(base_type)

		if !ok {
			err = fmt.Errorf("(NewCWLType_TypeFromString) base_type %s unkown", native)
			return
		}

		an_array_schema["items"] = []CWLType_Type{CWLType_Type_Basic(base_type)}

		result, err = NewCWLType_Type(schemata, an_array_schema, context)
		if err != nil {
			return
		}

		return

	}

	result, ok := IsValidType(native)

	if !ok {

		for _, schema := range schemata {
			id := schema.GetId()
			fmt.Printf("found id: \"%s\"\n", id)
			if id == native {
				result = schema
				return
			}

		}

		err = fmt.Errorf("(NewCWLType_TypeFromString) type %s unkown", native)
		return
	}

	return
}

func NewCWLType_Type(schemata []CWLType_Type, native interface{}, context string) (result CWLType_Type, err error) {

	native, err = MakeStringMap(native)
	if err != nil {
		return
	}

	switch native.(type) {
	case string:
		native_str := native.(string)

		return NewCWLType_TypeFromString(schemata, native_str, context)

	case CWLType_Type_Basic:

		native_tt := native.(CWLType_Type_Basic)
		native_str := string(native_tt)

		return NewCWLType_TypeFromString(schemata, native_str, context)

	case map[string]interface{}:
		native_map := native.(map[string]interface{})

		object_type, has_type := native_map["type"]
		if !has_type {
			err = fmt.Errorf("(NewCWLType_Type) map object has not field \"type\"")
			return
		}
		if object_type == "array" {

			switch context {
			case "Input":
				result, err = NewInputArraySchemaFromInterface(native, schemata)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputArraySchemaFromInterface returned: %s", err.Error())
				}
				return
			case "CommandOutput": // CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema
				result, err = NewCommandOutputArraySchemaFromInterface(native, schemata)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandOutputArraySchemaFromInterface returned: %s", err.Error())
				}
				return
			case "CommandInput":
				result, err = NewCommandInputArraySchemaFromInterface(native, schemata)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandInputArraySchemaFromInterface returned: %s", err.Error())
				}
				return
			case "WorkflowOutput":
				result, err = NewOutputArraySchemaFromInterface(native, schemata)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewWorkflowOutputOutputArraySchemaFromInterface returned: %s", err.Error())
				}
				return
			default:
				err = fmt.Errorf("(NewCWLType_Type) context %s unknown", context)
				return
			}

		} else if object_type == "record" {

			switch context {
			case "Input": // InputRecordSchema
				result, err = NewInputRecordSchemaFromInterface(native, schemata)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputRecordSchemaFromInterface returned: %s", err.Error())
				}
				return
			case "CommandInput":

			case "CommandOutput":

			default:
				err = fmt.Errorf("(NewCWLType_Type) context %s unknown", context)
				return
			}

		} else {

			fmt.Println("(NewCWLType_Type) native_map:")
			spew.Dump(native_map)
			err = fmt.Errorf("(NewCWLType_Type) object_type %s not supported yet", object_type)
			return
		}
	case OutputArraySchema:
		result = native.(OutputArraySchema)
		return
	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType_Type) type %s unkown", reflect.TypeOf(native))
		return
	}

	return
}

func NewCWLType_TypeArray(native interface{}, schemata []CWLType_Type, context string) (result []CWLType_Type, err error) {

	switch native.(type) {
	case map[string]interface{}:

		original_map := native.(map[string]interface{})

		type_value, has_type := original_map["type"]
		if !has_type {
			err = fmt.Errorf("(NewCWLType_TypeArray) type not found")
			return
		}

		if type_value == "array" {
			var array_schema *CommandOutputArraySchema
			array_schema, err = NewCommandOutputArraySchemaFromInterface(original_map, schemata)
			if err != nil {
				return
			}
			result = []CWLType_Type{array_schema}
			return
		} else {
			err = fmt.Errorf("(NewCWLType_TypeArray) type %s unknown", type_value)
			return
		}

	case string:
		native_str := native.(string)

		var a_type CWLType_Type
		a_type, err = NewCWLType_TypeFromString(schemata, native_str, context)
		if err != nil {
			return
		}
		result = []CWLType_Type{a_type}

	case []string:

		native_array := native.([]string)
		type_array := []CWLType_Type{}
		for _, element_str := range native_array {
			var element_type CWLType_Type
			element_type, err = NewCWLType_TypeFromString(schemata, element_str, context)
			if err != nil {
				return
			}
			type_array = append(type_array, element_type)
		}

		result = type_array

	case []interface{}:

		native_array := native.([]interface{})
		type_array := []CWLType_Type{}
		for _, element_str := range native_array {
			var element_type CWLType_Type
			element_type, err = NewCWLType_Type(schemata, element_str, context)
			if err != nil {
				return
			}
			type_array = append(type_array, element_type)
		}

		result = type_array

	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType_TypeArray) type unknown: %s", reflect.TypeOf(native))

	}
	return
}
