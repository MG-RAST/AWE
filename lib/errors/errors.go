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
	InvalidFileTypeForFilter = "Invalid file type for filter"
	InvalidIndex             = "Invalid Index"
	InvalidAuth              = "Invalid Auth Header"
	MongoDocNotFound         = "Document not found"
	NoAuth                   = "No Authorization"
	NoEligibleWorkunitFound  = "No eligible workunit found"
	QueueEmpty               = "Server queue is empty"
	UnAuth                   = "User Unauthorized"
)
