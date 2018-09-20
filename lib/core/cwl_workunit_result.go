package core

import (
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"
	//cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	//"github.com/davecgh/go-spew/spew"
	"fmt"
)

// TODO core.Notice and this should be the same !!!!

type Notice struct {
	Id          Workunit_Unique_Identifier `bson:"id" json:"id" mapstructure:"id"` // redundant field, for reporting
	WorkerId    string                     `bson:"worker_id" json:"worker_id" mapstructure:"worker_id"`
	Results     *cwl.Job_document          `bson:"results" json:"results" mapstructure:"results"`                            // subset of tool_results with Shock URLs
	Status      string                     `bson:"status,omitempty" json:"status,omitempty" mapstructure:"status,omitempty"` // this is redundant as workunit already has state, but this is only used for transfer
	ComputeTime int                        `bson:"computetime,omitempty" json:"computetime,omitempty" mapstructure:"computetime,omitempty"`
	Notes       string
	Stderr      string
}

//type Notice struct {
//	WorkId      Workunit_Unique_Identifier
//	Status      string
//	WorkerId    string
//	ComputeTime int
//	Notes       string
//	Stderr      string
//}

func NewNotice(native interface{}, context *cwl.WorkflowContext) (workunit_result *Notice, err error) {
	workunit_result = &Notice{}
	switch native.(type) {

	case map[string]interface{}:
		native_map, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewNotice) type error")
			return
		}

		results, has_results := native_map["results"]
		if has_results {
			if results != nil {

				var results_jobdoc *cwl.Job_document
				results_jobdoc, err = cwl.NewJob_documentFromNamedTypes(results, context)
				if err != nil {
					err = fmt.Errorf("(NewNotice) NewJob_documentFromNamedTypes returns %s", err.Error())
					return
				}

				workunit_result.Results = results_jobdoc
			} else {
				logger.Debug(2, "(NewNotice) results == nil")
			}
		} else {
			logger.Debug(2, "(NewNotice) no results")
		}

		id, ok := native_map["id"]
		if !ok {
			err = fmt.Errorf("(NewNotice) id is missing")
			return
		}

		workunit_result.Id, err = New_Workunit_Unique_Identifier_from_interface(id)
		if err != nil {
			err = fmt.Errorf("(NewNotice) New_Workunit_Unique_Identifier_from_interface returned: %s", err.Error())
			return
		}

		workunit_result.WorkerId, _ = native_map["worker_id"].(string)
		status, has_status := native_map["status"]
		if !has_status {
			err = fmt.Errorf("(NewNotice) status is missing")
			return
		}
		workunit_result.Status, _ = status.(string)
		workunit_result.ComputeTime, _ = native_map["computetime"].(int)

		return

	default:
		err = fmt.Errorf("(NewNotice) wrong type, map expected")
		return

	}

	return
}
