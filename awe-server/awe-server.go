package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"os"
	"path"
	"runtime"
	"strings"
	"time"

	"github.com/MG-RAST/AWE/lib/auth"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/controller"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/db"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/AWE/lib/versions"
	"github.com/MG-RAST/golib/go-uuid/uuid"
	"github.com/MG-RAST/golib/goweb"
)

// this function is deprecated ion favor of the stand-alone awe-monitor (https://github.com/MG-RAST/awe-monitor)
func launchSite(control chan int, port int) {

	r := &goweb.RouteManager{}

	siteDirectory := conf.SITE_PATH
	fileinfo, err := os.Stat(siteDirectory)
	if err != nil {
		message := fmt.Sprintf("ERROR: site, path %s does not exist: %s", siteDirectory, err.Error())
		if os.IsNotExist(err) {
			message += " IsNotExist"
		}

		fmt.Fprintf(os.Stderr, message, "\n")
		logger.Error(message)

		os.Exit(1)
	} else {
		if !fileinfo.IsDir() {
			message := fmt.Sprintf("ERROR: site, path %s exists, but is not a directory", siteDirectory)
			fmt.Fprintf(os.Stderr, message, "\n")
			logger.Error(message)
			os.Exit(1)
		}

	}

	templateConfFilename := path.Join(conf.SITE_PATH, "js/config.js.tt")
	targetConfFilename := path.Join(conf.SITE_PATH, "js/config.js")
	buf, err := ioutil.ReadFile(templateConfFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not read config template: %v\n", err)
		logger.Error("ERROR: could not read config template: " + err.Error())
	}
	templateConfString := string(buf)

	// add / replace AWE API url
	if conf.API_URL == "" {
		fmt.Fprintf(os.Stderr, "ERROR: API_URL is not defined. \n")
		logger.Error("ERROR: API_URL is not defined.")
	}
	templateConfString = strings.Replace(templateConfString, "[% api_url %]", conf.API_URL, -1)

	// add login auth
	authOn := "false"
	authOAuthserver := "false"
	authResources := ""
	authURL := ""
	if conf.HAS_OAUTH {
		authOn = "true"
		b, _ := json.Marshal(conf.LOGIN_RESOURCES)
		b = bytes.TrimPrefix(b, []byte("{"))
		b = bytes.TrimSuffix(b, []byte("}"))
		authResources = "," + string(b)
	}
	if conf.USE_OAUTH_SERVER {
		authOAuthserver = "true"
		authURL = conf.AUTH_URL
	}

	// replace auth
	templateConfString = strings.Replace(templateConfString, "[% auth_on %]", authOn, -1)
	templateConfString = strings.Replace(templateConfString, "[% auth_oauthserver %]", authOAuthserver, -1)
	templateConfString = strings.Replace(templateConfString, "[% auth_url %]", authURL, -1)
	templateConfString = strings.Replace(templateConfString, "[% auth_default %]", conf.LOGIN_DEFAULT, -1)
	templateConfString = strings.Replace(templateConfString, "[% auth_resources %]", authResources, -1)

	targetConfFile, err := os.Create(targetConfFilename)
	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not write config for Retina: %v\n", err)
		logger.Error("ERROR: could not write config for Retina: " + err.Error())
	}

	_, err = io.WriteString(targetConfFile, templateConfString)

	if err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: could not write config for Retina: %v\n", err)
		logger.Error("ERROR: could not write config for Retina: " + err.Error())
	}

	targetConfFile.Close()

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
	//goweb.ConfigureDefaultFormatters()
	r := &goweb.RouteManager{}
	r.Map("/job/{jid}/acl/{type}", c.JobAcl["typed"])
	r.Map("/job/{jid}/acl", c.JobAcl["base"])
	r.Map("/cgroup/{cgid}/acl/{type}", c.ClientGroupAcl["typed"])
	r.Map("/cgroup/{cgid}/acl", c.ClientGroupAcl["base"])
	r.Map("/cgroup/{cgid}/token", c.ClientGroupToken)
	r.MapRest("/job", c.Job)
	r.MapRest("/workflow_instances", c.WorkflowInstances)
	r.MapRest("/work", c.Work)
	r.MapRest("/cgroup", c.ClientGroup)
	r.MapRest("/client", c.Client)
	r.MapRest("/queue", c.Queue)
	r.MapRest("/logger", c.Logger)
	r.MapRest("/awf", c.Awf)
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

	if err := conf.Init_conf("server"); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: "+err.Error())
		os.Exit(1)
	}

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("DEBUG_LEVEL > 0")
	}
	if _, err := os.Stat(conf.DATA_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.DATA_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating data_path \"%s\", %s\n", conf.DATA_PATH, err.Error())
			os.Exit(1)
		}
	}

	if (conf.LOG_OUTPUT == "file") || (conf.LOG_OUTPUT == "both") {
		if _, err := os.Stat(conf.LOGS_PATH); err != nil && os.IsNotExist(err) {
			if err := os.MkdirAll(conf.LOGS_PATH, 0777); err != nil {
				fmt.Fprintf(os.Stderr, "ERROR in creating log_path \"%s\" %s\n", conf.LOGS_PATH, err.Error())
				os.Exit(1)
			}
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

	time.Sleep(time.Second * 3) // workaround to make sure logger is working correctly ; TODO better fix needed

	core.JM = core.NewJobMap()
	core.ServerUUID = uuid.New()

	logger.Info("init db...")

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

	logger.Info("init resource manager...")

	//init resource manager
	core.InitResMgr("server")

	logger.Info("InitAwfMgr...")
	core.InitAwfMgr()

	logger.Info("InitJobDB...")
	core.InitJobDB()

	logger.Info("InitClientGroupDB...")
	core.InitClientGroupDB()

	logger.Info("init auth...")
	//init auth
	auth.Initialize()

	controller.PrintLogo()
	conf.Print("server")

	logger.Info("launching server...")

	//launch server
	control := make(chan int)
	go core.Ttl.Handle() // deletes expired jobs
	go core.QMgr.ClientHandle()
	go core.QMgr.NoticeHandle()
	go core.QMgr.ClientChecker()
	go core.QMgr.UpdateQueueLoop()

	goweb.ConfigureDefaultFormatters()
	//go launchSite(control, conf.SITE_PORT) // deprecated
	go launchAPI(control, conf.API_PORT)

	logger.Info("API launched...")

	// reload job directory
	if conf.RELOAD != "" {
		fmt.Println("####### Reloading #######")
		if err := reload(conf.RELOAD); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		}
		fmt.Println("Done")
	}

	//if err := core.AwfMgr.LoadWorkflows(); err != nil {
	//	logger.Error("LoadWorkflows: " + err.Error())
	//}

	var host string
	if hostname, err := os.Hostname(); err == nil {
		host = fmt.Sprintf("%s:%d", hostname, conf.API_PORT)
	}

	//recover unfinished jobs before server went down last time
	if conf.RECOVER {
		if conf.RECOVER_MAX > 0 {
			logger.Info("####### Recovering %d unfinished jobs #######", conf.RECOVER_MAX)
		} else {
			logger.Info("####### Recovering all unfinished jobs #######")
		}

		recovered, total, err := core.QMgr.RecoverJobs()
		if err != nil {
			logger.Error("RecoverJobs error: %v\n", err)
		}
		fmt.Printf("%d total jobs from mongo\n", total)
		fmt.Printf("%d unfinished jobs recovered\n", recovered)

		logger.Info("Recovering done")
		logger.Event(event.SERVER_RECOVER, "host="+host)
	} else {
		logger.Event(event.SERVER_START, "host="+host)
	}

	if conf.PID_FILE_PATH != "" {
		f, err := os.Create(conf.PID_FILE_PATH)
		if err != nil {
			errMsg := "Could not create pid file: " + conf.PID_FILE_PATH + "\n"
			fmt.Fprintf(os.Stderr, errMsg)
			logger.Error("ERROR: " + errMsg)
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
