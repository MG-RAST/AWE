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
	r.MapRest("/user", c.User)
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

	//init logger
	logger.Initialize("server")

	//init db
	if err := db.Initialize(); err != nil {
		fmt.Printf("failed to initialize job db: %s\n", err.Error())
		os.Exit(1)
	}

	//init db collection for user
	user.Initialize()

	//init resource manager
	core.InitResMgr("server")
	core.InitAwfMgr()
	core.InitJobDB()

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

	//init max job number (jid)
	if err := core.QMgr.InitMaxJid(); err != nil {
		fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		os.Exit(1)
	}

	//recover unfinished jobs before server went down last time
	if conf.RECOVER {
		fmt.Println("####### Recovering unfinished jobs #######")
		if err := core.QMgr.RecoverJobs(); err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: %v\n", err)
		}
		fmt.Println("Done")
	}

	//launch server
	control := make(chan int)
	go core.QMgr.Handle()
	go core.QMgr.Timer()
	go core.QMgr.ClientChecker()
	go launchSite(control, conf.SITE_PORT)
	go launchAPI(control, conf.API_PORT)

	if err := core.AwfMgr.LoadWorkflows(); err != nil {
		logger.Error("LoadWorkflows: " + err.Error())
	}

	var host string
	if hostname, err := os.Hostname(); err == nil {
		host = fmt.Sprintf("%s:%d", hostname, conf.API_PORT)
	}
	if conf.RECOVER {
		logger.Event(event.SERVER_RECOVER, "host="+host)
	} else {
		logger.Event(event.SERVER_START, "host="+host)
	}

	<-control //block till something dies
}
