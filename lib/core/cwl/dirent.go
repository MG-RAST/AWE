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
	fmt.Println("(Dirent/Evaluate) start")
	//*** d.Entry
	var ok bool

	entry := d.Entry

	switch entry.(type) {
	case Expression, string:
		var entry_expr Expression

		entry_expr, ok = entry.(Expression)

		if !ok {
			var entry_str string
			entry_str, ok = entry.(string)
			if !ok {
				err = fmt.Errorf("(Dirent/Evaluate) neither string nor Expression !?")
				return
			}
			entry_expr = *NewExpressionFromString(entry_str)
		}

		fmt.Println("(Dirent/Evaluate) Entry")
		var new_value interface{}
		fmt.Printf("(Dirent/Evaluate) entry_expr: %s\n", entry_expr.String())
		new_value, err = entry_expr.EvaluateExpression(nil, inputs)
		if err != nil {
			err = fmt.Errorf("(Dirent/Evaluate) EvaluateExpression returned: %s", err.Error())
			return
		}

		fmt.Println("(Dirent/Evaluate) new value")
		// verify return type:
		switch new_value.(type) {
		case String, string:
			fmt.Printf("(Dirent/Evaluate) new_value is a string: %s\n", new_value.(string))
			// valid returns
			d.Entry = new_value
		default:
			err = fmt.Errorf("(Dirent/Evaluate) EvaluateExpression returned type %s, this is not expected", reflect.TypeOf(new_value))
			return

		}
		fmt.Println("(Dirent/Evaluate) new value, done")
	case nil:
		//ignore

	default:
		panic(fmt.Sprintf("(Dirent/Evaluate) type not expected: %s", reflect.TypeOf(entry)))
		err = fmt.Errorf("(Dirent/Evaluate) type not expected: %s", reflect.TypeOf(entry))
		return
	}

	// *** Entryname

	entryname := d.Entryname

	switch entryname.(type) {
	case Expression, string:
		var entryname_expr Expression

		entryname_expr, ok = entryname.(Expression)

		if !ok {
			var entryname_str string
			entryname_str, ok = entryname.(string)
			if !ok {
				err = fmt.Errorf("(Dirent/Evaluate) neither string nor Expression !?")
				return
			}
			entryname_expr = *NewExpressionFromString(entryname_str)
		}

		fmt.Println("(Dirent/Evaluate) entryname")
		var new_value interface{}
		fmt.Printf("(Dirent/Evaluate) entryname_expr: %s\n", entryname_expr.String())
		new_value, err = entryname_expr.EvaluateExpression(nil, inputs)
		if err != nil {
			err = fmt.Errorf("(Dirent/Evaluate) EvaluateExpression returned: %s", err.Error())
			return
		}

		// verify return type:
		switch new_value.(type) {
		case String, string:
			fmt.Printf("(Dirent/Evaluate) new_value is a string: %s\n", new_value.(string))
			// valid returns
			d.Entryname = new_value
		default:
			err = fmt.Errorf("(Dirent/Evaluate) EvaluateExpression returned type %s, this is not expected", reflect.TypeOf(new_value))
			return

		}
	case nil:
		//ignore

	default:
		panic(fmt.Sprintf("(Dirent/Evaluate) type not expected: %s", reflect.TypeOf(entryname)))
		err = fmt.Errorf("(Dirent/Evaluate) type not expected: %s", reflect.TypeOf(entryname))
		return
	}

	return

}
