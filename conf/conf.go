package conf

import ()

type idxOpts struct {
	unique   bool
	dropDups bool
	sparse   bool
}

// Setup conf variables
var (
	BasePriority = 1

	// Config File
	CONFIGFILE = ""

	// AWE 
	SITEPORT = 0
	APIPORT  = 0

	// Admin
	ADMINEMAIL = ""
	SECRETKEY  = ""

	// Directories
	DATAPATH = ""
	SITEPATH = ""
	LOGSPATH = ""

	// Mongodb 
	MONGODB = ""

	// Node Indices
	NODEIDXS map[string]idxOpts = nil
)

func init() {}
