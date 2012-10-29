package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"github.com/jaredwilkening/goweb"
	"net"
	"net/http"
	"strings"
	"time"
)

var (
	logo = "\n" +
		" +--------------+  +----+   +----+   +----+  +--------------+\n" +
		" |              |  |    |   |    |   |    |  |              |\n" +
		" |    +----+    |  |    |   |    |   |    |  |    +---------+\n" +
		" |    |    |    |  |    |   |    |   |    |  |    |          \n" +
		" |    +----+    |  |    |   |    |   |    |  |    +---------+\n" +
		" |              |  |    |   |    |   |    |  |              |\n" +
		" |    +----+    |  |    |   |    |   |    |  |    +---------+\n" +
		" |    |    |    |  |    \\---/    \\---/    |  |    |          \n" +
		" |    |    |    |  |                      |  |    +---------+\n" +
		" |    |    |    |   \\        /---\\       /   |              |\n" +
		" +----+    +----+     \\-----/     \\-----/    +--------------+\n"
)

func printLogo() {
	fmt.Println(logo)
	return
}

func Site(cx *goweb.Context) {
	LogRequest(cx.Request)
	http.ServeFile(cx.ResponseWriter, cx.Request, conf.SITE_PATH+"/pages/main.html")
}

func LogRequest(req *http.Request) {
	host, _, _ := net.SplitHostPort(req.RemoteAddr)
	prefix := fmt.Sprintf("%s [%s]", host, time.Now().Format(time.RFC1123))
	suffix := ""
	if _, auth := req.Header["Authorization"]; auth {
		suffix = "AUTH"
	}
	url := ""
	if req.URL.RawQuery != "" {
		url = fmt.Sprintf("%s %s?%s", req.Method, req.URL.Path, req.URL.RawQuery)
	} else {
		url = fmt.Sprintf("%s %s", req.Method, req.URL.Path)
	}
	fmt.Printf("%s %q %s\n", prefix, url, suffix)
}

func RawDir(cx *goweb.Context) {
	LogRequest(cx.Request)
	http.ServeFile(cx.ResponseWriter, cx.Request, fmt.Sprintf("%s%s", conf.DATA_PATH, cx.Request.URL.Path))
}

func AssetsDir(cx *goweb.Context) {
	LogRequest(cx.Request)
	http.ServeFile(cx.ResponseWriter, cx.Request, conf.SITE_PATH+cx.Request.URL.Path)
}

type resource struct {
	R []string `json:"resources"`
	U string   `json:"url"`
	D string   `json:"documentation"`
	C string   `json:"contact"`
	I string   `json:"id"`
	T string   `json:"type"`
}

func ResourceDescription(cx *goweb.Context) {
	LogRequest(cx.Request)
	host := ""
	if strings.Contains(cx.Request.Host, ":") {
		split := strings.Split(cx.Request.Host, ":")
		host = split[0]
	} else {
		host = cx.Request.Host
	}
	r := resource{
		R: []string{"job", "task", "user"},
		U: "http://" + host + ":" + fmt.Sprint(conf.API_PORT) + "/",
		D: "http://" + host + ":" + fmt.Sprint(conf.SITE_PORT) + "/",
		C: conf.ADMIN_EMAIL,
		I: "AWE",
		T: "AWE",
	}
	cx.WriteResponse(r, 200)
}
