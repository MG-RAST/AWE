package cwl

import (
	"fmt"
	"path"
	"reflect"
	"strings"

	uuid "github.com/MG-RAST/golib/go-uuid/uuid"
)

// ProcessImpl _
type ProcessImpl struct {
	CWLObjectImpl  `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	CWL_class_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	IdentifierImpl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"`
	Requirements   []Requirement `yaml:"requirements,omitempty" bson:"requirements,omitempty" json:"requirements,omitempty" mapstructure:"requirements,omitempty"`
	Hints          []Requirement `yaml:"hints,omitempty" bson:"hints,omitempty" json:"hints,omitempty" mapstructure:"hints,omitempty"`
}

// IsProcess _
func (p *ProcessImpl) IsProcess() {}

// ProcessImplInit _
// process is a pointer to the emebdded Process of Workflow and Tools.
func ProcessImplInit(generic interface{}, process *ProcessImpl, parentIdentifier string, objectIdentifier string, context *WorkflowContext) (err error) {

	object, ok := generic.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("other types than map[string]interface{} not supported yet (got %s)", reflect.TypeOf(generic))
		return
	}

	// id
	if objectIdentifier != "" {
		object["id"] = objectIdentifier
	} else {

		objectIDIf, hasID := object["id"]
		if hasID {

			objectIdentifier = objectIDIf.(string)

			if !strings.HasPrefix(objectIdentifier, "#") {
				if parentIdentifier == "" {
					err = fmt.Errorf("(NewCommandLineTool) A) parentIdentifier is needed but empty, objectIdentifier=%s", objectIdentifier)
					return
				}
				objectIdentifier = path.Join(parentIdentifier, objectIdentifier)
			}

			if !strings.HasPrefix(objectIdentifier, "#") {
				err = fmt.Errorf("(NewCommandLineTool) not absolute: objectIdentifier=%s , parentIdentifier=%s, objectIdentifier=%s", objectIdentifier, parentIdentifier, objectIdentifier)
				return
			}
			object["id"] = objectIdentifier
		} else {
			if parentIdentifier == "" {
				err = fmt.Errorf("(NewCommandLineTool) B) parentIdentifier is needed but empty")
				return
			}

			objectIdentifier = path.Join(parentIdentifier, uuid.New())
			object["id"] = objectIdentifier
		}
	}
	process.ID = objectIdentifier
	//object["id"] = objectIdentifier

	// SchemaDefRequirement
	// extract SchemaDefRequirement and puf into context
	//var schemaDefReq *SchemaDefRequirement
	//var schemataNew []CWLType_Type
	//hasSchemaDefReq := false

	requirementsIf, hasRequirements := object["requirements"]
	if hasRequirements {
		// 	schemaDefReq, hasSchemaDefReq,
		_, _, err = GetSchemaDefRequirement(requirementsIf, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) GetSchemaDefRequirement returned: %s", err.Error())
			return
		}

		// //spew.Dump(requirementsIf)
		// if hasSchemaDefReq {
		// 	fmt.Println("hasSchemaDefReq")
		// } else {
		// 	fmt.Println("not hasSchemaDefReq")
		// }
		//spew.Dump(schemaDefReq)
		//panic("schemaDefReq--")
	}

	return
}

// CreateRequirementAndHints _
func CreateRequirementAndHints(generic map[string]interface{}, process *ProcessImpl, injectedRequirements []Requirement, inputs interface{}, context *WorkflowContext) (err error) {

	var requirementsArray []Requirement

	requirementsIf, _ := generic["requirements"]
	// if hasRequirements {

	// 	requirementsArray, err = CreateRequirementArrayAndInject(requirementsIf, injectedRequirements, nil, context)
	// 	if err != nil {
	// 		err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (requirements): %s", err.Error())
	// 		return
	// 	}

	// 	//for i, _ := range schemataNew {
	// 	//	schemata = append(schemata, schemataNew[i])
	// 	//}

	// 	//p.Requirements = requirementsArray

	// }

	requirementsArray, err = CreateRequirementArrayAndInject(requirementsIf, injectedRequirements, nil, context)
	if err != nil {
		err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (requirements): %s", err.Error())
		return
	}
	process.Requirements = requirementsArray
	//	generic["requirements"] = requirementsArray

	hints, ok := generic["hints"]
	if ok && (hints != nil) {
		//var schemataNew []CWLType_Type

		var hintsArray []Requirement
		hintsArray, err = CreateHintsArray(hints, injectedRequirements, inputs, context)
		if err != nil {
			err = fmt.Errorf("(NewCommandLineTool) error in CreateRequirementArray (hints): %s", err.Error())
			return
		}
		//for i, _ := range schemataNew {
		//	schemata = append(schemata, schemataNew[i])
		//}
		//fmt.Println("hintsArray:")
		//spew.Dump(hintsArray)
		//generic["hints"] = hintsArray
		process.Hints = hintsArray
	}

	return
}
