package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	e "github.com/MG-RAST/AWE/errors"
	. "github.com/MG-RAST/AWE/logger"
	"io"
	"io/ioutil"
	"mime/multipart"
	"net"
	"net/http"
	"os"
	"runtime"
	"strings"
	"time"
)

type HeartbeatResponse struct {
	Code int      `bson:"status" json:"status"`
	Data string   `bson:"data" json:"data"`
	Errs []string `bson:"error" json:"error"`
}

type ClientResponse struct {
	Code int      `bson:"status" json:"status"`
	Data Client   `bson:"data" json:"data"`
	Errs []string `bson:"error" json:"error"`
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
	if err := heartbeating(conf.SERVER_URL, self.Id); err != nil {
		if err.Error() == e.ClientNotFound {
			ReRegisterWithSelf(conf.SERVER_URL)
		}
	}
}

func heartbeating(host string, clientid string) (err error) {
	response := new(HeartbeatResponse)
	res, err := http.Get(fmt.Sprintf("%s/client/%s?heartbeat", host, clientid))
	Log.Debug(3, fmt.Sprintf("client %s sent a heartbeat to %s", host, clientid))
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
			return errors.New(strings.Join(response.Errs, ","))
		}
		return
	}
	return
}

func RegisterWithProfile(host string, profile *Client) (client *Client, err error) {
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
		if len(response.Errs) > 0 {
			//or if err.Error() == "json: cannot unmarshal null into Go value of type core.Client"
			return nil, errors.New(strings.Join(response.Errs, ","))
		}
		return
	}
	client = &response.Data
	return
}

func ReRegisterWithSelf(host string) (client *Client, err error) {
	fmt.Printf("lost contact with server, try to re-register\n")
	client, err = RegisterWithProfile(host, self)
	if err != nil {
		Log.Error("Error: fail to re-register, clientid=" + self.Id)
		fmt.Printf("failed to re-register\n")
	} else {
		Log.Event(EVENT_CLIENT_AUTO_REREGI, "clientid="+self.Id)
		fmt.Printf("re-register successfully\n")
	}
	return
}

func ComposeProfile() (profile *Client, err error) {
	profile = new(Client)
	profile.Name = conf.CLIENT_NAME
	profile.Group = conf.CLIENT_GROUP
	profile.CPUs = runtime.NumCPU()

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
	return
}
