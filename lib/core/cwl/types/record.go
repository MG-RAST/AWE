package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
)

type Record struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Fields       []CWLType `yaml:"fields,omitempty" json:"fields,omitempty" bson:"fields,omitempty"`
}

func (r *Record) GetClass() string { return CWL_record }

func (r *Record) Is_CWL_minimal()                {}
func (r *Record) Is_CWLType()                    {}
func (r *Record) Is_CommandInputParameterType()  {}
func (r *Record) Is_CommandOutputParameterType() {}

func NewRecord(id string, native interface{}) (record *Record, err error) {

	record = &Record{}

	if id != "" {
		err = fmt.Errorf("not sure what to do with id here")
		return
	}

	switch native.(type) {
	case map[interface{}]interface{}:
		native_map, _ := native.(map[interface{}]interface{})
		for key, value := range native_map {

			key_str, ok := key.(string)
			if !ok {
				err = fmt.Errorf("Could not cast key to string")
				return
			}

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
		err = fmt.Errorf("Unknown Record")
		return
	}

	return
}

func (c *Record) String() string {
	return "an record (TODO implement this)"
}
