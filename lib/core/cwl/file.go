package cwl

import (
	"fmt"
	"github.com/mitchellh/mapstructure"
)

type File struct {
	Id             string `yaml:"id"`
	Path           string `yaml:"path"`
	Checksum       string `yaml:"checksum"`
	Size           int32  `yaml:"size"`
	SecondaryFiles []File `yaml:"secondaryFiles"`
	Format         string `yaml:"format"`
}

func (f File) GetClass() string { return "File" }
func (f File) GetId() string    { return f.Id }
func (f File) String() string   { return f.Path }

func MakeFile(id string, obj interface{}) (file File, err error) {
	file = File{}
	err = mapstructure.Decode(obj, &file)
	if err != nil {
		err = fmt.Errorf("Could not convert File object %s", id)
		return
	}
	if file.Id == "" {
		file.Id = id
	}
	return
}
