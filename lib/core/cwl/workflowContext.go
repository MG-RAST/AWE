package cwl

type WorkflowContext struct {
	Path       string
	Namespaces map[string]string

	// old ParsingContext
	If_objects map[string]interface{}
	Objects    map[string]CWL_object
}
