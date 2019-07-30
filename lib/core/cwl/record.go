package cwl

import (
	"encoding/json"
	"fmt"
	"path"
	"reflect"
	"strings"

	"github.com/davecgh/go-spew/spew"
)

// Record _
type Record map[string]CWLType

// IsCWLObject _
func (r *Record) IsCWLObject() {}

// GetClass _
func (r *Record) GetClass() string { return "record" }

// GetID _
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

// SetID _
func (r *Record) SetID(id string) {
	idString := NewString(id)
	rNptr := *r
	rNptr["id"] = idString
	//(*r)["id"] = ids
}

// GetType _
func (r *Record) GetType() CWLType_Type { return CWLRecord }

// IsCWLMinimal _
func (r *Record) IsCWLMinimal() {}

// Is_Type _
func (r *Record) Is_Type() {}

// Type2String _
func (r *Record) Type2String() string { return "record" }

//func (r *Record) Is_CommandInputParameterType()  {}
//func (r *Record) Is_CommandOutputParameterType() {}

// NewRecord _
func NewRecord(id string, parentID string, native interface{}, context *WorkflowContext) (record Record, err error) {

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

		for keyStr := range nativeMap {
			keys = append(keys, keyStr)
		}

		for _, keyStr := range keys {

			value, _ := nativeMap[keyStr]

			var valueCwl CWLType

			valueCwl, err = NewCWLType(keyStr, "", value, context)
			if err != nil {
				fmt.Println("Got a record:")
				spew.Dump(native)
				err = fmt.Errorf("(NewRecord) %s NewCWLType returned: %s", keyStr, err.Error())
				return
			}

			record[keyStr] = valueCwl

			//record.Fields = append(record.Fields, value_cwl)

		}

		if id != "" {
			record.SetID(id)
			//record["id"] = NewString(id)
		}

		if parentID != "" {
			idIf, ok := nativeMap["id"]
			if ok {
				var idStr string
				idStr, ok = idIf.(string)
				if ok {
					if !strings.HasPrefix(idStr, "#") {
						idStr = path.Join(parentID, idStr)
						record.SetID(idStr)

					}

				}

			}

		}

		return

	default:
		fmt.Println("Unknown Record:")
		spew.Dump(native)
		err = fmt.Errorf("(NewRecord) Unknown type: %s (%s)", reflect.TypeOf(native), spew.Sdump(native))
	}

	return
}

// String _
func (r *Record) String() string {
	bytes, err := json.Marshal(r)
	if err != nil {
		return err.Error()
	}
	return string(bytes[:])
}
