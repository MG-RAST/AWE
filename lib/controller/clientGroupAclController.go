package controller

import (
	"net/http"
	"strings"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/go-uuid/uuid"
	"github.com/MG-RAST/golib/goweb"
	mgo "gopkg.in/mgo.v2"
)

var (
	validClientGroupAclTypes = map[string]bool{"all": true, "read": true, "write": true, "delete": true, "owner": true, "execute": true,
		"public_all": true, "public_read": true, "public_write": true, "public_delete": true, "public_execute": true}
)

// GET: /cgroup/{cgid}/acl/ (only OPTIONS and GET are supported here)
var ClientGroupAclController goweb.ControllerFunc = func(cx *goweb.Context) {
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

	// If no auth was provided, and anonymous read is allowed, use the public user
	if u == nil {
		if conf.ANON_CG_READ == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	cgid := cx.PathParams["cgid"]

	// Load clientgroup by id
	cg, err := core.LoadClientGroup(cgid)
	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
			return
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("clientgroup not found: "+cgid, http.StatusBadRequest)
			return
		}
	}

	// Only the owner, an admin, or someone with read access can view acl's.
	//
	// NOTE: If the clientgroup is publicly owned, then anyone can view all acl's. The owner can only
	//       be "public" when anonymous clientgroup creation (ANON_CG_WRITE) is enabled in AWE config.

	rights := cg.ACL.Check(u.Uuid)
	if cg.ACL.Owner != u.Uuid && u.Admin == false && cg.ACL.Owner != "public" && rights["read"] == false {
		cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
		return
	}

	if cx.Request.Method == "GET" {
		cx.RespondWithData(cg.ACL)
	} else {
		cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
	}
	return
}

// GET, POST, PUT, DELETE, OPTIONS: /cgroup/{cgid}/acl/{type}
var ClientGroupAclControllerTyped goweb.ControllerFunc = func(cx *goweb.Context) {
	LogRequest(cx.Request)

	if cx.Request.Method == "OPTIONS" {
		cx.RespondWithOK()
		return
	}

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	cgid := cx.PathParams["cgid"]
	rtype := cx.PathParams["type"]
	rmeth := cx.Request.Method

	if !validClientGroupAclTypes[rtype] {
		cx.RespondWithErrorMessage("Invalid acl type", http.StatusBadRequest)
		return
	}

	// Load clientgroup by id
	cg, err := core.LoadClientGroup(cgid)
	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("clientgroup not found: "+cgid, http.StatusBadRequest)
		}
		return
	}

	// Parse user list
	ids, err := parseClientGroupAclRequestTyped(cx)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	// Users that are not an admin or clientgroup job owner can only delete themselves from an ACL.
	if cg.ACL.Owner != u.Uuid && u.Admin == false {
		if rmeth == "DELETE" {
			if len(ids) != 1 || (len(ids) == 1 && ids[0] != u.Uuid) {
				cx.RespondWithErrorMessage("Non-owners of clientgroups can delete one and only user from the ACLs (themselves)", http.StatusBadRequest)
				return
			}
			if rtype == "owner" {
				cx.RespondWithErrorMessage("Deleting ownership is not a supported request type.", http.StatusBadRequest)
				return
			} else if rtype == "all" {
				cg.ACL.UnSet(ids[0], map[string]bool{"read": true, "write": true, "delete": true, "execute": true})
			} else {
				cg.ACL.UnSet(ids[0], map[string]bool{rtype: true})
			}
			cg.Save()
			cx.RespondWithData(cg.ACL)
			return
		}
		cx.RespondWithErrorMessage("Users that are not clientgroup owners can only delete themselves from ACLs.", http.StatusBadRequest)
		return
	}

	// At this point we know we're dealing with an admin or the clientgroup owner.
	// Admins and clientgroup owners can view/edit/delete ACLs
	if rmeth == "GET" {
		cx.RespondWithData(cg.ACL)
		return
	} else if rmeth == "POST" || rmeth == "PUT" {
		if rtype == "owner" {
			if len(ids) == 1 {
				cg.ACL.SetOwner(ids[0])
			} else {
				cx.RespondWithErrorMessage("Clientgroups must have one owner.", http.StatusBadRequest)
				return
			}
		} else if rtype == "all" {
			for _, i := range ids {
				cg.ACL.Set(i, map[string]bool{"read": true, "write": true, "delete": true, "execute": true})
			}
		} else if rtype == "public_read" {
			cg.ACL.Set("public", map[string]bool{"read": true})
		} else if rtype == "public_write" {
			cg.ACL.Set("public", map[string]bool{"write": true})
		} else if rtype == "public_delete" {
			cg.ACL.Set("public", map[string]bool{"delete": true})
		} else if rtype == "public_execute" {
			cg.ACL.Set("public", map[string]bool{"execute": true})
		} else if rtype == "public_all" {
			cg.ACL.Set("public", map[string]bool{"read": true, "write": true, "delete": true, "execute": true})
		} else {
			for _, i := range ids {
				cg.ACL.Set(i, map[string]bool{rtype: true})
			}
		}
		cg.Save()
		cx.RespondWithData(cg.ACL)
		return
	} else if rmeth == "DELETE" {
		if rtype == "owner" {
			cx.RespondWithErrorMessage("Deleting ownership is not a supported request type.", http.StatusBadRequest)
			return
		} else if rtype == "all" {
			for _, i := range ids {
				cg.ACL.UnSet(i, map[string]bool{"read": true, "write": true, "delete": true, "execute": true})
			}
		} else if rtype == "public_read" {
			cg.ACL.UnSet("public", map[string]bool{"read": true})
		} else if rtype == "public_write" {
			cg.ACL.UnSet("public", map[string]bool{"write": true})
		} else if rtype == "public_delete" {
			cg.ACL.UnSet("public", map[string]bool{"delete": true})
		} else if rtype == "public_execute" {
			cg.ACL.UnSet("public", map[string]bool{"execute": true})
		} else if rtype == "public_all" {
			cg.ACL.UnSet("public", map[string]bool{"read": true, "write": true, "delete": true, "execute": true})
		} else {
			for _, i := range ids {
				cg.ACL.UnSet(i, map[string]bool{rtype: true})
			}
		}
		cg.Save()
		cx.RespondWithData(cg.ACL)
		return
	} else {
		cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
		return
	}
}

func parseClientGroupAclRequestTyped(cx *goweb.Context) (ids []string, err error) {
	var users []string
	query := cx.Request.URL.Query()
	params, _, err := ParseMultipartForm(cx.Request)
	if _, ok := query["users"]; ok && err != nil && err.Error() == "request Content-Type isn't multipart/form-data" {
		users = strings.Split(query.Get("users"), ",")
	} else if params["users"] != "" {
		users = strings.Split(params["users"], ",")
	} else {
		return nil, nil
	}
	for _, v := range users {
		if uuid.Parse(v) != nil {
			ids = append(ids, v)
		} else {
			u := user.User{Username: v}
			if err := u.SetMongoInfo(); err != nil {
				return nil, err
			}
			ids = append(ids, u.Uuid)
		}
	}
	return ids, nil
}
