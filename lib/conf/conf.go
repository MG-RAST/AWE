package conf

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/MG-RAST/golib/goconfig/config"
)

const VERSION string = "0.9.36"

var GIT_COMMIT_HASH string // use -ldflags "-X github.com/MG-RAST/AWE/lib/conf.GIT_COMMIT_HASH <value>"
const BasePriority int = 1

const DB_COLL_JOBS string = "Jobs"
const DB_COLL_PERF string = "Perf"
const DB_COLL_CGS string = "ClientGroups"
const DB_COLL_USERS string = "Users"

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
	GLOBUS_OAUTH       bool
	MGRAST_OAUTH       bool
	GLOBUS_TOKEN_URL   string
	GLOBUS_PROFILE_URL string
	MGRAST_OAUTH_URL   string
	MGRAST_LOGIN_URL   string
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
	Admin_Users    = make(map[string]bool)
	AUTH_RESOURCES = make(map[string]AuthResource)
	AUTH_DEFAULT   string

	// internal config control
	FAKE_VAR = false
)

// writes to target only if has been defined in config
// avoids overwriting of default values if config is not defined
func getDefinedValueInt(c *config.Config, section string, key string, target *int) {
	if c.HasOption(section, key) {
		if int_value, err := c.Int(section, key); err == nil {
			*target = int_value
		}
	}
}

func getDefinedValueBool(c *config.Config, section string, key string, target *bool) {
	if c.HasOption(section, key) {
		if bool_value, err := c.Bool(section, key); err == nil {
			*target = bool_value
		}
	}
}

func getDefinedValueString(c *config.Config, section string, key string, target *string) {
	if string_value, err := c.String(section, key); err == nil {
		string_value = os.ExpandEnv(string_value)
		*target = string_value
	}

}

type Config_value struct {
	Conf_type string
	Conf_str  *Config_value_string
	Conf_int  *Config_value_int
	Conf_bool *Config_value_bool
}

type Config_value_string struct {
	Target        *string
	Default_value string
	Section       string
	Key           string
	Descr_short   string
	Descr_long    string
}

type Config_value_int struct {
	Target        *int
	Default_value int
	Section       string
	Key           string
	Descr_short   string
	Descr_long    string
}

type Config_value_bool struct {
	Target        *bool
	Default_value bool
	Section       string
	Key           string
	Descr_short   string
	Descr_long    string
}

type Config_store struct {
	Store []*Config_value
	Fs    *flag.FlagSet
	Con   *config.Config
}

type AuthResource struct {
	Icon      string `json:"icon"`
	Prefix    string `json:"prefix"`
	Keyword   string `json:"keyword"`
	Url       string `json:"url"`
	UseHeader bool   `json:"useHeader"`
	Bearer    string `json:"bearer"`
}

func NewCS(c *config.Config) *Config_store {
	cs := &Config_store{Store: make([]*Config_value, 0, 100), Con: c} // length 0, capacity 100
	cs.Fs = flag.NewFlagSet("name", flag.ContinueOnError)
	cs.Fs.BoolVar(&FAKE_VAR, "fake_var", true, "ignore this")
	cs.Fs.BoolVar(&SHOW_HELP, "h", false, "ignore this") // for help: -h
	return cs
}

func (this *Config_store) AddString(target *string,
	default_value string,
	section string,
	key string,
	descr_short string,
	descr_long string) {

	*target = default_value
	new_val := &Config_value{Conf_type: "string"}
	new_val.Conf_str = &Config_value_string{target, default_value, section, key, descr_short, descr_long}

	this.Store = append(this.Store, new_val)
}

func (this *Config_store) AddInt(target *int,
	default_value int,
	section string,
	key string,
	descr_short string,
	descr_long string) {

	*target = default_value
	new_val := &Config_value{Conf_type: "int"}
	new_val.Conf_int = &Config_value_int{target, default_value, section, key, descr_short, descr_long}

	this.Store = append(this.Store, new_val)
}

func (this *Config_store) AddBool(target *bool,
	default_value bool,
	section string,
	key string,
	descr_short string,
	descr_long string) {

	*target = default_value
	new_val := &Config_value{Conf_type: "bool"}
	new_val.Conf_bool = &Config_value_bool{target, default_value, section, key, descr_short, descr_long}

	this.Store = append(this.Store, new_val)
}

func (this Config_store) Parse() {
	c := this.Con
	f := this.Fs
	for _, val := range this.Store {
		if val.Conf_type == "string" {
			get_my_config_string(c, f, val.Conf_str)
		} else if val.Conf_type == "int" {
			get_my_config_int(c, f, val.Conf_int)
		} else if val.Conf_type == "bool" {
			get_my_config_bool(c, f, val.Conf_bool)
		}
	}
	err := this.Fs.Parse(os.Args[1:])
	if err != nil {
		this.PrintHelp()
		fmt.Fprintf(os.Stderr, "error parsing command line args: "+err.Error()+"\n")
		os.Exit(1)
	}
}

func (this Config_store) PrintHelp() {
	current_section := ""
	prefix := "--"
	if PRINT_HELP {
		prefix = ""
	}
	for _, val := range this.Store {
		if val.Conf_type == "string" {
			d := val.Conf_str
			if current_section != d.Section {
				current_section = d.Section
				fmt.Printf("\n[%s]\n", current_section)
			}
			fmt.Printf("%s%-27s %s (default: \"%s\")\n", prefix, d.Key+"=<string>", d.Descr_short, d.Default_value)

			if PRINT_HELP && d.Descr_long != "" {
				fmt.Printf("     %s\n", d.Descr_long)
			}
		} else if val.Conf_type == "int" {
			d := val.Conf_int
			if current_section != d.Section {
				current_section = d.Section
				fmt.Printf("\n[%s]\n", current_section)
			}
			fmt.Printf("%s%-27s %s (default: %d)\n", prefix, d.Key+"=<int>", d.Descr_short, d.Default_value)

			if PRINT_HELP && d.Descr_long != "" {
				fmt.Printf("     %s\n", d.Descr_long)
			}
		} else if val.Conf_type == "bool" {
			d := val.Conf_bool
			if current_section != d.Section {
				current_section = d.Section
				fmt.Printf("\n[%s]\n", current_section)
			}
			fmt.Printf("%s%-27s %s (default: %t)\n", prefix, d.Key+"=<bool>", d.Descr_short, d.Default_value)

			if PRINT_HELP && d.Descr_long != "" {
				fmt.Printf("     %s\n", d.Descr_long)
			}
		}
	}

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
func getConfiguration(c *config.Config, mode string) (c_store *Config_store, err error) {
	c_store = NewCS(c)
	// examples:
	// c_store.AddString(&VARIABLE, "", "section", "key", "", "")
	// c_store.AddInt(&VARIABLE, 0, "section", "key", "", "")
	// c_store.AddBool(&VARIABLE, false, "section", "key", "", "")

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
		c_store.AddString(&MGRAST_OAUTH_URL, "", "Auth", "mgrast_oauth_url", "", "")
		c_store.AddString(&MGRAST_LOGIN_URL, "", "Auth", "mgrast_login_url", "", "")
		c_store.AddBool(&CLIENT_AUTH_REQ, false, "Auth", "client_auth_required", "", "")

		// Admin
		c_store.AddString(&ADMIN_USERS_VAR, "", "Admin", "users", "", "")
		c_store.AddString(&ADMIN_EMAIL, "", "Admin", "email", "", "")

		// Directories
		c_store.AddString(&SITE_PATH, os.Getenv("GOPATH")+"/src/github.com/MG-RAST/AWE/site", "Directories", "site", "the path to the website", "")
		c_store.AddString(&AWF_PATH, "", "Directories", "awf", "", "")
	}

	// Directories
	c_store.AddString(&DATA_PATH, "/mnt/data/awe/data", "Directories", "data", "a file path for storing system related data (job script, cached data, etc)", "")
	c_store.AddString(&LOGS_PATH, "/mnt/data/awe/logs", "Directories", "logs", "a path for storing logs", "")

	// Paths
	c_store.AddString(&PID_FILE_PATH, "", "Paths", "pidfile", "", "")

	if mode == "server" {
		// Mongodb
		c_store.AddString(&MONGODB_HOST, "localhost", "Mongodb", "hosts", "", "")
		c_store.AddString(&MONGODB_DATABASE, "AWEDB", "Mongodb", "database", "", "")
		c_store.AddString(&MONGODB_USER, "", "Mongodb", "user", "", "")
		c_store.AddString(&MONGODB_PASSWD, "", "Mongodb", "password", "", "")
		c_store.AddInt(&MONGODB_TIMEOUT, 1200, "Mongodb", "timeout", "", "")

		// Server
		c_store.AddString(&TITLE, "AWE Server", "Server", "title", "", "")
		c_store.AddBool(&PERF_LOG_WORKUNIT, true, "Server", "perf_log_workunit", "collecting performance log per workunit", "")
		c_store.AddInt(&EXPIRE_WAIT, 60, "Server", "expire_wait", "wait time for expiration reaper in minutes", "")
		c_store.AddString(&GLOBAL_EXPIRE, "", "Server", "global_expire", "default number and unit of time after job completion before it expires", "")
		c_store.AddString(&PIPELINE_EXPIRE, "", "Server", "pipeline_expire", "comma seperated list of pipeline_name=expire_days_unit, overrides global_expire", "")
		c_store.AddInt(&MAX_WORK_FAILURE, 3, "Server", "max_work_failure", "number of times that one workunit fails before the workunit considered suspend", "")
		c_store.AddInt(&MAX_CLIENT_FAILURE, 5, "Server", "max_client_failure", "number of times that one client consecutively fails running workunits before the client considered suspend", "")
		c_store.AddInt(&GOMAXPROCS, 0, "Server", "go_max_procs", "", "")
		c_store.AddString(&RELOAD, "", "Server", "reload", "path or url to awe job data. WARNING this will drop all current jobs", "")
		c_store.AddBool(&RECOVER, false, "Server", "recover", "load unfinished jobs from mongodb on startup", "")
		c_store.AddInt(&RECOVER_MAX, 0, "Server", "recover_max", "max number of jobs to recover, default (0) means recover all", "")
	}

	if mode == "client" {
		// Client
		c_store.AddString(&SERVER_URL, "http://localhost:8001", "Client", "serverurl", "URL of AWE server, including API port", "")
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
	c_store.AddString(&USE_DOCKER, "yes", "Docker", "use_docker", "\"yes\", \"no\" or \"only\"", "yes: allow docker tasks, no: do not allow docker tasks, only: allow only docker tasks; if docker is not installed on the clients, choose \"no\"")

	if mode == "client" {
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

	//Proxy
	c_store.AddInt(&P_SITE_PORT, 8082, "Proxy", "p-site-port", "", "")
	c_store.AddInt(&P_API_PORT, 8002, "Proxy", "p-api-port", "", "")

	//Other
	c_store.AddBool(&DEV_MODE, false, "Other", "dev", "dev or demo mode, print some msgs on screen", "")
	c_store.AddInt(&DEBUG_LEVEL, 0, "Other", "debuglevel", "debug level: 0-3", "")
	c_store.AddString(&CONFIG_FILE, "", "Other", "conf", "path to config file", "")
	c_store.AddString(&LOG_OUTPUT, "console", "Other", "logoutput", "log output stream, one of: file, console, both", "")
	c_store.AddBool(&SHOW_VERSION, false, "Other", "version", "show version", "")
	c_store.AddBool(&SHOW_GIT_COMMIT_HASH, false, "Other", "show_git_commit_hash", "", "")
	c_store.AddBool(&PRINT_HELP, false, "Other", "fullhelp", "show detailed usage without \"--\"-prefixes", "")
	c_store.AddBool(&SHOW_HELP, false, "Other", "help", "show usage", "")

	c_store.Parse()

	return c_store, nil
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
				return errors.New("ERROR: parsing command options, missing conf file")
			}
		}
	}

	var c *config.Config = nil
	if CONFIG_FILE != "" {
		c, err = config.ReadDefault(CONFIG_FILE)
		if err != nil {
			return errors.New("ERROR: error reading conf file: " + err.Error())
		}
	}

	c_store, err := getConfiguration(c, mode) // from config file and command line arguments
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: error reading conf file: %v\n", err)
		return
	}

	// ####### at this point configuration variables are set ########

	if FAKE_VAR == false {
		fmt.Fprintf(os.Stderr, "ERROR: config was not parsed\n")
		os.Exit(1)
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
	if mode == "client" {
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

	if GLOBUS_TOKEN_URL != "" && GLOBUS_PROFILE_URL != "" {
		GLOBUS_OAUTH = true
		AUTH_DEFAULT = "KBase"
		AUTH_RESOURCES["KBase"] = AuthResource{
			Icon:      "KBase_favicon.ico",
			Prefix:    "kbgo4711",
			Keyword:   "auth",
			Url:       MGRAST_LOGIN_URL,
			UseHeader: false,
			Bearer:    "OAuth",
		}
	}
	if MGRAST_OAUTH_URL != "" {
		MGRAST_OAUTH = true
		AUTH_DEFAULT = "MG-RAST"
		AUTH_RESOURCES["MG-RAST"] = AuthResource{
			Icon:      "MGRAST_favicon.ico",
			Prefix:    "mggo4711",
			Keyword:   "auth",
			Url:       MGRAST_LOGIN_URL,
			UseHeader: false,
			Bearer:    "mgrast",
		}
	}

	if ADMIN_USERS_VAR != "" {
		for _, name := range strings.Split(ADMIN_USERS_VAR, ",") {
			Admin_Users[strings.TrimSpace(name)] = true
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
		return errors.New("invalid option for logoutput, use one of: file, console, both")
	}

	WORK_PATH, _ = filepath.Abs(WORK_PATH)
	APP_PATH, _ = filepath.Abs(APP_PATH)
	SITE_PATH, _ = filepath.Abs(SITE_PATH)
	DATA_PATH, _ = filepath.Abs(DATA_PATH)
	LOGS_PATH, _ = filepath.Abs(LOGS_PATH)
	AWF_PATH, _ = filepath.Abs(AWF_PATH)

	VERSIONS["Job"] = 2

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
	fmt.Printf("##### Admin #####\nemail:\t%s\n\n", ADMIN_EMAIL)
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
	if MGRAST_LOGIN_URL != "" {
		fmt.Printf("mgrast_login_url:\t%s\n", MGRAST_LOGIN_URL)
	}
	if len(Admin_Users) > 0 {
		fmt.Printf("admin_auth:\ttrue\nadmin_users:\t")
		for name, _ := range Admin_Users {
			fmt.Printf("%s ", name)
		}
		fmt.Printf("\n")
	}
	fmt.Printf("\n")

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
			fmt.Printf("\n")
		}
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
		fmt.Printf("##### Mongodb #####\nhost(s):\t%s\ndatabase:\t%s\ntimeout:\t%d\n", MONGODB_HOST, MONGODB_DATABASE, MONGODB_TIMEOUT)
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
	fmt.Printf("Usage: awe-client -conf </path/to/cfg> [-debuglevel 0-3]\n")
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
