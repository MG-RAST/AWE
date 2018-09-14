package cwl

import (
	"encoding/json"

	"fmt"

	"github.com/MG-RAST/AWE/lib/logger"
	//"github.com/mitchellh/mapstructure"

	"gopkg.in/yaml.v2"
	//"io/ioutil"
	//"os"
	//"reflect"
	"strconv"
)

type ParsingContext struct {
	If_objects map[string]interface{}
	Objects    map[string]CWL_object
}

type CWL_object_generic map[string]interface{}

type CWLVersion string

type LinkMergeMethod string // merge_nested or merge_flattened

func Add_to_collection_deprecated(context *WorkflowContext, object_array CWL_object_array) (err error) {

	for i, object := range object_array {
		err = context.Add(strconv.Itoa(i), object) // TODO fix id
		if err != nil {
			err = fmt.Errorf("(Add_to_collection) collection.Add returned: %s", err.Error())
			return
		}
	}

	return
}

func Unmarshal(data_ptr *[]byte, v interface{}) (err error) {

	data := *data_ptr

	if data[0] == '{' {

		err_json := json.Unmarshal(data, v)
		if err_json != nil {
			logger.Debug(1, "CWL json unmarshal error: "+err_json.Error())
			err = err_json
			return
		}
	} else {
		err_yaml := yaml.Unmarshal(data, v)
		if err_yaml != nil {
			logger.Debug(1, "CWL yaml unmarshal error: "+err_yaml.Error())
			err = err_yaml
			return
		}

	}

	return
}
