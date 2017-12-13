package cwl

import ()

type InputEnumSchema struct {
	EnumSchema   `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // provides Symbols, Type, Label
	InputBinding *CommandLineBinding                                                   `yaml:"inputBinding,omitempty" json:"inputBinding,omitempty" bson:"inputBinding,omitempty"`
}

func NewInputEnumSchemaFromInterface(original interface{}) (ies *InputEnumSchema, err error) {


	original, err = MakeStringMap(original)
	if err != nil {
		return
	}
	
	switch (original.type()) {
		
		
	case map[string]interface{}:
		
		
		
		
		
	default:
		err = fmt.Errorf("(NewInputEnumSchemaFromInterface) error")
		return
		
	}
	

}
