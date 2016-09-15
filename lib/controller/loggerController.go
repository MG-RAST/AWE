package controller

import (
	"github.com/MG-RAST/AWE/lib/conf"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/golib/goweb"
	"net/http"
	"strconv"
)

type LoggerController struct{}

// OPTIONS: /logger
func (cr *LoggerController) Options(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithOK()
	return
}

// POST: /logger
func (cr *LoggerController) Create(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// GET: /logger/{id}
func (cr *LoggerController) Read(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// GET: /logger
func (cr *LoggerController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	// public logging info
	if query.Has("event") {
		cx.RespondWithData(event.EventDiscription)
		return
	}
	if query.Has("debug") {
		cx.RespondWithData(map[string]int{"debuglevel": conf.DEBUG_LEVEL})
		return
	}

	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// PUT: /logger/{id}
func (cr *LoggerController) Update(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// PUT: /logger
func (cr *LoggerController) UpdateMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	// Try to authenticate user.
	u, err := request.Authenticate(cx.Request)
	if err != nil && err.Error() != e.NoAuth {
		cx.RespondWithErrorMessage(err.Error(), http.StatusUnauthorized)
		return
	}
	// must be admin user
	if u == nil || u.Admin == false {
		cx.RespondWithErrorMessage(e.NoAuth, http.StatusUnauthorized)
		return
	}

	// Gather query params
	query := &Query{Li: cx.Request.URL.Query()}

	// currently can only reset debug level
	if query.Has("debug") {
		levelStr := query.Value("debug")
		levelInt, err := strconv.Atoi(levelStr)
		if err != nil {
			cx.RespondWithErrorMessage("invalid debug level: "+err.Error(), http.StatusBadRequest)
		}
		conf.DEBUG_LEVEL = levelInt
		logger.Event(event.DEBUG_LEVEL, "level="+levelStr+";user="+u.Username)
		cx.RespondWithData(map[string]int{"debuglevel": conf.DEBUG_LEVEL})
		return
	}

	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// DELETE: /logger/{id}
func (cr *LoggerController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}

// DELETE: /logger
func (cr *LoggerController) DeleteMany(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithError(http.StatusNotImplemented)
	return
}
