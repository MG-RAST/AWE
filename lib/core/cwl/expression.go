package cwl

type Expression string

func (e Expression) String() string { return string(e) }
func (e Expression) Evaluate(collection *CWL_collection) string {
	result_str := (*collection).Evaluate(string(e))
	return result_str
}
