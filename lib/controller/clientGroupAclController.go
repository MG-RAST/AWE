package controller

import (
	"errors"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/go-uuid/uuid"
	"github.com/MG-RAST/golib/goweb"
	"net/http"
	"strings"
)

// GET: /cgroup/{cgid}/acl/ (only GET is supported here)
var ClientGroupAclController goweb.ControllerFunc = func(cx *goweb.Context) {
	LogRequest(cx.Request)

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
		if err.Error() == e.MongoDocNotFound {
			cx.RespondWithNotFound()
			return
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("clientgroup not found: "+cgid, http.StatusBadRequest)
			return
		}
	}

	// User must have read permissions on clientgroup or be clientgroup owner or be an admin
	rights := cg.Acl.Check(u.Uuid)
	if cg.Acl.Owner != u.Uuid && rights["read"] == false && u.Admin == false {
		cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
		return
	}

	if cx.Request.Method == "GET" {
		cx.RespondWithData(cg.Acl)
		return
	} else {
		cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
		return
	}
	return
}

// GET, POST, PUT, DELETE: /cgroup/{cgid}/acl/{type}
var ClientGroupAclControllerTyped goweb.ControllerFunc = func(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	cgid := cx.PathParams["cgid"]
	rtype := cx.PathParams["type"]
	rmeth := cx.Request.Method

	if rtype != "all" && rtype != "owner" && rtype != "read" && rtype != "write" && rtype != "delete" && rtype != "execute" {
		cx.RespondWithErrorMessage("Invalid acl type", http.StatusBadRequest)
		return
	}

	// Load clientgroup by id
	cg, err := core.LoadClientGroup(cgid)

	if err != nil {
		if err.Error() == e.MongoDocNotFound {
			cx.RespondWithNotFound()
			return
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("clientgroup not found: "+cgid, http.StatusBadRequest)
			return
		}
	}

	// Users that are not the clientgroup owner can only delete themselves from an ACL.
	// Clientgroup owners can view/edit/delete ACLs
	if cg.Acl.Owner != u.Uuid {
		if rmeth == "DELETE" {
			ids, err := parseClientGroupAclRequestTyped(cx)
			if err != nil {
				cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
				return
			}
			if len(ids) != 1 || (len(ids) == 1 && ids[0] != u.Uuid) {
				cx.RespondWithErrorMessage("Non-owners of clientgroups can delete one and only user from the ACLs (themselves)", http.StatusBadRequest)
				return
			}
			if rtype == "owner" {
				cx.RespondWithErrorMessage("Deleting ownership is not a supported request type.", http.StatusBadRequest)
				return
			} else if rtype == "all" {
				for _, atype := range []string{"read", "write", "delete", "execute"} {
					cg.Acl.UnSet(ids[0], map[string]bool{atype: true})
				}
			} else {
				cg.Acl.UnSet(ids[0], map[string]bool{rtype: true})
			}
			cg.Save()
			cx.RespondWithOK()
			return
		}
		cx.RespondWithErrorMessage("Users that are not clientgroup owners can only delete themselves from ACLs.", http.StatusBadRequest)
		return
	}

	// At this point we know we're dealing with just the clientgroup owner.
	if rmeth != "GET" {
		ids, err := parseClientGroupAclRequestTyped(cx)
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
		if rmeth == "POST" || rmeth == "PUT" {
			if rtype == "owner" {
				if len(ids) == 1 {
					cg.Acl.SetOwner(ids[0])
				} else {
					cx.RespondWithErrorMessage("Too many users. Clientgroups may have only one owner.", http.StatusBadRequest)
					return
				}
			} else if rtype == "all" {
				for _, atype := range []string{"read", "write", "delete", "execute"} {
					for _, i := range ids {
						cg.Acl.Set(i, map[string]bool{atype: true})
					}
				}
			} else {
				for _, i := range ids {
					cg.Acl.Set(i, map[string]bool{rtype: true})
				}
			}
			cg.Save()
		} else if rmeth == "DELETE" {
			if rtype == "owner" {
				cx.RespondWithErrorMessage("Deleting ownership is not a supported request type.", http.StatusBadRequest)
				return
			} else if rtype == "all" {
				for _, atype := range []string{"read", "write", "delete", "execute"} {
					for _, i := range ids {
						cg.Acl.UnSet(i, map[string]bool{atype: true})
					}
				}
			} else {
				for _, i := range ids {
					cg.Acl.UnSet(i, map[string]bool{rtype: true})
				}
			}
			cg.Save()
		} else {
			cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
			return
		}
	}

	switch rtype {
	default:
		cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
	case "owner":
		cx.RespondWithData(map[string]string{"owner": cg.Acl.Owner})
	case "read":
		cx.RespondWithData(map[string][]string{"read": cg.Acl.Read})
	case "write":
		cx.RespondWithData(map[string][]string{"write": cg.Acl.Write})
	case "delete":
		cx.RespondWithData(map[string][]string{"delete": cg.Acl.Delete})
	case "execute":
		cx.RespondWithData(map[string][]string{"execute": cg.Acl.Execute})
	case "all":
		cx.RespondWithData(cg.Acl)
	}

	return
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
		return nil, errors.New("Action requires list of comma separated usernames in 'users' parameter")
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
