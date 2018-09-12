package cwl

import (
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"strings"
)

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
