package conf

import (
	"errors"
	"flag"
	"fmt"
	"github.com/MG-RAST/golib/goconfig/config"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

const VERSION string = "0.9.52testing2"

var GIT_COMMIT_HASH string // use -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH <value>"
const BasePriority int = 1

const DB_COLL_JOBS string = "Jobs"
const DB_COLL_PERF string = "Perf"
const DB_COLL_CGS string = "ClientGroups"
const DB_COLL_USERS string = "Users"

//prefix for site login
const LOGIN_PREFIX string = "go4711"

//default index type used for intermediate data
const DEFAULT_INDEX string = "chunkrecord"

//default chunk size, consistent with shock
const DEFAULT_CHUNK_SIZE int64 = 1048576 * 1

//Shock_TimeOut
const SHOCK_TIMEOUT time.Duration = 30 * time.Second

//Default page size
const DEFAULT_PAGE_SIZE int = 25

const INSTANCE_METADATA_TIMEOUT time.Duration = 5 * time.Second

const CLIEN_DIR_DELAY_FAIL time.Duration = 30 * time.Minute //clean failed workunit dir after 30 minutes
const CLIEN_DIR_DELAY_DONE time.Duration = 1 * time.Minute  // clean done workunit dir after 1 minute
const STDOUT_FILENAME string = "awe_stdout.txt"
const STDERR_FILENAME string = "awe_stderr.txt"
const WORKNOTES_FILENAME string = "awe_worknotes.txt"

const ALL_APP string = "*"

var WORKUNIT_LOGS = [3]string{"stdout", "stderr", "worknotes"}
var LOG_OUTPUTS = [3]string{"file", "console", "both"}

var checkExpire = regexp.MustCompile(`^(\d+)(M|H|D)$`)

// set defaults in function "getConfiguration" below !!!!!
var (
	SHOW_VERSION bool
	TITLE        string

	// Reload
	RELOAD      string
	RECOVER     bool
	RECOVER_MAX int

	// AWE server port
	SITE_PORT int
	API_PORT  int
	// AWE server external address
	SITE_URL string
	API_URL  string

	// AWE proxy port
	P_SITE_PORT int
	P_API_PORT  int

	// SSL
	SSL_ENABLED   bool
	SSL_KEY_FILE  string
	SSL_CERT_FILE string

	// Anonymous-Access-Control
	ANON_WRITE     bool
	ANON_READ      bool
	ANON_DELETE    bool
	ANON_CG_WRITE  bool
	ANON_CG_READ   bool
	ANON_CG_DELETE bool

	// Auth
	BASIC_AUTH         bool
	GLOBUS_TOKEN_URL   string
	GLOBUS_PROFILE_URL string
	OAUTH_URL_STR      string
	OAUTH_BEARER_STR   string
	SITE_LOGIN_URL     string
	CLIENT_AUTH_REQ    bool
	CLIENT_GROUP_TOKEN string

	// Admin
	ADMIN_EMAIL     string
	ADMIN_USERS_VAR string

	// Directories
	DATA_PATH     string
	SITE_PATH     string
	LOGS_PATH     string
	AWF_PATH      string
	PID_FILE_PATH string

	// Mongodb
	MONGODB_HOST     string
	MONGODB_DATABASE string
	MONGODB_USER     string
	MONGODB_PASSWD   string
	MONGODB_TIMEOUT  int

	// Server
	COREQ_LENGTH       int
	EXPIRE_WAIT        int
	GLOBAL_EXPIRE      string
	PIPELINE_EXPIRE    string
	PERF_LOG_WORKUNIT  bool
	MAX_WORK_FAILURE   int
	MAX_CLIENT_FAILURE int
	GOMAXPROCS         int

	// Client
	WORK_PATH                   string
	APP_PATH                    string
	SUPPORTED_APPS              string
	PRE_WORK_SCRIPT             string
	PRE_WORK_SCRIPT_ARGS_STRING string
	PRE_WORK_SCRIPT_ARGS        = []string{}
	METADATA                    string

	SERVER_URL     string
	CLIENT_NAME    string
	CLIENT_HOST    string
	CLIENT_GROUP   string
	CLIENT_DOMAIN  string
	WORKER_OVERLAP bool
	PRINT_APP_MSG  bool
	AUTO_CLEAN_DIR bool
	NO_SYMLINK     bool
	CACHE_ENABLED  bool

	CWL_TOOL  string
	CWL_JOB   string
	SHOCK_URL string

	// Docker
	USE_DOCKER                    string
	DOCKER_BINARY                 string
	USE_APP_DEFS                  string
	CGROUP_MEMORY_DOCKER_DIR      string
	MEM_CHECK_INTERVAL            = 0 * time.Second //in milliseconds e.g. use 10 * time.Second
	MEM_CHECK_INTERVAL_SECONDS    int
	APP_REGISTRY_URL              string
	DOCKER_SOCKET                 string
	DOCKER_WORK_DIR               string
	DOCKER_WORKUNIT_PREDATA_DIR   string
	SHOCK_DOCKER_IMAGE_REPOSITORY string

	// Other
	ERROR_LENGTH         int
	DEV_MODE             bool
	DEBUG_LEVEL          int
	CONFIG_FILE          string
	LOG_OUTPUT           string
	PRINT_HELP           bool // full usage
	SHOW_HELP            bool // simple usage
	SHOW_GIT_COMMIT_HASH bool

	// used to track changes in data structures
	VERSIONS = make(map[string]int)

	// used to track expiration for different pipelines
	PIPELINE_EXPIRE_MAP = make(map[string]string)

	// used to track admin users
	AdminUsers = []string{}

	// used for login
	LOGIN_RESOURCES = make(map[string]LoginResource)
	LOGIN_DEFAULT   string

	// used to map bearer token to oauth url
	AUTH_OAUTH    = make(map[string]string)
	HAS_OAUTH     bool
	OAUTH_DEFAULT string // first value in OAUTH_URL_STR

	// internal config control
	FAKE_VAR = false

	//
	ARGS []string
)

type LoginResource struct {
	Icon      string `json:"icon"`
	Prefix    string `json:"prefix"`
	Keyword   string `json:"keyword"`
	Url       string `json:"url"`
	UseHeader bool   `json:"useHeader"`
	Bearer    string `json:"bearer"`
}

func get_my_config_string(c *config.Config, f *flag.FlagSet, val *Config_value_string) {
	//overwrite variable if defined in config file
	if c != nil {
		getDefinedValueString(c, val.Section, val.Key, val.Target)
	}
	//overwrite variable if defined on command line (default values are overwritten by config file)
	f.StringVar(val.Target, val.Key, *val.Target, val.Descr_short)
}

func get_my_config_int(c *config.Config, f *flag.FlagSet, val *Config_value_int) {
	//overwrite variable if defined in config file
	if c != nil {
		getDefinedValueInt(c, val.Section, val.Key, val.Target)
	}
	//overwrite variable if defined on command line (default values are overwritten by config file)
	f.IntVar(val.Target, val.Key, *val.Target, val.Descr_short)
}

func get_my_config_bool(c *config.Config, f *flag.FlagSet, val *Config_value_bool) {
	//overwrite variable if defined in config file
	if c != nil {
		getDefinedValueBool(c, val.Section, val.Key, val.Target)
	}
	//overwrite variable if defined on command line (default values are overwritten by config file)
	f.BoolVar(val.Target, val.Key, *val.Target, val.Descr_short)
}

// wolfgang: I started to change it such that config values are only written when defined in the config file
func getConfiguration(c *config.Config, mode string) (c_store *Config_store) {
	c_store = NewCS(c)

	if mode == "server" {
		// Ports
		c_store.AddInt(&SITE_PORT, 8081, "Ports", "site-port", "Internal port to run AWE Monitor on", "")
		c_store.AddInt(&API_PORT, 8001, "Ports", "api-port", "Internal port for API", "")

		// External
		c_store.AddString(&SITE_URL, "http://localhost:8081", "External", "site-url", "External URL of AWE monitor, including port", "")
		c_store.AddString(&API_URL, "http://localhost:8001", "External", "api-url", "External API URL of AWE server, including port", "")

		// SSL
		c_store.AddBool(&SSL_ENABLED, false, "SSL", "enable", "", "")

		// if SSL_ENABLED { // TODO this needs to be checked !
		c_store.AddString(&SSL_KEY_FILE, "", "SSL", "key", "", "")
		c_store.AddString(&SSL_CERT_FILE, "", "SSL", "cert", "", "")

		// Access-Control
		c_store.AddBool(&ANON_WRITE, true, "Anonymous", "write", "", "")
		c_store.AddBool(&ANON_READ, true, "Anonymous", "read", "", "")
		c_store.AddBool(&ANON_DELETE, true, "Anonymous", "delete", "", "")
		c_store.AddBool(&ANON_CG_WRITE, false, "Anonymous", "cg_write", "", "")
		c_store.AddBool(&ANON_CG_READ, false, "Anonymous", "cg_read", "", "")
		c_store.AddBool(&ANON_CG_DELETE, false, "Anonymous", "cg_delete", "", "")

		// Auth
		c_store.AddBool(&BASIC_AUTH, false, "Auth", "basic", "", "")
		c_store.AddString(&GLOBUS_TOKEN_URL, "", "Auth", "globus_token_url", "", "")
		c_store.AddString(&GLOBUS_PROFILE_URL, "", "Auth", "globus_profile_url", "", "")
		c_store.AddString(&OAUTH_URL_STR, "", "Auth", "oauth_urls", "", "")
		c_store.AddString(&OAUTH_BEARER_STR, "", "Auth", "oauth_bearers", "", "")
		c_store.AddString(&SITE_LOGIN_URL, "", "Auth", "login_url", "", "")
		c_store.AddBool(&CLIENT_AUTH_REQ, false, "Auth", "client_auth_required", "", "")

		// Admin
		c_store.AddString(&ADMIN_USERS_VAR, "", "Admin", "users", "", "")
		c_store.AddString(&ADMIN_EMAIL, "", "Admin", "email", "", "")

		// Directories
		c_store.AddString(&SITE_PATH, os.Getenv("GOPATH")+"/src/github.com/MG-RAST/AWE/site", "Directories", "site", "the path to the website", "")
		c_store.AddString(&AWF_PATH, "", "Directories", "awf", "", "")
	}

	if mode == "server" || mode == "worker" {
		// Directories
		c_store.AddString(&DATA_PATH, "/mnt/data/awe/data", "Directories", "data", "a file path for storing system related data (job script, cached data, etc)", "")
		c_store.AddString(&LOGS_PATH, "/mnt/data/awe/logs", "Directories", "logs", "a path for storing logs", "")

		// Paths
		c_store.AddString(&PID_FILE_PATH, "", "Paths", "pidfile", "", "")
	}

	if mode == "server" {
		// Mongodb
		c_store.AddString(&MONGODB_HOST, "localhost", "Mongodb", "hosts", "", "")
		c_store.AddString(&MONGODB_DATABASE, "AWEDB", "Mongodb", "database", "", "")
		c_store.AddString(&MONGODB_USER, "", "Mongodb", "user", "", "")
		c_store.AddString(&MONGODB_PASSWD, "", "Mongodb", "password", "", "")
		c_store.AddInt(&MONGODB_TIMEOUT, 1200, "Mongodb", "timeout", "", "")

		// Server
		c_store.AddString(&TITLE, "AWE Server", "Server", "title", "", "")
		c_store.AddInt(&COREQ_LENGTH, 100, "Server", "coreq_length", "length of checkout request queue", "")
		c_store.AddInt(&EXPIRE_WAIT, 60, "Server", "expire_wait", "wait time for expiration reaper in minutes", "")
		c_store.AddString(&GLOBAL_EXPIRE, "", "Server", "global_expire", "default number and unit of time after job completion before it expires", "")
		c_store.AddString(&PIPELINE_EXPIRE, "", "Server", "pipeline_expire", "comma seperated list of pipeline_name=expire_days_unit, overrides global_expire", "")
		c_store.AddBool(&PERF_LOG_WORKUNIT, true, "Server", "perf_log_workunit", "collecting performance log per workunit", "")
		c_store.AddInt(&MAX_WORK_FAILURE, 3, "Server", "max_work_failure", "number of times that one workunit fails before the workunit considered suspend", "")
		c_store.AddInt(&MAX_CLIENT_FAILURE, 5, "Server", "max_client_failure", "number of times that one client consecutively fails running workunits before the client considered suspend", "")
		c_store.AddInt(&GOMAXPROCS, 0, "Server", "go_max_procs", "", "")
		c_store.AddString(&RELOAD, "", "Server", "reload", "path or url to awe job data. WARNING this will drop all current jobs", "")
		c_store.AddBool(&RECOVER, false, "Server", "recover", "load unfinished jobs from mongodb on startup", "")
		c_store.AddInt(&RECOVER_MAX, 0, "Server", "recover_max", "max number of jobs to recover, default (0) means recover all", "")
	}

	if mode == "worker" || mode == "submitter" {
		c_store.AddString(&SERVER_URL, "http://localhost:8001", "Client", "serverurl", "URL of AWE server, including API port", "")
		c_store.AddString(&CWL_TOOL, "", "Client", "cwl_tool", "CWL CommandLineTool file", "")
		c_store.AddString(&CWL_JOB, "", "Client", "cwl_job", "CWL job file", "")
	}

	if mode == "worker" || mode == "submitter" {
		c_store.AddString(&SHOCK_URL, "http://localhost:8001", "Client", "shockurl", "URL of SHOCK server, including port number", "")
	}

	if mode == "worker" {
		// Client/worker

		c_store.AddString(&CLIENT_GROUP, "default", "Client", "group", "name of client group", "")
		c_store.AddString(&CLIENT_NAME, "default", "Client", "name", "default determines client name by openstack meta data", "")
		c_store.AddString(&CLIENT_HOST, "127.0.0.1", "Client", "host", "host or ip address", "host or ip address to help finding machines where the clients runs on")
		c_store.AddString(&CLIENT_DOMAIN, "default", "Client", "domain", "", "")
		c_store.AddString(&CLIENT_GROUP_TOKEN, "", "Client", "clientgroup_token", "", "")

		c_store.AddString(&SUPPORTED_APPS, "", "Client", "supported_apps", "list of suported apps, comma separated", "")
		c_store.AddString(&APP_PATH, "", "Client", "app_path", "the file path of supported app", "")

		c_store.AddString(&WORK_PATH, "/mnt/data/awe/work", "Client", "workpath", "the root dir for workunit working dirs", "")
		c_store.AddString(&METADATA, "", "Client", "metadata", "", "e.g. ec2, openstack...")

		c_store.AddString(&PRE_WORK_SCRIPT, "", "Client", "pre_work_script", "", "")
		c_store.AddString(&PRE_WORK_SCRIPT_ARGS_STRING, "", "Client", "pre_work_script_args", "", "")

		c_store.AddBool(&PRINT_APP_MSG, true, "Client", "print_app_msg", "collect stdout/stderr for apps", "")
		c_store.AddBool(&WORKER_OVERLAP, false, "Client", "worker_overlap", "overlap client side computation and data movement", "")
		c_store.AddBool(&AUTO_CLEAN_DIR, true, "Client", "auto_clean_dir", "delete workunit directory to save space after completion, turn of for debugging", "")
		c_store.AddBool(&CACHE_ENABLED, false, "Client", "cache_enabled", "", "")
		c_store.AddBool(&NO_SYMLINK, false, "Client", "no_symlink", "copy files from predata to work dir, default is to create symlink", "")

	}

	// Docker
	if mode == "server" || mode == "worker" {
		c_store.AddString(&USE_DOCKER, "yes", "Docker", "use_docker", "\"yes\", \"no\" or \"only\"", "yes: allow docker tasks, no: do not allow docker tasks, only: allow only docker tasks; if docker is not installed on the clients, choose \"no\"")
	}
	if mode == "worker" {
		c_store.AddString(&DOCKER_BINARY, "API", "Docker", "docker_binary", "docker binary to use, default is the docker API (API recommended)", "")
		c_store.AddInt(&MEM_CHECK_INTERVAL_SECONDS, 0, "Docker", "mem_check_interval_seconds", "memory check interval in seconds (kernel needs to support that)", "0 seconds means disabled")
		c_store.AddString(&CGROUP_MEMORY_DOCKER_DIR, "/sys/fs/cgroup/memory/docker/[ID]/memory.stat", "Docker", "cgroup_memory_docker_dir", "path to cgroup directory for docker", "")
		c_store.AddString(&DOCKER_SOCKET, "unix:///var/run/docker.sock", "Docker", "docker_socket", "docker socket path", "")
		c_store.AddString(&DOCKER_WORK_DIR, "/workdir/", "Docker", "docker_workpath", "work dir in docker container started by client", "")
		c_store.AddString(&DOCKER_WORKUNIT_PREDATA_DIR, "/db/", "Docker", "docker_data", "predata dir in docker container started by client", "")
		c_store.AddString(&SHOCK_DOCKER_IMAGE_REPOSITORY, "http://shock.metagenomics.anl.gov", "Docker", "image_url", "url of shock server hosting docker images", "")
	}
	if mode == "server" {
		c_store.AddString(&USE_APP_DEFS, "no", "Docker", "use_app_defs", "\"yes\", \"no\" or \"only\"", "yes: allow app defs, no: do not allow app defs, only: allow only app defs")
		c_store.AddString(&APP_REGISTRY_URL, "https://raw.githubusercontent.com/MG-RAST/Skyport/master/app_definitions/", "Docker", "app_registry_url", "URL for app defintions", "")
	}

	if mode == "server" || mode == "worker" {
		//Proxy
		c_store.AddInt(&P_SITE_PORT, 8082, "Proxy", "p-site-port", "", "")
		c_store.AddInt(&P_API_PORT, 8002, "Proxy", "p-api-port", "", "")

		//Other
		c_store.AddInt(&ERROR_LENGTH, 5000, "Other", "errorlength", "amount of App STDERR to save in Job.Error", "")
		c_store.AddBool(&DEV_MODE, false, "Other", "dev", "dev or demo mode, print some msgs on screen", "")

		c_store.AddString(&CONFIG_FILE, "", "Other", "conf", "path to config file", "")
		c_store.AddString(&LOG_OUTPUT, "console", "Other", "logoutput", "log output stream, one of: file, console, both", "")

	}
	c_store.AddInt(&DEBUG_LEVEL, 0, "Other", "debuglevel", "debug level: 0-3", "")
	c_store.AddBool(&SHOW_VERSION, false, "Other", "version", "show version", "")
	c_store.AddBool(&SHOW_GIT_COMMIT_HASH, false, "Other", "show_git_commit_hash", "", "")
	c_store.AddBool(&PRINT_HELP, false, "Other", "fullhelp", "show detailed usage without \"--\"-prefixes", "")
	c_store.AddBool(&SHOW_HELP, false, "Other", "help", "show usage", "")

	c_store.Parse()
	return
}

func Init_conf(mode string) (err error) {
	// the flag libary needs to parse after config file has been read, thus the conf file is parsed manually here
	CONFIG_FILE = ""

	for i, elem := range os.Args {
		if strings.HasPrefix(elem, "-conf") || strings.HasPrefix(elem, "--conf") {
			parts := strings.SplitN(elem, "=", 2)
			if len(parts) == 2 {
				CONFIG_FILE = parts[1]
			} else if i+1 < len(os.Args) {
				CONFIG_FILE = os.Args[i+1]
			} else {
				return errors.New("missing conf file in command options")
			}
		}
	}

	var c *config.Config = nil
	if CONFIG_FILE != "" {
		c, err = config.ReadDefault(CONFIG_FILE)
		if err != nil {
			return errors.New("error reading conf file: " + err.Error())
		}
	}
	c_store := getConfiguration(c, mode)

	// ####### at this point configuration variables are set ########

	ARGS = c_store.Fs.Args()

	if FAKE_VAR == false {
		return errors.New("config was not parsed")
	}
	if PRINT_HELP || SHOW_HELP {
		c_store.PrintHelp()
		os.Exit(0)
	}
	if SHOW_VERSION {
		PrintVersionMsg()
		os.Exit(0)
	}
	if SHOW_GIT_COMMIT_HASH {
		fmt.Fprintf(os.Stdout, "GIT_COMMIT_HASH=%s\n", GIT_COMMIT_HASH)
		os.Exit(0)
	}

	// configuration post processing
	if mode == "worker" {
		if CLIENT_NAME == "" || CLIENT_NAME == "default" || CLIENT_NAME == "hostname" {
			hostname, err := os.Hostname()
			if err == nil {
				CLIENT_NAME = hostname
			}
		}
		if MEM_CHECK_INTERVAL_SECONDS > 0 {
			MEM_CHECK_INTERVAL = time.Duration(MEM_CHECK_INTERVAL_SECONDS) * time.Second
			// TODO
		}
	}

	// parse OAuth settings if used
	if OAUTH_URL_STR != "" && OAUTH_BEARER_STR != "" {
		ou := strings.Split(OAUTH_URL_STR, ",")
		ob := strings.Split(OAUTH_BEARER_STR, ",")
		if len(ou) != len(ob) {
			return errors.New("number of items in oauth_urls and oauth_bearers are not the same")
		}
		// first entry is default for "oauth" bearer token and login
		OAUTH_DEFAULT = ou[0]
		LOGIN_DEFAULT = strings.ToUpper(ob[0])
		HAS_OAUTH = true
		// process all entries
		for i := range ob {
			AUTH_OAUTH[ob[i]] = ou[i]
			NAME := strings.ToUpper(ob[i])
			LOGIN_RESOURCES[NAME] = buildLoginResource(NAME, ob[i])
		}
	}
	// globus has some hard-coded info
	if GLOBUS_TOKEN_URL != "" && GLOBUS_PROFILE_URL != "" {
		NAME := "KBase"
		HAS_OAUTH = true
		LOGIN_DEFAULT = NAME
		LOGIN_RESOURCES[NAME] = buildLoginResource(NAME, "globus")
		// use globus as defualt if no oauth_url / oauth_bearer
		if OAUTH_DEFAULT == "" {
			OAUTH_DEFAULT = "globus"
		}
	}

	if ADMIN_USERS_VAR != "" {
		for _, name := range strings.Split(ADMIN_USERS_VAR, ",") {
			AdminUsers = append(AdminUsers, strings.TrimSpace(name))
		}
	}

	if mode == "server" {
		if PIPELINE_EXPIRE != "" {
			for _, set := range strings.Split(PIPELINE_EXPIRE, ",") {
				parts := strings.Split(set, "=")
				if valid, _, _ := parseExpiration(parts[1]); !valid {
					return errors.New("expiration format in pipeline_expire is invalid")
				}
				PIPELINE_EXPIRE_MAP[parts[0]] = parts[1]
			}
		}
		if GLOBAL_EXPIRE != "" {
			if valid, _, _ := parseExpiration(GLOBAL_EXPIRE); !valid {
				return errors.New("expiration format in global_expire is invalid")
			}
		}
	}

	if PID_FILE_PATH == "" {
		PID_FILE_PATH = DATA_PATH + "/pidfile"
	}

	if PRE_WORK_SCRIPT_ARGS_STRING != "" {
		PRE_WORK_SCRIPT_ARGS = strings.Split(PRE_WORK_SCRIPT_ARGS_STRING, ",")
	}

	vaildLogout := false
	for _, logout := range LOG_OUTPUTS {
		if LOG_OUTPUT == logout {
			vaildLogout = true
		}
	}
	if !vaildLogout {
		return fmt.Errorf("\"%s\" is invalid option for logoutput, use one of: file, console, both", LOG_OUTPUT)
	}

	SITE_PATH = cleanPath(SITE_PATH)
	DATA_PATH = cleanPath(DATA_PATH)
	LOGS_PATH = cleanPath(LOGS_PATH)
	WORK_PATH = cleanPath(WORK_PATH)
	APP_PATH = cleanPath(APP_PATH)
	AWF_PATH = cleanPath(AWF_PATH)
	PID_FILE_PATH = cleanPath(PID_FILE_PATH)

	VERSIONS["Job"] = 2

	return
}

func buildLoginResource(name string, bearer string) (login LoginResource) {
	login.Icon = name + "_favicon.ico"
	login.Prefix = strings.ToLower(name[0:2]) + LOGIN_PREFIX
	login.Keyword = "auth"
	login.Url = SITE_LOGIN_URL
	login.UseHeader = false
	login.Bearer = bearer
	return
}

func parseExpiration(expire string) (valid bool, duration int, unit string) {
	match := checkExpire.FindStringSubmatch(expire)
	if len(match) == 0 {
		return
	}
	valid = true
	duration, _ = strconv.Atoi(match[1])
	switch match[2] {
	case "M":
		unit = "minutes"
	case "H":
		unit = "hours"
	case "D":
		unit = "days"
	}
	return
}

func Print(service string) {
	fmt.Printf("##### Admin #####\nemail:\t%s\nusers:\t%s\n\n", ADMIN_EMAIL, ADMIN_USERS_VAR)
	fmt.Printf("####### Anonymous ######\nread:\t%t\nwrite:\t%t\ndelete:\t%t\n", ANON_READ, ANON_WRITE, ANON_DELETE)
	fmt.Printf("clientgroup read:\t%t\nclientgroup write:\t%t\nclientgroup delete:\t%t\n\n", ANON_CG_READ, ANON_CG_WRITE, ANON_CG_DELETE)

	fmt.Printf("##### Auth #####\n")
	if BASIC_AUTH {
		fmt.Printf("basic_auth:\ttrue\n")
	}
	if GLOBUS_TOKEN_URL != "" && GLOBUS_PROFILE_URL != "" {
		fmt.Printf("type:\tglobus\ntoken_url:\t%s\nprofile_url:\t%s\n", GLOBUS_TOKEN_URL, GLOBUS_PROFILE_URL)
	}
	if len(AUTH_OAUTH) > 0 {
		fmt.Printf("type:\toauth\n")
		for b, u := range AUTH_OAUTH {
			fmt.Printf("bearer: %s\turl: %s\n", b, u)
		}
	}
	if SITE_LOGIN_URL != "" {
		fmt.Printf("login_url:\t%s\n", SITE_LOGIN_URL)
	}
	fmt.Println()

	if service == "server" {
		fmt.Printf("##### Expiration #####\nexpire_wait:\t%d minutes\n", EXPIRE_WAIT)
		fmt.Printf("global_expire:\t")
		if GLOBAL_EXPIRE == "" {
			fmt.Printf("disabled\n")
		} else {
			_, duration, unit := parseExpiration(GLOBAL_EXPIRE)
			fmt.Printf("%d %s\n", duration, unit)
		}
		fmt.Printf("pipeline_expire:")
		if PIPELINE_EXPIRE == "" {
			fmt.Printf("\tdisabled\n")
		} else {
			for name, expire := range PIPELINE_EXPIRE_MAP {
				_, duration, unit := parseExpiration(expire)
				fmt.Printf("\n\t%s:\t%d %s", name, duration, unit)
			}
			fmt.Println()
		}
		fmt.Println()
	}

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
		fmt.Printf("##### Mongodb #####\nhost(s):\t%s\ndatabase:\t%s\ntimeout:\t%d\n\n", MONGODB_HOST, MONGODB_DATABASE, MONGODB_TIMEOUT)
	}

	if service == "server" {
		fmt.Printf("##### Ports #####\nsite:\t%d\napi:\t%d\n\n", SITE_PORT, API_PORT)
	} else if service == "proxy" {
		fmt.Printf("##### Ports #####\nsite:\t%d\napi:\t%d\n\n", P_SITE_PORT, P_API_PORT)
	}
}

func cleanPath(p string) string {
	if p != "" {
		p, _ = filepath.Abs(p)
	}
	return p
}

func PrintClientCfg() {
	fmt.Printf("###AWE client running###\n")
	fmt.Printf("work_path=%s\n", WORK_PATH)
	fmt.Printf("server_url=%s\n", SERVER_URL)
	fmt.Printf("print_app_msg=%t\n", PRINT_APP_MSG)
}

func PrintClientUsage() {
	fmt.Printf("Usage: awe-worker -conf </path/to/cfg> [-debuglevel 0-3]\n")
}

func PrintServerUsage() {
	fmt.Printf("Usage: awe-server -conf </path/to/cfg> [-dev] [-recover] [-debuglevel 0-3]\n")
}

func PrintProxyUsage() {
	fmt.Printf("Usage: awe-proxy -conf </path/to/cfg> [-debuglevel 0-3]\n")
}

func PrintVersionMsg() {
	fmt.Printf("AWE version: %s\n", VERSION)
}
