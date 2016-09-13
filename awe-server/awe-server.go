package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/auth"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/controller"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/db"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/AWE/lib/versions"
	"github.com/MG-RAST/golib/goweb"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
)

func launchSite(control chan int, port int) {
	goweb.ConfigureDefaultFormatters()
	r := &goweb.RouteManager{}

	site_directory := conf.SITE_PATH
	fileinfo, err := os.Stat(site_directory)
	if err != nil {
		message := fmt.Sprintf("ERROR: site, path %s does not exist: %s", site_directory, err.Error())
		if os.IsNotExist(err) {
			message += " IsNotExist"
		}

		fmt.Fprintf(os.Stderr, message, "\n")
		logger.Error(message)

		os.Exit(1)
	} else {
		if !fileinfo.IsDir() {
			message := fmt.Sprintf("ERROR: site, path %s exists, but is not a directory", site_directory)
			fmt.Fprintf(os.Stderr, message, "\n")
			logger.Error(message)
			os.Exit(1)
		}

	}

	template_conf_filename := path.Join(conf.SITE_PATH, "js/config.js.tt")
	target_conf_filename := path.Join(conf.SITE_PATH, "js/config.js")
	buf, err := ioutil.ReadFile(template_conf_filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not read config template: %v\n", err)
		logger.Error("ERROR: could not read config template: " + err.Error())
	}
	template_conf_string := string(buf)

	// add / replace AWE API url
	if conf.API_URL == "" {
		fmt.Fprintf(os.Stderr, "ERROR: API_URL is not defined. \n")
		logger.Error("ERROR: API_URL is not defined.")
	}
	template_conf_string = strings.Replace(template_conf_string, "[% api_url %]", conf.API_URL, -1)

	// add auth
	auth_on := "false"
	auth_resources := ""
	if conf.GLOBUS_OAUTH || conf.MGRAST_OAUTH {
		auth_on = "true"
		b, _ := json.Marshal(conf.AUTH_RESOURCES)
		b = bytes.TrimPrefix(b, []byte("{"))
		b = bytes.TrimSuffix(b, []byte("}"))
		auth_resources = "," + string(b)
	}

	// replace auth
	template_conf_string = strings.Replace(template_conf_string, "[% auth_on %]", auth_on, -1)
	template_conf_string = strings.Replace(template_conf_string, "[% auth_default %]", conf.AUTH_DEFAULT, -1)
	template_conf_string = strings.Replace(template_conf_string, "[% auth_resources %]", auth_resources, -1)

	target_conf_file, err := os.Create(target_conf_filename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not write config for Retina: %v\n", err)
		logger.Error("ERROR: could not write config for Retina: " + err.Error())
	}

	_, err = io.WriteString(target_conf_file, template_conf_string)

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not write config for Retina: %v\n", err)
		logger.Error("ERROR: could not write config for Retina: " + err.Error())
	}

	target_conf_file.Close()

	r.MapFunc("*", controller.SiteDir)
	if conf.SSL_ENABLED {
		err := goweb.ListenAndServeRoutesTLS(fmt.Sprintf(":%d", conf.SITE_PORT), conf.SSL_CERT_FILE, conf.SSL_KEY_FILE, r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: site: %v\n", err)
			logger.Error("ERROR: site: " + err.Error())
		}
	} else {
		err := goweb.ListenAndServeRoutes(fmt.Sprintf(":%d", conf.SITE_PORT), r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: site: %v\n", err)
			logger.Error("ERROR: site: " + err.Error())
		}
	}
	control <- 1 //we are ending
}

func launchAPI(control chan int, port int) {
	c := controller.NewServerController()
	goweb.ConfigureDefaultFormatters()
	r := &goweb.RouteManager{}
	r.Map("/job/{jid}/acl/{type}", c.JobAcl["typed"])
	r.Map("/job/{jid}/acl", c.JobAcl["base"])
	r.Map("/cgroup/{cgid}/acl/{type}", c.ClientGroupAcl["typed"])
	r.Map("/cgroup/{cgid}/acl", c.ClientGroupAcl["base"])
	r.Map("/cgroup/{cgid}/token", c.ClientGroupToken)
	r.MapRest("/job", c.Job)
	r.MapRest("/work", c.Work)
	r.MapRest("/cgroup", c.ClientGroup)
	r.MapRest("/client", c.Client)
	r.MapRest("/queue", c.Queue)
	r.MapRest("/awf", c.Awf)
	r.MapFunc("/event", controller.EventDescription, goweb.GetMethod)
	r.MapFunc("*", controller.ResourceDescription, goweb.GetMethod)
	if conf.SSL_ENABLED {
		err := goweb.ListenAndServeRoutesTLS(fmt.Sprintf(":%d", conf.API_PORT), conf.SSL_CERT_FILE, conf.SSL_KEY_FILE, r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: api: %v\n", err)
			logger.Error("ERROR: api: " + err.Error())
		}
	} else {
		err := goweb.ListenAndServeRoutes(fmt.Sprintf(":%d", conf.API_PORT), r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: api: %v\n", err)
			logger.Error("ERROR: api: " + err.Error())
		}
	}
	control <- 1 //we are ending
}

func main() {

	err := conf.Init_conf("server")

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: error reading conf file: "+err.Error())
		os.Exit(1)
	}

	//if !conf.INIT_SUCCESS {
	//	conf.PrintServerUsage()
	//	os.Exit(1)
	//}
	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("DEBUG_LEVEL > 0")
	}
	if _, err := os.Stat(conf.DATA_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.DATA_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating data_path \"%s\", %s\n", conf.DATA_PATH, err.Error())
			os.Exit(1)
		}
	}

	if _, err := os.Stat(conf.LOGS_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.LOGS_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating log_path \"%s\" %s\n", conf.LOGS_PATH, err.Error())
			os.Exit(1)
		}
	}

	if _, err := os.Stat(conf.DATA_PATH + "/temp"); err != nil && os.IsNotExist(err) {
		if err := os.Mkdir(conf.DATA_PATH+"/temp", 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
			os.Exit(1)
		}
	}
	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("logger.Initialize...")
	}
	//init logger
	logger.Initialize("server")

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("init db...")
	}
	//init db
	if err := db.Initialize(); err != nil {
		fmt.Printf("failed to initialize job db: %s\n", err.Error())
		os.Exit(1)
	}

	//init versions
	if err := versions.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "Err@versions.Initialize: %v\n", err)
		logger.Error("Err@versions.Initialize: " + err.Error())
		os.Exit(1)
	}

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("init db collection for user...")
	}
	//init db collection for user
	if err := user.Initialize(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR initializing user database: %s\n", err.Error())
		os.Exit(1)
	}

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("init resource manager...")
	}
	//init resource manager
	core.InitResMgr("server")
	core.InitAwfMgr()
	core.InitJobDB()
	core.InitClientGroupDB()

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("init auth...")
	}
	//init auth
	auth.Initialize()

	controller.PrintLogo()
	conf.Print("server")

	// reload job directory
	if conf.RELOAD != "" {
		fmt.Println("####### Reloading #######")
		if err := reload(conf.RELOAD); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		}
		fmt.Println("Done")
	}

	// init max job number (jid), backwards compatible with jobid file
	if err := core.QMgr.InitMaxJid(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR from InitMaxJid : %v\n", err)
		os.Exit(1)
	}
	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("launching server...")
	}
	//launch server
	control := make(chan int)
	go core.Ttl.Handle()
	go core.QMgr.JidHandle()
	go core.QMgr.TaskHandle()
	go core.QMgr.ClientHandle()
	go core.QMgr.ClientChecker()
	go launchSite(control, conf.SITE_PORT)
	go launchAPI(control, conf.API_PORT)

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("API launched...")
	}
	if err := core.AwfMgr.LoadWorkflows(); err != nil {
		logger.Error("LoadWorkflows: " + err.Error())
	}

	var host string
	if hostname, err := os.Hostname(); err == nil {
		host = fmt.Sprintf("%s:%d", hostname, conf.API_PORT)
	}

	//recover unfinished jobs before server went down last time
	if conf.RECOVER {
		fmt.Println("####### Recovering unfinished jobs #######")
		if err := core.QMgr.RecoverJobs(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		}
		fmt.Println("Done")
		logger.Event(event.SERVER_RECOVER, "host="+host)
	} else {
		logger.Event(event.SERVER_START, "host="+host)
	}

	if conf.PID_FILE_PATH != "" {
		f, err := os.Create(conf.PID_FILE_PATH)
		if err != nil {
			err_msg := "Could not create pid file: " + conf.PID_FILE_PATH + "\n"
			fmt.Fprintf(os.Stderr, err_msg)
			logger.Error("ERROR: " + err_msg)
			os.Exit(1)
		}
		defer f.Close()

		pid := os.Getpid()
		fmt.Fprintln(f, pid)

		fmt.Println("##### pidfile #####")
		fmt.Printf("pid: %d saved to file: %s\n\n", pid, conf.PID_FILE_PATH)
	}

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("setting GOMAXPROCS...")
	}
	// setting GOMAXPROCS
	var procs int
	avail := runtime.NumCPU()
	if avail <= 2 {
		procs = 1
	} else if avail == 3 {
		procs = 2
	} else {
		procs = avail - 2
	}
	fmt.Println("##### Procs #####")
	fmt.Printf("Number of available CPUs = %d\n", avail)
	if conf.GOMAXPROCS > 0 {
		procs = conf.GOMAXPROCS
	}
	if procs <= avail {
		fmt.Printf("Running AWE server with GOMAXPROCS = %d\n\n", procs)
		runtime.GOMAXPROCS(procs)
	} else {
		fmt.Println("GOMAXPROCS config value is greater than available number of CPUs.")
		fmt.Printf("Running Shock server with GOMAXPROCS = %d\n\n", avail)
		runtime.GOMAXPROCS(avail)
	}

	<-control //block till something dies
}
