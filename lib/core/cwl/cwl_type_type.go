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

		switch context {

		case "WorkflowOutput":

			oas := NewOutputArraySchema()
			oas.Items = []CWLType_Type{CWLType_Type_Basic(base_type_str)}
			result = oas
			return
		default:
			err = fmt.Errorf("(NewCWLType_TypeFromString) context %s not supported yet", context)
		}

		//an_array_schema["items"] = []CWLType_Type{CWLType_Type_Basic(base_type)}

		//result, err = NewCWLType_Type(schemata, an_array_schema, context)
		//if err != nil {
		//	return
		//}

		//return

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

					//spew.Dump(native)
					//panic("done")

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
				result, err = NewCommandInputRecordSchemaFromInterface(native, schemata)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandInputRecordSchemaFromInterface returned: %s", err.Error())
				}
			case "CommandOutput":
				panic("CommandOutput not implemented yet")
			default:
				err = fmt.Errorf("(NewCWLType_Type) context %s unknown", context)
				return
			}

		} else if object_type == "enum" {

			switch context {
			case "Input":
				result, err = NewInputEnumSchemaFromInterface(native)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputEnumSchemaFromInterface returned: %s", err.Error())
				}
				return

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
	case *OutputArraySchema:
		oas_p := native.(*OutputArraySchema)
		result = *oas_p
	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType_Type) type %s unkown", reflect.TypeOf(native))
		return
	}

	return
}

func NewCWLType_TypeArray(native interface{}, schemata []CWLType_Type, context string, pass_schemata_along bool) (result []CWLType_Type, err error) {

	native, err = MakeStringMap(native)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:

		original_map := native.(map[string]interface{})

		var new_type CWLType_Type
		new_type, err = NewCWLType_Type(schemata, original_map, context)

		if err != nil {
			err = fmt.Errorf("(NewCWLType_TypeArray) NewCWLType_Type returns: %s", err.Error())
			return
		}

		result = []CWLType_Type{new_type}
		return

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
			if pass_schemata_along {
				schemata = append(schemata, element_type)
			}
		}

		result = type_array

	default:
		spew.Dump(native)
		err = fmt.Errorf("(NewCWLType_TypeArray) type unknown: %s", reflect.TypeOf(native))

	}
	return
}
