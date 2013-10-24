package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/httpclient"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"runtime"
	"strconv"
	"strings"
	"time"
)

type HeartbeatResponse struct {
	Code int        `bson:"status" json:"status"`
	Data core.HBmsg `bson:"data" json:"data"`
	Errs []string   `bson:"error" json:"error"`
}

type ClientResponse struct {
	Code int         `bson:"status" json:"status"`
	Data core.Client `bson:"data" json:"data"`
	Errs []string    `bson:"error" json:"error"`
}

func heartBeater(control chan int) {
	for {
		time.Sleep(10 * time.Second)
		SendHeartBeat()
	}
	control <- 2 //we are ending
}

//client sends heartbeat to server to maintain active status and re-register when needed
func SendHeartBeat() {
	hbmsg, err := heartbeating(conf.SERVER_URL, core.Self.Id)
	if err != nil {
		if err.Error() == e.ClientNotFound {
			ReRegisterWithSelf(conf.SERVER_URL)
		}
	}
	//handle requested ops from the server
	for op, objs := range hbmsg {
		if op == "discard" { //discard suspended workunits
			suspendedworks := strings.Split(objs, ",")
			for _, work := range suspendedworks {
				DiscardWorkunit(work)
			}
		} else if op == "restart" {
			RestartClient()
		} else if op == "stop" {
			StopClient()
		} else if op == "clean" {
			CleanDisk()
		}
	}
}

func heartbeating(host string, clientid string) (msg core.HBmsg, err error) {
	response := new(HeartbeatResponse)
	targeturl := fmt.Sprintf("%s/client/%s?heartbeat", host, clientid)
	//res, err := http.Get(targeturl)
	res, err := httpclient.Get(targeturl, httpclient.Header{}, nil, nil)
	logger.Debug(3, fmt.Sprintf("client %s sent a heartbeat to %s", host, clientid))
	if err != nil {
		return
	}
	defer res.Body.Close()
	jsonstream, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}
	if err = json.Unmarshal(jsonstream, response); err == nil {
		if len(response.Errs) > 0 {
			return msg, errors.New(strings.Join(response.Errs, ","))
		}
		return response.Data, nil
	}
	return
}

func RegisterWithProfile(host string, profile *core.Client) (client *core.Client, err error) {
	profile_jsonstream, err := json.Marshal(profile)
	profile_path := conf.DATA_PATH + "/clientprofile.json"
	ioutil.WriteFile(profile_path, []byte(profile_jsonstream), 0644)

	bodyBuf := &bytes.Buffer{}
	bodyWriter := multipart.NewWriter(bodyBuf)
	fileWriter, err := bodyWriter.CreateFormFile("profile", profile_path)
	if err != nil {
		return nil, err
	}
	fh, err := os.Open(profile_path)
	if err != nil {
		return nil, err
	}
	_, err = io.Copy(fileWriter, fh)
	if err != nil {
		return nil, err
	}
	contentType := bodyWriter.FormDataContentType()
	bodyWriter.Close()
	targetUrl := host + "/client"

	resp, err := http.Post(targetUrl, contentType, bodyBuf)

	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()

	jsonstream, err := ioutil.ReadAll(resp.Body)

	response := new(ClientResponse)
	if err = json.Unmarshal(jsonstream, response); err != nil {
		return nil, errors.New("fail to unmashal response:" + string(jsonstream))
	}
	if len(response.Errs) > 0 {
		return nil, errors.New(strings.Join(response.Errs, ","))
	}
	client = &response.Data
	return
}

func RegisterWithAuth(host string, profile *core.Client) (client *core.Client, err error) {
	if conf.CLIENT_USERNAME == "public" {
		fmt.Println("client username and password not configured, register as a public user (can only access public data)")
	}

	profile_jsonstream, err := json.Marshal(profile)
	profile_path := conf.DATA_PATH + "/clientprofile.json"
	ioutil.WriteFile(profile_path, []byte(profile_jsonstream), 0644)

	form := httpclient.NewForm()
	form.AddFile("profile", profile_path)
	if err := form.Create(); err != nil {
		return nil, err
	}
	headers := httpclient.Header{
		"Content-Type":   form.ContentType,
		"Content-Length": strconv.FormatInt(form.Length, 10),
	}
	user := httpclient.GetUserByBasicAuth(conf.CLIENT_USERNAME, conf.CLIENT_PASSWORD)
	targetUrl := host + "/client"

	resp, err := httpclient.Post(targetUrl, headers, form.Reader, user)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	jsonstream, err := ioutil.ReadAll(resp.Body)
	response := new(ClientResponse)
	if err = json.Unmarshal(jsonstream, response); err != nil {
		return nil, errors.New("fail to unmashal response:" + string(jsonstream))
	}
	if len(response.Errs) > 0 {
		return nil, errors.New(strings.Join(response.Errs, ","))
	}
	client = &response.Data
	return
}

func ReRegisterWithSelf(host string) (client *core.Client, err error) {
	fmt.Printf("lost contact with server, try to re-register\n")
	client, err = RegisterWithAuth(host, core.Self)
	if err != nil {
		logger.Error("Error: fail to re-register, clientid=" + core.Self.Id)
		fmt.Printf("failed to re-register\n")
	} else {
		logger.Event(event.CLIENT_AUTO_REREGI, "clientid="+core.Self.Id)
		fmt.Printf("re-register successfully\n")
	}
	return
}

func ComposeProfile() (profile *core.Client, err error) {
	profile = new(core.Client)
	profile.Name = conf.CLIENT_NAME
	profile.Group = conf.CLIENT_GROUP
	profile.CPUs = runtime.NumCPU()
	profile.User = conf.CLIENT_USERNAME

	//app list
	profile.Apps = []string{}
	if conf.SUPPORTED_APPS != "" { //apps configured in .cfg
		apps := strings.Split(conf.SUPPORTED_APPS, ",")
		for _, item := range apps {
			profile.Apps = append(profile.Apps, item)
		}
	} else { //apps not configured in .cfg, check the executables in APP_PATH)
		if files, err := ioutil.ReadDir(conf.APP_PATH); err == nil {
			for _, item := range files {
				profile.Apps = append(profile.Apps, item.Name())
			}
		}
	}

	if addrs, err := net.InterfaceAddrs(); err == nil {
		for _, a := range addrs {
			if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && len(strings.Split(ipnet.IP.String(), ".")) == 4 {
				profile.Host = ipnet.IP.String()
				break
			}
		}
	}
	if core.Service == "proxy" {
		profile.Proxy = true
	}
	return
}

func DiscardWorkunit(id string) (err error) {
	fmt.Printf("try to discard workunit %s\n", id)
	if stage, ok := workmap[id]; ok {
		if stage == ID_WORKER {
			chankill <- true
		}
	}
	return
}

func RestartClient() (err error) {
	fmt.Printf("try to restart client\n")
	//to-do: implementation here
	return
}

func StopClient() (err error) {
	fmt.Printf("try to stop client\n")
	//to-do: implementation here
	return
}

func CleanDisk() (err error) {
	fmt.Printf("try to clean disk space\n")
	//to-do: implementation here
	return
}
