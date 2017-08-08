package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	//"reflect"
)

//http://www.commonwl.org/v1.0/Workflow.html#CommandLineBinding
type CommandLineBinding struct {
	LoadContents  bool   `yaml:"loadContents,omitempty" bson:"loadContents,omitempty" json:"loadContents,omitempty"`
	Position      int    `yaml:"position,omitempty" bson:"position,omitempty" json:"position,omitempty"`
	Prefix        string `yaml:"prefix,omitempty" bson:"prefix,omitempty" json:"prefix,omitempty"`
	Separate      string `yaml:"separate,omitempty" bson:"separate,omitempty" json:"separate,omitempty"`
	ItemSeparator string `yaml:"itemSeparator,omitempty" bson:"itemSeparator,omitempty" json:"itemSeparator,omitempty"`
	ValueFrom     string `yaml:"valueFrom,omitempty" bson:"valueFrom,omitempty" json:"valueFrom,omitempty"`
	ShellQuote    bool   `yaml:"shellQuote,omitempty" bson:"shellQuote,omitempty" json:"shellQuote,omitempty"`
}

func NewCommandLineBinding(original interface{}) (clb *CommandLineBinding, err error) {

	var commandlinebinding CommandLineBinding

	switch original.(type) {
	case map[interface{}]interface{}:

		err = mapstructure.Decode(original, &commandlinebinding)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineBinding) %s", err.Error())
			return
		}
		clb = &commandlinebinding

	case string:

		org_str, _ := original.(string)

		commandlinebinding.ValueFrom = org_str

	default:
		err = fmt.Errorf("(NewCommandLineBinding) type unknown")
		return
	}

	return
}

func NewCommandLineBindingArray(original interface{}) (new_array []*CommandLineBinding, err error) {
	switch original.(type) {
	case []interface{}:
		for _, v := range original.([]interface{}) {

			clb, xerr := NewCommandLineBinding(v)
			if xerr != nil {
				err = fmt.Errorf("(NewCommandLineBindingArray) []interface{} NewCommandLineBinding returned: %s", xerr)
				return
			}

			//fmt.Printf("C")
			new_array = append(new_array, clb)
			//fmt.Printf("D")

		}
		return
	case string:

		clb, xerr := NewCommandLineBinding(original)
		if xerr != nil {
			err = fmt.Errorf("(NewCommandLineBindingArray) string NewCommandLineBinding returned: %s", xerr)
			return
		}

		new_array = append(new_array, clb)

	default:
		err = fmt.Errorf("(NewCommandLineBindingArray) type unknown")
		return
	}
	return
}
