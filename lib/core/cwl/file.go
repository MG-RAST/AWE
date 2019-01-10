package cwl

import (
	"fmt"
	"os"
	"path"
	"reflect"

	shock "github.com/MG-RAST/go-shock-client"
	"github.com/davecgh/go-spew/spew"

	//"github.com/davecgh/go-spew/spew"
	"net/url"
	"strings"

	"github.com/mitchellh/mapstructure"
)

// http://www.commonwl.org/v1.0/Workflow.html#File
type File struct {
	CWLType_Impl `yaml:",inline" json:",inline" bson:",inline" mapstructure:",squash"` // Provides: Id, Class, Type
	//Type           CWLType_Type  `yaml:"-" json:"-" bson:"-" mapstructure:"-"`
	Location string `yaml:"location,omitempty" json:"location,omitempty" bson:"location,omitempty" mapstructure:"location,omitempty"` // An IRI that identifies the file resource.
	//Location_url   *url.URL      `yaml:"-" json:"-" bson:"-" mapstructure:"-"`                                                                     // only for internal purposes
	Path           string        `yaml:"path,omitempty" json:"path,omitempty" bson:"path,omitempty" mapstructure:"path,omitempty"`                 // dirname + '/' + basename == path This field must be set by the implementation.
	Basename       string        `yaml:"basename,omitempty" json:"basename,omitempty" bson:"basename,omitempty" mapstructure:"basename,omitempty"` // dirname + '/' + basename == path // if not defined, take from location
	Dirname        string        `yaml:"dirname,omitempty" json:"dirname,omitempty" bson:"dirname,omitempty" mapstructure:"dirname,omitempty"`     // dirname + '/' + basename == path
	Nameroot       string        `yaml:"nameroot,omitempty" json:"nameroot,omitempty" bson:"nameroot,omitempty" mapstructure:"nameroot,omitempty"`
	Nameext        string        `yaml:"nameext,omitempty" json:"nameext,omitempty" bson:"nameext,omitempty" mapstructure:"nameext,omitempty"`
	Checksum       string        `yaml:"checksum,omitempty" json:"checksum,omitempty" bson:"checksum,omitempty" mapstructure:"checksum,omitempty"`
	Size           *int32        `yaml:"size,omitempty" json:"size,omitempty" bson:"size,omitempty" mapstructure:"size,omitempty"`
	SecondaryFiles []interface{} `yaml:"secondaryFiles,omitempty" json:"secondaryFiles,omitempty" bson:"secondaryFiles,omitempty" mapstructure:"secondaryFiles,omitempty"`
	Format         string        `yaml:"format,omitempty" json:"format,omitempty" bson:"format,omitempty" mapstructure:"format,omitempty"`
	Contents       string        `yaml:"contents,omitempty" json:"contents,omitempty" bson:"contents,omitempty" mapstructure:"contents,omitempty"`
	// Shock node
	Host   string `yaml:"-" json:"-" bson:"-" mapstructure:"-"`
	Node   string `yaml:"-" json:"-" bson:"-" mapstructure:"-"`
	Bearer string `yaml:"-" json:"-" bson:"-" mapstructure:"-"`
	Token  string `yaml:"-" json:"-" bson:"-" mapstructure:"-"`
}

func (f *File) Is_CWL_minimal() {}
func (f *File) Is_CWLType()     {}

//func (f *File) GetClass() string      { return "File" }
func (f *File) GetType() CWLType_Type { return CWL_File }

//func (f *File) GetId() string       { return f.Id }
//func (f *File) SetId(id string)     { f.Id = id }
func (f *File) String() string { return f.Path }

//func (f *File) Is_Array() bool      { return false }

//func (f *File) Is_CommandInputParameterType() {} // for CommandInputParameterType

func NewFile() (f *File) {
	f = &File{}
	f.CWLType_Impl = CWLType_Impl{}
	f.Class = string(CWL_File)
	f.Type = CWL_File
	return
}

func NewFileFromInterface(obj interface{}, context *WorkflowContext) (file File, err error) {

	file, err = MakeFile(obj, context)

	return
}

func MakeFile(obj interface{}, context *WorkflowContext) (file File, err error) {

	//fmt.Println("MakeFile:")
	//spew.Dump(obj)

	obj_map, ok := obj.(map[string]interface{})

	if !ok {
		err = fmt.Errorf("(MakeFile) File is not a map[string]interface{} (File is %s)", reflect.TypeOf(obj))
		return
	}

	secondaryFiles, has_secondaryFiles := obj_map["secondaryFiles"]
	if has_secondaryFiles {
		obj_map["secondaryFiles"], err = GetSecondaryFilesArray(secondaryFiles, context)
		if err != nil {
			err = fmt.Errorf("(MakeFile) GetSecondaryFilesArray returned: %s", err.Error())
			return
		}

	}

	// format, has_format := obj_map["format"]
	// if has_format {
	// 	switch format.(type) {
	// 	case []interface{}:
	// 		// ok
	// 	case interface{}:
	// 		// put element into array
	// 		obj_map["format"] = []interface{}{format}
	// 	default:
	// 		err = fmt.Errorf("(MakeFile) format type not supported: %s", reflect.TypeOf(format))
	// 		return
	// 	}

	// }

	file = *NewFile()
	err = mapstructure.Decode(obj_map, &file)
	if err != nil {
		spew.Dump(obj)
		err = fmt.Errorf("(MakeFile) Could not convert File object: %s", err.Error())
		return
	}
	//file.Class = string(CWL_File)
	//file.Type = CWL_File

	//fmt.Println("MakeFile input:")
	//spew.Dump(obj)
	//fmt.Println("MakeFile output:")
	//fmt.Printf("%+v\n", file)

	components_updated := false

	if file.Location != "" {

		// example shock://shock.metagenomics.anl.gov/node/92a76f64-d221-4947-9fd0-7106c3b9163a
		file_url, xerr := url.Parse(file.Location)
		if xerr != nil {
			err = fmt.Errorf("(MakeFile) Error parsing url %s %s", file.Location, xerr.Error())

			return
		}

		//file.Location_url = file_url
		scheme := strings.ToLower(file_url.Scheme)
		values := file_url.Query()

		// TODO add "node/uuid" matching to detect shock

		switch scheme {

		case "http", "https":
			_, has_download := values["download"]
			//extract filename ?
			if has_download && false {

				array := strings.Split(file_url.Path, "/")
				if len(array) != 3 {
					err = fmt.Errorf("(MakeFile) shock url cannot be parsed")
					return
				}
				if array[1] != "node" {
					err = fmt.Errorf("(MakeFile) node missing in shock url")
					return
				}
				file.Node = array[2]

				file.Host = scheme + "://" + file_url.Host
				shock_client := shock.ShockClient{Host: file.Host} // TODO Token: datatoken
				node, xerr := shock_client.GetNode(file.Node)

				if xerr != nil {
					err = fmt.Errorf("(MakeFile) Could not get shock node (%s, %s): %s", file.Host, file.Node, xerr.Error())
					return
				}
				//fmt.Println("---node:")
				//spew.Dump(node)

				if file.Basename == "" {
					// use filename from shocknode
					if node.File.Name != "" {
						//fmt.Printf("(MakeFile) node.File.Name: %s\n", node.File.Name)
						//file.Basename = node.File.Name
						file.UpdateComponents(node.File.Name)
					} else {
						// if user does not specify a filename and shock does not, use node_id
						//file.Basename =
						file.UpdateComponents(file.Node)
					}
					components_updated = true
				}
			}
		case "file":

			file.UpdateComponents(file_url.Path)
		case "":

		default:
			err = fmt.Errorf("(MakeFile) scheme %s not supported yet", scheme)
			return
		}
	}

	if context != nil && context.Initialzing && err == nil {
		err = context.Add("", &file, "MakeFile")
		if err != nil {
			err = fmt.Errorf("(MakeFile) context.Add returned: %s", err.Error())
			return
		}
	}

	// used to set Basename, Nameroot and Nameext
	if (!components_updated) && file.Path != "" {
		file.UpdateComponents(file.Path)
	}
	return
}

func (file *File) UpdateComponents(filename string) {
	file.Basename = path.Base(filename)

	file.Nameext = path.Ext(filename)
	file.Nameroot = strings.TrimSuffix(file.Basename, file.Nameext)
	//fmt.Printf("file.Basename: %s\n", file.Basename)
	//fmt.Printf("file.Nameext: %s\n", file.Nameext)
	//fmt.Printf("file.Nameroot: %s\n", file.Nameroot)

}

func (file *File) SetPath(path_str string) {
	file.Path = path_str

	if path_str == "" {
		// on upload to Shock we loose path, but need to keep basename

		//file.Basename = ""

		//file.Nameext = ""
		//file.Nameroot = ""
	} else {
		file.UpdateComponents(path_str)

	}
	//fmt.Printf("file.Path: %s\n", file.Path)

}

// returns array<File | Directory>
func GetSecondaryFilesArray(original interface{}, context *WorkflowContext) (array []interface{}, err error) {

	array = []interface{}{}

	switch original.(type) {
	case []interface{}:
		original_array := original.([]interface{})
		for i, _ := range original_array {
			var obj CWLType
			obj, err = NewCWLType("", original_array[i], context)
			if err != nil {
				err = fmt.Errorf("(GetSecondaryFilesArray) NewCWLType returned: %s", err.Error())
				return
			}

			obj_class := obj.GetClass()

			if obj_class != "File" && obj_class != "Directory" {
				err = fmt.Errorf("(GetSecondaryFilesArray) Object class %s not supported", obj_class)
				return
			}
			array = append(array, obj)

		}
		return
	}

	err = fmt.Errorf("(GetSecondaryFilesArray) type %s not supported", reflect.TypeOf(original))

	return
}

func (file *File) Exists(inputfile_path string) (ok bool, err error) {
	if file.Contents != "" {
		ok = true
		return
	}

	if file.Path != "" {
		file_path := file.Path

		if !path.IsAbs(file_path) {
			file_path = path.Join(inputfile_path, file_path)
		}

		//var file_info os.FileInfo
		_, err = os.Stat(file_path)
		if err != nil {
			err = nil
			ok = false
			//err = fmt.Errorf("(UploadFile) os.Stat returned: %s (file.Path: %s)", err.Error(), file.Path)
			return
		}
		ok = true
		return
	}

	if file.Location != "" {

		file_path := strings.TrimPrefix(file.Location, "file://")

		if strings.HasPrefix(file.Location, "http:") {
			err = fmt.Errorf("(File/Exists) schema http not yet supported (%s)", file.Location)
			return
		}
		if strings.HasPrefix(file.Location, "https:") {
			err = fmt.Errorf("(File/Exists) schema https not yet supported")
			return
		}
		if strings.HasPrefix(file.Location, "ftp:") {
			err = fmt.Errorf("(File/Exists) schema https not yet supported")
			return
		}

		if !path.IsAbs(file_path) {
			file_path = path.Join(inputfile_path, file_path)
		}

		//var file_info os.FileInfo
		_, err = os.Stat(file_path)
		if err != nil {
			err = nil
			ok = false
			//err = fmt.Errorf("(UploadFile) os.Stat returned: %s (file.Path: %s)", err.Error(), file.Path)
			return
		}
		ok = true
		return
	}
	return
}

//type FileArray struct {
//	CWLType_Impl
//	Id   string `yaml:"id" json:"id"`
//	Data []File
//}

//func (f *FileArray) GetClass() string { return "array" }

//func (f *FileArray) Is_Array() bool   { return true }

//func (f *FileArray) Is_CWL_array_type() {}
//func (f *FileArray) Get_Array() *[]File {
//	return &f.Data
//}

//func (f *FileArray) GetId() string   { return f.Id }
//func (f *FileArray) SetId(id string) { f.Id = id }

//func (f *FileArray) String() string      { return f.Path }
//func (f *FileArray) GetLocation() string { return f.Location } // for CWL_location

//func (s *FileArray) Is_CommandInputParameterType() {} // for CommandInputParameterType
