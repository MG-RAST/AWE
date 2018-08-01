package cwl

import (
	"fmt"

	"github.com/mitchellh/mapstructure"
)

type Directory struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Provides: Id, Class, Type

	Location string    `yaml:"location,omitempty" json:"location,omitempty" bson:"location,omitempty" mapstructure:"location,omitempty"` //
	Path     string    `yaml:"path,omitempty" json:"path,omitempty" bson:"path,omitempty" mapstructure:"path,omitempty"`
	Basename string    `yaml:"basename,omitempty" json:"basename,omitempty" bson:"basename,omitempty" mapstructure:"basename,omitempty"`
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

	//spew.Dump(obj)
	//err = fmt.Errorf("who invoked me ?")
	//return

	if !ok {
		err = fmt.Errorf("(NewDirectoryFromInterface) not a map")
		return
	}

	listing, has_listing := obj_map["listing"]
	if has_listing {
		var listing_types *[]CWLType
		listing_types, err = NewCWLTypeArray(listing, "")
		if err != nil {
			err = fmt.Errorf("(MakeFile) NewCWLTypeArray returns: %s", err.Error())
			return
		}
		// TODO check that elements are only File and Directory (make this a feature of NewCWLTypeArray??)
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
