package cwl

import (
	"encoding/json"
	"io/ioutil"
	"path"
	"reflect"
	"strings"

	"fmt"

	"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/mitchellh/mapstructure"

	"gopkg.in/mgo.v2/bson"
	"gopkg.in/yaml.v2"
	//"io/ioutil"
	//"os"
	//"reflect"
)

// ParsingContext _
type ParsingContext struct {
	If_objects map[string]interface{}
	Objects    map[string]CWLObject
}

// CWLObjectGeneric _
type CWLObjectGeneric map[string]interface{}

// LinkMergeMethod _
type LinkMergeMethod string // merge_nested or merge_flattened

// func Add_to_collection_deprecated(context *WorkflowContext, object_array CWLObjectArray) (err error) {

// 	for i, object := range object_array {
// 		err = context.Add(strconv.Itoa(i), object, "Add_to_collection_deprecated") // TODO fix id
// 		if err != nil {
// 			err = fmt.Errorf("(Add_to_collection_deprecated) collection.Add returned: %s", err.Error())
// 			return
// 		}
// 	}

// 	return
// }

// Unmarshal _
func Unmarshal(dataPtr *[]byte, v interface{}) (err error) {

	data := *dataPtr

	if data[0] == '{' {

		errJSON := json.Unmarshal(data, v)
		if errJSON != nil {
			logger.Debug(1, "CWL json unmarshal error: "+errJSON.Error())
			err = errJSON
			return
		}
	} else {
		errYAML := yaml.Unmarshal(data, v)
		if errYAML != nil {
			logger.Debug(1, "CWL yaml unmarshal error: "+errYAML.Error())
			err = errYAML
			return
		}

	}

	return
}

func MakeStringMap(v interface{}, context *WorkflowContext) (result interface{}, err error) {

	switch v.(type) {
	case bson.M:

		original_map := v.(bson.M)

		new_map := make(map[string]interface{})

		for key_str, value := range original_map {

			new_map[key_str] = value
		}

		result = new_map
		return
	case map[interface{}]interface{}:

		v_map, ok := v.(map[interface{}]interface{})
		if !ok {
			err = fmt.Errorf("casting problem (b)")
			return
		}
		v_string_map := make(map[string]interface{})

		for key, value := range v_map {
			key_string := key.(string)
			v_string_map[key_string] = value
		}

		result = v_string_map
		return
	case map[string]interface{}:
		v_map, ok := v.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("casting problem (x)")
			return
		}
		import_req, ok := v_map["$import"]

		if ok {
			//fmt.Println("(MakeStringMap) found import")
			var import_req_str string
			import_req_str, ok = import_req.(string)
			if !ok {
				err = fmt.Errorf("(MakeStringMap) map[string]interface{} found $import, expected string, got %s", reflect.TypeOf(import_req))
				return
			}
			if !strings.HasPrefix(import_req_str, "#") {
				err = fmt.Errorf("(MakeStringMap) $import does not start with #")
				return
			}

			import_path := strings.TrimPrefix(import_req_str, "#")
			//fmt.Printf("import_path: %s\n", import_path)

			if context.Path == "-" {
				err = fmt.Errorf("(MakeStringMap) context.Path empty\n")
				return
			}
			//fmt.Printf("context.Path: %s\n", context.Path)
			import_path = path.Join(context.Path, import_path)
			//fmt.Printf("import_path: %s\n", import_path)
			var doc_stream []byte
			doc_stream, err = ioutil.ReadFile(import_path)
			if err != nil {
				err = fmt.Errorf("(MakeStringMap) error in reading import file \"%s\": %s ", import_path, err.Error())
				return
			}

			var import_obj map[string]interface{}
			err = Unmarshal(&doc_stream, &import_obj)
			if err != nil {
				err = fmt.Errorf("(MakeStringMap) Unmarshal returned %s", err.Error())
				return
			}

			result = import_obj
			return
		}

	}
	result = v
	return
}
