package conf

import (
	"flag"
	"fmt"
	"github.com/MG-RAST/golib/goconfig/config"
	"os"
	"strings"
	"time"
)

// Setup conf variables
var (
	VERSION      = "0.9.3"
	SHOW_VERSION = false

	//Reload
	RELOAD   = ""
	RECOVER  = false
	DEV_MODE = false

	BasePriority = 1

	// Config File
	CONFIG_FILE = ""

	// AWE server port
	SITE_PORT = 8081
	API_PORT  = 8001
	// AWE server external address
	SITE_URL = ""
	API_URL  = ""

	// AWE proxy port
	P_SITE_PORT = 8082
	P_API_PORT  = 8002

	// SSL
	SSL_ENABLED   = false
	SSL_KEY_FILE  = ""
	SSL_CERT_FILE = ""

	// Anonymous-Access-Control
	ANON_WRITE     = true
	ANON_READ      = true
	ANON_DELETE    = true
	ANON_CG_WRITE  = false
	ANON_CG_READ   = false
	ANON_CG_DELETE = false

	// Auth
	BASIC_AUTH         = true
	GLOBUS_OAUTH       = false
	MGRAST_OAUTH       = false
	GLOBUS_TOKEN_URL   = ""
	GLOBUS_PROFILE_URL = ""
	MGRAST_OAUTH_URL   = ""
	CLIENT_AUTH_REQ    = false
	CLIENT_GROUP_TOKEN = ""

	// Admin
	ADMIN_EMAIL = ""
	SECRET_KEY  = ""

	// Directories
	DATA_PATH     = ""
	SITE_PATH     = ""
	LOGS_PATH     = ""
	AWF_PATH      = ""
	PID_FILE_PATH = ""

	// Mongodb
	MONGODB_HOST     = ""
	MONGODB_DATABASE = "AWEDB"
	MONGODB_USER     = ""
	MONGODB_PASSWD   = ""
	DB_COLL_JOBS     = "Jobs"
	DB_COLL_PERF     = "Perf"
	DB_COLL_CGS      = "ClientGroups"
	DB_COLL_USERS    = "Users"

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
	//default chunk size, consistent with shock
	DEFAULT_CHUNK_SIZE int64 = 1048576 * 1
	//Shock_TimeOut
	SHOCK_TIMEOUT = 30 * time.Second
	//Default page size
	DEFAULT_PAGE_SIZE = 25

	GOMAXPROCS = 0

	//APP
	APP_REGISTRY_URL = ""

	//[client]
	TOTAL_WORKER                  = 1
	WORK_PATH                     = ""
	APP_PATH                      = ""
	SUPPORTED_APPS                = ""
	PRE_WORK_SCRIPT               = ""
	PRE_WORK_SCRIPT_ARGS          = []string{}
	SERVER_URL                    = "http://localhost:8001"
	OPENSTACK_METADATA_URL        = "" //openstack metadata url, e.g. "http://169.254.169.254/2009-04-04/meta-data"
	INSTANCE_METADATA_TIMEOUT     = 5 * time.Second
	CLIENT_NAME                   = "default"
	CLIENT_GROUP                  = "default"
	CLIENT_DOMAIN                 = "default"
	WORKER_OVERLAP                = false
	PRINT_APP_MSG                 = true
	AUTO_CLEAN_DIR                = true
	CLIEN_DIR_DELAY_FAIL          = 30 * time.Minute //clean failed workunit dir after 30 minutes
	CLIEN_DIR_DELAY_DONE          = 1 * time.Minute  // clean done workunit dir after 1 minute
	STDOUT_FILENAME               = "awe_stdout.txt"
	STDERR_FILENAME               = "awe_stderr.txt"
	WORKNOTES_FILENAME            = "awe_worknotes.txt"
	MEM_CHECK_INTERVAL            = 10 * time.Second
	DOCKER_WORK_DIR               = "/workdir/"
	SHOCK_DOCKER_IMAGE_REPOSITORY = "http://shock.metagenomics.anl.gov"
	KB_AUTH_TOKEN                 = "KB_AUTH_TOKEN"
	CACHE_ENABLED                 = false
	//tag
	INIT_SUCCESS = true

	CGROUP_MEMORY_DOCKER_DIR = ""
	//const
	ALL_APP = "*"

	Admin_Users = make(map[string]bool)
)

func init() {
	flag.StringVar(&CONFIG_FILE, "conf", "", "path to config file")
	flag.StringVar(&RELOAD, "reload", "", "path or url to awe job data. WARNING this will drop all current jobs.")
	flag.BoolVar(&RECOVER, "recover", false, "path to awe job data.")
	flag.BoolVar(&SHOW_VERSION, "version", false, "show version.")
	flag.IntVar(&DEBUG_LEVEL, "debug", -1, "debug level: 0-3")
	flag.BoolVar(&DEV_MODE, "dev", false, "dev or demo mode, print some msgs on screen")
	flag.StringVar(&CGROUP_MEMORY_DOCKER_DIR, "cgroup_memory_docker_dir", "/sys/fs/cgroup/memory/docker/", "path to cgroup directory for docker")
	flag.StringVar(&SERVER_URL, "server_url", "", "URL of AWE server, including API port")
	flag.Parse()

	//	fmt.Printf("in conf.init(), flag=%v", flag)

	if SHOW_VERSION {
		PrintVersionMsg()
		os.Exit(0)
	}

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

	// URLs
	SITE_URL, _ = c.String("External", "site-url")
	API_URL, _ = c.String("External", "api-url")

	// SSL
	SSL_ENABLED, _ = c.Bool("SSL", "enable")
	if SSL_ENABLED {
		SSL_KEY_FILE, _ = c.String("SSL", "key")
		SSL_CERT_FILE, _ = c.String("SSL", "cert")
	}

	// Access-Control
	ANON_WRITE, _ = c.Bool("Anonymous", "write")
	ANON_READ, _ = c.Bool("Anonymous", "read")
	ANON_DELETE, _ = c.Bool("Anonymous", "delete")
	ANON_CG_WRITE, _ = c.Bool("Anonymous", "cg_write")
	ANON_CG_READ, _ = c.Bool("Anonymous", "cg_read")
	ANON_CG_DELETE, _ = c.Bool("Anonymous", "cg_delete")

	// Auth
	if basic_auth, err := c.Bool("Auth", "basic"); err == nil {
		BASIC_AUTH = basic_auth
	}
	GLOBUS_TOKEN_URL, _ = c.String("Auth", "globus_token_url")
	GLOBUS_PROFILE_URL, _ = c.String("Auth", "globus_profile_url")
	MGRAST_OAUTH_URL, _ = c.String("Auth", "mgrast_oauth_url")

	if GLOBUS_TOKEN_URL != "" && GLOBUS_PROFILE_URL != "" {
		GLOBUS_OAUTH = true
	}
	if MGRAST_OAUTH_URL != "" {
		MGRAST_OAUTH = true
	}

	CLIENT_AUTH_REQ, _ = c.Bool("Auth", "client_auth_required")

	// Admin
	if admin_users, err := c.String("Admin", "users"); err == nil {
		for _, name := range strings.Split(admin_users, ",") {
			Admin_Users[name] = true
		}
	}

	ADMIN_EMAIL, _ = c.String("Admin", "email")
	SECRET_KEY, _ = c.String("Admin", "secretkey")

	// Directories
	SITE_PATH, _ = c.String("Directories", "site")
	DATA_PATH, _ = c.String("Directories", "data")
	LOGS_PATH, _ = c.String("Directories", "logs")
	AWF_PATH, _ = c.String("Directories", "awf")

	// Paths
	PID_FILE_PATH, _ = c.String("Paths", "pidfile")
	if PID_FILE_PATH == "" {
		PID_FILE_PATH = DATA_PATH + "/pidfile"
	}

	// Mongodb
	MONGODB_HOST, _ = c.String("Mongodb", "hosts")
	MONGODB_DATABASE, _ = c.String("Mongodb", "database")
	MONGODB_USER, _ = c.String("Mongodb", "user")
	MONGODB_PASSWD, _ = c.String("Mongodb", "password")
	if MONGODB_DATABASE == "" {
		MONGODB_DATABASE = "AWEDB"
	}

	// APP
	APP_REGISTRY_URL, _ = c.String("App", "app_registry_url")

	// Server options
	if perf_log_workunit, err := c.Bool("Server", "perf_log_workunit"); err == nil {
		PERF_LOG_WORKUNIT = perf_log_workunit
	}
	if big_data_size, err := c.Int("Server", "big_data_size"); err == nil {
		BIG_DATA_SIZE = int64(big_data_size)
	}
	if max_work_failure, err := c.Int("Server", "max_work_failure"); err == nil {
		MAX_WORK_FAILURE = max_work_failure
	}
	if max_client_failure, err := c.Int("Server", "max_client_failure"); err == nil {
		MAX_CLIENT_FAILURE = max_client_failure
	}
	if go_max_procs, err := c.Int("Server", "go_max_procs"); err == nil {
		GOMAXPROCS = go_max_procs
	}

	// Client
	WORK_PATH, _ = c.String("Client", "workpath")
	APP_PATH, _ = c.String("Client", "app_path")
	if SERVER_URL == "" {
		SERVER_URL, _ = c.String("Client", "serverurl")
	}
	OPENSTACK_METADATA_URL, _ = c.String("Client", "openstack_metadata_url")
	SUPPORTED_APPS, _ = c.String("Client", "supported_apps")
	if clientname, err := c.String("Client", "name"); err == nil {
		CLIENT_NAME = clientname
	}

	if pre_work_script, err := c.String("Client", "pre_work_script"); err == nil {
		PRE_WORK_SCRIPT = pre_work_script
	}
	if pre_work_script_args, err := c.String("Client", "pre_work_script_args"); err == nil {
		PRE_WORK_SCRIPT_ARGS = strings.Split(pre_work_script_args, ",")
	}

	if CLIENT_NAME == "" || CLIENT_NAME == "default" || CLIENT_NAME == "hostname" {
		hostname, err := os.Hostname()
		if err == nil {
			CLIENT_NAME = hostname
		}
	}

	if clientgroup, err := c.String("Client", "group"); err == nil {
		CLIENT_GROUP = clientgroup
	}
	if clientdomain, err := c.String("Client", "domain"); err == nil {
		CLIENT_DOMAIN = clientdomain
	}
	if print_app_msg, err := c.Bool("Client", "print_app_msg"); err == nil {
		PRINT_APP_MSG = print_app_msg
	}
	if worker_overlap, err := c.Bool("Client", "worker_overlap"); err == nil {
		WORKER_OVERLAP = worker_overlap
	}
	if auto_clean_dir, err := c.Bool("Client", "auto_clean_dir"); err == nil {
		AUTO_CLEAN_DIR = auto_clean_dir
	}
	if cache_enabled, err := c.Bool("Client", "cache_enabled"); err == nil {
		CACHE_ENABLED = cache_enabled
	}

	CLIENT_GROUP_TOKEN, _ = c.String("Client", "clientgroup_token")

	//Proxy
	P_SITE_PORT, _ = c.Int("Proxy", "p-site-port")
	P_API_PORT, _ = c.Int("Proxy", "p-api-port")

	//Args
	if DEBUG_LEVEL == -1 { //if no debug level set in cmd line args, find values in config file.
		if dlevel, err := c.Int("Args", "debuglevel"); err == nil {
			DEBUG_LEVEL = dlevel
		} else {
			DEBUG_LEVEL = 0
		}
	}
}

func Print(service string) {
	fmt.Printf("##### Admin #####\nemail:\t%s\nsecretkey:\t%s\n\n", ADMIN_EMAIL, SECRET_KEY)
	fmt.Printf("####### Anonymous ######\nread:\t%t\nwrite:\t%t\ndelete:\t%t\n", ANON_READ, ANON_WRITE, ANON_DELETE)
	fmt.Printf("clientgroup read:\t%t\nclientgroup write:\t%t\nclientgroup delete:\t%t\n\n", ANON_CG_READ, ANON_CG_WRITE, ANON_CG_DELETE)
	fmt.Printf("##### Auth #####\n")
	if BASIC_AUTH {
		fmt.Printf("basic_auth:\ttrue\n")
	}
	if GLOBUS_TOKEN_URL != "" && GLOBUS_PROFILE_URL != "" {
		fmt.Printf("globus_token_url:\t%s\nglobus_profile_url:\t%s\n", GLOBUS_TOKEN_URL, GLOBUS_PROFILE_URL)
	}
	if MGRAST_OAUTH_URL != "" {
		fmt.Printf("mgrast_oauth_url:\t%s\n", MGRAST_OAUTH_URL)
	}
	if len(Admin_Users) > 0 {
		fmt.Printf("admin_auth:\ttrue\nadmin_users:\t")
		for name, _ := range Admin_Users {
			fmt.Printf("%s ", name)
		}
		fmt.Printf("\n")
	}
	fmt.Printf("\n")

	fmt.Printf("##### Directories #####\nsite:\t%s\ndata:\t%s\nlogs:\t%s\n", SITE_PATH, DATA_PATH, LOGS_PATH)
	if service == "server" {
		fmt.Printf("awf:\t%s\n", AWF_PATH)
	}
	fmt.Println()

	if SSL_ENABLED {
		fmt.Printf("##### SSL #####\nenabled:\t%t\nkey:\t%s\ncert:\t%s\n\n", SSL_ENABLED, SSL_KEY_FILE, SSL_CERT_FILE)
	} else {
		fmt.Printf("##### SSL #####\nenabled:\t%t\n\n", SSL_ENABLED)
	}

	if service == "server" {
		fmt.Printf("##### Mongodb #####\nhost(s):\t%s\ndatabase:\t%s\n\n", MONGODB_HOST, MONGODB_DATABASE)
	}
	fmt.Println()
	if service == "server" {
		fmt.Printf("##### Ports #####\nsite:\t%d\napi:\t%d\n\n", SITE_PORT, API_PORT)
	} else if service == "proxy" {
		fmt.Printf("##### Ports #####\nsite:\t%d\napi:\t%d\n\n", P_SITE_PORT, P_API_PORT)
	}
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
	fmt.Printf("Usage: awe-server -conf </path/to/cfg> [-dev] [-recover] [debug 0-3]\n")
}

func PrintProxyUsage() {
	fmt.Printf("Usage: awe-proxy -conf </path/to/cfg> [debug 0-3]\n")
}

func PrintVersionMsg() {
	fmt.Printf("AWE version: %s\n", VERSION)
}
