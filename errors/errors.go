package errors

import (
	"regexp"
)

var (
	MongoDupKeyRegex = regexp.MustCompile("duplicate\\s+key")
)

const (
	ClientNotFound           = "Client not found"
	InvalidIndex             = "Invalid Index"
	InvalidFileTypeForFilter = "Invalid file type for filter"
	MongoDocNotFound         = "Document not found"
	NoAuth                   = "No Authorization"
	QueueEmpty               = "Server queue is empty"
	NoEligibleWorkunitFound  = "No eligible workunit found"
	UnAuth                   = "User Unauthorized"
)
