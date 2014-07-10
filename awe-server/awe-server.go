package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/auth"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/controller"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/db"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/user"
	"github.com/MG-RAST/golib/goweb"
	"os"
	"runtime"
)

func launchSite(control chan int, port int) {
	goweb.ConfigureDefaultFormatters()
	r := &goweb.RouteManager{}
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
	r.MapRest("/job", c.Job)
	r.MapRest("/work", c.Work)
	r.MapRest("/client", c.Client)
	r.MapRest("/queue", c.Queue)
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

	if !conf.INIT_SUCCESS {
		conf.PrintServerUsage()
		os.Exit(1)
	}
	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("DEBUG_LEVEL > 0")
	}
	if _, err := os.Stat(conf.DATA_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.DATA_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating data_path %s\n", err.Error())
			os.Exit(1)
		}
	}

	if _, err := os.Stat(conf.LOGS_PATH); err != nil && os.IsNotExist(err) {
		if err := os.MkdirAll(conf.LOGS_PATH, 0777); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR in creating log_path %s\n", err.Error())
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

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("init db collection for user...")
	}
	//init db collection for user
	user.Initialize()

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("init resource manager...")
	}
	//init resource manager
	core.InitResMgr("server")
	core.InitAwfMgr()
	core.InitJobDB()

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

	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("init max job number (jid)...")
	}
	//init max job number (jid)
	if err := core.QMgr.InitMaxJid(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR from InitMaxJid : %v\n", err)
		os.Exit(1)
	}
	if conf.DEBUG_LEVEL > 0 {
		fmt.Println("launching server...")
	}
	//launch server
	control := make(chan int)
	go core.QMgr.Handle()
	go core.QMgr.Timer()
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
