package controller

import (
	"net/http"

	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/golib/goweb"
)

type WorkflowInstancesController struct{}

//  returns all workflow_instances or only those that match job_id, .i.e. /workflow_instances?job_id=<id>

// pagination should be efficient if list is sorted by an indexed field
// multiple sort keys are ok, but maybe should only be allowed when indexed fields are used

// GET: /workflow_instances
func (cr *WorkflowInstancesController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// get user
	u, done := GetAuthorizedUser(cx)
	if done {
		return
	}

	// get database query for user-specific ACLs
	db_query := GetAclQuery(u)

	// get limit, offset, order
	var err error
	var options *core.DefaultQueryOptions
	options, err = QueryParseDefaultOptions(cx)
	if err != nil {
		return
	}

	request_query := &Query{Li: cx.Request.URL.Query()}

	if request_query.Has("job_id") {
		db_query["job_id"] = request_query.Value("job_id")
	}

	//var result []core.WorkflowInstance
	var result_if interface{}
	var total int
	//var count int
	result_if, total, err = core.DBGetJobWorkflow_instances(db_query, options, false)

	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	cx.RespondWithPaginatedData(result_if, options.Limit, options.Offset, total)

	return
}
