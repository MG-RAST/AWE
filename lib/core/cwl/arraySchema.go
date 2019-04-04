package cwl

import "fmt"

//"fmt"

// ArraySchema _
type ArraySchema struct {
	Schema `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"` // provides Type, Label
	Items  []CWLType_Type                                                        `yaml:"items,omitempty" bson:"items,omitempty" json:"items,omitempty" mapstructure:"items,omitempty"` // string or []string ([] speficies which types are possible, e.g ["File" , "null"])
}

func (c *ArraySchema) Is_Type()            {}
func (c *ArraySchema) Type2String() string { return "array" }
func (c *ArraySchema) GetID() string       { return "" }

func NewArraySchema() (as *ArraySchema) {
	as = &ArraySchema{}
	as.Schema = Schema{}
	as.Schema.Type = CWLArray
	return
}

func NewArraySchemaFromMap(original_map map[string]interface{}, schemata []CWLType_Type, context_p string, context *WorkflowContext) (as *ArraySchema, err error) {
	as = &ArraySchema{}
	as.Schema = Schema{}
	as.Schema.Type = CWLArray

	items, ok := original_map["items"]
	if !ok {

		err = fmt.Errorf("(NewArraySchemaFromMap) items are missing")
		return
	}
	var items_type []CWLType_Type
	items_type, err = NewCWLType_TypeArray(items, schemata, context_p, false, context)
	if err != nil {
		err = fmt.Errorf("(NewOutputArraySchemaFromInterface) NewCWLType_TypeArray returns: %s", err.Error())
		return
	}

	as.Items = items_type

	return
}
