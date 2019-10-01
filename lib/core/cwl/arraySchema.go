package cwl

import (
	"fmt"

	"github.com/davecgh/go-spew/spew"
)

//"fmt"

// ArraySchema _
type ArraySchema struct {
	Schema `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"` // provides Type, Label
	Items  []CWLType_Type                                                        `yaml:"items,omitempty" bson:"items,omitempty" json:"items,omitempty" mapstructure:"items,omitempty"` // string or []string ([] speficies which types are possible, e.g ["File" , "null"])
}

// Is_Type _
func (c ArraySchema) Is_Type() {}

// Type2String _
func (c ArraySchema) Type2String() string { return "array" }

// GetID _
func (c ArraySchema) GetID() string { return "" }

// NewArraySchema _
func NewArraySchema() (as *ArraySchema) {
	as = &ArraySchema{}
	as.Schema = Schema{}
	as.Schema.Type = CWLArray
	return
}

// NewArraySchemaFromMap _
func NewArraySchemaFromMap(originalMap map[string]interface{}, schemata []CWLType_Type, contextP string, context *WorkflowContext) (as *ArraySchema, err error) {
	as = &ArraySchema{}
	as.Schema = Schema{}
	as.Schema.Type = CWLArray

	items, ok := originalMap["items"]
	if !ok {
		fmt.Println("NewArraySchemaFromMap:")
		spew.Dump(originalMap)
		err = fmt.Errorf("(NewArraySchemaFromMap) items are missing")
		return
	}
	var itemsType []CWLType_Type
	itemsType, err = NewCWLType_TypeArray(items, schemata, contextP, false, context)
	if err != nil {
		err = fmt.Errorf("(NewOutputArraySchemaFromInterface) NewCWLType_TypeArray returns: %s", err.Error())
		return
	}

	if len(itemsType) == 0 {
		err = fmt.Errorf("(NewOutputArraySchemaFromInterface) len(items_type) == 0")
		return
	}

	as.Items = itemsType

	return
}
