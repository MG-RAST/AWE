package cwl

import (
	"fmt"
	"reflect"

	"github.com/mitchellh/mapstructure"
)

type Directory struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Provides: Id, Class, Type

	Location string    `yaml:"location,omitempty" json:"location,omitempty" bson:"location,omitempty" mapstructure:"location,omitempty"` //
	Path     string    `yaml:"path,omitempty" json:"path,omitempty" bson:"path,omitempty" mapstructure:"path,omitempty"`
	Basename string    `yaml:"basename,omitempty" json:"basename,omitempty" bson:"basename,omitempty" mapstructure:"basename,omitempty"` //the basename property defines the name of the File or Subdirectory when staged to disk
	Listing  []CWLType `yaml:"listing,omitempty" json:"listing,omitempty" bson:"listing,omitempty" mapstructure:"listing,omitempty"`
}

func (d Directory) GetClass() string { return string(CWLDirectory) }

func (d Directory) String() string { return d.Path }

func NewDirectory() (d *Directory) {

	d = &Directory{}
	d.Class = string(CWLDirectory)
	d.Type = CWLDirectory
	return
}

func NewDirectoryFromInterface(obj interface{}, context *WorkflowContext) (d *Directory, err error) {

	obj_map, ok := obj.(map[string]interface{})

	//fmt.Println("(NewDirectoryFromInterface)")
	//spew.Dump(obj)
	//err = fmt.Errorf("who invoked me ?")
	//return

	if !ok {
		err = fmt.Errorf("(NewDirectoryFromInterface) not a map")
		return
	}

	listing, has_listing := obj_map["listing"]
	if has_listing {
		//fmt.Println("(NewDirectoryFromInterface) listing:")
		//spew.Dump(listing)
		var listing_types *[]CWLType
		listing_types, err = NewFDArray(listing, "", context)
		if err != nil {
			err = fmt.Errorf("(MakeFile) NewFDArray returns: %s", err.Error())
			return
		}
		//fmt.Println("(NewDirectoryFromInterface) listing_types:")
		//spew.Dump(listing_types)
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
func NewFDArray(native interface{}, parent_id string, context *WorkflowContext) (cwl_array_ptr *[]CWLType, err error) {

	switch native.(type) {
	case []map[string]interface{}:

		nativeArray, ok := native.([]map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewFDArray) could not parse []map[string]interface{}")
			return
		}

		cwl_array := []CWLType{}

		for _, value := range nativeArray {

			var value_cwl CWLType
			value_cwl, err = NewCWLType("", "", value, context)
			if err != nil {
				err = fmt.Errorf("(NewFDArray) in loop, NewCWLType returned: %s", err.Error())
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
	case []interface{}:

		nativeArray, ok := native.([]interface{})
		if !ok {
			err = fmt.Errorf("(NewFDArray) could not parse []interface{}")
			return
		}

		cwl_array := []CWLType{}

		for _, value := range nativeArray {

			var value_cwl CWLType
			value_cwl, err = NewCWLType("", "", value, context)
			if err != nil {
				err = fmt.Errorf("(NewFDArray) in loop, NewCWLType returned: %s", err.Error())
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
		//fmt.Println("(NewFDArray) array element")
		//spew.Dump(native)
		ct, xerr := NewCWLType("", parent_id, native, context)
		if xerr != nil {
			err = xerr
			return
		}
		//fmt.Println("(NewFDArray) array element, after parsing")
		//spew.Dump(ct)
		switch ct.(type) {
		case *File, *Directory:
			//fmt.Println("(NewFDArray) ok, file or directory")
			// ok
		default:
			//fmt.Println("(NewFDArray) ERROR")
			err = fmt.Errorf("(NewFDArray) (B) expected *File or *Directory, got %s ", reflect.TypeOf(ct))
			return
		}

		cwl_array_ptr = &[]CWLType{ct}
	}

	return

}
