package errors

import (
	"regexp"
)

var (
	MongoDupKeyRegex = regexp.MustCompile("duplicate\\s+key")
)

const (
	ClientNotFound           = "Client not found"
	ClientSuspended          = "Client suspended"
	InvalidIndex             = "Invalid Index"
	InvalidFileTypeForFilter = "Invalid file type for filter"
	MongoDocNotFound         = "Document not found"
	NoAuth                   = "No Authorization"
	NoEligibleWorkunitFound  = "No eligible workunit found"
	QueueEmpty               = "Server queue is empty"
	//ShockNotReachable        = "Shock not reachable"
	UnAuth = "User Unauthorized"
)
