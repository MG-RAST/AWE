package cwl

import (
	"crypto/sha1"
	"encoding/hex"
	"fmt"
	"io"
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

// File http://www.commonwl.org/v1.0/Workflow.html#File
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

// IsCWLMinimal _
func (file *File) IsCWLMinimal() {}

// Is_CWLType _
func (file *File) Is_CWLType() {}

//func (f *File) GetClass() string      { return "File" }

// GetType _
func (file *File) GetType() CWLType_Type { return CWLFile }

//func (f *File) GetID() string       { return f.Id }
//func (f *File) SetID(id string)     { f.Id = id }
func (file *File) String() string { return file.Path }

//func (f *File) Is_Array() bool      { return false }

//func (f *File) Is_CommandInputParameterType() {} // for CommandInputParameterType

// NewFile _
func NewFile() (file *File) {
	file = &File{}
	file.CWLType_Impl = CWLType_Impl{}
	file.Class = string(CWLFile)
	file.Type = CWLFile
	return
}

// NewFileFromInterface _
func NewFileFromInterface(obj interface{}, context *WorkflowContext, external_id string) (file File, err error) {

	file, err = MakeFile(obj, context, external_id)

	return
}

// MakeFile _
func MakeFile(obj interface{}, context *WorkflowContext, external_id string) (file File, err error) {

	// external_id is used to add File to context, not to set the Id or Name field!

	//fmt.Println("MakeFile:")
	//spew.Dump(obj)

	objMap, ok := obj.(map[string]interface{})

	if !ok {
		err = fmt.Errorf("(MakeFile) File is not a map[string]interface{} (File is %s)", reflect.TypeOf(obj))
		return
	}

	secondaryFiles, hasSecondaryFiles := objMap["secondaryFiles"]
	if hasSecondaryFiles {
		objMap["secondaryFiles"], err = GetSecondaryFilesArray(secondaryFiles, context)
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
	err = mapstructure.Decode(objMap, &file)
	if err != nil {
		spew.Dump(obj)
		err = fmt.Errorf("(MakeFile) Could not convert File object: %s", err.Error())
		return
	}
	//file.Class = string(CWLFile)
	//file.Type = CWLFile

	//fmt.Println("MakeFile input:")
	//spew.Dump(obj)
	//fmt.Println("MakeFile output:")
	//fmt.Printf("%+v\n", file)

	componentsUpdated := false

	if file.Contents != "" {
		file.Location = file.Basename

		h := sha1.New()
		io.WriteString(h, file.Contents)
		file.Checksum = "sha1$" + hex.EncodeToString(h.Sum(nil))

		//contentLen := len(file.Contents)
		contentLenInt32 := int32(len(file.Contents))
		file.Size = &contentLenInt32

	}

	if file.Location != "" && file.Contents != "" {

		// example shock://shock.metagenomics.anl.gov/node/92a76f64-d221-4947-9fd0-7106c3b9163a
		fileURL, xerr := url.Parse(file.Location)
		if xerr != nil {
			err = fmt.Errorf("(MakeFile) Error parsing url file.Location=%s : %s", file.Location, xerr.Error())

			return
		}

		//file.Location_url = file_url
		scheme := strings.ToLower(fileURL.Scheme)
		values := fileURL.Query()

		// TODO add "node/uuid" matching to detect shock

		switch scheme {

		case "http", "https":
			_, hasDownload := values["download"]
			//extract filename ?
			if hasDownload && false {

				array := strings.Split(fileURL.Path, "/")
				if len(array) != 3 {
					err = fmt.Errorf("(MakeFile) shock url cannot be parsed")
					return
				}
				if array[1] != "node" {
					err = fmt.Errorf("(MakeFile) node missing in shock url")
					return
				}
				file.Node = array[2]

				file.Host = scheme + "://" + fileURL.Host
				shockClient := shock.ShockClient{Host: file.Host} // TODO Token: datatoken
				node, xerr := shockClient.GetNode(file.Node)

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
					componentsUpdated = true
				}
			}
		case "file":

			file.UpdateComponents(fileURL.Path)
		case "":

		default:
			err = fmt.Errorf("(MakeFile) scheme %s not supported yet", scheme)
			return
		}
	}

	if context != nil && context.Initialzing && external_id != "" {

		err = context.AddObject(external_id, &file, "MakeFile")
		if err != nil {
			err = fmt.Errorf("(MakeFile) (external_id: %s) context.Add returned: %s", external_id, err.Error())
			return
		}
	}

	// used to set Basename, Nameroot and Nameext
	if (!componentsUpdated) && file.Path != "" {
		file.UpdateComponents(file.Path)
	}
	return
}

// UpdateComponents _
func (file *File) UpdateComponents(filename string) {
	file.Basename = path.Base(filename)

	file.Nameext = path.Ext(filename)
	file.Nameroot = strings.TrimSuffix(file.Basename, file.Nameext)
	//fmt.Printf("file.Basename: %s\n", file.Basename)
	//fmt.Printf("file.Nameext: %s\n", file.Nameext)
	//fmt.Printf("file.Nameroot: %s\n", file.Nameroot)

}

// SetPath _
func (file *File) SetPath(pathStr string) {
	//fmt.Printf("(SetPath) %s\n", pathStr)
	oldPath := file.Path
	file.Path = pathStr

	if pathStr == "" {
		file.UpdateComponents(oldPath)
		file.Path = ""
		// on upload to Shock we loose path, but need to keep basename

		//file.Basename = ""

		//file.Nameext = ""
		//file.Nameroot = ""
	} else {
		file.UpdateComponents(pathStr)

	}
	//fmt.Printf("file.Path: %s\n", file.Path)
	return
}

// GetSecondaryFilesArray returns array<File | Directory>
func GetSecondaryFilesArray(original interface{}, context *WorkflowContext) (array []interface{}, err error) {

	array = []interface{}{}

	switch original.(type) {
	case []interface{}:
		originalArray := original.([]interface{})
		for i := range originalArray {
			var obj CWLType
			obj, err = NewCWLType("", "", originalArray[i], context)
			if err != nil {
				err = fmt.Errorf("(GetSecondaryFilesArray) NewCWLType returned: %s", err.Error())
				return
			}

			objClass := obj.GetClass()

			if objClass != "File" && objClass != "Directory" {
				err = fmt.Errorf("(GetSecondaryFilesArray) Object class %s not supported", objClass)
				return
			}
			array = append(array, obj)

		}
		return
	}

	err = fmt.Errorf("(GetSecondaryFilesArray) type %s not supported", reflect.TypeOf(original))

	return
}

// Exists _
func (file *File) Exists(inputfilePath string) (ok bool, err error) {
	if file.Contents != "" {
		ok = true
		return
	}

	//fmt.Printf("file.Path: %s\n", file.Path)
	//fmt.Printf("file.Location: %s\n", file.Location)

	if file.Path != "" {
		filePath := file.Path
		filePath = strings.TrimPrefix(filePath, "file://")

		if !path.IsAbs(filePath) {
			filePath = path.Join(inputfilePath, filePath)
		}

		//var file_info os.FileInfo
		_, err = os.Stat(filePath)
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

		filePath := strings.TrimPrefix(file.Location, "file://")

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

		if !path.IsAbs(filePath) {
			filePath = path.Join(inputfilePath, filePath)
		}

		//var file_info os.FileInfo
		_, err = os.Stat(filePath)
		if err != nil {
			err = nil
			ok = false
			//err = fmt.Errorf("(UploadFile) os.Stat returned: %s (file.Path: %s)", err.Error(), file.Path)
			return
		}
		ok = true

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

//func (f *FileArray) IsArrayType() {}
//func (f *FileArray) Get_Array() *[]File {
//	return &f.Data
//}

//func (f *FileArray) GetID() string   { return f.Id }
//func (f *FileArray) SetID(id string) { f.Id = id }

//func (f *FileArray) String() string      { return f.Path }
//func (f *FileArray) GetLocation() string { return f.Location } // for CWL_location

//func (s *FileArray) Is_CommandInputParameterType() {} // for CommandInputParameterType
