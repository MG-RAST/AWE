package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
)

type Expression string

func (e Expression) String() string { return string(e) }

//var CWL_Expression CWLType_Type = "expression"
func NewExpressionFromString(original string) (expression *Expression, err error) {

	expression_npr := Expression(original)
	expression = &expression_npr

	return
}

func NewExpression(original interface{}) (expression *Expression, err error) {
	switch original.(type) {
	case string:
		expression_str := original.(string)
		//fmt.Printf("--------------------------------- expression_str: %s\n", expression_str)
		expression, err = NewExpressionFromString(expression_str)
		if err != nil {
			err = fmt.Errorf("(NewExpression)  NewExpressionFromString returned: %s", err.Error())
			return
		}

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
