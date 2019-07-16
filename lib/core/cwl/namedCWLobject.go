package cwl

import (
	"fmt"
	"reflect"
)

type NamedCWLObject struct {
	CWL_id_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides id
	Value       CWLObject                                                             `yaml:"value,omitempty" bson:"value,omitempty" json:"value,omitempty" mapstructure:"value,omitempty"`
}

//type NamedCWLObject_array []NamedCWLObject

func NewNamedCWLObject(id string, value CWLObject) NamedCWLObject {
	x := NamedCWLObject{Value: value}
	x.Id = id
	return x
}

func NewNamedCWLObject_from_interface(original interface{}, context *WorkflowContext) (x NamedCWLObject, schemata []CWLType_Type, err error) {

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	switch original.(type) {
	case map[string]interface{}:
		var original_map map[string]interface{}
		var ok bool
		original_map, ok = original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewNamedCWLObject_from_interface) not a map (%s)", reflect.TypeOf(original))
			return
		}

		x = NamedCWLObject{}

		var id interface{}
		id, ok = original_map["id"]
		if !ok {
			err = fmt.Errorf("(NewNamedCWLObject_from_interface) id not found")
			return
		}
		var id_str string
		id_str, ok = id.(string)
		if !ok {
			err = fmt.Errorf("(NewNamedCWLObject_from_interface) id not a string")
			return
		}
		x.Id = id_str

		var value interface{}
		value, ok = original_map["value"]
		if !ok {
			err = fmt.Errorf("(NewNamedCWLObject_from_interface) value not found")
			return
		}

		var obj CWLObject
		obj, schemata, err = NewCWLObject(value, "", nil, context)
		if err != nil {
			err = fmt.Errorf("(NewNamedCWLObject_from_interface) NewCWLObject returned: %s", err.Error())
			return
		}

		x.Value = obj
		return
	default:
		err = fmt.Errorf("(NewNamedCWLObject_from_interface) not a map (%s)", reflect.TypeOf(original))
	}
	return
}

func NewNamedCWLObject_array(original interface{}, context *WorkflowContext) (array []NamedCWLObject, schemata []CWLType_Type, err error) {

	//original, err = makeStringMap(original)
	//if err != nil {
	//	return
	//}

	if original == nil {
		err = fmt.Errorf("(NewNamedCWLObject_array) original == nil")
		return
	}

	array = []NamedCWLObject{}

	switch original.(type) {

	case []interface{}:

		org_a := original.([]interface{})

		for _, element := range org_a {
			var schemataNew []CWLType_Type
			var cwl_object NamedCWLObject
			cwl_object, schemataNew, err = NewNamedCWLObject_from_interface(element, context)
			if err != nil {
				err = fmt.Errorf("(NewNamedCWLObject_array) NewCWLObject returned %s", err.Error())
				return
			}

			array = append(array, cwl_object)

			for i, _ := range schemataNew {
				schemata = append(schemata, schemataNew[i])
			}
		}

		return

	default:
		err = fmt.Errorf("(NewNamedCWLObject_array), unknown type %s", reflect.TypeOf(original))
	}
	return

}
