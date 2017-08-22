package cwl

import (
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"reflect"
)

type Expression string

func (e Expression) String() string { return string(e) }

func NewExpression(original interface{}) (expression *Expression, err error) {
	switch original.(type) {
	case string:
		expression_str := original.(string)

		expression_nptr := Expression(expression_str)

		expression = &expression_nptr

	default:
		err = fmt.Errorf("cannot parse Expression, wrong type")
	}
	return

}

func NewExpressionArray(original interface{}) (expressions *[]Expression, err error) {

	switch original.(type) {
	case string:
		expression, xerr := NewExpression(original)
		if xerr != nil {
			err = fmt.Errorf("(NewExpressionArray) NewExpression returns: %s", xerr.Error)
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

			exp, xerr := NewExpression(value)
			if xerr != nil {
				err = fmt.Errorf("(NewExpressionArray) NewExpression returns: %s", xerr.Error())
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
