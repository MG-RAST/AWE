package cwl

import (
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"strings"
)

// func Resolve_ImportsArray(obj_array []interface{}, context *WorkflowContext) (err error) {
// 	fmt.Println("(Resolve_ImportsArray) start")

// 	if obj_array == nil {
// 		return
// 	}

// 	for i, _ := range obj_array {

// 		switch obj_array[i].(type) {
// 		case map[string]interface{}: // this is done to make sure a map[string]interface{} is inserted into array
// 			obj_map := obj_array[i].(map[string]interface{})
// 			var new_element map[string]interface{}
// 			new_element, err = Resolve_ImportsMap(obj_map, context)
// 			if new_element != nil {
// 				obj_array[i] = new_element
// 			}

// 		case map[interface{}]interface{}: // this is done to make sure a map[string]interface{} is inserted into array

// 			var obj interface{}
// 			obj, err = MakeStringMap(obj_array[i], context)
// 			if err != nil {
// 				err = fmt.Errorf("(Resolve_Imports) MakeStringMap returned: %s", err.Error())
// 				return
// 			}
// 			var obj_map map[string]interface{}
// 			var ok bool
// 			obj_map, ok = obj.(map[string]interface{})
// 			if !ok {
// 				err = fmt.Errorf("(Resolve_Imports) obj not map[string]interface{} ")
// 				return
// 			}
// 			//var new_element map[string]interface{}
// 			obj_map, err = Resolve_ImportsMap(obj_map, context)
// 			if obj_map != nil {
// 				obj_array[i] = obj_map
// 			}
// 		case []interface{}:
// 			obj_array := obj_array[i].([]interface{})

// 			err = Resolve_ImportsArray(obj_array, context)
// 			if err != nil {
// 				err = fmt.Errorf("(Resolve_Imports) Resolve_ImportsArray returned: %s", err.Error())
// 				return
// 			}

// 			obj_array[i] = obj_array
// 		case string:
// 			continue
// 		default:
// 			panic(reflect.TypeOf(obj_array[i]))
// 			// var new_element interface{}
// 			// new_element, err = Resolve_Imports(obj_array[i], context)
// 			// if err != nil {
// 			// 	err = fmt.Errorf("(Resolve_Imports) []interface{}  Resolve_Imports returned: %s", err.Error())
// 			// 	return
// 			// }
// 			// if new_element != nil {
// 			// 	obj_array[i] = new_element
// 			// }
// 		}

// 	}

// 	return
// }

// func Resolve_ImportsMap(obj_map map[string]interface{}, context *WorkflowContext) (new_obj map[string]interface{}, err error) {

// 	err = fmt.Errorf("deprecated")
// 	return
// 	if obj_map == nil {
// 		return
// 	}

// 	fmt.Println("(Resolve_ImportsMap) start")
// 	spew.Dump(obj_map)

// 	import_req, ok := obj_map["$import"]

// 	if ok {
// 		fmt.Println("(Resolve_ImportsMap) found import")
// 		var import_req_str string
// 		import_req_str, ok = import_req.(string)
// 		if !ok {
// 			err = fmt.Errorf("(Resolve_Imports) map[string]interface{} found $import, expected string, got %s", reflect.TypeOf(import_req))
// 			return
// 		}
// 		if !strings.HasPrefix(import_req_str, "#") {
// 			err = fmt.Errorf("(Resolve_Imports) $import does not start with #")
// 			return
// 		}

// 		import_path := strings.TrimPrefix(import_req_str, "#")
// 		//fmt.Printf("import_path: %s\n", import_path)

// 		if context.Path == "-" {
// 			err = fmt.Errorf("context.Path empty\n")
// 			return
// 		}
// 		//fmt.Printf("context.Path: %s\n", context.Path)
// 		import_path = path.Join(context.Path, import_path)
// 		//fmt.Printf("import_path: %s\n", import_path)
// 		var doc_stream []byte
// 		doc_stream, err = ioutil.ReadFile(import_path)
// 		if err != nil {
// 			err = fmt.Errorf("(Resolve_Imports) error in reading file %s: %s ", import_path, err.Error())
// 			return
// 		}

// 		var import_obj map[string]interface{}
// 		err = Unmarshal(&doc_stream, &import_obj)
// 		if err != nil {
// 			err = fmt.Errorf("(Resolve_Imports) Unmarshal returned %s", err.Error())
// 			return
// 		}
// 		fmt.Println("(Resolve_ImportsMap) reading import")
// 		spew.Dump(import_obj)
// 		var modified_import_obj map[string]interface{}
// 		modified_import_obj, err = Resolve_ImportsMap(import_obj, context)
// 		if err != nil {
// 			err = fmt.Errorf("(Resolve_Imports) (on $import) Resolve_ImportsMap returned %s", err.Error())
// 			return
// 		}

// 		if modified_import_obj != nil {
// 			new_obj = modified_import_obj
// 		} else {
// 			new_obj = import_obj
// 		}

// 		fmt.Println("(Resolve_ImportsMap) returning")
// 		spew.Dump(new_obj)
// 		panic("done X")
// 		return
// 	}

// 	for i, _ := range obj_map {
// 		// var new_element interface{}
// 		// new_element, err = Resolve_Imports(obj_map[i], context)
// 		// if err != nil {
// 		// 	err = fmt.Errorf("(Resolve_Imports) map[string]interface{} Resolve_Imports returned: %s", err.Error())
// 		// 	return
// 		// }
// 		// if new_element != nil {
// 		// 	obj_map[i] = new_element
// 		// }
// 		switch obj_map[i].(type) {
// 		case map[string]interface{}: // this is done to make sure a map[string]interface{} is inserted into array
// 			obj_map := obj_map[i].(map[string]interface{})
// 			var new_element map[string]interface{}
// 			new_element, err = Resolve_ImportsMap(obj_map, context)
// 			if new_element != nil {
// 				obj_map[i] = new_element
// 			} else {
// 				obj_map[i] = obj_map
// 			}

// 		case map[interface{}]interface{}: // this is done to make sure a map[string]interface{} is inserted into array

// 			var obj interface{}
// 			obj, err = MakeStringMap(obj_map[i], context)
// 			if err != nil {
// 				err = fmt.Errorf("(Resolve_Imports) MakeStringMap returned: %s", err.Error())
// 				return
// 			}
// 			var obj_map map[string]interface{}
// 			var ok bool
// 			obj_map, ok = obj.(map[string]interface{})
// 			if !ok {
// 				err = fmt.Errorf("(Resolve_Imports) obj not map[string]interface{} ")
// 				return
// 			}
// 			var new_element map[string]interface{}
// 			new_element, err = Resolve_ImportsMap(obj_map, context)
// 			if err != nil {
// 				err = fmt.Errorf("(Resolve_Imports) Resolve_ImportsMap returned: %s", err.Error())
// 				return
// 			}
// 			if new_element != nil {
// 				obj_map[i] = new_element
// 			} else {
// 				obj_map[i] = obj_map
// 			}
// 		case []interface{}:
// 			obj_array := obj_map[i].([]interface{})

// 			err = Resolve_ImportsArray(obj_array, context)
// 			if err != nil {
// 				err = fmt.Errorf("(Resolve_Imports) Resolve_ImportsArray returned: %s", err.Error())
// 				return
// 			}

// 			obj_map[i] = obj_array

// 		case string:
// 			continue
// 		default:
// 			panic(reflect.TypeOf(obj_map[i]))
// 			// var new_element interface{}
// 			// new_element, err = Resolve_Imports(obj_map[i], context)
// 			// if err != nil {
// 			// 	err = fmt.Errorf("(Resolve_Imports) []interface{}  Resolve_Imports returned: %s", err.Error())
// 			// 	return
// 			// }
// 			// if new_element != nil {
// 			// 	obj_map[i] = new_element
// 			// }
// 		}
// 	}

// 	return
// }

// returns new_obj only when object has to be replaced

// func Resolve_Imports_deprecated(obj interface{}, context *WorkflowContext) (new_obj interface{}, err error) {
// 	fmt.Println("(Resolve_Imports) start")
// 	if context == nil {
// 		err = fmt.Errorf("(Resolve_Imports) context == nil")
// 		return
// 	}

// 	if obj == nil {
// 		return
// 	}

// 	switch obj.(type) {
// 	case []interface{}:
// 		obj_array := obj.([]interface{})
// 		err = Resolve_ImportsArray(obj_array, context)

// 	case map[string]interface{}:

// 		obj_map := obj.(map[string]interface{})
// 		new_obj, err = Resolve_ImportsMap(obj_map, context)

// 	case map[interface{}]interface{}:

// 		//obj_map := obj.(map[interface{}]interface{})

// 		obj, err = MakeStringMap(obj, context)

// 		obj_map, ok := obj.(map[string]interface{})
// 		if !ok {
// 			err = fmt.Errorf("(Resolve_Imports) expected map[string]interface{}, got: %s ", reflect.TypeOf(obj))
// 			return
// 		}

// 		new_obj, err = Resolve_ImportsMap(obj_map, context)
// 		if err != nil {
// 			err = fmt.Errorf("(Resolve_Imports) Resolve_ImportsMap returned: %s ", err.Error())
// 			return
// 		}

// 	case string:
// 		return
// 	default:
// 		err = fmt.Errorf("(Resolve_Imports) type %s unkown", reflect.TypeOf(obj))
// 		return
// 	}

// 	fmt.Printf("(Resolve_Imports) result: %s\n", reflect.TypeOf(new_obj))

// 	return
// }

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
	//fmt.Printf("import_path: %s\n", import_path)

	if context.Path == "-" {
		err = fmt.Errorf("context.Path empty\n")
		return
	}

	import_path = path.Join(context.Path, import_path)
	//fmt.Printf("import_path: %s\n", import_path)
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
