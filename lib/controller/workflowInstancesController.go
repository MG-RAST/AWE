package controller

import (
	"net/http"
	"strconv"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/golib/goweb"
	"gopkg.in/mgo.v2/bson"
)

type WorkflowInstancesController struct{}

// GET: /workflow_instances

func (cr *WorkflowInstancesController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	permisson_query := bson.M{}

	// cg, err := request.AuthenticateClientGroup(cx.Request)
	// if err != nil {
	// 	if err.Error() == e.NoAuth || err.Error() == e.UnAuth || err.Error() == e.InvalidAuth {
	// 		if conf.CLIENT_AUTH_REQ == true {
	// 			cx.RespondWithError(http.StatusUnauthorized)
	// 			return
	// 		}
	// 	} else {
	// 		logger.Error("Err@AuthenticateClientGroup: " + err.Error())
	// 		cx.RespondWithError(http.StatusInternalServerError)
	// 		return
	// 	}
	// }

	if u != nil {
		// Add authorization checking to query if the user is not an admin
		if u.Admin == false {
			permisson_query["$or"] = []bson.M{bson.M{"acl.read": "public"}, bson.M{"acl.read": u.Uuid}, bson.M{"acl.owner": u.Uuid}, bson.M{"acl": bson.M{"$exists": "false"}}}
		}
	} else {
		// User is anonymous
		if conf.ANON_READ {
			// select on only jobs that are publicly readable
			permisson_query["acl.read"] = "public"
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	// // check that clientgroup auth token matches group of client
	// clientid := query.Value("client")
	// client, ok, err := core.QMgr.GetClient(clientid, true)
	// if err != nil {
	// 	cx.RespondWithErrorMessage(err.Error(), http.StatusInternalServerError)
	// 	return
	// }
	// logger.Debug(3, "WorkflowInstances request with clientid=%s", clientid)
	// if !ok {
	// 	cx.RespondWithErrorMessage(e.ClientNotFound, http.StatusBadRequest)
	// 	return
	// }

	// if cg != nil && client.Group != cg.Name {
	// 	cx.RespondWithErrorMessage("Clientgroup name in token does not match that in the client configuration.", http.StatusBadRequest)
	// 	return
	// }

	limit := conf.DEFAULT_PAGE_SIZE
	offset := 0
	order := ""

	if query.Has("limit") {
		limit, err = strconv.Atoi(query.Value("limit"))
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
	}
	if query.Has("offset") {
		offset, err = strconv.Atoi(query.Value("offset"))
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
	}
	if query.Has("order") {
		order = query.Value("order")
	}

	if query.Has("direction") {
		cx.RespondWithErrorMessage("direction is not supported, use minus to change direction", http.StatusBadRequest)
	}

	//var result []core.WorkflowInstance
	var result_if interface{}

	options := map[string]int{"limit": limit, "offset": offset}
	//var count int
	result_if, _, err = core.DBGetJobWorkflow_instances(permisson_query, options, order, false)

	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
	}

	cx.RespondWithData(result_if)

	// //checkout a workunit in FCFS order
	// workunits, err := core.QMgr.CheckoutWorkunits("FCFS", clientid, client, availableBytes, 1)

	// if err != nil {

	// 	err_str := err.Error()

	// 	err = fmt.Errorf("(ReadMany GET /work) CheckoutWorkunits returns: clientid=%s;available=%d;error=%s", clientid, availableBytes, err.Error())

	// 	if strings.Contains(err_str, e.QueueEmpty) ||
	// 		strings.Contains(err_str, e.QueueSuspend) ||
	// 		strings.Contains(err_str, e.NoEligibleWorkunitFound) ||
	// 		strings.Contains(err_str, e.ClientNotFound) ||
	// 		strings.Contains(err_str, e.ClientSuspended) {

	// 		logger.Debug(3, err.Error())
	// 	} else {
	// 		logger.Error(err.Error())
	// 	}

	// 	cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
	// 	return
	// }

	// //log event about workunit checkout (WO)
	// workids := []string{}
	// for _, work := range workunits {
	// 	workids = append(workids, work.Id)
	// }
	// logger.Event(event.WORK_CHECKOUT, fmt.Sprintf("workids=%s;clientid=%s;available=%d", strings.Join(workids, ","), clientid, availableBytes))

	// // Base case respond with node in json

	// if len(workunits) == 0 {
	// 	err = fmt.Errorf("(ReadMany GET /work) No workunits found, clientid=%s", clientid)
	// 	logger.Error(err.Error())
	// 	cx.RespondWithErrorMessage("(ReadMany GET /work) No workunits found", http.StatusBadRequest)
	// 	return
	// }

	// workunit := workunits[0]
	// workunit.State = core.WORK_STAT_RESERVED
	// workunit.Client = clientid

	// //test, err := json.Marshal(workunit)
	// //if err != nil {
	// //	panic("did not work")
	// //}
	// //fmt.Println("workunit: ")
	// //fmt.Printf("workunit:\n %s\n", test)

	// cx.RespondWithData(workunit)
	return
}
