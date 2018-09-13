package cwl

import (
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"strings"
)

func Resolve_Imports(obj interface{}, context *WorkflowContext) (new_obj interface{}, err error) {

	if context == nil {
		err = fmt.Errorf("(Resolve_Imports) context == nil")
		return
	}

	switch obj.(type) {
	case []interface{}:
		obj_array := obj.([]interface{})
		for i, _ := range obj_array {

			var new_element interface{}
			new_element, err = Resolve_Imports(obj_array[i], context)
			if err != nil {
				err = fmt.Errorf("(Resolve_Imports) []interface{}  Resolve_Imports returned: %s", err.Error())
				return
			}
			if new_element != nil {
				obj_array[i] = new_element
			}

		}
	case map[string]interface{}:

		obj_map := obj.(map[string]interface{})

		import_req, ok := obj_map["$import"]

		if ok {
			var import_req_str string
			import_req_str, ok = import_req.(string)
			if !ok {
				err = fmt.Errorf("(Resolve_Imports) map[string]interface{} found $import, expected string, got %s", reflect.TypeOf(import_req))
				return
			}
			if !strings.HasPrefix(import_req_str, "#") {
				err = fmt.Errorf("(Resolve_Imports) $import does not start with #")
				return
			}

			import_path := strings.TrimPrefix(import_req_str, "#")
			fmt.Printf("import_path: %s\n", import_path)

			if context.Path == "-" {
				err = fmt.Errorf("context.Path empty\n")
				return
			}

			import_path = path.Join(context.Path, import_path)
			fmt.Printf("import_path: %s\n", import_path)
			var doc_stream []byte
			doc_stream, err = ioutil.ReadFile(import_path)
			if err != nil {
				err = fmt.Errorf("(Resolve_Imports) error in reading file %s: %s ", import_path, err.Error())
				return
			}

			//var new_obj interface{}
			err = Unmarshal(&doc_stream, &new_obj)
			if err != nil {
				err = fmt.Errorf("(Resolve_Imports) Unmarshal returned %s", err.Error())
				return
			}
			return
		}

		for i, _ := range obj_map {
			var new_element interface{}
			new_element, err = Resolve_Imports(obj_map[i], context)
			if err != nil {
				err = fmt.Errorf("(Resolve_Imports) map[string]interface{} Resolve_Imports returned: %s", err.Error())
				return
			}
			if new_element != nil {
				obj_map[i] = new_element
			}

		}
	case map[interface{}]interface{}:

		obj_map := obj.(map[interface{}]interface{})

		for i, _ := range obj_map {
			var new_element interface{}
			new_element, err = Resolve_Imports(obj_map[i], context)
			if err != nil {
				err = fmt.Errorf("(Resolve_Imports) map[interface{}]interface{} Resolve_Imports returned: %s", err.Error())
				return
			}
			if new_element != nil {
				obj_map[i] = new_element
			}

		}
	case string:
		return
	default:
		err = fmt.Errorf("(Resolve_Imports) type %s unkown", reflect.TypeOf(obj))
		return
	}

	return
}

func Evaluate_import(obj interface{}, context *WorkflowContext) (new_doc interface{}, ok bool, err error) {

	var obj_map map[string]interface{}

	obj_map, ok = obj.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(Evaluate_import) expected map[string]interface{}, got %s", reflect.TypeOf(obj))
		return
	}

	import_req, ok := obj_map["$import"]

	if !ok {
		ok = false
		return
	}

	var import_req_str string
	import_req_str, ok = import_req.(string)
	if !ok {
		err = fmt.Errorf("(Evaluate_import) found $import, expected string, got %s", reflect.TypeOf(import_req))
		return
	}
	if !strings.HasPrefix(import_req_str, "#") {
		err = fmt.Errorf("(Evaluate_import) $import does not start with #")
		return
	}

	import_path := strings.TrimPrefix(import_req_str, "#")
	fmt.Printf("import_path: %s\n", import_path)

	if context.Path == "-" {
		err = fmt.Errorf("context.Path empty\n")
		return
	}

	import_path = path.Join(context.Path, import_path)
	fmt.Printf("import_path: %s\n", import_path)
	var doc_stream []byte
	doc_stream, err = ioutil.ReadFile(import_path)
	if err != nil {
		err = fmt.Errorf("(Evaluate_import) error in reading file %s: %s ", import_path, err.Error())
		return
	}

	//var new_doc interface{}
	err = Unmarshal(&doc_stream, &new_doc)
	if err != nil {
		err = fmt.Errorf("(Evaluate_import) Unmarshal returned %s", err.Error())
		return
	}

	ok = true
	return
}
