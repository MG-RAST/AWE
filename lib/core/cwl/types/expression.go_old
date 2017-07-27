package cwl

import (
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
)

func EvaluateExpression(collection *CWL_collection, e cwl_types.Expression) string {
	result_str := (*collection).Evaluate(string(e))
	return result_str
}
