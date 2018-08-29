package cwl

import (
	"fmt"
	"reflect"

	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
)

type Directory struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Provides: Id, Class, Type

	Location string    `yaml:"location,omitempty" json:"location,omitempty" bson:"location,omitempty" mapstructure:"location,omitempty"` //
	Path     string    `yaml:"path,omitempty" json:"path,omitempty" bson:"path,omitempty" mapstructure:"path,omitempty"`
	Basename string    `yaml:"basename,omitempty" json:"basename,omitempty" bson:"basename,omitempty" mapstructure:"basename,omitempty"` //the basename property defines the name of the File or Subdirectory when staged to disk
	Listing  []CWLType `yaml:"listing,omitempty" json:"listing,omitempty" bson:"listing,omitempty" mapstructure:"listing,omitempty"`
}

func (d Directory) GetClass() string { return string(CWL_Directory) }

func (d Directory) String() string { return d.Path }

func NewDirectory() (d *Directory) {

	d = &Directory{}
	d.Class = string(CWL_Directory)
	d.Type = CWL_Directory
	return
}

func NewDirectoryFromInterface(obj interface{}) (d *Directory, err error) {

	obj_map, ok := obj.(map[string]interface{})

	fmt.Println("(NewDirectoryFromInterface)")
	spew.Dump(obj)
	//err = fmt.Errorf("who invoked me ?")
	//return

	if !ok {
		err = fmt.Errorf("(NewDirectoryFromInterface) not a map")
		return
	}

	listing, has_listing := obj_map["listing"]
	if has_listing {
		var listing_types *[]CWLType
		listing_types, err = NewFDArray(listing, "")
		if err != nil {
			err = fmt.Errorf("(MakeFile) NewFDArray returns: %s", err.Error())
			return
		}

		obj_map["listing"] = listing_types

	}

	d = NewDirectory()

	err = mapstructure.Decode(obj, d)
	if err != nil {
		err = fmt.Errorf("(MakeFile) Could not convert Directory object %s", err.Error())
		return
	}

	return
}

// Array of Files and/or Directories
func NewFDArray(native interface{}, parent_id string) (cwl_array_ptr *[]CWLType, err error) {

	switch native.(type) {
	case []interface{}:

		native_array, ok := native.([]interface{})
		if !ok {
			err = fmt.Errorf("(NewFDArray) could not parse []interface{}")
			return
		}

		cwl_array := []CWLType{}

		for _, value := range native_array {
			value_cwl, xerr := NewCWLType("", value)
			if xerr != nil {
				err = xerr
				return
			}

			switch value_cwl.(type) {
			case *File, *Directory:
				// ok
			default:
				err = fmt.Errorf("(NewFDArray) (A) expected *File or *Directory, got %s ", reflect.TypeOf(value_cwl))
				return
			}

			cwl_array = append(cwl_array, value_cwl)
		}
		cwl_array_ptr = &cwl_array
	default:

		ct, xerr := NewCWLType("", native)
		if xerr != nil {
			err = xerr
			return
		}

		switch ct.(type) {
		case *File, *Directory:
			// ok
		default:
			err = fmt.Errorf("(NewFDArray) (B) expected *File or *Directory, got %s ", reflect.TypeOf(ct))
			return
		}

		cwl_array_ptr = &[]CWLType{ct}
	}

	return

}
