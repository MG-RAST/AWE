package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/MG-RAST/AWE/lib/shock"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"net/url"
	"strings"
)

// http://www.commonwl.org/v1.0/Workflow.html#File
type File struct {
	CWLType_Impl
	Id             string         `yaml:"id" json:"id"`
	Location       string         `yaml:"location" json:"location"` // An IRI that identifies the file resource.
	Path           string         `yaml:"path" json:"path"`         // dirname + '/' + basename == path This field must be set by the implementation.
	Basename       string         `yaml:"basename" json:"basename"` // dirname + '/' + basename == path // if not defined, take from location
	Dirname        string         `yaml:"dirname" json:"dirname"`   // dirname + '/' + basename == path
	Nameroot       string         `yaml:"nameroot" json:"nameroot"`
	Nameext        string         `yaml:"nameext" json:"nameext"`
	Checksum       string         `yaml:"checksum" json:"checksum"`
	Size           int32          `yaml:"size" json:"size"`
	SecondaryFiles []CWL_location `yaml:"secondaryFiles" json:"secondaryFiles"`
	Format         string         `yaml:"format" json:"format"`
	Contents       string         `yaml:"contents" json:"contents"`
	// Shock node
	Host   string
	Node   string
	Bearer string
	Token  string
}

func (f *File) GetClass() string    { return CWL_File }
func (f *File) GetId() string       { return f.Id }
func (f *File) SetId(id string)     { f.Id = id }
func (f *File) String() string      { return f.Path }
func (f *File) GetLocation() string { return f.Location } // for CWL_location
//func (f *File) Is_Array() bool      { return false }

func (f *File) Is_CommandInputParameterType() {} // for CommandInputParameterType

func NewFile(obj interface{}) (file File, err error) {

	file, err = MakeFile("unknown", obj)
	return
}

func MakeFile(id string, obj interface{}) (file File, err error) {
	file = File{}
	err = mapstructure.Decode(obj, &file)
	if err != nil {
		err = fmt.Errorf("(MakeFile) Could not convert File object %s", id)
		return
	}

	fmt.Println("MakeFile input:")
	spew.Dump(obj)
	fmt.Println("MakeFile output:")
	fmt.Printf("%+v\n", file)

	if file.Location != "" {

		// example shock://shock.metagenomics.anl.gov/node/92a76f64-d221-4947-9fd0-7106c3b9163a
		file_url, errx := url.Parse(file.Location)
		if errx != nil {
			err = fmt.Errorf("Error parsing url %s %s", file.Location, errx.Error())

			return
		}
		scheme := strings.ToLower(file_url.Scheme)
		values := file_url.Query()

		// TODO add "node/uuid" matching to detect shock

		switch scheme {

		case "http":
			_, has_download := values["download"]
			//extract filename ?
			if has_download {

				array := strings.Split(file_url.Path, "/")
				if len(array) != 3 {
					err = fmt.Errorf("shock url cannot be parsed")
					return
				}
				if array[1] != "node" {
					err = fmt.Errorf("node missing in shock url")
					return
				}
				file.Node = array[2]

				file.Host = "http://" + file_url.Host
				shock_client := shock.ShockClient{Host: file.Host} // TODO Token: datatoken
				node, xerr := shock_client.Get_node(file.Node)

				if xerr != nil {
					err = fmt.Errorf("(Get_node) Could not get shock node (%s, %s): %s", file.Host, file.Node, xerr.Error())
					return
				}
				//fmt.Println("---node:")
				//spew.Dump(node)

				if file.Basename == "" {
					// use filename from shocknode
					if node.File.Name != "" {
						fmt.Printf("node.File.Name: %s\n", node.File.Name)
						file.Basename = node.File.Name
					} else {
						// if user does not specify a filename and shock does not, use node_id
						file.Basename = file.Node
					}

				}
			}
		}
	}
	if file.Id == "" {
		file.Id = id
	}
	return
}

type Directory struct {
	Id       string         `yaml:"id"`
	Location string         `yaml:"location"`
	Path     string         `yaml:"path"`
	Basename string         `yaml:"basename"`
	Listing  []CWL_location `yaml:"basename"`
}

func (d Directory) GetClass() string    { return "Directory" }
func (d Directory) GetId() string       { return d.Id }
func (d Directory) String() string      { return d.Path }
func (d Directory) GetLocation() string { return d.Location } // for CWL_location

type FileArray struct {
	CWLType_Impl
	Id   string `yaml:"id" json:"id"`
	Data []File
}

func (f *FileArray) GetClass() string { return CWL_File_array }

//func (f *FileArray) Is_Array() bool   { return true }

func (f *FileArray) Is_CWL_array_type() {}
func (f *FileArray) Get_Array() *[]File {
	return &f.Data
}

func (f *FileArray) GetId() string   { return f.Id }
func (f *FileArray) SetId(id string) { f.Id = id }

//func (f *FileArray) String() string      { return f.Path }
//func (f *FileArray) GetLocation() string { return f.Location } // for CWL_location

func (s *FileArray) Is_CommandInputParameterType() {} // for CommandInputParameterType
