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
	ClientGroupBadName       = "Clientgroup name in token does not match that in the client."
	InvalidFileTypeForFilter = "Invalid file type for filter"
	InvalidIndex             = "Invalid Index"
	InvalidAuth              = "Invalid Auth Header"
	NoAuth                   = "No Authorization"
	NoEligibleWorkunitFound  = "No eligible workunit found"
	QueueEmpty               = "Server queue is empty"
	UnAuth                   = "User Unauthorized"
)
