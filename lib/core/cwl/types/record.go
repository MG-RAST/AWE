package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
)

type Record struct {
	CWLType_Impl
	Id     string
	Fields []CWLType
}

func (r *Record) GetClass() string { return CWL_record }
func (r *Record) GetId() string    { return r.Id }
func (r *Record) SetId(id string)  { r.Id = id }

func (r *Record) Is_CWL_minimal()                {}
func (r *Record) Is_CWLType()                    {}
func (r *Record) Is_CommandInputParameterType()  {}
func (r *Record) Is_CommandOutputParameterType() {}

func NewRecord(native interface{}) (record *Record, err error) {

	record = &Record{}

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
