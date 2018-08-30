package cwl

import (
	"bytes"
	"encoding/json"
	"fmt"
	"reflect"
	"regexp"
	"strings"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	"github.com/robertkrimen/otto"
)

type Expression string

func (e Expression) String() string { return string(e) }

func (e Expression) EvaluateExpression(self interface{}, inputs interface{}) (result interface{}, err error) {
	vm := otto.New()
	var self_json []byte
	self_json, err = json.Marshal(self)
	if err != nil {
		err = fmt.Errorf("(LongStringExpression) json.Marshal returns: %s", err.Error())
		return
	}
	logger.Debug(3, "SET self=%s\n", self_json)

	spew.Dump("inputs")

	var inputs_json []byte
	inputs_json, err = json.Marshal(inputs)
	if err != nil {
		err = fmt.Errorf("(LongStringExpression) json.Marshal returns: %s", err.Error())
		return
	}
	logger.Debug(3, "SET inputs=%s\n", inputs_json)

	reg := regexp.MustCompile(`\$\(.+\)`)
	// CWL documentation: http://www.commonwl.org/v1.0/Workflow.html#Expressions

	parsed_str := e.String()

	matches := reg.FindAll([]byte(parsed_str), -1)
	fmt.Printf("()Matches: %d\n", len(matches))
	if len(matches) > 0 {

		concatenate := false
		if len(matches) > 1 {
			concatenate = true
		}

		for _, match := range matches {
			expression_string := bytes.TrimPrefix(match, []byte("$("))
			expression_string = bytes.TrimSuffix(expression_string, []byte(")"))

			javascript_function := fmt.Sprintf("(function(){\n self=%s ; inputs=%s; return %s;\n})()", self_json, inputs_json, expression_string)
			fmt.Printf("%s\n", javascript_function)

			value, xerr := vm.Run(javascript_function)
			if xerr != nil {
				err = fmt.Errorf("(EvaluateExpression) Javascript complained: A) %s", xerr.Error())
				return
			}
			//fmt.Printf("(EvaluateExpression) reflect.TypeOf(value): %s\n", reflect.TypeOf(value))

			//if value.IsNumber()
			if concatenate {
				value_str, xerr := value.ToString()
				if xerr != nil {
					err = fmt.Errorf("(EvaluateExpression) Cannot convert value to string: %s", xerr.Error())
					return
				}
				parsed_str = strings.Replace(parsed_str, string(match), value_str, 1)
			} else {

				var value_returned CWLType
				var exported_value interface{}
				//https://godoc.org/github.com/robertkrimen/otto#Value.Export
				exported_value, err = value.Export()

				if err != nil {
					err = fmt.Errorf("(EvaluateExpression)  value.Export() returned: %s", err.Error())
					return
				}
				logger.Debug(3, "(EvaluateExpression) exported_value type: %s", reflect.TypeOf(exported_value))
				switch exported_value.(type) {

				case string:

					value_returned = NewString(exported_value.(string))

				case bool:

					value_returned = NewBooleanFrombool(exported_value.(bool))

				case int:
					value_returned = NewInt(exported_value.(int))
				case int64:
					value_returned = NewLong(exported_value.(int64))
				case float32:
					value_returned = NewFloat(exported_value.(float32))
				case float64:
					//fmt.Println("got a double")
					value_returned = NewDouble(exported_value.(float64))
				case uint64:
					value_returned = NewInt(exported_value.(int))

				case []interface{}: //Array
					err = fmt.Errorf("(EvaluateExpression) array not supported yet")
					return
				case interface{}: //Object

					//fmt.Println("(EvaluateExpression) object:")
					//spew.Dump(exported_value)
					//err = fmt.Errorf("(EvaluateExpression) record not supported yet")

					value_returned, err = NewCWLType("", exported_value)
					if err != nil {
						err = fmt.Errorf("(EvaluateExpression) NewCWLType returned: %s", err.Error())
						return
					}

				case nil:
					value_returned = NewNull()
				default:
					err = fmt.Errorf("(EvaluateExpression) js return type not supoported: (%s)", reflect.TypeOf(exported_value))
					return
				}

				logger.Debug(3, "(EvaluateExpression) value_returned type: %s", reflect.TypeOf(value_returned))
				//fmt.Println("value_returned:")
				//spew.Dump(value_returned)
				result = value_returned
				return
			}
		} // for matches

		//if concatenate
		result = NewString(parsed_str)
		return

	} // if matches
	//}

	//fmt.Printf("parsed_str: %s\n", parsed_str)

	// evaluate ${...} ECMAScript function body
	reg = regexp.MustCompile(`(?s)\${.+}`) // s-flag is needed to include newlines

	// CWL documentation: http://www.commonwl.org/v1.0/Workflow.html#Expressions

	matches = reg.FindAll([]byte(parsed_str), -1)
	//fmt.Printf("{}Matches: %d\n", len(matches))
	if len(matches) == 0 {
		result = NewString(parsed_str)
		return
	}

	if len(matches) == 1 {
		match := matches[0]
		expression_string := bytes.TrimPrefix(match, []byte("${"))
		expression_string = bytes.TrimSuffix(expression_string, []byte("}"))

		javascript_function := fmt.Sprintf("(function(){\n self=%s ; inputs=%s; %s \n})()", self_json, inputs_json, expression_string)
		fmt.Printf("%s\n", javascript_function)

		value, xerr := vm.Run(javascript_function)
		if xerr != nil {
			err = fmt.Errorf("Javascript complained: B) %s", xerr.Error())
			return
		}

		value_exported, _ := value.Export()

		fmt.Printf("reflect.TypeOf(value_exported): %s\n", reflect.TypeOf(value_exported))

		var value_cwl CWLType
		value_cwl, err = NewCWLType("", value_exported)
		if err != nil {
			err = fmt.Errorf("(NewWorkunit) Error parsing javascript VM result value, cwl.NewCWLType returns: %s", err.Error())
			return
		}

		result = value_cwl
		return
	}

	err = fmt.Errorf("(NewWorkunit) ValueFrom contains more than one ECMAScript function body")
	return

}

//var CWL_Expression CWLType_Type = "expression"
func NewExpressionFromString(original string) (expression *Expression) {

	expression_npr := Expression(original)
	expression = &expression_npr

	return
}

func NewExpression(original interface{}) (expression *Expression, err error) {
	switch original.(type) {
	case string:
		expression_str := original.(string)
		//fmt.Printf("--------------------------------- expression_str: %s\n", expression_str)
		expression = NewExpressionFromString(expression_str)

	default:
		err = fmt.Errorf("cannot parse Expression, unkown type %s", reflect.TypeOf(original))
	}
	return

}

func NewExpressionArray(original interface{}) (expressions *[]Expression, err error) {

	switch original.(type) {
	case string:
		original_str := original.(string)

		expression, xerr := NewExpression(original_str)
		if xerr != nil {
			err = fmt.Errorf("(NewExpressionArray) NewExpression returns: %s", xerr.Error())
			return
		}
		expressions_nptr := []Expression{*expression}
		expressions = &expressions_nptr
	case []string:
		original_array, ok := original.([]string)
		if !ok {
			err = fmt.Errorf("(NewExpressionArray) type assertion error")
			return
		}

		expression_array := []Expression{}

		for _, value := range original_array {

			var exp *Expression
			exp, err = NewExpression(value)
			if err != nil {
				err = fmt.Errorf("(NewExpressionArray) NewExpression returns: %s", err.Error())
				return
			}

			expression_array = append(expression_array, *exp)
		}

		expressions = &expression_array

	case []interface{}:

		string_array := []string{}
		original_array := original.([]interface{})
		for _, value := range original_array {
			value_str, ok := value.(string)
			if !ok {
				err = fmt.Errorf("(NewExpressionArray) value was not string")
				return
			}
			string_array = append(string_array, value_str)
		}

		return NewExpressionArray(string_array)

	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewExpressionArray) cannot parse Expression array, unknown type %s", reflect.TypeOf(original))
	}
	return

}
