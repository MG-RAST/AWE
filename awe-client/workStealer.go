package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	. "github.com/MG-RAST/AWE/core"
	e "github.com/MG-RAST/AWE/errors"
	. "github.com/MG-RAST/AWE/logger"
	"io/ioutil"
	"net/http"
	"os"
	"strings"
	"time"
)

type WorkResponse struct {
	Code int       `bson:"status" json:"status"`
	Data *Workunit `bson:"data" json:"data"`
	Errs []string  `bson:"error" json:"error"`
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
				time.Sleep(2 * time.Hour)
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
		Log.Debug(2, "workStealer: checked out a workunit: id="+wu.Id)
		//log event about work checktout (WC)
		Log.Event(EVENT_WORK_CHECKOUT, "workid="+wu.Id)
		self.Total_checkout += 1
		self.Current_work[wu.Id] = true

		//hand the work to the next step handler: dataMover
		workstat := NewWorkPerf(wu.Id)
		workstat.Checkout = time.Now().Unix()
		rawWork := &rawWork{
			workunit: wu,
			perfstat: workstat,
		}
		chanRaw <- rawWork

		//if worker overlap is inhibited, wait until deliverer finishes processing the workunit
		if conf.WORKER_OVERLAP == false {
			chanPermit <- true
		}
	}
	control <- ID_WORKSTEALER //we are ending
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
		return nil, err
	}

	if err = json.Unmarshal(jsonstream, response); err != nil {
		return
	}

	if len(response.Errs) > 0 {
		return nil, errors.New(strings.Join(response.Errs, ","))
	}

	if response.Code == 200 {
		workunit = response.Data
		return workunit, nil
	}
	return
}
