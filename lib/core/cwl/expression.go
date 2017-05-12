package cwl

import (
	"fmt"
)

type Expression string

func (e Expression) String() string { return string(e) }
func (e Expression) Evaluate(collection *CWL_collection) string {
	result_str := (*collection).Evaluate(string(e))
	return result_str
}

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
			err = xerr
			return
		}
		expressions_nptr := []Expression{*expression}
		expressions = &expressions_nptr
	case []string:
		expressions_nptr := original.([]Expression)
		expressions = &expressions_nptr
	default:
		err = fmt.Errorf("cannot parse Expression array, unknown type")
	}
	return

}
