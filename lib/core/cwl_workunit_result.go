package core

import (
	"github.com/MG-RAST/AWE/lib/core/cwl"
	"github.com/MG-RAST/AWE/lib/logger"

	//cwl_types "github.com/MG-RAST/AWE/lib/core/cwl/types"
	//"github.com/davecgh/go-spew/spew"
	"fmt"
)

// TODO core.Notice and this should be the same !!!!

// Notice _
type Notice struct {
	ID          Workunit_Unique_Identifier `bson:"id" json:"id" mapstructure:"id"` // redundant field, for reporting
	WorkerID    string                     `bson:"worker_id" json:"worker_id" mapstructure:"worker_id"`
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

// NewNotice _
func NewNotice(native interface{}, context *cwl.WorkflowContext) (workunitResult *Notice, err error) {
	workunitResult = &Notice{}
	switch native.(type) {

	case map[string]interface{}:
		nativeMap, ok := native.(map[string]interface{})
		if !ok {
			err = fmt.Errorf("(NewNotice) type error")
			return
		}

		results, hasResults := nativeMap["results"]
		if hasResults {
			if results != nil {

				var resultsJobdoc *cwl.Job_document
				resultsJobdoc, err = cwl.NewJob_documentFromNamedTypes(results, context)
				if err != nil {
					err = fmt.Errorf("(NewNotice) NewJob_documentFromNamedTypes returns %s", err.Error())
					return
				}

				workunitResult.Results = resultsJobdoc
			} else {
				logger.Debug(2, "(NewNotice) results == nil")
			}
		} else {
			logger.Debug(2, "(NewNotice) no results")
		}

		id, ok := nativeMap["id"]
		if !ok {
			err = fmt.Errorf("(NewNotice) id is missing")
			return
		}

		workunitResult.ID, err = New_Workunit_Unique_Identifier_from_interface(id)
		if err != nil {
			err = fmt.Errorf("(NewNotice) New_Workunit_Unique_Identifier_from_interface returned: %s", err.Error())
			return
		}

		workunitResult.WorkerID, _ = nativeMap["worker_id"].(string)
		status, hasStatus := nativeMap["status"]
		if !hasStatus {
			err = fmt.Errorf("(NewNotice) status is missing")
			return
		}
		workunitResult.Status, _ = status.(string)
		workunitResult.ComputeTime, _ = nativeMap["computetime"].(int)

	default:
		err = fmt.Errorf("(NewNotice) wrong type, map expected")

	}

	return
}
