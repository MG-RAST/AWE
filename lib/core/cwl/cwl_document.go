package cwl

import (
	"fmt"
	"io/ioutil"
	"path"
	"reflect"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
)

// GraphDocument this is used by YAML or JSON library for inital parsing
type GraphDocument struct {
	CwlVersion CWLVersion        `yaml:"cwlVersion,omitempty" json:"cwlVersion,omitempty" bson:"cwlVersion,omitempty" mapstructure:"cwlVersion,omitempty"`
	Base       interface{}       `yaml:"$base,omitempty" json:"$base,omitempty" bson:"base,omitempty" mapstructure:"$base,omitempty"`
	Graph      []interface{}     `yaml:"$graph" json:"$graph" bson:"graph" mapstructure:"$graph"` // can only be used for reading, yaml has problems writing interface objetcs
	Namespaces map[string]string `yaml:"$namespaces,omitempty" json:"$namespaces,omitempty" bson:"namespaces,omitempty" mapstructure:"$namespaces,omitempty"`
	Schemas    []interface{}     `yaml:"$schemas,omitempty" json:"$schemas,omitempty" bson:"schemas,omitempty" mapstructure:"$schemas,omitempty"`
}

// ParseCWLGraphDocument _
// entrypoint defauyts to #main but can anything else...
func ParseCWLGraphDocument(yamlStr string, entrypoint string, context *WorkflowContext) (objectArray []NamedCWLObject, schemata []CWLType_Type, schemas []interface{}, err error) {

	cwlGen := GraphDocument{}

	yamlByte := []byte(yamlStr)
	err = Unmarshal(&yamlByte, &cwlGen)
	if err != nil {
		logger.Debug(1, "CWL unmarshal error")
		err = fmt.Errorf("(Parse_cwl_graph_document) Unmarshal returned: " + err.Error())
		return
	}
	//fmt.Println("-------------- yamlStr")
	//fmt.Println(yamlStr)
	//fmt.Println("-------------- raw CWL")
	//spew.Dump(cwl_gen)
	//fmt.Println("-------------- Start parsing")

	if len(cwlGen.Graph) == 0 {
		err = fmt.Errorf("(Parse_cwl_graph_document) len(cwlGen.Graph) == 0")
		return
	}

	// resolve $import
	//var new_obj []interface{}
	//err = Resolve_ImportsArray(cwl_gen.Graph, context)
	//if err != nil {
	//	err = fmt.Errorf("(Parse_cwl_graph_document) Resolve_Imports returned: " + err.Error())
	//	return
	//}

	//fmt.Println("############# cwl_gen.Graph:")
	//spew.Dump(cwl_gen.Graph)
	//panic("done")
	//if new_obj != nil {
	//	err = fmt.Errorf("(Parse_cwl_graph_document) $import in $graph not supported yet")
	//	return
	//}

	//context.Graph = cwlGen.Graph
	context.GraphDocument = cwlGen
	context.CwlVersion = cwlGen.CwlVersion

	if cwlGen.Namespaces != nil {
		context.Namespaces = cwlGen.Namespaces
	}

	if cwlGen.Schemas != nil {
		schemas = cwlGen.Schemas
	}

	// iterate over Graph

	// try to find CWL version!
	if context.CwlVersion == "" {
		for _, elem := range cwlGen.Graph {
			elemMap, ok := elem.(map[string]interface{})
			if ok {
				cwlVersionIf, hasVersion := elemMap["cwlVersion"]
				if hasVersion {

					var cwlVersionStr string
					cwlVersionStr, ok = cwlVersionIf.(string)
					if !ok {
						err = fmt.Errorf("(Parse_cwl_graph_document) Could not read CWLVersion (%s)", reflect.TypeOf(cwlVersionIf))
						return
					}
					context.CwlVersion = CWLVersion(cwlVersionStr)
					break
				}

			}
		}

	}

	if context.CwlVersion == "" {
		// see raw
		err = fmt.Errorf("(Parse_cwl_graph_document) cwl_version empty")
		return
	}

	//fmt.Println("-------------- A Parse_cwl_document")
	if entrypoint == "" {
		entrypoint = "#main"
	}

	if !strings.HasPrefix(entrypoint, "#") {
		err = fmt.Errorf("(Parse_cwl_graph_document) entrypoint not absolute: %s", entrypoint)
		return
	}

	err = context.Init(entrypoint)
	if err != nil {
		err = fmt.Errorf("(Parse_cwl_graph_document) (using entrypoint=%s) context.Init returned: %s", entrypoint, err.Error())
		return
	}

	for id, object := range context.Objects {
		namedObj := NewNamedCWLObject(id, object)
		objectArray = append(objectArray, namedObj)
	}

	// object_array []NamedCWLObject
	//fmt.Println("############# object_array:")
	//spew.Dump(object_array)

	return
}

// ParseCWLSimpleDocument _
// returns: objectID may be used an entrypoint later
func ParseCWLSimpleDocument(yamlStr string, useObjectID string, context *WorkflowContext) (objectArray []NamedCWLObject, schemata []CWLType_Type, schemas []interface{}, objectID string, err error) {
	// Here I expect a single object, Workflow or CommandLIneTool
	//fmt.Printf("-------------- yaml_str: %s\n", yaml_str)

	//fmt.Printf("ParseCWLSimpleDocument start\n")
	//defer fmt.Printf("ParseCWLSimpleDocument end\n")

	var objectIf interface{}

	yamlByte := []byte(yamlStr)
	err = Unmarshal(&yamlByte, &objectIf)
	if err != nil {
		fmt.Println("(ParseCWLSimpleDocument) yamlStr:")
		fmt.Println(yamlStr)
		//logger.Debug(1, "CWL unmarshal error")
		err = fmt.Errorf("(ParseCWLSimpleDocument) Unmarshal returns: %s", err.Error())
		return
	}
	//fmt.Println("objectIf:")
	//spew.Dump(objectIf)
	var objectIfTranslated interface{}
	objectIfTranslated, err = translate(objectIf, context.Path)
	if err != nil {
		err = fmt.Errorf("(ParseCWLSimpleDocument) translate returned: %s", err.Error())
		return
	}

	objectIfTranslated, err = MakeStringMap(objectIfTranslated, context)
	if err != nil {
		return
	}

	var ok bool
	//objectIf = nil

	switch objectIfTranslated.(type) {
	case map[string]interface{}:

		var objectMap map[string]interface{}

		objectMap, ok = objectIfTranslated.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(ParseCWLSimpleDocument) could not translate: %s", err.Error())
			return
		}

		if context.CwlVersion == "" {

			cwlVersionIf, hasVersion := objectMap["cwlVersion"]
			if hasVersion {

				cwlVersionStr, ok := cwlVersionIf.(string)
				if ok {
					context.CwlVersion = NewCWLVersion(cwlVersionStr)
				} else {
					err = fmt.Errorf("(ParseCWLSimpleDocument) version not string (type: %s)", reflect.TypeOf(cwlVersionIf))
					return
				}
			}
		}

		if objectMap == nil {
			err = fmt.Errorf("(ParseCWLSimpleDocument) objectMap == nil")
			return
		}

		//fmt.Println("object_if:")
		//spew.Dump(object_if)
		//var ok bool
		var namespacesIf interface{}
		namespacesIf, ok = objectMap["$namespaces"]
		if ok {

			namespacesIf, err = MakeStringMap(namespacesIf, nil)
			if err != nil {
				return
			}

			var namespacesMap map[string]interface{}
			namespacesMap, ok = namespacesIf.(map[string]interface{})
			if !ok {
				err = fmt.Errorf("(ParseCWLSimpleDocument) namespaces_if error (type: %s)", reflect.TypeOf(namespacesIf))
				return
			}
			context.Namespaces = make(map[string]string)

			for key, value := range namespacesMap {

				switch value := value.(type) {
				case string:
					context.Namespaces[key] = value
				default:
					err = fmt.Errorf("(ParseCWLSimpleDocument) value is not string (%s)", reflect.TypeOf(value))
					return
				}

			}

		} else {
			context.Namespaces = nil
			//fmt.Println("no namespaces")
		}

		var schemasIf interface{}
		schemasIf, ok = objectMap["$schemas"]
		if ok {

			schemas, ok = schemasIf.([]interface{})
			if !ok {
				err = fmt.Errorf("(ParseCWLSimpleDocument) could not parse $schemas (%s)", reflect.TypeOf(schemasIf))
				return
			}

		}
		//var this_class string
		//this_class, err = GetClass(object_if)
		//if err != nil {
		//	err = fmt.Errorf("(Parse_cwl_document) GetClass returns %s", err.Error())
		//	return
		//}
		//fmt.Printf("this_class: %s\n", this_class)
		//fmt.Printf("AAAA\n")
		setObjectID := false
		//fmt.Printf("ParseCWLSimpleDocument: testsssssss \n")
		//var objectID string
		objectID, err = GetID(objectMap)
		if err != nil {
			//fmt.Printf("ParseCWLSimpleDocument: got no id, got v=%s\n", useObjectID)

			if useObjectID == "" {
				err = fmt.Errorf("(ParseCWLSimpleDocument) B) GetId returns %s", err.Error())
				return
			}
			err = nil

			//if len(strings.Split(useObjectID, "/")) > 1 {
			//	err = fmt.Errorf("(ParseCWLSimpleDocument) contains path !?  %s ", useObjectID)
			//	return
			//}

			objectID = useObjectID
			//fmt.Printf("ParseCWLSimpleDocument: objectID %s\n", objectID)
			setObjectID = true
			//err = fmt.Errorf("(ParseCWLSimpleDocument) GetId returns %s", err.Error())
			//return
		}

		if objectID == "#." {
			err = fmt.Errorf("(ParseCWLSimpleDocument) objectID wrong (useObjectID: %s)", useObjectID)
			return
		}

		//fmt.Printf("objectID: %s\n", objectID)

		var object CWLObject
		var schemataNew []CWLType_Type
		object, schemataNew, err = NewCWLObject(objectMap, objectID, "", nil, context)
		if err != nil {
			err = fmt.Errorf("(ParseCWLSimpleDocument) B NewCWLObject returns %s", err.Error())
			return
		}

		//fmt.Println("-------------- raw CWL")
		//spew.Dump(commandlinetool_if)
		//fmt.Println("-------------- Start parsing")

		//var commandlinetool *CommandLineTool
		//var schemataNew []CWLType_Type
		//commandlinetool, schemataNew, err = NewCommandLineTool(commandlinetool_if)
		//if err != nil {
		//	err = fmt.Errorf("(Parse_cwl_document) NewCommandLineTool returned: %s", err.Error())
		//	return
		//}

		switch object.(type) {
		case *Workflow:
			thisWorkflow, _ := object.(*Workflow)
			if setObjectID {
				thisWorkflow.ID = objectID
			}
			context.CwlVersion = thisWorkflow.CwlVersion
		case *CommandLineTool:
			thisCLT, _ := object.(*CommandLineTool)
			if setObjectID {
				thisCLT, _ := object.(*CommandLineTool)
				thisCLT.ID = objectID
			}
			context.CwlVersion = thisCLT.CwlVersion
		case *ExpressionTool:
			thisET, _ := object.(*ExpressionTool)
			if setObjectID {
				thisET.ID = objectID
			}
			context.CwlVersion = thisET.CwlVersion

		}

		logger.Debug(3, "(ParseCWLSimpleDocument) adding %s to context", objectID)
		err = context.AddObject(objectID, object, "ParseCWLSimpleDocument")
		if err != nil {
			err = fmt.Errorf("(ParseCWLSimpleDocument) context.Add returned %s", err.Error())
			return
		}

		namedObj := NewNamedCWLObject(objectID, object)
		//named_obj := NewNamedCWLObject(commandlinetool.Id, commandlinetool)

		//cwl_version = commandlinetool.CwlVersion // TODO

		objectArray = append(objectArray, namedObj)
		for i := range schemataNew {
			schemata = append(schemata, schemataNew[i])
		}

	case []interface{}:
		var objectIfArray []interface{}

		objectIfArray, ok = objectIfTranslated.([]interface{})
		if !ok {
			err = fmt.Errorf("(ParseCWLSimpleDocument) []interface{} , could not translate: %s", err.Error())
			return
		}

		for i := range objectIfArray {

			var object CWLObject
			var schemataNew []CWLType_Type
			object, schemataNew, err = NewCWLObject(objectIfArray[i], "", "", nil, context)
			if err != nil {
				err = fmt.Errorf("(ParseCWLSimpleDocument) B NewCWLObject returns %s", err.Error())
				return
			}

			for i := range schemataNew {
				schemata = append(schemata, schemataNew[i])
			}

			namedObj := NewNamedCWLObject(objectID, object)
			objectArray = append(objectArray, namedObj)
		}

	default:

		err = fmt.Errorf("(ParseCWLSimpleDocument) type unknown %s", reflect.TypeOf(objectIfTranslated))
		return

	}

	//fmt.Println("objectIfTranslated:")
	//spew.Dump(objectIfTranslated)

	return
}

// ReadYamlFile _
func ReadYamlFile(filePath string) (objectIf interface{}, err error) {
	var fileByte []byte
	fileByte, err = ioutil.ReadFile(filePath)
	if err != nil {
		err = fmt.Errorf("error in reading file %s: %s", filePath, err.Error())
		return
	}

	//var objectIf interface{}

	//yamlByte := []byte(yamlStr)
	err = Unmarshal(&fileByte, &objectIf)
	if err != nil {
		fmt.Println("(ParseCWLSimpleDocument) yamlStr:")
		fmt.Println(fileByte)
		//logger.Debug(1, "CWL unmarshal error")
		err = fmt.Errorf("(ParseCWLSimpleDocument) Unmarshal returns: %s", err.Error())
		return
	}

	return
}

// ParseCWLDocumentFile _
func ParseCWLDocumentFile(existingContext *WorkflowContext, filePath string, entrypoint string, inputfilePath string, identifier string) (objectArray []NamedCWLObject, schemata []CWLType_Type, context *WorkflowContext, schemas []interface{}, newEntrypoint string, err error) {

	logger.Debug(3, "(ParseCWLDocumentFile) loading %s", filePath)
	// skip parsing if files have been loaded before
	if existingContext != nil {

		var ok bool
		if existingContext.FilesLoaded != nil {
			_, ok = existingContext.FilesLoaded[filePath]
			if ok {
				return
			}
		}
	}

	var fileByte []byte
	fileByte, err = ioutil.ReadFile(filePath)
	if err != nil {
		err = fmt.Errorf("error in reading file %s: %s", filePath, err.Error())
		return
	}

	fileStr := string(fileByte)

	//filePathBase := path.Base(filePath)

	objectArray, schemata, context, schemas, newEntrypoint, err = ParseCWLDocument(existingContext, fileStr, entrypoint, inputfilePath, identifier)
	if err != nil {
		err = fmt.Errorf("(ParseCWLDocumentFile) ParseCWLDocument returned: %s", err.Error())
		return
	}

	if context.FilesLoaded == nil {
		context.FilesLoaded = make(map[string]bool)
	}

	context.FilesLoaded[filePath] = true
	return
}

// ParseCWLDocument _
// format: inputfilePath  / fileName # entrypoint , example: /path/tool.cwl#main
// a CWL document can be a graph document or a single object document
// an entrypoint can only be specified for a graph document
// useObjectID in case of single object
func ParseCWLDocument(existingContext *WorkflowContext, yamlStr string, entrypoint string, inputfilePath string, useObjectID string) (objectArray []NamedCWLObject, schemata []CWLType_Type, context *WorkflowContext, schemas []interface{}, newEntrypoint string, err error) {
	//fmt.Printf("(Parse_cwl_document) starting\n")

	if useObjectID != "" {
		if !strings.HasPrefix(useObjectID, "#") {
			err = fmt.Errorf("(NewCWLObject) useObjectID has not # as prefix (%s)", useObjectID)
			return
		}
	}

	if existingContext != nil {
		context = existingContext
	} else {
		context = NewWorkflowContext()
		context.Path = inputfilePath
		context.InitBasic()
	}

	graphPos := strings.Index(yamlStr, "$graph")

	//yamlStr = strings.Replace(yamlStr, "$namespaces", "namespaces", -1)
	//fmt.Println("yamlStr:")
	//fmt.Println(yamlStr)

	if graphPos != -1 {
		// *** graph file ***
		//yamlStr = strings.Replace(yamlStr, "$graph", "graph", -1) // remove dollar sign
		logger.Debug(3, "(Parse_cwl_document) graph document")
		//fmt.Printf("(Parse_cwl_document) ParseCWLGraphDocument\n")
		objectArray, schemata, schemas, err = ParseCWLGraphDocument(yamlStr, entrypoint, context)
		if err != nil {
			err = fmt.Errorf("(Parse_cwl_document) Parse_cwl_graph_document returned: %s", err.Error())
			return
		}

	} else {
		logger.Debug(3, "(Parse_cwl_document) simple document")
		//fmt.Printf("(Parse_cwl_document) ParseCWLSimpleDocument\n")

		//useObjectID := "#" + inputfilePath

		var objectID string

		objectArray, schemata, schemas, objectID, err = ParseCWLSimpleDocument(yamlStr, useObjectID, context)
		if err != nil {
			err = fmt.Errorf("(Parse_cwl_document) Parse_cwl_simple_document returned: %s", err.Error())
			return
		}
		newEntrypoint = objectID
	}
	if len(objectArray) == 0 {
		err = fmt.Errorf("(Parse_cwl_document) len(objectArray) == 0 (graphPos: %d)", graphPos)
		return
	}
	//fmt.Printf("(Parse_cwl_document) end\n")
	return
}

// source: https://gist.github.com/hvoecking/10772475
func translate(obj interface{}, workingPath string) (tObj interface{}, err error) {
	// Wrap the original in a reflect.Value
	original := reflect.ValueOf(obj)

	copy := reflect.New(original.Type()).Elem()
	_, _, err = translateRecursive(copy, original, workingPath)

	tObj = copy.Interface()

	//fmt.Println("tObj:")
	//spew.Dump(tObj)

	// Remove the reflection wrapper
	return
}

// extended version of https://gist.github.com/hvoecking/10772475
func translateRecursive(copy, original reflect.Value, workingPath string) (includeString string, importString string, err error) {

	var childIncludeString string
	var childImportString string

	includeStr := "$include" // get text only
	includeStrValue := reflect.ValueOf(includeStr)

	importStr := "$import" // get real document
	importStrValue := reflect.ValueOf(importStr)

	switch original.Kind() {
	// The first cases handle nested structures and translate them recursively

	// If it is a pointer we need to unwrap and call once again
	case reflect.Ptr:
		// To get the actual value of the original we have to call Elem()
		// At the same time this unwraps the pointer so we don't end up in
		// an infinite recursion
		originalValue := original.Elem()
		// Check if the pointer is nil
		if !originalValue.IsValid() {
			return
		}
		// Allocate a new object and set the pointer to it
		copy.Set(reflect.New(originalValue.Type()))
		// Unwrap the newly created pointer
		childIncludeString, importString, err = translateRecursive(copy.Elem(), originalValue, workingPath)
		if err != nil {
			return
		}
		if childIncludeString != "" {
			fmt.Println("A")
		}

		if childImportString != "" {
			fmt.Println("childImportString A")
		}

	// If it is an interface (which is very similar to a pointer), do basically the
	// same as for the pointer. Though a pointer is not the same as an interface so
	// note that we have to call Elem() after creating a new object because otherwise
	// we would end up with an actual pointer
	case reflect.Interface:
		// Get rid of the wrapping interface
		originalValue := original.Elem()
		if !originalValue.IsValid() {
			return
		}
		//if originalValue.IsNil() {
		//	return
		//}
		// Create a new object. Now new gives us a pointer, but we want the value it
		// points to, so we have to call Elem() to unwrap it

		originalValueType := originalValue.Type()
		//fmt.Println("originalValueType:")
		//spew.Dump(originalValueType)
		copyValueIf := reflect.New(originalValueType)
		copyValue := copyValueIf.Elem()
		childIncludeString, childImportString, err = translateRecursive(copyValue, originalValue, workingPath)
		if err != nil {
			return
		}
		if childIncludeString != "" {
			includeString = childIncludeString
			//fmt.Println("B")
			//originalValue = reflect.Indirect(reflect.ValueOf(&childIncludeString))
			//originalValue.SetString(childIncludeString)
		}
		if childImportString != "" {
			//fmt.Println("childImportString B")
			importString = childImportString
		}
		copy.Set(copyValue)

	// If it is a struct we translate each field
	case reflect.Struct:
		for i := 0; i < original.NumField(); i++ {
			childIncludeString, childImportString, err = translateRecursive(copy.Field(i), original.Field(i), workingPath)
			if err != nil {
				return
			}
			if childIncludeString != "" {
				//fmt.Println("C")
			}
			if childImportString != "" {
				fmt.Println("childImportString C")
			}
		}

	// If it is a slice we create a new slice and translate each element
	case reflect.Slice:
		copy.Set(reflect.MakeSlice(original.Type(), original.Len(), original.Cap()))
		for i := 0; i < original.Len(); i++ {
			childIncludeString, childImportString, err = translateRecursive(copy.Index(i), original.Index(i), workingPath)
			if err != nil {
				return
			}
			if childIncludeString != "" {
				//fmt.Println("D")

				var includeContentBytes []byte

				fullPath := path.Join(workingPath, childIncludeString)
				includeContentBytes, err = ioutil.ReadFile(fullPath)
				if err != nil {
					err = fmt.Errorf("(translateRecursive) could not read file %s: %s", fullPath, err.Error())
					return
				}

				includeContentStr := string(includeContentBytes)
				//fmt.Printf("File contents: %s", includeContentStr)

				includeContentStrValue := reflect.ValueOf(&includeContentStr)
				includeContentStrValuePValue := reflect.Indirect(includeContentStrValue)

				copy.Index(i).Set(includeContentStrValuePValue)

			}
			if childImportString != "" {
				//fmt.Println("D")

				//var importContentBytes []byte

				fullPath := path.Join(workingPath, childImportString)

				fullPathArray := strings.Split(fullPath, "#")
				fullPath = fullPathArray[0]
				//entrypoint := ""
				// if len(fullPathArray) > 1 {
				// 	entrypoint = fullPathArray[1]
				// }

				var objectIf interface{}
				objectIf, err = ReadYamlFile(fullPath)
				//var objectArray []NamedCWLObject
				//objectArray, _, _, _, _, err = ParseCWLDocumentFile(nil, fullPath, entrypoint, workingPath, path.Base(fullPath))

				if err != nil {
					err = fmt.Errorf("(translateRecursive) ReadYamlFile returned: %s", err.Error())
					return
				}

				// if len(objectArray) == 0 {
				// 	err = fmt.Errorf("(translateRecursive) len(objectArray) == 0 ")
				// 	return
				// }
				importContentStrValue := reflect.ValueOf(&objectIf)

				importContentStrValuePValue := reflect.Indirect(importContentStrValue)

				copy.Index(i).Set(importContentStrValuePValue)

			}
		}

	// If it is a map we create a new map and translate each value
	case reflect.Map:
		copy.Set(reflect.MakeMap(original.Type()))

		// test if object has "$import" key
		importField := original.MapIndex(importStrValue)
		if importField.IsValid() {

			importString = importField.Elem().String()
			return
			//fmt.Println("string: " + showText)
			//panic("found include field !")

		}

		// test if object has "$include" key
		includeField := original.MapIndex(includeStrValue)
		if includeField.IsValid() {

			includeString = includeField.Elem().String()
			return
			//fmt.Println("string: " + showText)
			//panic("found include field !")

		}

		for _, key := range original.MapKeys() {
			originalValue := original.MapIndex(key)
			// New gives us a pointer, but again we want the value
			copyValue := reflect.New(originalValue.Type()).Elem()
			childIncludeString, childImportString, err = translateRecursive(copyValue, originalValue, workingPath)
			if err != nil {
				return
			}

			if childIncludeString != "" {
				panic("E")
			}
			if childImportString != "" {
				//fmt.Println("childImportString E")
				fullPath := path.Join(workingPath, childImportString)

				fullPathArray := strings.Split(fullPath, "#")
				fullPath = fullPathArray[0]
				// entrypoint := ""
				// if len(fullPathArray) > 1 {
				// 	entrypoint = fullPathArray[1]
				// }

				var objectIf interface{}
				objectIf, err = ReadYamlFile(fullPath)

				//use document reader here ?
				//var objectArray []NamedCWLObject
				//objectArray, _, _, _, _, err = ParseCWLDocumentFile(nil, fullPath, entrypoint, workingPath, path.Base(fullPath))

				if err != nil {
					err = fmt.Errorf("(translateRecursive) ReadYamlFile returned: %s", err.Error())
					return
				}

				// if len(objectArray) == 0 {
				// 	err = fmt.Errorf("(translateRecursive) len(objectArray) == 0 ")
				// 	return
				// }
				importContentStrValue := reflect.ValueOf(&objectIf)
				// var importContentStrValue reflect.Value
				// if len(objectArray) == 1 {
				// 	//fmt.Println("objectArray[0].Value:")
				// 	//spew.Dump(objectArray[0].Value)

				// 	importContentStrValue = reflect.ValueOf(&objectArray[0].Value)
				// } else {
				// 	err = fmt.Errorf("(translateRecursive) not implemented yet")
				// 	return
				// 	//importContentStrValue = reflect.ValueOf(&objectArray)
				// }

				//importContentStrValuePValue := reflect.Indirect(importContentStrValue)
				copyValue = reflect.Indirect(importContentStrValue)
			}
			copy.SetMapIndex(key, copyValue)
		}

	// Otherwise we cannot traverse anywhere so this finishes the the recursion

	// If it is a string translate it (yay finally we're doing what we came for)
	//case reflect.String:
	//	translatedString := dict[original.Interface().(string)]
	//	copy.SetString(translatedString)

	// And everything else will simply be taken from the original
	default:
		copy.Set(original)
	}

	return
}
