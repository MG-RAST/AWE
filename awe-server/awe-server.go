package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"github.com/MG-RAST/AWE/core"
	"github.com/MG-RAST/AWE/logger"
	"github.com/jaredwilkening/goweb"
	"os"
)

var (
	log      = logger.New()
	queueMgr = core.NewQueueMgr()
)

func launchSite(control chan int, port int) {
	goweb.ConfigureDefaultFormatters()
	r := &goweb.RouteManager{}
	r.MapFunc("/raw", RawDir)
	r.MapFunc("/assets", AssetsDir)
	r.MapFunc("*", Site)
	if conf.SSL_ENABLED {
		err := goweb.ListenAndServeRoutesTLS(fmt.Sprintf(":%d", conf.SITE_PORT), conf.SSL_CERT_FILE, conf.SSL_KEY_FILE, r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: site: %v\n", err)
			log.Error("ERROR: site: " + err.Error())
		}
	} else {
		err := goweb.ListenAndServeRoutes(fmt.Sprintf(":%d", conf.SITE_PORT), r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: site: %v\n", err)
			log.Error("ERROR: site: " + err.Error())
		}
	}
	control <- 1 //we are ending
}

func launchAPI(control chan int, port int) {
	goweb.ConfigureDefaultFormatters()
	r := &goweb.RouteManager{}
	r.MapRest("/job", new(JobController))
	//r.MapRest("/user", new(UserController))
	r.MapFunc("*", ResourceDescription, goweb.GetMethod)
	if conf.SSL_ENABLED {
		err := goweb.ListenAndServeRoutesTLS(fmt.Sprintf(":%d", conf.API_PORT), conf.SSL_CERT_FILE, conf.SSL_KEY_FILE, r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: api: %v\n", err)
			log.Error("ERROR: api: " + err.Error())
		}
	} else {
		err := goweb.ListenAndServeRoutes(fmt.Sprintf(":%d", conf.API_PORT), r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: api: %v\n", err)
			log.Error("ERROR: api: " + err.Error())
		}
	}
	control <- 1 //we are ending
}

func main() {
	printLogo()
	conf.Print()

	//launch server
	control := make(chan int)
	go log.Handle()
	go queueMgr.Handle()
	go queueMgr.Timer()
	go launchSite(control, conf.SITE_PORT)
	go launchAPI(control, conf.API_PORT)
	<-control //block till something dies
}
