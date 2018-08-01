package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type EnvironmentDef struct {
	EnvName  string     `yaml:"envName,omitempty" bson:"envName,omitempty" json:"envName,omitempty" mapstructure:"envName,omitempty"`
	EnvValue Expression `yaml:"envValue,omitempty" bson:"envValue,omitempty" json:"envValue,omitempty" mapstructure:"envValue,omitempty"`
}

func NewEnvironmentDefFromInterface(original interface{}) (enfDev EnvironmentDef, err error) {

	err = mapstructure.Decode(original, &enfDev)
	if err != nil {
		err = fmt.Errorf("(NewEnvironmentDefFromInterface) mapstructure.Decode returned: %s", err.Error())
		return
	}
	return
}

func GetEnfDefArray(original interface{}) (array []EnvironmentDef, err error) {

	array = []EnvironmentDef{}

	switch original.(type) {
	case []interface{}:
		original_array := original.([]interface{})
		for i, _ := range original_array {
			var enfDev EnvironmentDef
			enfDev, err = NewEnvironmentDefFromInterface(original_array[i])
			if err != nil {
				err = fmt.Errorf("(GetEnfDefArray) NewEnvironmentDefFromInterface returned: %s", err.Error())
				return
			}

			array = append(array, enfDev)

		}
		return
	}

	err = fmt.Errorf("(GetEnfDefArray) type %s not supported", reflect.TypeOf(original))

	return
}
