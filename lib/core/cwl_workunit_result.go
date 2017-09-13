package core

import (
	"github.com/MG-RAST/AWE/lib/core/cwl"
	//cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	//"github.com/davecgh/go-spew/spew"
	"fmt"
)

type CWL_workunit_result struct {
	Id          string            `bson:"id" json:"id" mapstructure:"id"` // redundant field, for reporting
	WorkerId    string            `bson:"worker_id" json:"worker_id" mapstructure:"worker_id"`
	Results     *cwl.Job_document `bson:"results" json:"results" mapstructure:"results"`                         // subset of tool_results with Shock URLs
	State       string            `bson:"state,omitempty" json:"state,omitempty" mapstructure:"state,omitempty"` // this is redundant as workunit already has state, but this is only used for transfer
	ComputeTime int               `bson:"computetime,omitempty" json:"computetime,omitempty" mapstructure:"computetime,omitempty"`
}

func NewCWL_workunit_result(native interface{}) (workunit_result *CWL_workunit_result, err error) {
	workunit_result = &CWL_workunit_result{}
	switch native.(type) {

	case map[string]interface{}:
		native_map, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewCWL_workunit_result) type error")
			return
		}

		results, has_results := native_map["results"]
		if has_results {

			var results_jobdoc *cwl.Job_document
			results_jobdoc, err = cwl.NewJob_document(results)
			if err != nil {
				return
			}

			workunit_result.Results = results_jobdoc
		}

		workunit_result.Id, _ = native_map["id"].(string)
		workunit_result.WorkerId, _ = native_map["worker_id"].(string)
		workunit_result.State, _ = native_map["state"].(string)
		workunit_result.ComputeTime, _ = native_map["computetime"].(int)

		return

	default:
		err = fmt.Errorf("(NewCWL_workunit_result) wrong type, map expected")
		return

	}

	return
}
