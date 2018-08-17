package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

//"github.com/davecgh/go-spew/spew"

// Dirent http://www.commonwl.org/v1.0/CommandLineTool.html#Dirent
type Dirent struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Entry        interface{} `yaml:"entry" json:"entry" bson:"entry" mapstructure:"entry"`                 // string, Expression
	Entryname    interface{} `yaml:"entryname" json:"entryname" bson:"entryname" mapstructure:"entryname"` // string, Expression
	Writable     bool        `yaml:"writable" json:"writable" bson:"writable" mapstructure:"writable"`
}

func NewDirentFromInterface(id string, original interface{}) (dirent *Dirent, err error) {

	dirent = &Dirent{}
	err = mapstructure.Decode(original, dirent)
	if err != nil {
		err = fmt.Errorf("(NewDirentFromInterface) Could not convert Dirent object: %s", err.Error())
		return
	}

	return
}

func (d *Dirent) Evaluate(inputs interface{}) (err error) {

	if inputs == nil {
		err = fmt.Errorf("(Dirent/Evaluate) no inputs")
		return
	}

	//*** d.Entry
	var ok bool
	var entry_expr Expression
	entry_expr, ok = d.Entry.(Expression)
	if ok {
		var new_value interface{}
		new_value, err = entry_expr.EvaluateExpression(nil, inputs)
		if err != nil {
			err = fmt.Errorf("(Dirent/Evaluate) EvaluateExpression returned: %s", err.Error())
			return
		}

		// verify return type:
		switch new_value.(type) {
		case String, string:
			// valid returns
			d.Entry = new_value
		default:
			err = fmt.Errorf("(Dirent/Evaluate) EvaluateExpression returned type %s, this is not expected", reflect.TypeOf(new_value))
			return

		}
	}

	//*** d.Entryname
	var entryname_expr Expression
	entryname_expr, ok = d.Entryname.(Expression)
	if ok {
		var new_value interface{}
		new_value, err = entryname_expr.EvaluateExpression(nil, inputs)
		if err != nil {
			err = fmt.Errorf("(Dirent/Evaluate) EvaluateExpression returned: %s", err.Error())
			return
		}

		// verify return type:
		switch new_value.(type) {
		case String, string:
			// valid returns
			d.Entryname = new_value
		default:
			err = fmt.Errorf("(Dirent/Evaluate) EvaluateExpression returned type %s, this is not expected", reflect.TypeOf(new_value))
			return

		}
	}

	return

}
