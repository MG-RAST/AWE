package cwl

import (
	"fmt"
	"reflect"
)

type Named_CWL_object struct {
	CWL_id_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides id
	Value       CWL_object                                                            `yaml:"value,omitempty" bson:"value,omitempty" json:"value,omitempty" mapstructure:"value,omitempty"`
}

func NewNamed_CWL_object(id string, value CWL_object) Named_CWL_object {
	x := Named_CWL_object{Value: value}
	x.Id = id
	return x
}

func NewNamed_CWL_object_from_interface(original interface{}) (x Named_CWL_object, schemata []CWLType_Type, err error) {

	var original_map map[string]interface{}
	var ok bool
	original_map, ok = original.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(NewNamed_CWL_object_from_interface) not a map")
		return
	}

	x = Named_CWL_object{}

	var id interface{}
	id, ok = original_map["id"]
	if !ok {
		err = fmt.Errorf("(NewNamed_CWL_object_from_interface) id not found")
		return
	}
	var id_str string
	id_str, ok = id.(string)
	if !ok {
		err = fmt.Errorf("(NewNamed_CWL_object_from_interface) id not a string")
		return
	}
	x.Id = id_str

	var value interface{}
	value, ok = original_map["value"]
	if !ok {
		err = fmt.Errorf("(NewNamed_CWL_object_from_interface) value not found")
		return
	}

	var obj CWL_object
	obj, schemata, err = New_CWL_object(value, "")
	if err != nil {
		err = fmt.Errorf("(NewNamed_CWL_object_from_interface) New_CWL_object returned: %s", err.Error())
		return
	}

	x.Value = obj

	return
}

type Named_CWL_object_array []Named_CWL_object

func NewNamed_CWL_object_array(original interface{}) (array Named_CWL_object_array, schemata []CWLType_Type, err error) {

	//original, err = makeStringMap(original)
	//if err != nil {
	//	return
	//}

	array = Named_CWL_object_array{}

	switch original.(type) {

	case []interface{}:

		org_a := original.([]interface{})

		for _, element := range org_a {
			var schemata_new []CWLType_Type
			var cwl_object Named_CWL_object
			cwl_object, schemata_new, err = NewNamed_CWL_object_from_interface(element)
			if err != nil {
				err = fmt.Errorf("(NewNamed_CWL_object_array) New_CWL_object returned %s", err.Error())
				return
			}

			array = append(array, cwl_object)

			for i, _ := range schemata_new {
				schemata = append(schemata, schemata_new[i])
			}
		}

		return

	default:
		err = fmt.Errorf("(NewNamed_CWL_object_array), unknown type %s", reflect.TypeOf(original))
	}
	return

}
