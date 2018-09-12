package cwl

//"fmt"

type ArraySchema struct {
	Schema `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"` // provides Type, Label
	Items  []CWLType_Type                                                        `yaml:"items,omitempty" bson:"items,omitempty" json:"items,omitempty" mapstructure:"items,omitempty"` // string or []string ([] speficies which types are possible, e.g ["File" , "null"])
}

func (c *ArraySchema) Is_Type()            {}
func (c *ArraySchema) Type2String() string { return "array" }
func (c *ArraySchema) GetId() string       { return "" }

func NewArraySchema() (as *ArraySchema) {
	as = &ArraySchema{}
	as.Schema = Schema{}
	as.Schema.Type = CWL_array
	return
}
