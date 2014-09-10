package controller

import (
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"
	"github.com/MG-RAST/golib/mgo"
	"github.com/MG-RAST/golib/mgo/bson"
	"net/http"
	"strconv"
	"strings"
)

type ClientGroupController struct{}

// OPTIONS: /clientgroup
func (cr *ClientGroupController) Options(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithOK()
	return
}

// POST: /clientgroup/{name}
func (cr *ClientGroupController) CreateWithId(name string, cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided and ANON_CG_WRITE is true, use the public user.
	// Otherwise if no auth was provided or user is not an admin, and ANON_CG_WRITE is false, throw an error.
	// Otherwise, proceed with creation of the clientgroup with the user.
	if u == nil && conf.ANON_CG_WRITE == true {
		u = &user.User{Uuid: "public"}
	} else if u == nil || !u.Admin {
		if conf.ANON_CG_WRITE == false {
			cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
			return
		}
	}

	cg, err := core.CreateClientGroup(name, u)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	cx.RespondWithData(cg)
	return
}

// GET: /clientgroup/{id}
func (cr *ClientGroupController) Read(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided and ANON_CG_READ is true, use the public user.
	// Otherwise if no auth was provided, throw an error.
	// Otherwise, proceed with retrieval of the clientgroup using the user.
	if u == nil {
		if conf.ANON_CG_READ == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
			return
		}
	}

	// Load clientgroup by id
	cg, err := core.LoadClientGroup(id)

	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("clientgroup id not found:"+id, http.StatusBadRequest)
		}
		return
	}

	// User must have read permissions on clientgroup or be clientgroup owner or be an admin or the clientgroup is publicly readable.
	// The other possibility is that public read of clientgroups is enabled and the clientgroup is publicly readable.
	rights := cg.Acl.Check(u.Uuid)
	public_rights := cg.Acl.Check("public")
	if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || rights["read"] == true || u.Admin == true || public_rights["read"] == true)) ||
		(u.Uuid == "public" && conf.ANON_CG_READ == true && public_rights["read"] == true) {
		cx.RespondWithData(cg)
		return
	}

	cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
	return
}

// GET: /clientgroup
func (cr *ClientGroupController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided and ANON_CG_READ is true, use the public user.
	// Otherwise if no auth was provided, throw an error.
	// Otherwise, proceed with retrieval of the clientgroups using the user.
	if u == nil {
		if conf.ANON_CG_READ == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
			return
		}
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	// Setup query and clientgroups objects
	q := bson.M{}
	cgs := core.ClientGroups{}

	// Add authorization checking to query if the user is not an admin
	if u.Admin == false {
		q["$or"] = []bson.M{bson.M{"acl.read": "public"}, bson.M{"acl.read": u.Uuid}, bson.M{"acl.owner": u.Uuid}}
	}

	limit := conf.DEFAULT_PAGE_SIZE
	offset := 0
	order := "last_modified"
	direction := "desc"
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
		direction = query.Value("direction")
	}

	// Gather params to make db query. Do not include the following list.
	skip := map[string]int{
		"limit":     1,
		"offset":    1,
		"order":     1,
		"direction": 1,
	}

	for key, val := range query.All() {
		_, s := skip[key]
		if !s {
			queryvalues := strings.Split(val[0], ",")
			q[key] = bson.M{"$in": queryvalues}
		}
	}

	total, err := cgs.GetPaginated(q, limit, offset, order, direction)
	if err != nil {
		cx.RespondWithError(http.StatusBadRequest)
		return
	}

	cx.RespondWithPaginatedData(cgs, limit, offset, total)
	return
}

// DELETE: /clientgroup/{id}
func (cr *ClientGroupController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided and ANON_CG_DELETE is true, use the public user.
	// Otherwise if no auth was provided, throw an error.
	// Otherwise, proceed with deletion of the clientgroup using the user.
	if u == nil {
		if conf.ANON_CG_DELETE == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
			return
		}
	}

	// Load clientgroup by id
	cg, err := core.LoadClientGroup(id)

	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("clientgroup id not found:"+id, http.StatusBadRequest)
		}
		return
	}

	// User must have delete permissions on clientgroup or be clientgroup owner or be an admin or the clientgroup is publicly deletable.
	// The other possibility is that public deletion of clientgroups is enabled and the clientgroup is publicly deletable.
	rights := cg.Acl.Check(u.Uuid)
	public_rights := cg.Acl.Check("public")
	if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || rights["delete"] == true || u.Admin == true || public_rights["delete"] == true)) ||
		(u.Uuid == "public" && conf.ANON_CG_DELETE == true && public_rights["delete"] == true) {
		err := core.DeleteClientGroup(id)
		if err != nil {
			cx.RespondWithErrorMessage("Could not delete clientgroup.", http.StatusInternalServerError)
			return
		}
		cx.RespondWithOK()
		return
	}

	cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
	return
}
