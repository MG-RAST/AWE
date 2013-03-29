package conf

import (
	"flag"
	"fmt"
	"github.com/jaredwilkening/goconfig/config"
	"os"
)

// Setup conf variables
var (
	//Reload
	RELOAD  = ""
	RECOVER = false

	BasePriority = 1

	// Config File
	CONFIG_FILE = ""

	// AWE 
	SITE_PORT = 8081
	API_PORT  = 8001

	// SSL
	SSL_ENABLED   = false
	SSL_KEY_FILE  = ""
	SSL_CERT_FILE = ""

	// Anonymous-Access-Control 
	ANON_WRITE      = false
	ANON_READ       = true
	ANON_CREATEUSER = false

	// Auth
	AUTH_TYPE               = "" //globus, oauth, basic
	GLOBUS_TOKEN_URL        = ""
	GLOBUS_PROFILE_URL      = ""
	OAUTH_REQUEST_TOKEN_URL = ""
	OAUTH_AUTH_TOKEN_URL    = ""
	OAUTH_ACCESS_TOKEN_URL  = ""

	// Admin
	ADMIN_EMAIL = ""
	SECRET_KEY  = ""

	// Directories
	DATA_PATH = ""
	SITE_PATH = ""
	LOGS_PATH = ""

	// Mongodb 
	MONGODB = ""

	//debug log level
	DEBUG_LEVEL = 0

	//[server] options
	//whether perf log including workunit info.
	PERF_LOG_WORKUNIT = true
	//number of times that one workunit fails before the workunit considered suspend
	MAX_WORK_FAILURE = 3
	//number of times that one clinet consecutively fails running workunits before the clinet considered suspend
	MAX_CLIENT_FAILURE = 5
	//big data threshold
	BIG_DATA_SIZE int64 = 1048576 * 1024
	//default index type used for intermediate data
	DEFAULT_INDEX = "chunkrecord"

	//[client]
	TOTAL_WORKER   = 1
	WORK_PATH      = ""
	APP_PATH       = ""
	SERVER_URL     = "http://localhost:8001"
	CLIENT_NAME    = "default"
	CLIENT_GROUP   = "default"
	CLIENT_PROFILE = ""
	WORKER_OVERLAP = false
	PRINT_APP_MSG  = false

	//tag
	INIT_SUCCESS = true
)

func init() {
	flag.StringVar(&CONFIG_FILE, "conf", "", "path to config file")
	flag.StringVar(&RELOAD, "reload", "", "path or url to awe job data. WARNING this will drop all current jobs.")
	flag.BoolVar(&RECOVER, "recover", false, "path to awe job data.")
	flag.StringVar(&CLIENT_PROFILE, "profile", "", "path to awe client profile.")
	flag.IntVar(&DEBUG_LEVEL, "debug", 0, "debug level: 0-3")
	flag.Parse()

	//	fmt.Printf("in conf.init(), flag=%v", flag)

	if len(CONFIG_FILE) == 0 {
		fmt.Fprintf(os.Stderr, "ERROR: conf file not specified\n")
		INIT_SUCCESS = false
		return
	}

	c, err := config.ReadDefault(CONFIG_FILE)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: error reading conf file: %v\n", err)
		INIT_SUCCESS = false
		return
	}

	// Ports
	SITE_PORT, _ = c.Int("Ports", "site-port")
	API_PORT, _ = c.Int("Ports", "api-port")

	// SSL
	SSL_ENABLED, _ = c.Bool("SSL", "enable")
	if SSL_ENABLED {
		SSL_KEY_FILE, _ = c.String("SSL", "key")
		SSL_CERT_FILE, _ = c.String("SSL", "cert")
	}

	// Access-Control 
	ANON_WRITE, _ = c.Bool("Anonymous", "write")
	ANON_READ, _ = c.Bool("Anonymous", "read")
	ANON_CREATEUSER, _ = c.Bool("Anonymous", "create-user")

	// Auth
	AUTH_TYPE, _ = c.String("Auth", "type")
	switch AUTH_TYPE {
	case "globus":
		GLOBUS_TOKEN_URL, _ = c.String("Auth", "globus_token_url")
		GLOBUS_PROFILE_URL, _ = c.String("Auth", "globus_profile_url")
	case "oauth":
		OAUTH_REQUEST_TOKEN_URL, _ = c.String("Auth", "oauth_request_token_url")
		OAUTH_AUTH_TOKEN_URL, _ = c.String("Auth", "oauth_auth_token_url")
		OAUTH_ACCESS_TOKEN_URL, _ = c.String("Auth", "oauth_access_token_url")
	case "basic":
		// nothing yet
	}

	// Admin
	ADMIN_EMAIL, _ = c.String("Admin", "email")
	SECRET_KEY, _ = c.String("Admin", "secretkey")

	// Directories
	SITE_PATH, _ = c.String("Directories", "site")
	DATA_PATH, _ = c.String("Directories", "data")
	LOGS_PATH, _ = c.String("Directories", "logs")

	// Mongodb
	MONGODB, _ = c.String("Mongodb", "hosts")

	// Server options
	if perf_log_workunit, err := c.Bool("Server", "perf_log_workunit"); err == nil {
		PERF_LOG_WORKUNIT = perf_log_workunit
	}
	if big_data_size, err := c.Int("Server", "big_data_size"); err == nil {
		BIG_DATA_SIZE = int64(big_data_size)
	}
	/*
		if default_index, err := c.String("Server", "default_index"); err == nil {
			DEFAULT_INDEX = default_index
		}
	*/

	// Client
	WORK_PATH, _ = c.String("Client", "workpath")
	APP_PATH, _ = c.String("Client", "app_path")
	SERVER_URL, _ = c.String("Client", "serverurl")
	if clientname, err := c.String("Client", "name"); err == nil {
		CLIENT_NAME = clientname
	}
	if clientgroup, err := c.String("Client", "group"); err == nil {
		CLIENT_GROUP = clientgroup
	}
	if clientprofile, err := c.String("Client", "clientprofile"); err == nil {
		CLIENT_PROFILE = clientprofile
	}
	if print_app_msg, err := c.Bool("Client", "print_app_msg"); err == nil {
		PRINT_APP_MSG = print_app_msg
	}
	if worker_overlap, err := c.Bool("Client", "worker_overlap"); err == nil {
		WORKER_OVERLAP = worker_overlap
	}
}

func Print() {
	fmt.Printf("##### Admin #####\nemail:\t%s\nsecretkey:\t%s\n\n", ADMIN_EMAIL, SECRET_KEY)
	fmt.Printf("####### Anonymous ######\nread:\t%t\nwrite:\t%t\ncreate-user:\t%t\n\n", ANON_READ, ANON_WRITE, ANON_CREATEUSER)
	if AUTH_TYPE == "basic" {
		fmt.Printf("##### Auth #####\ntype:\tbasic\n\n")
	} else if AUTH_TYPE == "globus" {
		fmt.Printf("##### Auth #####\ntype:\tglobus\ntoken_url:\t%s\nprofile_url:\t%s\n\n", GLOBUS_TOKEN_URL, GLOBUS_PROFILE_URL)
	}
	fmt.Printf("##### Directories #####\nsite:\t%s\ndata:\t%s\nlogs:\t%s\n\n", SITE_PATH, DATA_PATH, LOGS_PATH)
	if SSL_ENABLED {
		fmt.Printf("##### SSL #####\nenabled:\t%t\nkey:\t%s\ncert:\t%s\n\n", SSL_ENABLED, SSL_KEY_FILE, SSL_CERT_FILE)
	} else {
		fmt.Printf("##### SSL #####\nenabled:\t%t\n\n", SSL_ENABLED)
	}
	fmt.Printf("##### Mongodb #####\nhost(s):\t%s\n\n", MONGODB)
	fmt.Printf("##### Ports #####\nsite:\t%d\napi:\t%d\n\n", SITE_PORT, API_PORT)
}

func PrintClientCfg() {
	fmt.Printf("###AWE client running###\n")
	fmt.Printf("work_path=%s\n", WORK_PATH)
	fmt.Printf("server_url=%s\n", SERVER_URL)
	fmt.Printf("print_app_msg=%t\n", PRINT_APP_MSG)
}

func PrintClientUsage() {
	fmt.Printf("Usage: awe-client -conf </path/to/cfg> [-debug 0-3]\n")
}

func PrintServerUsage() {
	fmt.Printf("Usage: awe-server -conf </path/to/cfg> [-recover] [debug 0-3]\n")
}
