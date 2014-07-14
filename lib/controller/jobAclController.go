package controller

import (
	"github.com/MG-RAST/golib/goweb"
)

var (
	validAclTypes = map[string]bool{"read": true, "write": true, "delete": true, "owner": true}
)

// GET, POST, PUT, DELETE: /job/{jid}/acl/
var JobAclController goweb.ControllerFunc = func(cx *goweb.Context) {
	return
}

// GET, POST, PUT, DELETE: /job/{jid}/acl/{type}
var JobAclControllerTyped goweb.ControllerFunc = func(cx *goweb.Context) {
	return
}
