package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"github.com/jaredwilkening/goweb"
)

func main() {
	fmt.Printf("%s\n######### Conf #########\nmongodb:\t%s\nsecretkey:\t%s\nsite-port:\t%d\napi-port:\t%d\n\n####### Starting #######\n",
		logo,
		conf.MONGODB,
		conf.SECRETKEY,
		conf.SITEPORT,
		conf.APIPORT,
	)

	c := make(chan int)
	goweb.ConfigureDefaultFormatters()
	// start site
	go func() {
		r := &goweb.RouteManager{}
		r.MapFunc("*", Site)
		c <- 1
		goweb.ListenAndServeRoutes(fmt.Sprintf(":%d", conf.SITEPORT), r)
		c <- 1
	}()
	<-c
	fmt.Printf("site :%d... running\n", conf.SITEPORT)
	fmt.Printf("\n######### Log  #########\n")
	<-c
}
