package errors

import (
	"regexp"
)

var (
	MongoDupKeyRegex = regexp.MustCompile("duplicate\\s+key")
)

const (
	MongoDocNotFound         = "Document not found"
	UnAuth                   = "User Unauthorized"
	NoAuth                   = "No Authorization"
	InvalidIndex             = "Invalid Index"
	InvalidFileTypeForFilter = "Invalid file type for filter"
)
