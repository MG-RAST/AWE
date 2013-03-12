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
	"net/http"
	"os"
	"strings"
	"time"
)

type HeartbeatResponse struct {
	Code int      `bson:"S" json:"S"`
	Data string   `bson:"D" json:"D"`
	Errs []string `bson:"E" json:"E"`
}

type ClientResponse struct {
	Code int      `bson:"S" json:"S"`
	Data Client   `bson:"D" json:"D"`
	Errs []string `bson:"E" json:"E"`
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
	if err = json.Unmarshal(jsonstream, response); err != nil {
		if len(response.Errs) > 0 {
			return errors.New(strings.Join(response.Errs, ","))
		}
		return
	}
	return
}

func Register(host string) (client *Client, err error) {
	var res *http.Response
	serverUrl := fmt.Sprintf("%s/client", host)
	res, err = http.Post(serverUrl, "", strings.NewReader(""))
	if err != nil {
		return
	}

	jsonstream, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	//fmt.Printf("json=%s\n", jsonstream)

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

func RegisterWithProfile(host string, profile_path string) (client *Client, err error) {

	if _, err := os.Stat(profile_path); err != nil {
		return nil, errors.New("profile file not found: " + profile_path)
	}

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
	self_jsonstream, err := json.Marshal(self)
	tmpProfile := conf.DATA_PATH + "/temp.profile"
	ioutil.WriteFile(tmpProfile, []byte(self_jsonstream), 0644)
	client, err = RegisterWithProfile(host, tmpProfile)
	if err != nil {
		Log.Error("Error: fail to re-register, clientid=" + self.Id)
		fmt.Printf("failed to re-register\n")
	} else {
		Log.Event(EVENT_CLIENT_AUTO_REREGI, "clientid="+self.Id)
		fmt.Printf("re-register successfully\n")
	}
	return
}
