package cwl

import (
	//"errors"
	"fmt"
	cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
	"io/ioutil"
	//"os"
	//"strings"
	"github.com/MG-RAST/AWE/lib/logger"
	"gopkg.in/yaml.v2"
	"reflect"
	//"github.com/MG-RAST/AWE/lib/logger/event"
)

//type Job_document map[string]interface{}

type Job_document []cwl_types.CWLType

func (job_input *Job_document) GetMap() (job_input_map map[string]cwl_types.CWLType) {
	job_input_map = make(map[string]cwl_types.CWLType)

	for _, value := range *job_input {
		id := value.GetId()
		job_input_map[id] = value
	}
	return
}

func NewJob_document(original interface{}) (job *Job_document, err error) {

	logger.Debug(3, "(NewJob_document) starting")

	job_nptr := Job_document{}

	job = &job_nptr

	switch original.(type) {
	case map[interface{}]interface{}:
		original_map := original.(map[interface{}]interface{})

		for key, value := range original_map {
			key_str, ok := key.(string)
			if !ok {
				err = fmt.Errorf("key is not string")
				return
			}
			cwl_obj, xerr := cwl_types.NewCWLType(key_str, value)
			if xerr != nil {
				err = xerr
				return
			}
			job_nptr = append(job_nptr, cwl_obj)

		}
		return
	case map[string]interface{}:

		original_map := original.(map[string]interface{})

		for key_str, value := range original_map {

			cwl_obj, xerr := cwl_types.NewCWLType(key_str, value)
			if xerr != nil {
				err = xerr
				return
			}
			job_nptr = append(job_nptr, cwl_obj)

		}
		return

	case []interface{}:
		original_array := original.([]interface{})

		for _, value := range original_array {

			cwl_obj, xerr := cwl_types.NewCWLType("", value)
			if xerr != nil {
				err = xerr
				return
			}
			job_nptr = append(job_nptr, cwl_obj)

		}

		return
	default:
		spew.Dump(original)
		err = fmt.Errorf("(NewJob_document) type %s unknown", reflect.TypeOf(original))
	}
	return
}

func ParseJobFile(job_file string) (job_input *Job_document, err error) {

	job_stream, err := ioutil.ReadFile(job_file)
	if err != nil {
		return
	}

	job_gen := map[interface{}]interface{}{}
	err = Unmarshal(&job_stream, &job_gen)
	if err != nil {
		return
	}

	job_input, err = NewJob_document(job_gen)

	if err != nil {
		return
	}

	return
}

func ParseJob(job_byte_ptr *[]byte) (job_input *Job_document, err error) {

	// since Unmarshal (json and yaml) cannot unmarshal into interface{}, we try array and map

	job_byte := *job_byte_ptr
	if job_byte[0] == '-' {
		fmt.Println("yaml list")
		// I guess this is a yaml list
		//job_array := []cwl_types.CWLType{}
		//job_array := []map[string]interface{}{}
		job_array := []interface{}{}
		err = yaml.Unmarshal(job_byte, &job_array)

		if err != nil {
			err = fmt.Errorf("Could, not parse job input as yaml list: %s", err.Error())
			return
		}
		job_input, err = NewJob_document(job_array)
	} else {
		fmt.Println("yaml map")

		job_map := make(map[string]cwl_types.CWLType)

		err = yaml.Unmarshal(job_byte, &job_map)
		if err != nil {
			err = fmt.Errorf("Could, not parse job input as yaml map: %s", err.Error())
			return
		}
		job_input, err = NewJob_document(job_map)
	}

	if err != nil {
		return
	}

	return
}
