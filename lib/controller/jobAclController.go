package controller

import (
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/go-uuid/uuid"
	"github.com/MG-RAST/golib/goweb"
	"github.com/MG-RAST/golib/mgo"
	"net/http"
	"strings"
)

// GET: /job/{jid}/acl/ (only GET is supported here)
var JobAclController goweb.ControllerFunc = func(cx *goweb.Context) {
	LogRequest(cx.Request)

	jid := cx.PathParams["jid"]

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	// If no auth was provided, and anonymous read is allowed, use the public user
	if u == nil {
		if conf.ANON_READ == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	// Load job by id
	job, err := core.LoadJob(jid)
	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
			return
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("job not found: "+jid, http.StatusBadRequest)
			return
		}
	}

	// User must be job owner or be an admin
	if job.Acl.Owner != u.Uuid && u.Admin == false && job.Acl.Owner != "public" {
		cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
		return
	}

	if cx.Request.Method == "GET" {
		cx.RespondWithData(job.Acl)
		return
	} else {
		cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
		return
	}
	return
}

// GET, POST, PUT, DELETE: /job/{jid}/acl/{type}
var JobAclControllerTyped goweb.ControllerFunc = func(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}

	jid := cx.PathParams["jid"]
	rtype := cx.PathParams["type"]
	rmeth := cx.Request.Method

	if rtype != "all" && rtype != "owner" && rtype != "read" && rtype != "write" && rtype != "delete" {
		cx.RespondWithErrorMessage("Invalid acl type", http.StatusBadRequest)
		return
	}

	// Load job by id
	job, err := core.LoadJob(jid)

	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
			return
		} else {
			cx.RespondWithErrorMessage("job not found: "+jid, http.StatusBadRequest)
			return
		}
	}

	// Users that are not an admin or the clientgroup owner can only delete themselves from an ACL.
	if job.Acl.Owner != u.Uuid && u.Admin == false {
		if rmeth == "DELETE" {
			ids, err := parseJobAclRequestTyped(cx)
			if err != nil {
				cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
				return
			}
			if len(ids) != 1 || (len(ids) == 1 && ids[0] != u.Uuid) {
				cx.RespondWithErrorMessage("Non-owners of a job can delete one and only user from the ACLs (themselves).", http.StatusBadRequest)
				return
			}
			if rtype == "owner" {
				cx.RespondWithErrorMessage("Deleting job ownership is not a supported request type.", http.StatusBadRequest)
				return
			}
			if rtype == "all" {
				for _, atype := range []string{"read", "write", "delete"} {
					job.Acl.UnSet(ids[0], map[string]bool{atype: true})
				}
			} else {
				job.Acl.UnSet(ids[0], map[string]bool{rtype: true})
			}
			job.Save()
			cx.RespondWithOK()
			return
		}
		cx.RespondWithErrorMessage("Users that are not job owners can only delete themselves from ACLs.", http.StatusBadRequest)
		return
	}

	// At this point we know we're dealing with an admin or the clientgroup owner.
	// Admins and clientgroup owners can view/edit/delete ACLs
	if rmeth != "GET" {
		ids, err := parseJobAclRequestTyped(cx)
		if err != nil {
			cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
			return
		}
		if rmeth == "POST" || rmeth == "PUT" {
			if rtype == "owner" {
				if len(ids) == 1 {
					job.Acl.SetOwner(ids[0])
				} else {
					cx.RespondWithErrorMessage("Too many users. Jobs may have only one owner.", http.StatusBadRequest)
					return
				}
			} else if rtype == "all" {
				for _, atype := range []string{"read", "write", "delete"} {
					for _, i := range ids {
						job.Acl.Set(i, map[string]bool{atype: true})
					}
				}
			} else if rtype == "public_read" {
				job.Acl.Set("public", map[string]bool{"read": true})
			} else if rtype == "public_write" {
				job.Acl.Set("public", map[string]bool{"write": true})
			} else if rtype == "public_delete" {
				job.Acl.Set("public", map[string]bool{"delete": true})
			} else if rtype == "public_all" {
				for _, atype := range []string{"read", "write", "delete"} {
					job.Acl.Set("public", map[string]bool{atype: true})
				}
			} else {
				for _, i := range ids {
					job.Acl.Set(i, map[string]bool{rtype: true})
				}
			}
			job.Save()
		} else if rmeth == "DELETE" {
			if rtype == "owner" {
				cx.RespondWithErrorMessage("Deleting ownership is not a supported request type.", http.StatusBadRequest)
				return
			} else if rtype == "all" {
				for _, atype := range []string{"read", "write", "delete", "execute"} {
					for _, i := range ids {
						job.Acl.UnSet(i, map[string]bool{atype: true})
					}
				}
			} else if rtype == "public_read" {
				job.Acl.UnSet("public", map[string]bool{"read": true})
			} else if rtype == "public_write" {
				job.Acl.UnSet("public", map[string]bool{"write": true})
			} else if rtype == "public_delete" {
				job.Acl.UnSet("public", map[string]bool{"delete": true})
			} else if rtype == "public_all" {
				for _, atype := range []string{"read", "write", "delete"} {
					job.Acl.UnSet("public", map[string]bool{atype: true})
				}
			} else {
				for _, i := range ids {
					job.Acl.UnSet(i, map[string]bool{rtype: true})
				}
			}
			job.Save()
		} else {
			cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
			return
		}
	}

	switch rtype {
	default:
		cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
	case "owner":
		cx.RespondWithData(map[string]string{"owner": job.Acl.Owner})
	case "read", "public_read":
		cx.RespondWithData(map[string][]string{"read": job.Acl.Read})
	case "write", "public_write":
		cx.RespondWithData(map[string][]string{"write": job.Acl.Write})
	case "delete", "public_delete":
		cx.RespondWithData(map[string][]string{"delete": job.Acl.Delete})
	case "all", "public_all":
		cx.RespondWithData(job.Acl)
	}

	return
}

func parseJobAclRequestTyped(cx *goweb.Context) (ids []string, err error) {
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
