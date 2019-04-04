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
	GetID() string
}

type CWLType_Type_Basic string

func (s CWLType_Type_Basic) Is_Type()            {}
func (s CWLType_Type_Basic) Type2String() string { return string(s) }
func (s CWLType_Type_Basic) GetID() string       { return "" }

func NewCWLType_TypeFromString(schemata []CWLType_Type, native string, context string) (result CWLType_Type, err error) {

	if native == "" {
		err = fmt.Errorf("(NewCWLType_TypeFromString) string is empty")
		return
	}

	if strings.HasPrefix(native, "#") {
		result = NewPointerFromstring(native)

		return
	}

	if !strings.HasSuffix(native, "[]") {

		var ok bool
		result, ok = IsValidType(native)

		if !ok {

			err = fmt.Errorf("(NewCWLType_TypeFromString) type %s unkown", native)
			return
		}

		return
	}

	// is array

	//an_array_schema := make(map[string]interface{})
	//an_array_schema["type"] = "array"

	baseTypeStr := strings.TrimSuffix(native, "[]")
	var ok bool
	_, ok = IsValidType(baseTypeStr)

	if !ok {
		err = fmt.Errorf("(NewCWLType_TypeFromString) base_type %s unkown", baseTypeStr)
		return
	}

	// recurse:
	var baseType CWLType_Type
	baseType, err = NewCWLType_TypeFromString(schemata, baseTypeStr, context)
	if err != nil {
		err = fmt.Errorf("(NewCWLType_TypeFromString) recurisve call of NewCWLType_TypeFromString returned: %s", err.Error())
		return
	}

	switch context {

	case "WorkflowOutput":

		oas := NewOutputArraySchema()
		oas.Items = []CWLType_Type{baseType}
		result = oas
		return
	case "CommandInput":
		cias := NewCommandInputArraySchema()
		result = cias
		return

	default:
		err = fmt.Errorf("(NewCWLType_TypeFromString) context %s not supported yet (baseTypeStr: %s)", context, baseTypeStr)
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

		result, err = NewCWLType_TypeFromString(schemata, native_str, context_p)
		if err != nil {
			err = fmt.Errorf("(NewCWLType_Type) A NewCWLType_TypeFromString returned: %s", err.Error())
		}
		return

	case CWLType_Type_Basic:

		native_tt := native.(CWLType_Type_Basic)
		native_str := string(native_tt)

		result, err = NewCWLType_TypeFromString(schemata, native_str, context_p)
		if err != nil {
			err = fmt.Errorf("(NewCWLType_Type) B NewCWLType_TypeFromString returned: %s", err.Error())
		}
		return

	case map[string]interface{}:
		nativeMap := native.(map[string]interface{})

		object_type, has_type := nativeMap["type"]
		if !has_type {
			fmt.Println("(NewCWLType_Type) nativeMap:")
			spew.Dump(nativeMap)

			err = fmt.Errorf("(NewCWLType_Type) map object has not field \"type\" (context: %s)", context_p)
			return
		}
		switch object_type {
		case "array":

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
				err = fmt.Errorf("(NewCWLType_Type) array, context_p %s unknown", context_p)
				return
			}

		case "record":

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
				err = fmt.Errorf("(NewCWLType_Type) record, context %s unknown", context_p)
				return
			}

		case "enum":

			switch context_p {
			case "Input":
				result, err = NewInputEnumSchemaFromInterface(native, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewInputEnumSchemaFromInterface returned: %s", err.Error())
				}
				return

			case "CommandInput":
				result, err = NewCommandInputEnumSchemaFromInterface(native, context)
				if err != nil {
					err = fmt.Errorf("(NewCWLType_Type) NewCommandInputEnumSchemaFromInterface returned: %s", err.Error())
				}
				return
			default:
				err = fmt.Errorf("(NewCWLType_Type) enum, context_p %s unknown", context_p)
				return
			}
		default:

			fmt.Println("(NewCWLType_Type) nativeMap:")
			spew.Dump(nativeMap)
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
	case *CommandOutputArraySchema:
		coas_p := native.(*CommandOutputArraySchema)
		result = coas_p
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

		nativeArray := native.([]string)
		type_array := []CWLType_Type{}
		for _, element_str := range nativeArray {
			var element_type CWLType_Type
			element_type, err = NewCWLType_TypeFromString(schemata, element_str, context_p)
			if err != nil {
				return
			}
			type_array = append(type_array, element_type)
		}

		result = type_array

	case []interface{}:

		nativeArray := native.([]interface{})
		type_array := []CWLType_Type{}
		for _, element_str := range nativeArray {
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
