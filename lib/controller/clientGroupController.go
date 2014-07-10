package controller

import (
	"github.com/MG-RAST/golib/goweb"
)

type ClientGroupController struct{}

// OPTIONS: /clientgroup
func (cr *ClientGroupController) Options(cx *goweb.Context) {
	LogRequest(cx.Request)
	cx.RespondWithOK()
	return
}

// POST: /clientgroup
func (cr *ClientGroupController) Create(cx *goweb.Context) {
	LogRequest(cx.Request)

	return
}

// GET: /clientgroup/{id}
func (cr *ClientGroupController) Read(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	return
}

// GET: /clientgroup
func (cr *ClientGroupController) ReadMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	return
}

// PUT: /clientgroup
func (cr *ClientGroupController) UpdateMany(cx *goweb.Context) {
	LogRequest(cx.Request)

	return
}

// PUT: /clientgroup/{id}
func (cr *ClientGroupController) Update(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	return
}

// DELETE: /clientgroup/{id}
func (cr *ClientGroupController) Delete(id string, cx *goweb.Context) {
	LogRequest(cx.Request)

	return
}
