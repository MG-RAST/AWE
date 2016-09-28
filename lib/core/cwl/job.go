package cwl

import (
	//"errors"
	"fmt"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"io/ioutil"
	//"os"
	"strings"
)

type Job_document map[string]interface{}

func ParseJob(collection *CWL_collection, job_file string) (err error) {

	job_stream, err := ioutil.ReadFile(job_file)
	if err != nil {
		return
	}

	job := Job_document{}
	Unmarshal(job_stream, &job)

	for key, value := range job {

		switch value.(type) {
		case string:
			err = collection.Add(String{Id: key, Value: value.(string)})
			if err != nil {
				return
			}
		case int:
			err = collection.Add(&Int{Id: key, Value: value.(int)})
			if err != nil {
				return
			}
		case map[interface{}]interface{}:

			obj_empty := &Empty{}

			err = mapstructure.Decode(value, &obj_empty)
			if err != nil {
				err = fmt.Errorf("Could not convert into CWL object object")
				return
			}
			spew.Dump(obj_empty)

			class := strings.ToLower(obj_empty.Class)

			if class == "" {
				err = fmt.Errorf("CWL object \"%s\" has unknown class", key)
				return
			}

			switch class {
			case "file":
				f, xerr := MakeFile(key, value)
				if xerr != nil {
					err = xerr
					return
				}
				err = collection.Add(f)
				if err != nil {
					return
				}
			default:
				err = fmt.Errorf("Object class %s in %s unknown ", class, key)
				return
			}

		default:
			err = fmt.Errorf("Object \"%s\" unknown.", key)
			return
		}

	}

	fmt.Printf("-------MyCollection")
	spew.Dump(collection.All)

	fmt.Printf("-------")
	//os.Exit(0)

	return
}
