package cwl

type WorkflowOutputParameterType struct {
	Type               string
	OutputRecordSchema *OutputRecordSchema
	OutputEnumSchema   *OutputEnumSchema
	OutputArraySchema  *OutputArraySchema
}

type OutputRecordSchema struct{}

type OutputEnumSchema struct{}

type OutputArraySchema struct{}

func NewWorkflowOutputParameterType(original interface{}) (wopt_ptr *WorkflowOutputParameterType, err error) {
	wopt := WorkflowOutputParameterType{}
	wopt_ptr = &wopt

	switch original.(type) {
	case string:

		wopt.Type = original.(string)

		return
	case map[interface{}]interface{}:

		output_type, ok := original_map["type"]

		if !ok {
			fmt.Printf("unknown type")
			spew.Dump(original)
			err = fmt.Errorf("(NewWorkflowOutputParameterType) Map-Type unknown")
		}

		switch output_type {
		case "OutputRecordSchema":
			wopt.OutputRecordSchema = &OutputRecordSchema{}
		case "OutputEnumSchema":
			wopt.OutputEnumSchema = &OutputEnumSchema{}
		case "OutputArraySchema":
			wopt.OutputArraySchema = &OutputArraySchema{}
		default:
			err = fmt.Errorf("(NewWorkflowOutputParameterType) type %s is unknown", output_type)

		}

	default:
		err = fmt.Errorf("(NewWorkflowOutputParameterType) unknown type")
		return
	}

	return
}
