package cwl

import (
	//"errors"
	"fmt"
	//"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"io/ioutil"
	//"os"
	//"strings"
)

//type Job_document map[string]interface{}

type Job_document map[string]CWLType

func NewJob_document(original interface{}) (job *Job_document, err error) {

	job = &Job_document{}

	switch original.(type) {
	case map[interface{}]interface{}:
		original_map := original.(map[interface{}]interface{})

		keys := []string{}
		for key, _ := range original_map {
			key_str, ok := key.(string)
			if !ok {
				err = fmt.Errorf("key is not string")
				return
			}
			keys = append(keys, key_str)

		}

		for _, key := range keys {
			value := original_map[key]
			cwl_obj, xerr := NewCWLType(value)
			if xerr != nil {
				err = xerr
				return
			}
			original_map[key] = cwl_obj
		}

		err = mapstructure.Decode(original, job)
		if err != nil {
			err = fmt.Errorf("(NewCommandOutputBinding) %s", err.Error())
			return
		}
	case []interface{}:
		err = fmt.Errorf("(NewJob_document) type array not supported yet")
		return
	default:
		err = fmt.Errorf("(NewJob_document) type unknown")
	}
	return
}

func ParseJob(job_file string) (job_input *Job_document, err error) {

	job_stream, err := ioutil.ReadFile(job_file)
	if err != nil {
		return
	}

	job_gen := map[interface{}]interface{}{}
	err = Unmarshal(job_stream, &job_gen)
	if err != nil {
		return
	}

	job_input, err = NewJob_document(job_gen)

	if err != nil {
		return
	}

	//fmt.Printf("-------MyCollection")
	//spew.Dump(collection.All)

	//fmt.Printf("-------")
	//os.Exit(0)

	return
}
