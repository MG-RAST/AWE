package controller

import (
	"encoding/json"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/request"
	"github.com/MG-RAST/golib/goweb"
	"io"
	"math/rand"
	"mime/multipart"
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

func PrintLogo() {
	fmt.Println(logo)
	return
}

type Query struct {
	Li map[string][]string
}

func (q *Query) Has(key string) bool {
	if _, has := q.Li[key]; has {
		return true
	}
	return false
}

func (q *Query) Value(key string) string {
	return q.Li[key][0]
}

func (q *Query) List(key string) []string {
	return q.Li[key]
}

func (q *Query) All() map[string][]string {
	return q.Li
}

func (q *Query) Empty() bool {
	if len(q.Li) == 0 {
		return true
	}
	return false
}

func LogRequest(req *http.Request) {
	host, _, _ := net.SplitHostPort(req.RemoteAddr)
	//	prefix := fmt.Sprintf("%s [%s]", host, time.Now().Format(time.RFC1123))
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
	logger.Log.Access(host + " \"" + url + suffix + "\"")
}

func RawDir(cx *goweb.Context) {
	LogRequest(cx.Request)
	http.ServeFile(cx.ResponseWriter, cx.Request, fmt.Sprintf("%s/%s", conf.DATA_PATH, cx.Request.URL.Path))
}

func SiteDir(cx *goweb.Context) {
	LogRequest(cx.Request)
	if cx.Request.URL.Path == "/" {
		http.ServeFile(cx.ResponseWriter, cx.Request, conf.SITE_PATH+"/main.html")
	} else {
		http.ServeFile(cx.ResponseWriter, cx.Request, conf.SITE_PATH+cx.Request.URL.Path)
	}
}

type resource struct {
	R             []string `json:"resources"`
	F             []string `json:"info_indexes"`
	U             string   `json:"url"`
	D             string   `json:"documentation"`
	Title         string   `json:"title"` // title to show in AWE monitor
	C             string   `json:"contact"`
	I             string   `json:"id"`
	T             string   `json:"type"`
	S             string   `json:"queue_status"`
	V             string   `json:"version"`
	Time          string   `json:"server_time"`
	GitCommitHash string   `json:"git_commit_hash"`
}

func ResourceDescription(cx *goweb.Context) {
	LogRequest(cx.Request)
	r := resource{
		R:             []string{},
		F:             core.JobInfoIndexes,
		U:             apiUrl(cx) + "/",
		D:             siteUrl(cx) + "/",
		Title:         conf.TITLE,
		C:             conf.ADMIN_EMAIL,
		I:             "AWE",
		T:             core.Service,
		S:             core.QMgr.QueueStatus(),
		V:             conf.VERSION,
		Time:          time.Now().String(),
		GitCommitHash: conf.GIT_COMMIT_HASH,
	}
	if core.Service == "server" {
		r.R = []string{"job", "work", "client", "queue", "awf", "event"}
	} else if core.Service == "proxy" {
		r.R = []string{"client", "work"}
	}

	cx.WriteResponse(r, 200)
	return
}

func apiUrl(cx *goweb.Context) string {
	if conf.API_URL != "" {
		return conf.API_URL
	}
	return "http://" + cx.Request.Host
}

func siteUrl(cx *goweb.Context) string {
	if conf.SITE_URL != "" {
		return conf.SITE_URL
	} else if strings.Contains(cx.Request.Host, ":") {
		return fmt.Sprintf("http://%s:%d", strings.Split(cx.Request.Host, ":")[0], conf.SITE_PORT)
	}
	return "http://" + cx.Request.Host
}

// helper function for create & update
func ParseMultipartForm(r *http.Request) (params map[string]string, files core.FormFiles, err error) {
	params = make(map[string]string)
	files = make(core.FormFiles)

	reader, xerr := r.MultipartReader()
	if xerr != nil {
		err = fmt.Errorf("(ParseMultipartForm) MultipartReader not created: %s", xerr.Error())
		return
	}
	for {
		var part *multipart.Part
		part, err = reader.NextPart()
		if err != nil {
			if err == io.EOF {
				err = nil
				break
			}
			err = fmt.Errorf("(ParseMultipartForm) reader.NextPart() error: %s", err.Error())
			return
		}

		if part.FileName() == "" {
			buffer := make([]byte, 32*1024)
			n, err := part.Read(buffer)
			if n == 0 {
				break
			}
			if err != nil {
				if err == io.EOF {
					err = nil
					break
				}
				err = fmt.Errorf("(ParseMultipartForm) part.Read(buffer) error: %s", err.Error())
				return nil, nil, err
			}

			//buf_len := 50
			//if n < 50 {
			//	buf_len = n
			//}
			//logger.Debug(3, "FormName: %s Content: %s", part.FormName(), buffer[0:buf_len])

			params[part.FormName()] = fmt.Sprintf("%s", buffer[0:n])
		} else {

			tmpPath := fmt.Sprintf("%s/temp/%d%d", conf.DATA_PATH, rand.Int(), rand.Int())
			//logger.Debug(3, "FormName: %s tmpPath: %s", part.FormName(), tmpPath)
			files[part.FormName()] = core.FormFile{Name: part.FileName(), Path: tmpPath, Checksum: make(map[string]string)}
			bytes_written := 0
			var tmpFile *os.File
			tmpFile, err = os.Create(tmpPath)
			if err != nil {
				err = fmt.Errorf("(ParseMultipartForm) os.Create(tmpPath) error: %s", err.Error())
				return nil, nil, err
			}

			last_loop := false
			buffer := make([]byte, 32*1024)
			for {
				n := 0

				n, err = part.Read(buffer)
				//logger.Debug(3, "read from part: %d", n)
				if err != nil {
					//logger.Debug(3, "err != nil")
					if err == io.EOF {
						err = nil
						last_loop = true
					} else {
						err = fmt.Errorf("part.Read(buffer) error: %s", err.Error())
						return
					}

				}
				//logger.Debug(3, "after reading.... n: %d", n)
				if n == 0 {
					break
				}
				bytes_written += n
				//logger.Debug(3, "after reading, bytes_written: %d", bytes_written)
				m := 0
				m, err = tmpFile.Write(buffer[0:n])
				if err != nil {
					err = fmt.Errorf("(ParseMultipartForm) tmpFile.Write error: %s", err.Error())
					return
				}
				if m != n {
					err = fmt.Errorf("(ParseMultipartForm) m != n ")
					return
				}
				if last_loop {
					break
				}
			}
			tmpFile.Close()

			//logger.Debug(3, "FormName: %s bytes_written: %d", part.FormName(), bytes_written)
		}

	}

	return
}

func RespondTokenInHeader(cx *goweb.Context, token string) {
	cx.ResponseWriter.Header().Set("Datatoken", token)
	cx.Respond(nil, http.StatusOK, nil, cx)
	return
}

func RespondPrivateEnvInHeader(cx *goweb.Context, Envs map[string]string) (err error) {
	env_stream, err := json.Marshal(Envs)
	if err != nil {
		return err
	}
	cx.ResponseWriter.Header().Set("Privateenv", string(env_stream[:]))
	cx.Respond(nil, http.StatusOK, nil, cx)
	return
}

func AdminAuthenticated(cx *goweb.Context) bool {
	user, err := request.Authenticate(cx.Request)
	if err != nil {
		if err.Error() == e.NoAuth || err.Error() == e.UnAuth {
			cx.RespondWithError(http.StatusUnauthorized)
		} else {
			request.AuthError(err, cx)
		}
		return false
	}
	if _, ok := conf.Admin_Users[user.Username]; !ok {
		msg := fmt.Sprintf("user %s has no admin right", user.Username)
		cx.RespondWithErrorMessage(msg, http.StatusBadRequest)
		return false
	}
	return true
}
