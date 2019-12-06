package errors

import (
	"regexp"
)

var (
	MongoDupKeyRegex = regexp.MustCompile("duplicate\\s+key")
)

const (
	ClientNotFound           = "Client not found"
	ClientNotActive          = "Client not active"
	ClientSuspended          = "Client suspended"
	ClientNotSuspended       = "Client not suspended"
	ClientDeleted            = "Client deleted"
	ClientBusy               = "Client busy"
	ClientGroupBadName       = "Clientgroup name in token does not match that in the client."
	InvalidFileTypeForFilter = "Invalid file type for filter"
	InvalidIndex             = "Invalid Index"
	InvalidAuth              = "Invalid Auth Header"
	NoAuth                   = "No Authorization"
	NoEligibleWorkunitFound  = "No eligible workunit found"
	QueueEmpty               = "Server queue is empty"
	QueueFull                = "Server queue is full"
	QueueSuspend             = "Server queue is suspended"
	UnAuth                   = "User Unauthorized"
	ServerNotFound           = "Server not found"
	LockTimeout              = "Did not get lock"
)
