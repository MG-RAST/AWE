package cwl

import (
	"errors"
	"fmt"
	"reflect"

	//"github.com/MG-RAST/AWE/lib/logger"

	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
)

// Requirement _
type Requirement interface {
	CWLObject
	GetClass() string
	Evaluate(inputs interface{}, context *WorkflowContext) error
}

// DummyRequirement _
type DummyRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
}

// NewRequirementFromInterface _
func NewRequirementFromInterface(obj interface{}, inputs interface{}, context *WorkflowContext) (r Requirement, err error) {

	obj, err = MakeStringMap(obj, context)
	if err != nil {
		return
	}

	switch obj.(type) {
	case map[string]interface{}:

		var classStr string
		classStr, err = GetClass(obj)
		if err != nil {
			err = fmt.Errorf("(NewRequirementFromInterface) GetClass returned: %s", err.Error())
			return
		}
		//var schemata []CWLType_Type
		r, err = NewRequirement(classStr, obj, inputs, context)
		if err != nil {
			err = fmt.Errorf("(NewRequirementFromInterface) NewRequirement retured: %s", err.Error())
			return
		}
		return
	}

	var ok bool
	r, ok = obj.(Requirement)
	if ok {
		return
	}

	err = fmt.Errorf("(NewRequirementFromInterface) type not recognized: %s", reflect.TypeOf(obj))

	return

}

// NewRequirement _
func NewRequirement(class string, obj interface{}, inputs interface{}, context *WorkflowContext) (r Requirement, err error) {

	if class == "" {
		err = fmt.Errorf("class name empty")
		return
	}

	switch class {
	case "DockerRequirement":
		r, err = NewDockerRequirement(obj)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewDockerRequirement returns: %s", err.Error())
			return
		}
		return
	case "ShellCommandRequirement":
		r, err = NewShellCommandRequirement(obj)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewShellCommandRequirement returns: %s", err.Error())
			return
		}
		return
	case "ResourceRequirement":
		r, err = NewResourceRequirement(obj, inputs, context)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewResourceRequirement returns: %s", err.Error())
			return
		}
		return
	case "InlineJavascriptRequirement":
		r, err = NewInlineJavascriptRequirementFromInterface(obj)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewInlineJavascriptRequirement returns: %s", err.Error())
			return
		}
		return
	case "EnvVarRequirement":
		r, err = NewEnvVarRequirement(obj, context)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewEnvVarRequirement returns: %s", err.Error())
			return
		}
		return
	case "StepInputExpressionRequirement":
		r, err = NewStepInputExpressionRequirement(obj)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewStepInputExpressionRequirement returns: %s", err.Error())
			return
		}
		return
	case "ShockRequirement":
		r, err = NewShockRequirementFromInterface(obj)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewShockRequirementFromInterface returns: %s", err.Error())
			return
		}
		return
	case "InitialWorkDirRequirement":
		r, err = NewInitialWorkDirRequirement(obj, context)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewInitialWorkDirRequirement returns: %s", err.Error())
			return
		}
		return
	case "ScatterFeatureRequirement":
		r, err = NewScatterFeatureRequirement(obj)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewScatterFeatureRequirement returns: %s", err.Error())
			return
		}
		return
	case "MultipleInputFeatureRequirement":
		r, err = NewMultipleInputFeatureRequirement(obj)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewMultipleInputFeatureRequirement returns: %s", err.Error())
			return
		}
		return
	case "SchemaDefRequirement":
		r, err = NewSchemaDefRequirement(obj, context)
		if err != nil {
			err = fmt.Errorf("(NewRequirement) NewSchemaDefRequirement returns: %s", err.Error())
			return
		}
		return

	case "SubworkflowFeatureRequirement":
		thisR := DummyRequirement{}
		thisR.Class = "SubworkflowFeatureRequirement"
		r = &thisR
	default:
		err = errors.New("Requirement class not supported " + class)

	}
	return
}

// GetRequirement _
func GetRequirement(rName string, arrayPtr []Requirement) (requirement *Requirement, err error) {

	if arrayPtr == nil {
		err = fmt.Errorf("(GetRequirement) requirement array is empty, %s not found", rName)
		return
	}

	for i := range arrayPtr {

		if (arrayPtr)[i].GetClass() == rName {
			requirement = &(arrayPtr)[i]
			return
		}
	}
	fmt.Println("Requirement array:")
	spew.Dump(arrayPtr)
	err = fmt.Errorf("(GetRequirement) requirement %s not found", rName)

	return
}

// GetShockRequirement _
func GetShockRequirement(arrayPtr []Requirement) (shockRequirement *ShockRequirement, err error) {
	var requirementPtr *Requirement
	requirementPtr, err = GetRequirement("ShockRequirement", arrayPtr)
	if err != nil {
		return
	}

	requirement := *requirementPtr

	var ok bool
	shockRequirement, ok = requirement.(*ShockRequirement)
	if !ok {
		err = fmt.Errorf("(GetShockRequirement) could not convert ShockRequirement (type: %s)", reflect.TypeOf(requirement))
		return
	}
	//shockRequirement = &shockRequirement_nptr

	return
}

// AddRequirement _
func AddRequirement(newR Requirement, oldArrayPtr []Requirement) (newArrayPtr []Requirement, err error) {

	var newArray []Requirement

	newReqClass := newR.GetClass()
	if oldArrayPtr != nil {
		for i, _ := range oldArrayPtr {
			r := (oldArrayPtr)[i]
			if r.GetClass() == newReqClass {
				newArrayPtr = oldArrayPtr
				return
			}
		}
		newArray = append(oldArrayPtr, newR)
	} else {
		newArray = []Requirement{newR}
	}

	newArrayPtr = newArray

	return
}

// DeleteRequirement _
func DeleteRequirement(requirementClass string, oldArrayPtr []Requirement) (newArrayPtr []Requirement, err error) {

	// if old array is empty anyway, there is nothing to delete
	if oldArrayPtr == nil {
		newArrayPtr = nil
		return
	}

	var newArray []Requirement

	for i, _ := range oldArrayPtr {
		r := (oldArrayPtr)[i]
		if r.GetClass() != requirementClass {
			newArray = append(newArray, r)
		}
	}

	newArrayPtr = newArray

	return
}

// , injectedRequirements []Requirement

// CreateHintsArray _
func CreateHintsArray(original interface{}, injectedRequirements []Requirement, inputs interface{}, context *WorkflowContext) (hintsArray []Requirement, err error) {
	if original != nil {
		hintsArray, err = CreateRequirementArray(original, true, inputs, context)
		if err != nil {
			err = fmt.Errorf("(CreateRequirementArrayAndInject) CreateRequirementArray returned: %s", err.Error())
			return
		}
	}

	// if a hint is also in injectedRequirements, do not keep it ! It is now a real Requirement, with possibly different values
	if injectedRequirements != nil && hintsArray != nil {
		filteredHints := []Requirement{}
		for h := range hintsArray {

			isInjected := false
			for _, ir := range injectedRequirements {

				irClass := ir.GetClass()
				if hintsArray[h].GetClass() == irClass {
					isInjected = true
				}

			}
			if !isInjected {
				filteredHints = append(filteredHints, hintsArray[h])
			}
		}
		//	object["hints"] = new_hints
		hintsArray = filteredHints
	}

	return
}

// CreateRequirementArrayAndInject Tools inherit Requirements, but should not overwrite !
func CreateRequirementArrayAndInject(original interface{}, injectedRequirements []Requirement, inputs interface{}, context *WorkflowContext) (requirementsArray []Requirement, err error) {

	if original != nil {
		requirementsArray, err = CreateRequirementArray(original, false, inputs, context)
		if err != nil {
			err = fmt.Errorf("(CreateRequirementArrayAndInject) CreateRequirementArray returned: %s", err.Error())
			return
		}
	}

	logger.Debug(3, "(CreateRequirementArrayAndInject) requirementsArray: %d      injectedRequirements: %d", len(requirementsArray), len(injectedRequirements))

	//fmt.Println("requirementsArray:")
	//spew.Dump(requirementsArray)

	if injectedRequirements == nil {
		return
	}

	for _, ir := range injectedRequirements {

		irClass := ir.GetClass()

		found := false
		for j := range requirementsArray {
			currentRequirement := requirementsArray[j]
			if currentRequirement.GetClass() == irClass {
				found = true

				// merge SchemaDefRequirement !
				if irClass == "SchemaDefRequirement" {

					existingSchemaDefRequirement := currentRequirement.(*SchemaDefRequirement)
					injectedSchemaDefRequirement := ir.(*SchemaDefRequirement)

					existingSchemaDefRequirementMap := make(map[string]bool)

					for _, existingDefTypeIf := range existingSchemaDefRequirement.Types {

						var existingDefTypeName string

						switch existingDefTypeIf.(type) {
						case *InputRecordSchema:
							existingDefType := existingDefTypeIf.(*InputRecordSchema)
							existingDefTypeName = existingDefType.GetName()
						case *InputEnumSchema:
							existingDefType := existingDefTypeIf.(*InputEnumSchema)
							existingDefTypeName = existingDefType.Name
						case *InputArraySchema:
							// does not have a name , no way to compare here..
							continue
						default:
							err = fmt.Errorf("(CreateRequirementArrayAndInject) type not expected: %s", reflect.TypeOf(existingDefTypeIf))
							return
						}

						existingSchemaDefRequirementMap[existingDefTypeName] = true
					}
					// array<InputRecordSchema | InputEnumSchema | InputArraySchema>

					// abstract InputSchema should have Get/SetName

					for _, injectedDefTypeIf := range injectedSchemaDefRequirement.Types {
						//injectedDefTypeIfNamed := injectedDefTypeIf.(NamedSchema)
						//injectedDefTypeName := injectedDefTypeIfNamed.GetName()
						foundDefType := false
						var injectedDefTypeName string
						switch injectedDefTypeIf.(type) {
						case *InputRecordSchema:
							injectedDefType := injectedDefTypeIf.(*InputRecordSchema)
							injectedDefTypeName = injectedDefType.GetName()
							_, foundDefType = existingSchemaDefRequirementMap[injectedDefTypeName]
						case *InputEnumSchema:
							injectedDefType := injectedDefTypeIf.(*InputEnumSchema)
							injectedDefTypeName = injectedDefType.Name
							_, foundDefType = existingSchemaDefRequirementMap[injectedDefTypeName]
						case *InputArraySchema:
							// does not have a name , no way to compare here..
							continue
						default:
							err = fmt.Errorf("(CreateRequirementArrayAndInject) type not expected: %s", reflect.TypeOf(injectedDefTypeIf))
							return
						}

						if !foundDefType { // injectedDefTypeIf was not found
							// insert injectedDefTypeIf
							existingSchemaDefRequirement.Types = append(existingSchemaDefRequirement.Types, injectedDefTypeIf)
						}

					}

				}

				break
			}

		}
		if !found {
			requirementsArray = append(requirementsArray, ir)
		}

	}

	return
}

// CreateRequirementArray hints are optional, requirements are not
func CreateRequirementArray(original interface{}, optional bool, inputs interface{}, context *WorkflowContext) (newArray []Requirement, err error) {
	// here the keynames are actually class names

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	if original == nil {
		err = fmt.Errorf("(CreateRequirementArray) original == nil")
	}
	newArray = []Requirement{}

	reqMap := make(map[string]bool)

	switch original.(type) {
	case map[string]interface{}:

		for classStr, v := range original.(map[string]interface{}) {

			_, ok := reqMap[classStr]
			if ok {
				err = fmt.Errorf("(CreateRequirementArray) double entry: %s", classStr)
				return
			}
			reqMap[classStr] = true
			//var schemataNew []CWLType_Type
			var requirement Requirement
			requirement, err = NewRequirement(classStr, v, inputs, context)
			if err != nil {
				if optional {
					logger.Debug(1, "(CreateRequirementArray) A NewRequirement returns: %s", err.Error())
					err = nil
					continue
				}
				err = fmt.Errorf("(CreateRequirementArray) A NewRequirement returns: %s", err.Error())
				return
			}
			//for i, _ := range schemataNew {
			//	schemata = append(schemata, schemataNew[i])
			//}

			newArray = append(newArray, requirement)
		}
	case []interface{}:
		originalArray := original.([]interface{})

		for i, _ := range originalArray {
			v := originalArray[i]

			var requirement Requirement
			requirement, err = NewRequirementFromInterface(v, inputs, context)
			if err != nil {
				if optional {
					logger.Debug(1, "(CreateRequirementArray) A NewRequirement returns: %s", err.Error())
					err = nil
					continue
				}
				//fmt.Println("CreateRequirementArray:")
				//spew.Dump(original)
				//fmt.Println("CreateRequirementArray done")
				err = fmt.Errorf("(CreateRequirementArray) B NewRequirement returns: %s (%s)", err, spew.Sdump(v))
				return
			}

			newArray = append(newArray, requirement)

		}

	default:
		err = fmt.Errorf("(CreateRequirementArray) type %s unknown", reflect.TypeOf(original))
	}

	return
}
