package cwl

import (
	"encoding/json"
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
)

type Record map[string]CWLType

func (r *Record) Is_CWL_object() {}

func (r *Record) GetClass() string { return "record" }

func (r *Record) GetID() string {

	id, ok := (*r)["id"]
	if ok {

		id_str, ok := id.(*String)
		if ok {
			return string(*id_str)
		}
	}
	return ""
}

func (r *Record) SetId(id string) {
	//ids := NewString(id)
	//(*r)["id"] = ids
}

func (r *Record) GetType() CWLType_Type { return CWL_record }

func (r *Record) Is_CWL_minimal()     {}
func (r *Record) Is_Type()            {}
func (r *Record) Type2String() string { return "record" }

//func (r *Record) Is_CommandInputParameterType()  {}
//func (r *Record) Is_CommandOutputParameterType() {}

func NewRecord(id string, native interface{}, context *WorkflowContext) (record Record, err error) {

	native, err = MakeStringMap(native, context)
	if err != nil {
		return
	}
	record = Record{}

	//record = &Record{}
	//record.Id = id
	switch native.(type) {
	case map[string]interface{}:

		//fmt.Println("Got a record:")
		//spew.Dump(native)
		nativeMap, _ := native.(map[string]interface{})

		var keys []string

		for key_str, _ := range nativeMap {
			keys = append(keys, key_str)
		}

		for _, key_str := range keys {

			value, _ := nativeMap[key_str]

			var value_cwl CWLType

			value_cwl, err = NewCWLType(key_str, value, context)
			if err != nil {
				fmt.Println("Got a record:")
				spew.Dump(native)
				err = fmt.Errorf("(NewRecord) %s NewCWLType returned: %s", key_str, err.Error())
				return
			}

			record[key_str] = value_cwl

			//record.Fields = append(record.Fields, value_cwl)

		}

		_, has_id := nativeMap["id"]
		if !has_id {
			//record["id"] = NewString(id)
		}

		return

	default:
		fmt.Println("Unknown Record:")
		spew.Dump(native)
		err = fmt.Errorf("(NewRecord) Unknown type: %s (%s)", reflect.TypeOf(native), spew.Sdump(native))
	}

	return
}

func (c Record) String() string {
	bytes, err := json.Marshal(c)
	if err != nil {
		return err.Error()
	}
	return string(bytes[:])
}
