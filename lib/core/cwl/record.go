package cwl

import (
	"encoding/json"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"reflect"
)

type Record struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Fields       []CWLType `yaml:"fields,omitempty" json:"fields,omitempty" bson:"fields,omitempty"`
}

func (r *Record) GetClass() string { return "record" }

func (r *Record) Is_CWL_minimal() {}

//func (r *Record) Is_CommandInputParameterType()  {}
//func (r *Record) Is_CommandOutputParameterType() {}

func NewRecord(id string, native interface{}) (record *Record, err error) {

	native, err = MakeStringMap(native)
	if err != nil {
		return
	}

	record = &Record{}

	if id != "" {
		err = fmt.Errorf("not sure what to do with id here")
		return
	}

	switch native.(type) {
	case map[string]interface{}:

		//fmt.Println("Got a record:")
		//spew.Dump(native)

		native_map, _ := native.(map[string]interface{})
		for key_str, value := range native_map {

			value_cwl, xerr := NewCWLType(key_str, value)
			if xerr != nil {
				err = xerr
				return
			}
			record.Fields = append(record.Fields, value_cwl)
		}
		return

	default:
		fmt.Println("Unknown Record:")
		spew.Dump(native)
		err = fmt.Errorf("(NewRecord) Unknown type: %s (%s)", reflect.TypeOf(native), spew.Sdump(native))
		return
	}

	return
}

func (c *Record) String() string {
	bytes, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}
	return string(bytes[:])
}
