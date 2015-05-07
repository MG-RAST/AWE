// Package conf parses start up args and config file
package conf

import (
	"flag"
	"fmt"
	"github.com/MG-RAST/golib/goconfig/config"
	"os"
	"strconv"
	"strings"
)

type idxOpts struct {
	unique   bool
	dropDups bool
	sparse   bool
}

var (
	// Admin
	ADMIN_EMAIL = ""
	ADMIN_USERS = ""

	// Permissions for anonymous user
	ANON_READ   = true
	ANON_WRITE  = false
	ANON_DELETE = false

	// Address
	API_IP   = ""
	API_PORT = ""
	API_URL  = "" // for external address only

	// Auth
	AUTH_BASIC              = false
	AUTH_GLOBUS_TOKEN_URL   = ""
	AUTH_GLOBUS_PROFILE_URL = ""
	AUTH_MGRAST_OAUTH_URL   = ""

	// Default Chunksize for size virtual index
	CHUNK_SIZE int64 = 1048576

	// Config File
	CONFIG_FILE = ""

	GOMAXPROCS = ""

	// Logs
	LOG_PERF   = false // Indicates whether performance logs should be stored
	LOG_ROTATE = true  // Indicates whether logs should be rotated daily

	// Mongo information
	MONGODB_HOSTS             = ""
	MONGODB_DATABASE          = ""
	MONGODB_USER              = ""
	MONGODB_PASSWORD          = ""
	MONGODB_ATTRIBUTE_INDEXES = ""

	// Node Indices
	NODE_IDXS map[string]idxOpts = nil

	// Paths
	PATH_SITE    = ""
	PATH_DATA    = ""
	PATH_LOGS    = ""
	PATH_LOCAL   = ""
	PATH_PIDFILE = ""

	// Reload
	RELOAD = ""

	// SSL
	SSL      = false
	SSL_KEY  = ""
	SSL_CERT = ""

	// Versions
	VERSIONS = make(map[string]int)
)

// Initialize is an explicit init. Enables outside use
// of shock-server packages. Parses config and populates
// the conf variables.
func Initialize() {
	gopath := os.Getenv("GOPATH")
	flag.StringVar(&CONFIG_FILE, "conf", gopath+"/src/github.com/MG-RAST/Shock/shock-server.conf.template", "path to config file")
	flag.StringVar(&RELOAD, "reload", "", "path or url to shock data. WARNING this will drop all current data.")
	flag.Parse()
	c, err := config.ReadDefault(CONFIG_FILE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: error reading conf file: %v\n", err)
		os.Exit(1)
	}

	// Admin
	ADMIN_EMAIL, _ = c.String("Admin", "email")
	ADMIN_USERS, _ = c.String("Admin", "users")

	// Access-Control
	ANON_READ, _ = c.Bool("Anonymous", "read")
	ANON_WRITE, _ = c.Bool("Anonymous", "write")
	ANON_DELETE, _ = c.Bool("Anonymous", "delete")

	// Address
	API_IP, _ = c.String("Address", "api-ip")
	API_PORT, _ = c.String("Address", "api-port")

	// URLs
	API_URL, _ = c.String("External", "api-url")

	// Auth
	AUTH_BASIC, _ = c.Bool("Auth", "basic")
	AUTH_GLOBUS_TOKEN_URL, _ = c.String("Auth", "globus_token_url")
	AUTH_GLOBUS_PROFILE_URL, _ = c.String("Auth", "globus_profile_url")
	AUTH_MGRAST_OAUTH_URL, _ = c.String("Auth", "mgrast_oauth_url")

	// Runtime
	GOMAXPROCS, _ = c.String("Runtime", "GOMAXPROCS")

	LOG_PERF, _ = c.Bool("Log", "perf_log")
	LOG_ROTATE, _ = c.Bool("Log", "rotate")

	// Mongodb
	MONGODB_ATTRIBUTE_INDEXES, _ = c.String("Mongodb", "attribute_indexes")
	if MONGODB_DATABASE, err = c.String("Mongodb", "database"); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: Mongodb database must be set in config file.")
		os.Exit(1)
	}
	MONGODB_HOSTS, _ = c.String("Mongodb", "hosts")
	MONGODB_PASSWORD, _ = c.String("Mongodb", "password")
	MONGODB_USER, _ = c.String("Mongodb", "user")

	// parse Node-Indices
	NODE_IDXS = map[string]idxOpts{}
	nodeIdx, _ := c.Options("Node-Indices")
	for _, opt := range nodeIdx {
		val, _ := c.String("Node-Indices", opt)
		opts := idxOpts{}
		for _, parts := range strings.Split(val, ",") {
			p := strings.Split(parts, ":")
			if p[0] == "unique" {
				if p[1] == "true" {
					opts.unique = true
				} else {
					opts.unique = false
				}
			} else if p[0] == "dropDups" {
				if p[1] == "true" {
					opts.dropDups = true
				} else {
					opts.dropDups = false
				}
			} else if p[0] == "sparse" {
				if p[1] == "true" {
					opts.sparse = true
				} else {
					opts.sparse = false
				}
			}
		}
		NODE_IDXS[opt] = opts
	}

	// Paths
	PATH_SITE, _ = c.String("Paths", "site")
	PATH_DATA, _ = c.String("Paths", "data")
	PATH_LOGS, _ = c.String("Paths", "logs")
	PATH_LOCAL, _ = c.String("Paths", "local_paths")
	PATH_PIDFILE, _ = c.String("Paths", "pidfile")

	// SSL
	SSL, _ = c.Bool("SSL", "enable")
	if SSL {
		SSL_KEY, _ = c.String("SSL", "key")
		SSL_CERT, _ = c.String("SSL", "cert")
	}

	VERSIONS["ACL"] = 2
	VERSIONS["Auth"] = 1
	VERSIONS["Node"] = 4
}

// Bool is a convenience wrapper around strconv.ParseBool
func Bool(s string) bool {
	b, _ := strconv.ParseBool(s)
	return b
}

// Print prints the configuration loads to stdout
func Print() {
	fmt.Printf("####### Anonymous ######\nread:\t%v\nwrite:\t%v\ndelete:\t%v\n\n", ANON_READ, ANON_WRITE, ANON_DELETE)
	if (AUTH_GLOBUS_TOKEN_URL != "" && AUTH_GLOBUS_PROFILE_URL != "") || AUTH_MGRAST_OAUTH_URL != "" {
		fmt.Printf("##### Auth #####\n")
		if AUTH_GLOBUS_TOKEN_URL != "" && AUTH_GLOBUS_PROFILE_URL != "" {
			fmt.Printf("type:\tglobus\ntoken_url:\t%s\nprofile_url:\t%s\n\n", AUTH_GLOBUS_TOKEN_URL, AUTH_GLOBUS_PROFILE_URL)
		}
		if AUTH_MGRAST_OAUTH_URL != "" {
			fmt.Printf("type:\tmgrast\noauth_url:\t%s\n\n", AUTH_MGRAST_OAUTH_URL)
		}
	}
	fmt.Printf("##### Admin #####\nusers:\t%s\n\n", ADMIN_USERS)
	fmt.Printf("##### Paths #####\nsite:\t%s\ndata:\t%s\nlogs:\t%s\nlocal_paths:\t%s\n\n", PATH_SITE, PATH_DATA, PATH_LOGS, PATH_LOCAL)
	if SSL {
		fmt.Printf("##### SSL enabled #####\n")
		fmt.Printf("##### SSL key:\t%s\n##### SSL cert:\t%s\n\n", SSL_KEY, SSL_CERT)
	} else {
		fmt.Printf("##### SSL disabled #####\n\n")
	}
	fmt.Printf("##### Mongodb #####\nhost(s):\t%s\ndatabase:\t%s\n\n", MONGODB_HOSTS, MONGODB_DATABASE)
	fmt.Printf("##### Address #####\nip:\t%s\nport:\t%s\n\n", API_IP, API_PORT)
	if LOG_PERF {
		fmt.Printf("##### PerfLog enabled #####\n\n")
	}
	if LOG_ROTATE {
		fmt.Printf("##### Log rotation enabled #####\n\n")
	} else {
		fmt.Printf("##### Log rotation disabled #####\n\n")
	}
}
