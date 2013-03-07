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

type WorkResponse struct {
	Code int      `bson:"S" json:"S"`
	Data Workunit `bson:"D" json:"D"`
	Errs []string `bson:"E" json:"E"`
}

type ClientResponse struct {
	Code int      `bson:"S" json:"S"`
	Data Client   `bson:"D" json:"D"`
	Errs []string `bson:"E" json:"E"`
}

func workStealer(control chan int) {
	fmt.Printf("workStealer lanched, client=%s\n", self.Id)
	defer fmt.Printf("workStealer exiting...\n")
	retry := 0
	for {
		wu, err := CheckoutWorkunitRemote(conf.SERVER_URL)
		if err != nil {
			if err.Error() == e.QueueEmpty || err.Error() == e.NoEligibleWorkunitFound {
				//normal, do nothing
			} else if err.Error() == e.ClientNotFound {
				//server may be restarted, waiting for the hearbeater goroutine to try re-register
			} else if err.Error() == e.ClientSuspended {
				fmt.Printf("client suspended, waiting for repair...\n")
				//to-do: send out email notice that this client has problem and been suspended
				time.Sleep(1 * time.Hour)
				retry += 1
			} else {
				//something is wrong, server may be down
				fmt.Printf("error in checking out workunits: %v\n", err)
				retry += 1
			}
			if retry == 3 {
				os.Exit(1)
			}
			time.Sleep(10 * time.Second)
			continue
		} else {
			retry = 0
		}
		fmt.Printf("workStealer: checked out a workunit: id=%s\n", wu.Id)
		//log event about work checktout (WC)
		Log.Event(EVENT_WORK_CHECKOUT, "workid="+wu.Id)
		self.Total_checkout += 1
		self.Current_work[wu.Id] = true
		workChan <- wu
	}
	control <- 0 //we are ending
}

func CheckoutWorkunitRemote(serverhost string) (workunit *Workunit, err error) {

	response := new(WorkResponse)

	res, err := http.Get(fmt.Sprintf("%s/work?client=%s", serverhost, self.Id))

	Log.Debug(3, fmt.Sprintf("client %s sent a checkout request to %s", self.Id, serverhost))

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
			return nil, errors.New(strings.Join(response.Errs, ","))
		}
		return
	}

	if response.Code == 200 {
		workunit = &response.Data
		return workunit, nil
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
