package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	e "github.com/MG-RAST/AWE/errors"
	. "github.com/MG-RAST/AWE/logger"
	"io/ioutil"
	"net/http"
	"strings"
	"time"
)

type HeartbeatResponse struct {
	Code int      `bson:"S" json:"S"`
	Data string   `bson:"D" json:"D"`
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
