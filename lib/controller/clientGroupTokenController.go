package controller

import (
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"
	mgo "gopkg.in/mgo.v2"
	"net/http"
)

// GET, POST, PUT, DELETE, OPTIONS: /cgroup/{cgid}/token/ (only OPTIONS, PUT and DELETE are implemented)
var ClientGroupTokenController goweb.ControllerFunc = func(cx *goweb.Context) {
	LogRequest(cx.Request)

	if cx.Request.Method == "OPTIONS" {
		cx.RespondWithOK()
		return
	}

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided and ANON_CG_WRITE is true, use the public user.
	// Otherwise if no auth was provided, throw an error.
	// Otherwise, proceed with generation of the clientgroup token using the user.
	if u == nil {
		if conf.ANON_CG_WRITE == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
			return
		}
	}

	cgid := cx.PathParams["cgid"]
	cg, err := core.LoadClientGroup(cgid)

	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("clientgroup id not found:"+cgid, http.StatusBadRequest)
		}
		return
	}

	// User must have write permissions on clientgroup or be clientgroup owner or be an admin or the clientgroup is publicly writable.
	// The other possibility is that public write of clientgroups is enabled and the clientgroup is publicly writable.
	rights := cg.Acl.Check(u.Uuid)
	public_rights := cg.Acl.Check("public")
	if (u.Uuid != "public" && (cg.Acl.Owner == u.Uuid || rights["write"] == true || u.Admin == true || public_rights["write"] == true)) ||
		(u.Uuid == "public" && conf.ANON_CG_WRITE == true && public_rights["write"] == true) {

		switch cx.Request.Method {
		case "PUT":
			if cg.Token != "" {
				cx.RespondWithErrorMessage("Clientgroup has existing token.  This must be deleted before a new token can be generated.", http.StatusBadRequest)
				return
			}
			cg.SetToken()
			if err = cg.Save(); err != nil {
				cx.RespondWithErrorMessage("Could not save clientgroup.", http.StatusInternalServerError)
				return
			}
			cx.RespondWithData(cg)
			return
		case "DELETE":
			cg.Token = ""
			if err = cg.Save(); err != nil {
				cx.RespondWithErrorMessage("Could not save clientgroup.", http.StatusInternalServerError)
				return
			}
			cx.RespondWithData(cg)
			return
		default:
			cx.RespondWithError(http.StatusNotImplemented)
			return
		}
	}

	cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
	return
}
