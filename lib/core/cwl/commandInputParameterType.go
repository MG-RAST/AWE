package cwl

//"fmt"
//"github.com/davecgh/go-spew/spew"
//"strings"
//"github.com/mitchellh/mapstructure"
//"reflect"

//type CommandInputParameterType struct {
//	Type string
//}

// CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string | array<CWLType | CommandInputRecordSchema | CommandInputEnumSchema | CommandInputArraySchema | string>

// NewCommandInputParameterTypeArray _
func NewCommandInputParameterTypeArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (returnValue interface{}, err error) { // []CWLType_Type did not work

	//fmt.Println("NewCommandInputParameterTypeArray:")
	//spew.Dump(original)

	//result = []CWLType_Type{}

	switch original.(type) {
	case []interface{}:

		originalArray := original.([]interface{})

		returnArray := []CWLType_Type{}

		for _, element := range originalArray {

			var ciptArray []CWLType_Type
			ciptArray, err = NewCWLType_Type(schemata, element, "CommandInput", context)
			if err != nil {
				return
			}

			for _, ciptElement := range ciptArray {
				returnArray = append(returnArray, ciptElement)
			}

		}
		returnValue = returnArray
		return

	default:

		var ciptArray []CWLType_Type
		ciptArray, err = NewCWLType_Type(schemata, original, "CommandInput", context)
		if err != nil {
			return
		}
		returnValue = ciptArray[0]
		//for _, ciptElement := range ciptArray {
		//	returnArray = append(returnArray, ciptElement)
		//}

	}
	return
}
