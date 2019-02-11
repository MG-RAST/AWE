package cwl

import (
	//"errors"
	"fmt"

	"github.com/davecgh/go-spew/spew"
	//"github.com/mitchellh/mapstructure"
	"io/ioutil"
	//"os"
	//"strings"
	"reflect"

	"github.com/MG-RAST/AWE/lib/logger"
	"gopkg.in/yaml.v2"
	//"github.com/MG-RAST/AWE/lib/logger/event"
)

//type Job_document map[string]interface{}
type Job_document []NamedCWLType // JobDocArray

type JobDocMap map[string]CWLType

func (j *Job_document) Add(id string, value CWLType) (new_doc *Job_document) {
	x := NewNamedCWLType(id, value)

	j_npt := *j

	array := []NamedCWLType(j_npt)

	new_array_nptr := Job_document(append(array, x))

	new_doc = &new_array_nptr
	return
}

func (j *Job_document) Get(id string) (value CWLType, ok bool) {

	array := []NamedCWLType(*j)

	for i, _ := range array {
		named_type := array[i]
		if named_type.Id == id {

			value = named_type.Value
			ok = true
			return
		}

	}

	ok = false
	//err = fmt.Errorf("(Job_document/Get) Element %s not found.", id)

	return
}

func (jd_map JobDocMap) GetArray() (result Job_document, err error) {
	result = Job_document{}

	for key, value := range jd_map {
		if value == nil {
			err = fmt.Errorf("(GetArray) value of key %s is nil", key)
			return
		}
		named := NewNamedCWLType(key, value)
		result = append(result, named)
	}

	return
}

func (job_input *Job_document) GetMap() (job_input_map JobDocMap) {
	job_input_map = make(JobDocMap)

	for _, value := range *job_input {
		//id := value.GetId()
		job_input_map[value.Id] = value.Value
	}
	return
}

func NewJob_document(original interface{}, context *WorkflowContext) (job *Job_document, err error) {

	logger.Debug(3, "(NewJob_document) starting")

	original, err = MakeStringMap(original, context)
	if err != nil {
		return
	}

	job_nptr := Job_document{}

	job = &job_nptr

	switch original.(type) {

	case map[string]interface{}:

		original_map := original.(map[string]interface{})

		for key_str, value := range original_map {

			cwl_obj, xerr := NewCWLType(key_str, value, context)
			if xerr != nil {
				err = fmt.Errorf("(NewJob_document) NewCWLType returned: %s", xerr.Error())
				return
			}
			job_nptr = append(job_nptr, NewNamedCWLType(key_str, cwl_obj))

		}
		return

	case []interface{}:
		original_array := original.([]interface{})

		for _, value := range original_array {

			cwl_obj, xerr := NewCWLType("", value, context)
			if xerr != nil {
				err = fmt.Errorf("(NewJob_document) NewCWLType returned: %s", xerr.Error())
				return
			}
			job_nptr = append(job_nptr, NewNamedCWLType("not supported", cwl_obj))

		}

		return
	default:

		spew.Dump(original)
		err = fmt.Errorf("(NewJob_document) (A) type %s unknown", reflect.TypeOf(original))
	}
	return
}

func NewJob_documentFromNamedTypes(original interface{}, context *WorkflowContext) (job *Job_document, err error) {

	logger.Debug(3, "(NewJob_documentFromNamedTypes) starting")

	if original == nil {
		err = fmt.Errorf("(NewJob_documentFromNamedTypes) original == nil")
		return
	}

	original, err = MakeStringMap(original, context)
	if err != nil {
		err = fmt.Errorf("(NewJob_documentFromNamedTypes) MakeStringMap returned: %s", err.Error())
		return
	}

	job_nptr := Job_document{}

	switch original.(type) {

	case []interface{}:
		original_array := original.([]interface{})

		for _, named_thing := range original_array {

			var cwl_obj_named NamedCWLType
			cwl_obj_named, err = NewNamedCWLTypeFromInterface(named_thing, context)
			if err != nil {
				err = fmt.Errorf("(NewJob_documentFromNamedTypes) A) NewNamedCWLTypeFromInterface returns: %s", err.Error())
				return
			}

			job_nptr = append(job_nptr, cwl_obj_named)

		}
		job = &job_nptr
		return
	default:

		err = fmt.Errorf("(NewJob_document) (B) type %s unknown: %s", reflect.TypeOf(original), spew.Sdump(original))
	}
	return
}

func ParseJobFile(job_file string) (job_input *Job_document, err error) {

	job_stream, err := ioutil.ReadFile(job_file)
	if err != nil {
		return
	}

	job_gen := map[string]interface{}{}
	err = Unmarshal(&job_stream, &job_gen)
	if err != nil {
		err = fmt.Errorf("(ParseJobFile) Unmarshal returned: %s (json: %s)", err.Error(), string(job_stream[:]))
		return
	}

	job_input, err = NewJob_document(job_gen, nil)
	if err != nil {
		err = fmt.Errorf("(ParseJobFile) NewJob_document returned: %s", err.Error())
		return
	}

	return
}

func ParseJob(job_byte_ptr *[]byte) (job_input *Job_document, err error) {

	// since Unmarshal (json and yaml) cannot unmarshal into interface{}, we try array and map

	job_byte := *job_byte_ptr
	if job_byte[0] == '-' {
		//fmt.Println("yaml list")
		// I guess this is a yaml list
		//job_array := []CWLType{}
		//job_array := []map[string]interface{}{}
		job_array := []interface{}{}
		err = yaml.Unmarshal(job_byte, &job_array)

		if err != nil {
			err = fmt.Errorf("(ParseJob) Could, not parse job input as yaml list: %s", err.Error())
			return
		}
		job_input, err = NewJob_document(job_array, nil)
		if err != nil {
			err = fmt.Errorf("(ParseJob) A NewJob_document returns: %s", err.Error())
		}

	} else {
		//fmt.Println("yaml map")

		job_map := make(map[string]interface{})

		err = yaml.Unmarshal(job_byte, &job_map)
		if err != nil {
			err = fmt.Errorf("Could, not parse job input as yaml map: %s", err.Error())
			return
		}
		job_input, err = NewJob_document(job_map, nil)
		if err != nil {
			err = fmt.Errorf("(ParseJob) B NewJob_document returns: %s", err.Error())
		}
	}

	if err != nil {
		return
	}

	return
}
