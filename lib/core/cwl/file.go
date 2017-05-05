package cwl

import (
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/MG-RAST/AWE/lib/shock"
	//"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"net/url"
	"strings"
)

// http://www.commonwl.org/v1.0/Workflow.html#File
type File struct {
	Id             string         `yaml:"id"`
	Location       string         `yaml:"location"` // An IRI that identifies the file resource.
	Path           string         `yaml:"path"`     // dirname + '/' + basename == path This field must be set by the implementation.
	Basename       string         `yaml:"basename"` // dirname + '/' + basename == path // if not defined, take from location
	Dirname        string         `yaml:"dirname"`  // dirname + '/' + basename == path
	Nameroot       string         `yaml:"nameroot"`
	Nameext        string         `yaml:"nameext"`
	Checksum       string         `yaml:"checksum"`
	Size           int32          `yaml:"size"`
	SecondaryFiles []CWL_location `yaml:"secondaryFiles"`
	Format         string         `yaml:"format"`
	Contents       string         `yaml:"contents"`
	// Shock node
	Host   string
	Node   string
	Bearer string
	Token  string
}

func (f *File) GetClass() string    { return "File" }
func (f *File) GetId() string       { return f.Id }
func (f *File) SetId(id string)     { f.Id = id }
func (f *File) String() string      { return f.Path }
func (f *File) GetLocation() string { return f.Location } // for CWL_location

func MakeFile(id string, obj interface{}) (file File, err error) {
	file = File{}
	err = mapstructure.Decode(obj, &file)
	if err != nil {
		err = fmt.Errorf("Could not convert File object %s", id)
		return
	}

	if file.Location == "" {
		err = fmt.Errorf("Error file.Location is empty, id=%s", id)

		return
	}

	// example shock://shock.metagenomics.anl.gov/node/92a76f64-d221-4947-9fd0-7106c3b9163a
	file_url, errx := url.Parse(file.Location)
	if errx != nil {
		err = fmt.Errorf("Error parsing url %s %s", file.Location, errx.Error())

		return
	}

	scheme := strings.ToLower(file_url.Scheme)

	switch scheme {
	case "shock":

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

	case "http":
		//extract filename ?
		err = fmt.Errorf("Location scheme not supported yet, %s", id) // TODO
		return
	case "https":
		//extract filename ?
		err = fmt.Errorf("Location scheme not supported yet, %s", id) // TODO
		return
	case "ftp":
		//extract filename ?
		err = fmt.Errorf("Location scheme not supported yet, %s", id) // TODO
		return
	case "":
		err = fmt.Errorf("Location scheme missing, %s", id)
		return
	default:
		err = fmt.Errorf("Location scheme \"%s\" unknown, %s", scheme, id)
		return
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
