package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"github.com/MG-RAST/AWE/controller"
	"github.com/MG-RAST/AWE/core"
	. "github.com/MG-RAST/AWE/lib/util"
	. "github.com/MG-RAST/AWE/logger"
	"github.com/jaredwilkening/goweb"
	"os"
)

func launchAPI(control chan int, port int) {
	c := controller.NewProxyController()
	goweb.ConfigureDefaultFormatters()
	r := &goweb.RouteManager{}
	r.MapRest("/work", c.Work)
	r.MapRest("/client", c.Client)
	if conf.SSL_ENABLED {
		err := goweb.ListenAndServeRoutesTLS(fmt.Sprintf(":%d", conf.P_API_PORT), conf.SSL_CERT_FILE, conf.SSL_KEY_FILE, r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: api: %v\n", err)
			Log.Error("ERROR: api: " + err.Error())
		}
	} else {
		err := goweb.ListenAndServeRoutes(fmt.Sprintf(":%d", conf.P_API_PORT), r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: api: %v\n", err)
			Log.Error("ERROR: api: " + err.Error())
		}
	}
	control <- 1 //we are ending
}

func main() {

	if !conf.INIT_SUCCESS {
		conf.PrintServerUsage()
		os.Exit(1)
	}

	PrintLogo()
	conf.Print()

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

	//init queue mgr
	core.InitQueueMgr()

	//init logger
	Log = NewLogger("proxy")

	//launch server
	control := make(chan int)
	go Log.Handle()
	go core.QMgr.Handle()
	go core.QMgr.Timer()
	go core.QMgr.ClientChecker()
	go launchAPI(control, conf.API_PORT)

	var host string
	if hostname, err := os.Hostname(); err == nil {
		host = fmt.Sprintf("%s:%d", hostname, conf.API_PORT)
	}
	Log.Event(EVENT_SERVER_START, "host="+host)
	<-control //block till something dies
}
