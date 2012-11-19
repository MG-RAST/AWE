package main

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	"github.com/MG-RAST/AWE/core"
	"github.com/jaredwilkening/goweb"
	"math/rand"
	"net"
	"net/http"
	"os"
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

type Query struct {
	list map[string][]string
}

func (q *Query) Has(key string) bool {
	if _, has := q.list[key]; has {
		return true
	}
	return false
}

func (q *Query) Value(key string) string {
	return q.list[key][0]
}

func (q *Query) List(key string) []string {
	return q.list[key]
}

func (q *Query) All() map[string][]string {
	return q.list
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
		R: []string{"job", "work", "user"},
		U: "http://" + host + ":" + fmt.Sprint(conf.API_PORT) + "/",
		D: "http://" + host + ":" + fmt.Sprint(conf.SITE_PORT) + "/",
		C: conf.ADMIN_EMAIL,
		I: "AWE",
		T: "AWE",
	}
	cx.WriteResponse(r, 200)
}

// helper function for create & update
func ParseMultipartForm(r *http.Request) (params map[string]string, files core.FormFiles, err error) {
	params = make(map[string]string)
	files = make(core.FormFiles)

	reader, err := r.MultipartReader()
	if err != nil {
		return
	}
	for {
		if part, err := reader.NextPart(); err == nil {

			if part.FileName() == "" {
				buffer := make([]byte, 32*1024)
				n, err := part.Read(buffer)
				if n == 0 || err != nil {
					break
				}
				params[part.FormName()] = fmt.Sprintf("%s", buffer[0:n])
			} else {

				tmpPath := fmt.Sprintf("%s/temp/%d%d", conf.DATA_PATH, rand.Int(), rand.Int())
				files[part.FormName()] = core.FormFile{Name: part.FileName(), Path: tmpPath, Checksum: make(map[string]string)}
				if tmpFile, err := os.Create(tmpPath); err == nil {
					buffer := make([]byte, 32*1024)
					for {
						n, err := part.Read(buffer)
						if n == 0 || err != nil {
							break
						}
						tmpFile.Write(buffer[0:n])
					}
					tmpFile.Close()
				} else {
					return nil, nil, err
				}
			}
		} else if err.Error() != "EOF" {
			fmt.Println("err here")
			return nil, nil, err
		} else {
			break
		}
	}

	return
}
