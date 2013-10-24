package worker

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/httpclient"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"io/ioutil"
	"os"
	"strings"
	"time"
)

type WorkResponse struct {
	Code int            `bson:"status" json:"status"`
	Data *core.Workunit `bson:"data" json:"data"`
	Errs []string       `bson:"error" json:"error"`
}

type TokenResponse struct {
	Code int      `bson:"status" json:"status"`
	Data string   `bson:"data" json:"data"`
	Errs []string `bson:"error" json:"error"`
}

func workStealer(control chan int) {
	fmt.Printf("workStealer lanched, client=%s\n", core.Self.Id)
	defer fmt.Printf("workStealer exiting...\n")
	retry := 0
	for {
		if core.Service == "proxy" {
			<-core.ProxyWorkChan
		}
		wu, err := CheckoutWorkunitRemote()
		if err != nil {
			if err.Error() == e.QueueEmpty || err.Error() == e.NoEligibleWorkunitFound {
				//normal, do nothing
			} else if err.Error() == e.ClientNotFound {
				//server may be restarted, waiting for the hearbeater goroutine to try re-register
				ReRegisterWithSelf(conf.SERVER_URL)
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
			if core.Service != "proxy" { //proxy: event driven, client: timer driven
				time.Sleep(10 * time.Second)
			}
			continue
		} else {
			retry = 0
		}
		logger.Debug(2, "workStealer: checked out a workunit: id="+wu.Id)
		//log event about work checktout (WC)
		logger.Event(event.WORK_CHECKOUT, "workid="+wu.Id)
		core.Self.Total_checkout += 1
		core.Self.Current_work[wu.Id] = true
		workmap[wu.Id] = ID_WORKSTEALER

		//hand the work to the next step handler: dataMover
		workstat := core.NewWorkPerf(wu.Id)
		workstat.Checkout = time.Now().Unix()
		rawWork := &mediumwork{
			workunit: wu,
			perfstat: workstat,
		}
		fromStealer <- rawWork

		//if worker overlap is inhibited, wait until deliverer finishes processing the workunit
		if conf.WORKER_OVERLAP == false && core.Service != "proxy" {
			chanPermit <- true
		}
	}
	control <- ID_WORKSTEALER //we are ending
}

func CheckoutWorkunitRemote() (workunit *core.Workunit, err error) {
	response := new(WorkResponse)
	targeturl := fmt.Sprintf("%s/work?client=%s", conf.SERVER_URL, core.Self.Id)
	//res, err := http.Get(fmt.Sprintf("%s/work?client=%s", conf.SERVER_URL, core.Self.Id))
	res, err := httpclient.Get(targeturl, httpclient.Header{}, nil, nil)
	logger.Debug(3, fmt.Sprintf("client %s sent a checkout request to %s", core.Self.Id, conf.SERVER_URL))
	if err != nil {
		fmt.Printf("err=%s\n", err.Error())
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
		if workunit.Info.Auth == true {
			if token, err := FetchDataTokenByWorkId(workunit.Id); err == nil {
				workunit.Info.DataToken = token
			} else {
				return workunit, errors.New("need data token but failed to fetch one")
			}
		}
		return workunit, nil
	}
	return
}

func FetchDataTokenByWorkId(workid string) (token string, err error) {
	response := new(TokenResponse)
	targeturl := fmt.Sprintf("%s/work/%s?datatoken&client=%s", conf.SERVER_URL, workid, core.Self.Id)
	//res, err := http.Get(requrl)
	res, err := httpclient.Get(targeturl, httpclient.Header{}, nil, nil)
	logger.Debug(3, "GET:"+targeturl)
	if err != nil {
		return
	}
	defer res.Body.Close()
	jsonstream, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return "", err
	}
	if err = json.Unmarshal(jsonstream, response); err != nil {
		return
	}
	if len(response.Errs) > 0 {
		return "", errors.New(strings.Join(response.Errs, ","))
	}
	if response.Code == 200 {
		token = response.Data
		return token, nil
	}
	return
}

func CheckoutTokenByJobId(jobid string) (token string, err error) {
	return
}
