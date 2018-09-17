package cwl

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
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

	if strings.HasPrefix(native, "#") {
		result = NewPointerFromstring(native)

		return
	}

	if strings.HasSuffix(native, "[]") {

		// is array

		//an_array_schema := make(map[string]interface{})
		//an_array_schema["type"] = "array"

		base_type_str := strings.TrimSuffix(native, "[]")
		var ok bool
		_, ok = IsValidType(base_type_str)

		if !ok {
			err = fmt.Errorf("(NewCWLType_TypeFromString) base_type %s unkown", base_type_str)
			return
		}

		// recurse:
		var base_type CWLType_Type
		base_type, err = NewCWLType_TypeFromString(schemata, base_type_str, context)
		if err != nil {
			err = fmt.Errorf("(NewCWLType_TypeFromString) recurisve call of NewCWLType_TypeFromString returned: %s", err.Error())
			return
		}

		switch context {

		case "WorkflowOutput":

			oas := NewOutputArraySchema()
			oas.Items = []CWLType_Type{base_type}
			result = oas
			return
		default:
			err = fmt.Errorf("(NewCWLType_TypeFromString) context %s not supported yet", context)
		}

	}

	result, ok := IsValidType(native)

	if !ok {

		//for _, schema := range schemata {
		//	id := schema.GetId()
		//	fmt.Printf("found id: \"%s\"\n", id)
		//	if id == native {
		//		result = schema
		//		return
		//	}

		//}

		err = fmt.Errorf("(NewCWLType_TypeFromString) type %s unkown", native)
		return
	}

	return
}

func NewCWLType_Type(schemata []CWLType_Type, native interface{}, context_p string, context *WorkflowContext) (result CWLType_Type, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case string:
		native_str := native.(string)

		return NewCWLType_TypeFromString(schemata, native_str, context_p)

	case CWLType_Type_Basic:

		native_tt := native.(CWLType_Type_Basic)
		native_str := string(native_tt)

		return NewCWLType_TypeFromString(schemata, native_str, context_p)

	case map[string]interface{}:
		native_map := native.(map[string]interface{})

		object_type, has_type := native_map["type"]
		if !has_type {
			fmt.Println("(NewCWLType_Type) native_map:")
			spew.Dump(native_map)

			err = fmt.Errorf("(NewCWLType_Type) map object has not field \"type\" (context: %s)", context_p)
			return
		}
		if object_type == "array" {

			switch context_p {
			case "Input":
				result, err = NewInputArraySchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputArraySchemaFromInterface returned: %s", err.Error())
				}
				return
			case "Output":
				result, err = NewOutputArraySchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewOutputArraySchemaFromInterface returned: %s", err.Error())
				}
				return
			case "CommandOutput": // CommandOutputRecordSchema | CommandOutputEnumSchema | CommandOutputArraySchema
				result, err = NewCommandOutputArraySchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandOutputArraySchemaFromInterface returned: %s", err.Error())
				}
				return
			case "CommandInput":
				result, err = NewCommandInputArraySchemaFromInterface(native, schemata, context)
				if err != nil {

					err = fmt.Errorf("(NewCWLType_Type) NewCommandInputArraySchemaFromInterface returned: %s", err.Error())
				}
				return
			case "WorkflowOutput":
				result, err = NewOutputArraySchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewWorkflowOutputOutputArraySchemaFromInterface returned: %s", err.Error())
				}
				return
			default:
				err = fmt.Errorf("(NewCWLType_Type) array, context %s unknown", context)
				return
			}

		} else if object_type == "record" {

			switch context_p {
			case "Input": // InputRecordSchema
				result, err = NewInputRecordSchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputRecordSchemaFromInterface returned: %s", err.Error())
				}
				return
			case "CommandInput":
				result, err = NewCommandInputRecordSchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandInputRecordSchemaFromInterface returned: %s", err.Error())
				}
			case "CommandOutput":

				result, err = NewCommandOutputRecordSchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandOutputRecordSchemaFromInterface returned: %s", err.Error())
				}

			case "Output":

				result, err = NewOutputRecordSchemaFromInterface(native, schemata, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewOutputRecordSchemaFromInterface returned: %s", err.Error())
				}

			default:
				err = fmt.Errorf("(NewCWLType_Type) record, context %s unknown", context)
				return
			}

		} else if object_type == "enum" {

			switch context_p {
			case "Input":
				result, err = NewInputEnumSchemaFromInterface(native, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputEnumSchemaFromInterface returned: %s", err.Error())
				}
				return

			default:
				err = fmt.Errorf("(NewCWLType_Type) enum, context %s unknown", context)
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
	case *OutputArraySchema:
		oas_p := native.(*OutputArraySchema)
		result = *oas_p
	case *OutputRecordSchema:
		ors_p := native.(*OutputRecordSchema)
		result = ors_p
	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType_Type) type %s unkown", reflect.TypeOf(native))
		return
	}

	return
}

func NewCWLType_TypeArray(native interface{}, schemata []CWLType_Type, context_p string, pass_schemata_along bool, context *WorkflowContext) (result []CWLType_Type, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:

		original_map := native.(map[string]interface{})

		var new_type CWLType_Type
		new_type, err = NewCWLType_Type(schemata, original_map, context_p, context)

		if err != nil {
			err = fmt.Errorf("(NewCWLType_TypeArray) NewCWLType_Type returns: %s", err.Error())
			return
		}

		result = []CWLType_Type{new_type}
		return

	case string:
		native_str := native.(string)

		var a_type CWLType_Type
		a_type, err = NewCWLType_TypeFromString(schemata, native_str, context_p)
		if err != nil {
			return
		}
		result = []CWLType_Type{a_type}

	case []string:

		native_array := native.([]string)
		type_array := []CWLType_Type{}
		for _, element_str := range native_array {
			var element_type CWLType_Type
			element_type, err = NewCWLType_TypeFromString(schemata, element_str, context_p)
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
			element_type, err = NewCWLType_Type(schemata, element_str, context_p, context)
			if err != nil {
				return
			}
			type_array = append(type_array, element_type)
			if pass_schemata_along {
				schemata = append(schemata, element_type)
			}
		}

		result = type_array

	case []CWLType_Type:
		result = native.([]CWLType_Type)
	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType_TypeArray) type unknown: %s", reflect.TypeOf(native))

	}
	return
}
