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
func NewCommandInputParameterTypeArray(original interface{}, schemata []CWLType_Type, context *WorkflowContext) (returnArray []CWLType_Type, err error) {

	//fmt.Println("NewCommandInputParameterTypeArray:")
	//spew.Dump(original)

	//result = []CWLType_Type{}
	returnArray = []CWLType_Type{}

	switch original.(type) {
	case []interface{}:

		originalArray := original.([]interface{})

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

		return

	default:

		var ciptArray []CWLType_Type
		ciptArray, err = NewCWLType_Type(schemata, original, "CommandInput", context)
		if err != nil {
			return
		}
		for _, ciptElement := range ciptArray {
			returnArray = append(returnArray, ciptElement)
		}

	}
	return
}
