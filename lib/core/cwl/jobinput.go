package cwl

import (
	//"errors"
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/davecgh/go-spew/spew"
	"github.com/mitchellh/mapstructure"
	"io/ioutil"
	//"os"
	//"strings"
	"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/MG-RAST/AWE/lib/logger/event"
)

//type Job_document map[string]interface{}

type Job_document map[string]cwl_types.CWLType

func NewJob_document(original interface{}) (job *Job_document, err error) {

	logger.Debug(3, "(NewJob_document) starting")

	job_nptr := Job_document{}

	job = &job_nptr

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
			cwl_obj, xerr := cwl_types.NewCWLType(key, value)
			if xerr != nil {
				err = xerr
				return
			}
			original_map[key] = cwl_obj
		}

		err = mapstructure.Decode(original, job)
		if err != nil {
			err = fmt.Errorf("(NewJob_document) %s", err.Error())
			return
		}
	case map[string]interface{}:

		original_map := original.(map[string]interface{})

		keys := []string{}
		for key_str, _ := range original_map {

			keys = append(keys, key_str)

		}

		for _, key := range keys {
			value := original_map[key]
			cwl_obj, xerr := cwl_types.NewCWLType(key, value)
			if xerr != nil {
				err = xerr
				return
			}
			job_nptr[key] = cwl_obj
		}

		//err = mapstructure.Decode(original, job)
		//if err != nil {
		//	err = fmt.Errorf("(NewJob_document) %s", err.Error())
		//	return
		//}
		return

	case []interface{}:
		err = fmt.Errorf("(NewJob_document) type array not supported yet")
		return
	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewJob_document) type unknown")
	}
	return
}

func ParseJobFile(job_file string) (job_input *Job_document, err error) {

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

	return
}

func ParseJob(job_str string) (job_input *Job_document, err error) {

	job_gen := map[interface{}]interface{}{}
	err = Unmarshal([]byte(job_str), &job_gen)
	if err != nil {
		return
	}

	job_input, err = NewJob_document(job_gen)

	if err != nil {
		return
	}

	return
}
