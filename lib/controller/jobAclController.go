package controller

import (
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/AWE/vendor/github.com/MG-RAST/golib/go-uuid/uuid"
	"github.com/MG-RAST/AWE/vendor/github.com/MG-RAST/golib/goweb"
	mgo "github.com/MG-RAST/AWE/vendor/gopkg.in/mgo.v2"
	"net/http"
	"strings"
)

var (
	validJobAclTypes = map[string]bool{"all": true, "read": true, "write": true, "delete": true, "owner": true,
		"public_all": true, "public_read": true, "public_write": true, "public_delete": true}
)

// GET: /job/{jid}/acl/ (only OPTIONS and GET are supported here)
var JobAclController goweb.ControllerFunc = func(cx *goweb.Context) {
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
		if conf.ANON_READ == true {
			u = &user.User{Uuid: "public"}
		} else {
			cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
			return
		}
	}

	jid := cx.PathParams["jid"]

	// Load job by id
	job, err := core.LoadJob(jid)
	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			// In theory the db connection could be lost between
			// checking user and load but seems unlikely.
			cx.RespondWithErrorMessage("job not found: "+jid, http.StatusBadRequest)
		}
		return
	}

	// Only the owner, an admin, or someone with read access can view acl's.
	//
	// NOTE: If the job is publicly owned, then anyone can view all acl's. The owner can only
	//       be "public" when anonymous job creation (ANON_WRITE) is enabled in AWE config.

	rights := job.Acl.Check(u.Uuid)
	if job.Acl.Owner != u.Uuid && u.Admin == false && job.Acl.Owner != "public" && rights["read"] == false {
		cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
		return
	}

	if cx.Request.Method == "GET" {
		cx.RespondWithData(job.Acl)
	} else {
		cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
	}
	return
}

// GET, POST, PUT, DELETE, OPTIONS: /job/{jid}/acl/{type}
var JobAclControllerTyped goweb.ControllerFunc = func(cx *goweb.Context) {
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

	jid := cx.PathParams["jid"]
	rtype := cx.PathParams["type"]
	rmeth := cx.Request.Method

	if !validJobAclTypes[rtype] {
		cx.RespondWithErrorMessage("Invalid acl type", http.StatusBadRequest)
		return
	}

	// Load job by id
	job, err := core.LoadJob(jid)
	if err != nil {
		if err == mgo.ErrNotFound {
			cx.RespondWithNotFound()
		} else {
			cx.RespondWithErrorMessage("job not found: "+jid, http.StatusBadRequest)
		}
		return
	}

	// Parse user list
	ids, err := parseJobAclRequestTyped(cx)
	if err != nil {
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}

	// Users that are not an admin or the job owner can only delete themselves from an ACL.
	if job.Acl.Owner != u.Uuid && u.Admin == false {
		if rmeth == "DELETE" {
			if len(ids) != 1 || (len(ids) == 1 && ids[0] != u.Uuid) {
				cx.RespondWithErrorMessage("Non-owners of a job can delete one and only user from the ACLs (themselves).", http.StatusBadRequest)
				return
			}
			if rtype == "owner" {
				cx.RespondWithErrorMessage("Deleting job ownership is not a supported request type.", http.StatusBadRequest)
				return
			}
			if rtype == "all" {
				job.Acl.UnSet(ids[0], map[string]bool{"read": true, "write": true, "delete": true})
			} else {
				job.Acl.UnSet(ids[0], map[string]bool{rtype: true})
			}
			job.Save()
			cx.RespondWithData(job.Acl)
			return
		}
		cx.RespondWithErrorMessage("Users that are not job owners can only delete themselves from ACLs.", http.StatusBadRequest)
		return
	}

	// At this point we know we're dealing with an admin or the job owner.
	// Admins and job owners can view/edit/delete ACLs
	if rmeth == "GET" {
		cx.RespondWithData(job.Acl)
		return
	} else if rmeth == "POST" || rmeth == "PUT" {
		if rtype == "owner" {
			if len(ids) == 1 {
				job.Acl.SetOwner(ids[0])
			} else {
				cx.RespondWithErrorMessage("Jobs must have one owner.", http.StatusBadRequest)
				return
			}
		} else if rtype == "all" {
			for _, i := range ids {
				job.Acl.Set(i, map[string]bool{"read": true, "write": true, "delete": true})
			}
		} else if rtype == "public_read" {
			job.Acl.Set("public", map[string]bool{"read": true})
		} else if rtype == "public_write" {
			job.Acl.Set("public", map[string]bool{"write": true})
		} else if rtype == "public_delete" {
			job.Acl.Set("public", map[string]bool{"delete": true})
		} else if rtype == "public_all" {
			job.Acl.Set("public", map[string]bool{"read": true, "write": true, "delete": true})
		} else {
			for _, i := range ids {
				job.Acl.Set(i, map[string]bool{rtype: true})
			}
		}
		job.Save()
		cx.RespondWithData(job.Acl)
		return
	} else if rmeth == "DELETE" {
		if rtype == "owner" {
			cx.RespondWithErrorMessage("Deleting ownership is not a supported request type.", http.StatusBadRequest)
			return
		} else if rtype == "all" {
			for _, i := range ids {
				job.Acl.UnSet(i, map[string]bool{"read": true, "write": true, "delete": true})
			}
		} else if rtype == "public_read" {
			job.Acl.UnSet("public", map[string]bool{"read": true})
		} else if rtype == "public_write" {
			job.Acl.UnSet("public", map[string]bool{"write": true})
		} else if rtype == "public_delete" {
			job.Acl.UnSet("public", map[string]bool{"delete": true})
		} else if rtype == "public_all" {
			job.Acl.UnSet("public", map[string]bool{"read": true, "write": true, "delete": true})
		} else {
			for _, i := range ids {
				job.Acl.UnSet(i, map[string]bool{rtype: true})
			}
		}
		job.Save()
		cx.RespondWithData(job.Acl)
		return
	} else {
		cx.RespondWithErrorMessage("This request type is not implemented.", http.StatusNotImplemented)
		return
	}
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
