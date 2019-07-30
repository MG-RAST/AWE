package cwl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
)

type NamedCWLType struct {
	CWL_id_Impl `yaml:",inline" bson:",inline" json:",inline" mapstructure:",squash"` // provides id
	Value       CWLType                                                               `yaml:"value,omitempty" bson:"value,omitempty" json:"value,omitempty" mapstructure:"value,omitempty"`
}

func NewNamedCWLType(id string, value CWLType) NamedCWLType {

	x := NamedCWLType{Value: value}

	x.ID = id

	return x
}

func NewNamedCWLTypeFromInterface(native interface{}, context *WorkflowContext) (cwl_obj_named NamedCWLType, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:
		named_thing_map := native.(map[string]interface{})

		named_thing_value, ok := named_thing_map["value"]
		if !ok {
			fmt.Println("Expected NamedCWLType field \"value\" , but not found:")
			spew.Dump(native)
			err = fmt.Errorf("(NewNamedCWLTypeFromInterface) Expected NamedCWLType field \"value\" , but not found: %s", spew.Sdump(native))
			return
		}

		named_thing_value, err = MakeStringMap(named_thing_value, context)
		if err != nil {
			return
		}

		var id string
		var id_interface interface{}
		id_interface, ok = named_thing_map["id"]
		if ok {
			id, ok = id_interface.(string)
			if !ok {
				err = fmt.Errorf("(NewNamedCWLTypeFromInterface) id has wrong type")
				return
			}
		} else {
			id = ""
		}

		var cwl_obj CWLType
		cwl_obj, err = NewCWLType(id, "", named_thing_value, context)
		if err != nil {
			err = fmt.Errorf("(NewNamedCWLTypeFromInterface) B NewCWLType returns: %s", err.Error())
			return
		}

		//var cwl_obj_named NamedCWLType
		cwl_obj_named = NewNamedCWLType(id, cwl_obj)

	default:
		fmt.Println("Expected a NamedCWLType, but did not find a map:")
		spew.Dump(native)
		err = fmt.Errorf("(NewNamedCWLTypeFromInterface) Expected a NamedCWLType, but did not find a map")
		return
	}
	return
}
