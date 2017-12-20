package cwl

import (
	//"errors"
	"fmt"

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
type Job_document []NamedCWLType // JobDocArray

type JobDocMap map[string]CWLType

type NamedCWLType struct {
	CWL_id_Impl         // provides id
	Value       CWLType `yaml:"value,omitempty" bson:"value,omitempty" json:"value,omitempty" mapstructure:"value,omitempty"`
}

func NewNamedCWLType(id string, value CWLType) NamedCWLType {

	x := NamedCWLType{Value: value}

	x.Id = id

	return x
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

func NewNamedCWLTypeFromInterface(native interface{}) (cwl_obj_named NamedCWLType, err error) {

	native, err = MakeStringMap(native)
	if err != nil {
		return
	}

	switch native.(type) {
	case map[string]interface{}:
		named_thing_map := native.(map[string]interface{})

		named_thing_value, ok := named_thing_map["value"]
		if !ok {
			fmt.Println("Expected NamedCWLType field \"value\" , but not found:")
			spew.Dump(native)
			err = fmt.Errorf("(NewNamedCWLTypeFromInterface) Expected NamedCWLType field \"value\" , but not found: %s", spew.Sdump(native))
			return
		}

		var id string
		var id_interface interface{}
		id_interface, ok = named_thing_map["id"]
		if ok {
			id, ok = id_interface.(string)
			if !ok {
				err = fmt.Errorf("(NewNamedCWLTypeFromInterface) id has wrong type")
				return
			}
		} else {
			id = ""
		}

		var cwl_obj CWLType
		cwl_obj, err = NewCWLType(id, named_thing_value)
		if err != nil {
			err = fmt.Errorf("(NewNamedCWLTypeFromInterface) B NewCWLType returns: %s", err.Error())
			return
		}

		//var cwl_obj_named NamedCWLType
		cwl_obj_named = NewNamedCWLType(id, cwl_obj)

	default:
		fmt.Println("Expected a NamedCWLType, but did not find a map:")
		spew.Dump(native)
		err = fmt.Errorf("(NewNamedCWLTypeFromInterface) Expected a NamedCWLType, but did not find a map")
		return
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

func NewJob_document(original interface{}) (job *Job_document, err error) {

	logger.Debug(3, "(NewJob_document) starting")

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	job_nptr := Job_document{}

	job = &job_nptr

	switch original.(type) {

	case map[string]interface{}:

		original_map := original.(map[string]interface{})

		for key_str, value := range original_map {

			cwl_obj, xerr := NewCWLType(key_str, value)
			if xerr != nil {
				err = xerr
				return
			}
			job_nptr = append(job_nptr, NewNamedCWLType(key_str, cwl_obj))

		}
		return

	case []interface{}:
		original_array := original.([]interface{})

		for _, value := range original_array {

			cwl_obj, xerr := NewCWLType("", value)
			if xerr != nil {
				err = xerr
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

func NewJob_documentFromNamedTypes(original interface{}) (job *Job_document, err error) {

	logger.Debug(3, "(NewJob_documentFromNamedTypes) starting")

	if original == nil {
		err = fmt.Errorf("(NewJob_documentFromNamedTypes) original == nil")
		return
	}

	original, err = MakeStringMap(original)
	if err != nil {
		return
	}

	job_nptr := Job_document{}

	job = &job_nptr

	switch original.(type) {

	case []interface{}:
		original_array := original.([]interface{})

		for _, named_thing := range original_array {

			var cwl_obj_named NamedCWLType
			cwl_obj_named, err = NewNamedCWLTypeFromInterface(named_thing)
			if err != nil {
				err = fmt.Errorf("(NewJob_documentFromNamedTypes) A) NewNamedCWLTypeFromInterface returns: %s", err.Error())
				return
			}

			job_nptr = append(job_nptr, cwl_obj_named)

		}

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

	job_input, err = NewJob_document(job_gen)
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
		job_input, err = NewJob_document(job_array)
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
		job_input, err = NewJob_document(job_map)
		if err != nil {
			err = fmt.Errorf("(ParseJob) B NewJob_document returns: %s", err.Error())
		}
	}

	if err != nil {
		return
	}

	return
}
