package controller

import (
	"fmt"
	"net/http"
	"net/url"

	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/golib/goweb"
)

// WorkflowInstancesController _
type WorkflowInstancesController struct{}

//  returns all workflow_instances or only those that match job_id, .i.e. /workflow_instances?job_id=<id>

// pagination should be efficient if list is sorted by an indexed field
// multiple sort keys are ok, but maybe should only be allowed when indexed fields are used

// GET: /workflow_instances/{uuid}
func (cr *WorkflowInstancesController) Read(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	// get user
	u, done := GetAuthorizedUser(cx)
	if done {
		return
	}

	// get database query for user-specific ACLs
	dbQuery := GetAclQuery(u)

	// get limit, offset, order
	var err error
	var options *core.DefaultQueryOptions
	options, err = QueryParseDefaultOptions(cx)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	//logger.Debug(3, "(WorkflowInstancesController/Read) A id: %s", id)
	//request_query := &Query{Li: cx.Request.URL.Query()}
	id, err = url.QueryUnescape(id)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	//logger.Debug(3, "(WorkflowInstancesController/Read) B id: %s", id)
	//if request_query.Has("job_id") {
	dbQuery["id"] = id
	//}

	//var result []core.WorkflowInstance
	var resultIf interface{}

	//var count int
	resultIf, err = core.DBGetJobWorkflowInstanceOne(dbQuery, options, false)

	if err != nil {
		cx.RespondWithErrorMessage(fmt.Sprintf("(WorkflowInstancesController/Read) %s not found: %s", id, err.Error()), http.StatusBadRequest)
		return
	}

	cx.RespondWithData(resultIf)

	return
}

// GET: /workflow_instances/
//   example: /workflow_instances?query&job_id=<job_id>
func (cr *WorkflowInstancesController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// get user
	u, done := GetAuthorizedUser(cx)
	if done {
		return
	}

	// get database query for user-specific ACLs
	dbQuery := GetAclQuery(u)

	// get limit, offset, order
	var err error
	var options *core.DefaultQueryOptions
	options, err = QueryParseDefaultOptions(cx)
	if err != nil {
		return
	}

	requestQuery := &Query{Li: cx.Request.URL.Query()}

	if !requestQuery.Has("query") {
		cx.RespondWithErrorMessage("(WorkflowInstancesController/ReadMany) not a query", http.StatusBadRequest)
		return
	}

	if requestQuery.Has("job_id") {
		dbQuery["job_id"] = requestQuery.Value("job_id")
	}

	//var result []core.WorkflowInstance
	var resultIf interface{}
	var total int
	//var count int
	resultIf, total, err = core.DBGetJobWorkflowInstances(dbQuery, options, false)

	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	cx.RespondWithPaginatedData(resultIf, options.Limit, options.Offset, total)

	return
}
