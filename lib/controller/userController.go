package controller

import (
	"encoding/base64"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"
	"net/http"
	"strings"
)

type UserController struct{}

// Options: /user
func (cr *UserController) Options(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithOK()
	return
}

// POST: /user
// To create a new user make a empty POST to /user with user:password
// Basic Auth encoded in the header. Return new user object.
func (cr *UserController) Create(cx *goweb.Context) {
	// Log Request
	LogRequest(cx.Request)

	if _, ok := cx.Request.Header["Authorization"]; !ok {
		cx.RespondWithError(http.StatusUnauthorized)
		return
	}
	header := cx.Request.Header.Get("Authorization")
	tmpAuthArray := strings.Split(header, " ")

	authValues, err := base64.URLEncoding.DecodeString(tmpAuthArray[1])
	if err != nil {
		err = errors.New("Failed to decode encoded auth settings in http request.")
		cx.RespondWithError(http.StatusBadRequest)
		return
	}

	authValuesArray := strings.Split(string(authValues), ":")
	if conf.ANON_CREATEUSER == false && len(authValuesArray) != 4 {
		if len(authValuesArray) == 2 {
			cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
			return
		} else {
			cx.RespondWithError(http.StatusBadRequest)
			return
		}
	}
	name := authValuesArray[0]
	passwd := authValuesArray[1]
	admin := false
	if len(authValuesArray) == 4 {
		if authValuesArray[2] != fmt.Sprint(conf.SECRET_KEY) {
			cx.RespondWithErrorMessage(e.UnAuth, http.StatusUnauthorized)
			return
		} else if authValuesArray[3] == "true" {
			admin = true
		}
	}

	u, err := user.New(name, passwd, admin)
	if err != nil {
		// Duplicate key check
		if e.MongoDupKeyRegex.MatchString(err.Error()) {
			logger.Error("Err@user_Create: duplicate key error")
			cx.RespondWithErrorMessage("Username not available", http.StatusBadRequest)
			return
		} else {
			logger.Error("Err@user_Create: " + err.Error())
			cx.RespondWithError(http.StatusInternalServerError)
			return
		}
	}
	cx.RespondWithData(u)
	return
}

// DELETE: /user/{id}
func (cr *UserController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
}

// DELETE: /user
func (cr *UserController) DeleteMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
}

// GET: /user/{id}
func (cr *UserController) Read(id string, cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)
	u, err := request.Authenticate(cx.Request)
	if err != nil {
		if err.Error() == e.NoAuth || err.Error() == e.UnAuth {
			cx.RespondWithError(http.StatusUnauthorized)
			return
		} else {
			logger.Error("Err@user_Read: " + err.Error())
			cx.RespondWithError(http.StatusInternalServerError)
			return
		}
	}
	// Any user can access their own user info. Only admins can
	// access other's info
	if u.Uuid == id {
		cx.RespondWithData(u)
		return
	} else if u.Admin {
		nu, err := user.FindByUuid(id)
		if err != nil {
			if err.Error() == e.MongoDocNotFound {
				cx.RespondWithNotFound()
				return
			} else {
				logger.Error("Err@user_Read:Admin: " + err.Error())
				cx.RespondWithError(http.StatusInternalServerError)
				return
			}
		}
		cx.RespondWithData(nu)
		return
	} else {
		// Not sure how we could end up here but its probably the
		// user's fault
		cx.RespondWithError(http.StatusBadRequest)
		return
	}
}

// GET: /user
func (cr *UserController) ReadMany(cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)
	u, err := request.Authenticate(cx.Request)

	if err != nil {
		if err.Error() == e.NoAuth || err.Error() == e.UnAuth {
			cx.RespondWithError(http.StatusUnauthorized)
			return
		} else {
			logger.Error("Err@user_Read: " + err.Error())
			cx.RespondWithError(http.StatusInternalServerError)
			return
		}
	}
	if u.Admin {
		users := user.Users{}
		user.AdminGet(&users)
		if len(users) > 0 {
			cx.RespondWithData(users)
			return
		} else {
			cx.RespondWithNotFound()
			return
		}
	} else {
		cx.RespondWithError(http.StatusUnauthorized)
		return
	}
}

// PUT: /user/{id}
func (cr *UserController) Update(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
}

// PUT: /user
func (cr *UserController) UpdateMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
}
