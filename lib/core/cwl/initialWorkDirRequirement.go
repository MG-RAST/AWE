package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/CommandLineTool.html#InitialWorkDirRequirement
type InitialWorkDirRequirement struct {
	BaseRequirement `bson:",inline" yaml:",inline" json:",inline" mapstructure:",squash"`
	Listing         interface{} `yaml:"listing,omitempty" bson:"listing,omitempty" json:"listing,omitempty" mapstructure:"listing,omitempty"` // TODO: array<File | Directory | Dirent | string | Expression> | string | Expression
}

func (c InitialWorkDirRequirement) GetId() string { return "" }

func NewInitialWorkDirRequirement(original interface{}) (r *InitialWorkDirRequirement, err error) {
	var requirement InitialWorkDirRequirement
	r = &requirement

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	switch original.(type) {

	case map[string]interface{}:

		original_map, ok := original.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewInitialWorkDirRequirement) type error")
			return
		}

		//listing, has_listing := original_map["listing"]
		//if has_listing {
		//	original_map["listing"], err = NewCWLType("", listing)
		//	if err != nil {
		//		err = fmt.Errorf("(NewInitialWorkDirRequirement) NewCWLType returned: %s", err.Error())
		//	}
		//}

		err = mapstructure.Decode(original, &requirement)

		requirement.Class = "InitialWorkDirRequirement"

	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewInitialWorkDirRequirement) unknown type %s", reflect.TypeOf(original))
	}

	return
}
