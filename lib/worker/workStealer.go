package worker

import (
	"encoding/json"

	"github.com/MG-RAST/AWE/lib/core/cwl"
	//"errors"
	"fmt"

	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"

	//"github.com/MG-RAST/AWE/lib/core/cwl"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/golib/httpclient"

	//"github.com/davecgh/go-spew/spew"
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"time"

	"github.com/mitchellh/mapstructure"
)

type WorkResponse struct {
	core.BaseResponse `bson:",inline" json:",inline" mapstructure:",squash"`
	Data              *core.Workunit `bson:"data" json:"data" mapstructure:"data"`
}

type TokenResponse struct {
	Code int      `bson:"status" json:"status"`
	Data string   `bson:"data" json:"data"`
	Errs []string `bson:"error" json:"error"`
}

func workStealerRun(control chan int, retry_previous int) (retry int, err error) {
	retry = retry_previous

	if core.Service == "proxy" {
		<-core.ProxyWorkChan
	}

	// do not check out work if client is in a bad state
	for core.Self.WorkerState.Healthy == false {
		time.Sleep(time.Second * 10)
	}

	workunit, err := CheckoutWorkunitRemote()
	if err != nil {
		_ = core.Self.SetBusy(false, false)
		if err.Error() == e.QueueEmpty || err.Error() == e.QueueSuspend || err.Error() == e.NoEligibleWorkunitFound {
			//normal, do nothing
			logger.Debug(3, "(workStealer) client %s received status %s from server %s", core.Self.ID, err.Error(), conf.SERVER_URL)
		} else if err.Error() == e.ClientBusy {
			// client asked for work, but server has not finished processing its last delivered work
			logger.Error("(workStealer) server responds: last work delivered by client not yet processed, retry=%d", retry)
			retry += 1
		} else if err.Error() == e.ClientNotFound {
			logger.Error("(workStealer) server responds: client not found. will wait for heartbeat process to fix this")
			//server may be restarted, waiting for the hearbeater goroutine to try re-register
			//ReRegisterWithSelf(conf.SERVER_URL)
			// this has to be done by the heartbeat
		} else if err.Error() == e.ClientSuspended {
			logger.Error("(workStealer) client suspended, waiting for repair or resume request...")
			//TODO: send out email notice that this client has problem and been suspended
			time.Sleep(2 * time.Minute)
		} else if err.Error() == e.ClientDeleted {
			fmt.Printf("(workStealer) client deleted, exiting...\n")
			os.Exit(1) // TODO is there a better way of exiting ? E.g. in regard of the logger who wants to flush....
		} else if err.Error() == e.ServerNotFound {
			logger.Error("(workStealer) ServerNotFound...\n")
			retry += 1
		} else {
			//something is wrong, server may be down
			logger.Error("(workStealer) checking out workunit: %s, retry=%d", err.Error(), retry)
			retry += 1
		}
		if core.Service != "proxy" { //proxy: event driven, client: timer driven
			if retry <= 10 {
				logger.Debug(3, "(workStealer) sleep 10 seconds")
				time.Sleep(10 * time.Second)
			} else {
				logger.Debug(3, "(workStealer) sleep 30 seconds")
				time.Sleep(30 * time.Second)
			}
		}
		return
	} else {
		retry = 0
	}

	work_id := workunit.Workunit_Unique_Identifier
	var work_str string
	work_str, err = work_id.String()
	if err != nil {
		err = fmt.Errorf("(workStealer) work_id.String() returned: %s", err.Error())
		return
	}
	logger.Debug(1, "(workStealer) checked out workunit, id="+work_str)
	//log event about work checktout (WC)
	logger.Event(event.WORK_CHECKOUT, "workid="+work_str)

	err = core.Self.CurrentWork.Add(work_id)
	if err != nil {
		logger.Error("(workStealer) error: %s", err.Error())
		return
	}

	workmap.Set(work_id, ID_WORKSTEALER, "workStealer")

	//hand the work to the next step handler: dataMover
	workstat := core.NewWorkPerf()
	workstat.Checkout = time.Now().Unix()
	//rawWork := &Mediumwork{
	//	Workunit: wu,
	//	Perfstat: workstat,
	//}
	workunit.WorkPerf = workstat

	// make sure cwl-runner is invoked
	if workunit.CWLWorkunit != nil {
		workunit.Cmd.Name = "cwl-runner"
		// "--provenance", "cwl_tool_provenance", "--disable-pull"
		workunit.Cmd.ArgsArray = []string{"--leave-outputs", "--leave-tmpdir", "--tmp-outdir-prefix", "./tmp/", "--tmpdir-prefix", "./tmp/", "--rm-container", "--on-error", "stop", "./cwl_tool.yaml", "./cwl_job_input.yaml"}

	}

	//FromStealer <- rawWork // sends to dataMover
	FromStealer <- workunit // sends to dataMover

	//if worker overlap is inhibited, wait until deliverer finishes processing the workunit
	if conf.WORKER_OVERLAP == false && core.Service != "proxy" {
		chanPermit <- true
		// sleep short time to allow server to finish processing last delivered work
		time.Sleep(2 * time.Second)
	}
	return
}

func workStealer(control chan int) {

	fmt.Printf("workStealer launched, client=%s\n", core.Self.ID)
	logger.Debug(0, fmt.Sprintf("workStealer launched, client=%s\n", core.Self.ID))

	defer fmt.Printf("workStealer exiting...\n")
	retry := 0
	var err error
	for {
		retry, err = workStealerRun(control, retry)
		if err != nil {
			logger.Error("(workStealer) workStealerRun returns: %s", err.Error())
		}
	}
	//control <- ID_WORKSTEALER //we are ending
}

// CheckoutWorkunitRemote _
func CheckoutWorkunitRemote() (workunit *core.Workunit, err error) {
	logger.Debug(3, "(CheckoutWorkunitRemote) start")
	// get available work dir disk space
	var stat syscall.Statfs_t
	syscall.Statfs(conf.WORK_PATH, &stat)
	availableBytes := stat.Bavail * uint64(stat.Bsize)

	if core.Self == nil {
		err = fmt.Errorf("(CheckoutWorkunitRemote) core.Self == nil")
		return
	}
	targeturl := fmt.Sprintf("%s/work?client=%s&available=%d&server_uuid=%s", conf.SERVER_URL, core.Self.ID, availableBytes, core.ServerUUID)

	var headers httpclient.Header
	if conf.CLIENT_GROUP_TOKEN != "" {
		headers = httpclient.Header{
			"Authorization": []string{"CG_TOKEN " + conf.CLIENT_GROUP_TOKEN},
		}
	}
	logger.Debug(3, fmt.Sprintf("(CheckoutWorkunitRemote) client %s sends a checkout request to %s with available %d (targeturl = %s)", core.Self.ID, conf.SERVER_URL, availableBytes, targeturl))
	res, err := httpclient.DoTimeout("GET", targeturl, headers, nil, nil, time.Second*0)
	logger.Debug(3, fmt.Sprintf("(CheckoutWorkunitRemote) client %s sent a checkout request to %s with available %d", core.Self.ID, conf.SERVER_URL, availableBytes))
	if err != nil {
		err = fmt.Errorf("(CheckoutWorkunitRemote) error sending checkout request: %s", err.Error())
		return
	}
	defer res.Body.Close()

	if res.StatusCode == 404 {
		err = fmt.Errorf("(404) AWE server not found")
		return
	}

	jsonstream, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return
	}

	if len(jsonstream) == 0 {
		err = fmt.Errorf("response is empty")
		return
	}

	var response core.StandardResponse
	response.Status = -1

	err = json.Unmarshal(jsonstream, &response)
	if err != nil { // response
		jsonstreamStr := string(jsonstream)
		fmt.Printf("statuscode: %d\njsonstream:\n%s (len: %d) (%s)\n", res.StatusCode, jsonstreamStr, len(jsonstreamStr), err.Error())
		err = fmt.Errorf("(CheckoutWorkunitRemote) json.Unmarshal error: %s", err.Error())
		return
	}

	//spew.Dump(response)

	if response.Status == -1 {
		err = fmt.Errorf(e.ServerNotFound)
		return
	}

	if len(response.Error) > 0 {
		message := strings.Join(response.Error, ",")
		err = fmt.Errorf("%s", message)
		return
	}

	data_generic := response.Data
	if data_generic == nil {
		err = fmt.Errorf("(CheckoutWorkunitRemote) Data field missing (jsonstream:\n%s\n)", jsonstream)
		return
	}

	// remove CWL
	data_map, ok := data_generic.(map[string]interface{})
	if !ok {
		err = fmt.Errorf("(CheckoutWorkunitRemote) Could not make data field map[string]interface{}")
		return
	}
	var cwl_object *core.CWLWorkunit

	cwl_generic, has_cwl := data_map["cwl"]
	if has_cwl {
		if cwl_generic == nil {
			has_cwl = false
		}
		delete(data_map, "cwl")

	}

	info := &core.Info{}
	info_if, has_info := data_map["info"]
	if has_info { // interface -> json -> struct  // this is a bit ugly

		info_byte, xerr := json.Marshal(info_if)
		if xerr != nil {
			err = xerr
			return
		}

		yerr := json.Unmarshal(info_byte, info)
		if yerr != nil {
			err = yerr
			return
		}
		delete(data_map, "info")

	}

	command := &core.Command{}
	command_if, has_command := data_map["command"]
	if has_command { // interface -> json -> struct  // this is a bit ugly

		command_byte, xerr := json.Marshal(command_if)
		if xerr != nil {
			err = xerr
			return
		}

		yerr := json.Unmarshal(command_byte, command)
		if yerr != nil {
			err = yerr
			return
		}
		delete(data_map, "command")

	}

	partinfo := &core.PartInfo{}
	partinfo_if, has_partinfo := data_map["partinfo"]
	if has_partinfo { // interface -> json -> struct  // this is a bit ugly

		partinfo_byte, xerr := json.Marshal(partinfo_if)
		if xerr != nil {
			err = xerr
			return
		}

		yerr := json.Unmarshal(partinfo_byte, partinfo)
		if yerr != nil {
			err = yerr
			return
		}
		delete(data_map, "partinfo")

	}

	_, has_checkout_time := data_map["checkout_time"]
	if has_checkout_time {
		delete(data_map, "checkout_time") // TODO add checkout_time as time.Time

	}

	//delete(data_map, "info")

	workunit = &core.Workunit{}
	workunit.Info = info
	//workunit.Workunit_Unique_Identifier = core.Workunit_Unique_Identifier{}
	//if has_checkout_time {
	//	workunit_checkout_time_str, ok := workunit_checkout_time_if.(string)
	//	if !ok {
	//		err = fmt.Errorf("(CheckoutWorkunitRemote) cannot type assert checkout_time")
	//		return
	//	}
	//	workunit.CheckoutTime = workunit_checkout_time
	//}

	err = mapstructure.Decode(data_map, workunit)
	if err != nil {
		err = fmt.Errorf("(CheckoutWorkunitRemote) mapstructure.Decode error: %s", err.Error())
		return
	}

	logger.Debug(1, "(CheckoutWorkunitRemote) workunit.ID: %s", workunit.ID)

	if has_cwl {

		var xerr error
		//var schemata []cwl.CWLType_Type
		workunit.Context = cwl.NewWorkflowContext()
		workunit.Context.Init("")
		cwl_object, _, xerr = core.NewCWLWorkunitFromInterface(cwl_generic, "", workunit.Context)
		if xerr != nil {
			err = fmt.Errorf("(CheckoutWorkunitRemote) NewCWLWorkunit_from_interface failed: %s", xerr.Error())
			logger.Debug(1, err.Error())
			workunit.State = core.WORK_STAT_ERROR
			workunit.Notes = append(workunit.Notes, err.Error())
			err = nil // Pass error-workunit along to maintain error message
			return
		}

		//fmt.Println("CWL Workunit from server from server:")
		//cwl_object_bytes, _ := json.Marshal(cwl_object)
		//fmt.Printf("%s\n", string(cwl_object_bytes))

		workunit.CWLWorkunit = cwl_object
		workunit.CWLWorkunit.Notice = core.Notice{ID: workunit.Workunit_Unique_Identifier, WorkerID: core.Self.ID}

		if workunit.CWLWorkunit.Tool == nil {
			err = fmt.Errorf("(CheckoutWorkunitRemote) Tool == nil")
			return
		}
	}

	//test, err := json.Marshal(workunit)
	//if err != nil {
	//	panic("did not work")
	//}
	//fmt.Println("workunit: ")
	//fmt.Printf("workunit:\n %s\n", test)

	//panic("done...")

	if response.Status == 0 { // this is ugly
		err = fmt.Errorf(e.ServerNotFound)
		return
	}

	if response.Status != 200 {
		err = fmt.Errorf("(CheckoutWorkunitRemote) response_generic.Status != 200 : %d", response.Status)
		return
	}

	if workunit.TaskName == "" {
		err = fmt.Errorf("(CheckoutWorkunitRemote) TaskName empty !")
		return
	}
	logger.Debug(3, "(CheckoutWorkunitRemote) TaskName: %s", workunit.TaskName)

	//workunit = response.Data

	logger.Debug(3, "(CheckoutWorkunitRemote) workunit.Info.Auth == %t", workunit.Info.Auth)
	if workunit.Info.Auth == true {

		var token string
		token, err = workunit.FetchDataToken()
		if err != nil {
			err = fmt.Errorf("(CheckoutWorkunitRemote) need data token but failed to fetch it: %s", err.Error())
			return
		}
		logger.Debug(3, "(CheckoutWorkunitRemote) token length: %d", len(token))
		if token != "" {
			workunit.Info.DataToken = token
		}
	}

	logger.Debug(3, "(CheckoutWorkunitRemote) workunit id: %s", workunit.ID)

	logger.Debug(3, "(CheckoutWorkunitRemote) workunit Rank:%d TaskId:%s JobId:%s", workunit.Rank, workunit.TaskName, workunit.JobId)

	logger.Debug(3, fmt.Sprintf("(CheckoutWorkunitRemote) client %s got a workunit", core.Self.ID))
	workunit.State = core.WORK_STAT_CHECKOUT
	err = core.Self.SetBusy(true, false)
	if err != nil {
		return
	}
	return
}

// CheckoutTokenByJobId _
func CheckoutTokenByJobId(jobid string) (token string, err error) {
	return
}
