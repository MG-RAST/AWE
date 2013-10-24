package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/controller"
	"github.com/MG-RAST/AWE/lib/core"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/AWE/lib/worker"
	"github.com/MG-RAST/golib/goweb"
	"os"
)

func launchAPI(control chan int, port int) {
	c := controller.NewProxyController()
	goweb.ConfigureDefaultFormatters()
	r := &goweb.RouteManager{}
	r.MapRest("/work", c.Work)
	r.MapRest("/client", c.Client)
	r.MapFunc("*", controller.ResourceDescription, goweb.GetMethod)
	if conf.SSL_ENABLED {
		err := goweb.ListenAndServeRoutesTLS(fmt.Sprintf(":%d", conf.P_API_PORT), conf.SSL_CERT_FILE, conf.SSL_KEY_FILE, r)
		if err != nil {
			fmt.Fprintf(os.Stderr, "ERROR: api: %v\n", err)
			logger.Error("ERROR: api: " + err.Error())
		}
	} else {
		err := goweb.ListenAndServeRoutes(fmt.Sprintf(":%d", conf.P_API_PORT), r)
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
	fmt.Printf("--------AWE Proxy running--------\n\n")
	conf.Print("proxy")

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

	//init proxy mgr
	core.InitResMgr("proxy")

	//init logger
	logger.Initialize("proxy")

	//launch server
	control := make(chan int)
	go core.QMgr.Handle()
	go core.QMgr.Timer()
	go core.QMgr.ClientChecker()
	go launchAPI(control, conf.API_PORT)

	var host string
	if hostname, err := os.Hostname(); err == nil {
		host = fmt.Sprintf("%s:%d", hostname, conf.API_PORT)
	}
	logger.Event(event.SERVER_START, "host="+host)

	profile, err := worker.ComposeProfile()
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to compose profile: %s\n", err.Error())
		os.Exit(1)
	}

	self, err := worker.RegisterWithProfile(conf.SERVER_URL, profile)
	if err != nil {
		fmt.Fprintf(os.Stderr, "fail to register: %s\n", err.Error())
		os.Exit(1)
	}
	core.InitClientProfile(self)
	core.InitProxyWorkChan()

	fmt.Printf("Proxy registered, name=%s, id=%s\n", self.Name, self.Id)
	logger.Event(event.CLIENT_REGISTRATION, "clientid="+self.Id)

	if err := worker.InitWorkers(self); err == nil {
		worker.StartProxyWorkers()
	} else {
		fmt.Printf("failed to initialize and start workers:" + err.Error())
	}

	<-control //block till something dies
}
