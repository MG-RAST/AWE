package worker

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	"github.com/MG-RAST/AWE/lib/core"
	e "github.com/MG-RAST/AWE/lib/errors"
	"github.com/MG-RAST/AWE/lib/logger"
	"github.com/MG-RAST/AWE/lib/logger/event"
	"github.com/MG-RAST/golib/httpclient"
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
	fmt.Printf("heartBeater launched, client=%s\n", core.Self.Id)
	logger.Debug(0, fmt.Sprintf("heartBeater launched, client=%s\n", core.Self.Id))
	defer fmt.Printf("heartBeater exiting...\n")

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
		logger.Debug(3, "(SendHeartBeat) heartbeat returned error: "+err.Error())
	}

	val, ok := hbmsg["server-uuid"]
	if ok {
		if len(val) > 0 {
			if core.Server_UUID == "" {
				logger.Debug(1, "(SendHeartBeat) Setting Server UUID to %s", val)
				core.Server_UUID = val
			} else {
				if core.Server_UUID != val {
					// server has been restarted, stop work on client (TODO in future we will try to recover work)
					logger.Warning("(SendHeartBeat) Server UUID has changed (%s -> %s). Will stop all work units.", core.Server_UUID, val)
					all_work, _ := workmap.GetKeys()

					for _, work := range all_work {
						DiscardWorkunit(work)
					}
					core.Server_UUID = val
				}
			}

		} else {
			logger.Debug(1, "(SendHeartBeat) No Server UUID received")
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
	var headers httpclient.Header
	if conf.CLIENT_GROUP_TOKEN != "" {
		headers = httpclient.Header{
			"Authorization": []string{"CG_TOKEN " + conf.CLIENT_GROUP_TOKEN},
		}
	}
	res, err := httpclient.Get(targeturl, headers, nil, nil)
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

// not used, deprecated ?
func RegisterWithProfile(host string, profile *core.Client) (client *core.Client, err error) {
	profile_jsonstream, err := json.Marshal(profile)
	profile_path := conf.DATA_PATH + "/clientprofile.json"
	logger.Debug(3, "profile_path: %s", profile_path)
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
	response.Data.Init()
	client = &response.Data
	return
}

// invoked on start of AWE worker AND on ReRegisterWithSelf
func RegisterWithAuth(host string, pclient *core.Client) (client *core.Client, err error) {
	logger.Debug(3, "Try to register client")
	if conf.CLIENT_GROUP_TOKEN == "" {
		fmt.Println("clientgroup token not set, register as a public client (can only access public data)")
	}

	//serialize profile
	client_jsonstream, err := pclient.Marshal()
	//client_jsonstream, err := json.Marshal(pclient)
	if err != nil {
		err = fmt.Errorf("json.Marshal(client) error: %s", err.Error())
		return
	}

	// write profile to file
	logger.Debug(3, "client_jsonstream: %s ", string(client_jsonstream))
	profile_path := conf.DATA_PATH + "/clientprofile.json"
	logger.Debug(3, "profile_path: %s", profile_path)
	err = ioutil.WriteFile(profile_path, []byte(client_jsonstream), 0644)
	if err != nil {
		err = fmt.Errorf("(RegisterWithAuth) error in ioutil.WriteFile: %s", err.Error())
		return
	}

	// create http form
	form := httpclient.NewForm()
	form.AddFile("profile", profile_path)
	if err = form.Create(); err != nil {
		err = fmt.Errorf("form.Create() error: %s", err.Error())
		return
	}
	var headers httpclient.Header
	if conf.CLIENT_GROUP_TOKEN == "" {
		headers = httpclient.Header{
			"Content-Type":   []string{form.ContentType},
			"Content-Length": []string{strconv.FormatInt(form.Length, 10)},
		}
	} else {
		headers = httpclient.Header{
			"Content-Type":   []string{form.ContentType},
			"Content-Length": []string{strconv.FormatInt(form.Length, 10)},
			"Authorization":  []string{"CG_TOKEN " + conf.CLIENT_GROUP_TOKEN},
		}
	}

	// send profile
	targetUrl := host + "/client"
	logger.Debug(3, "Try to register client: %s", targetUrl)

	resp, err := httpclient.DoTimeout("POST", targetUrl, headers, form.Reader, nil, time.Second*10)
	if err != nil {
		err = fmt.Errorf("POST %s error: %s", targetUrl, err.Error())
		return
	}
	defer resp.Body.Close()

	// evaluate response
	response := new(ClientResponse)
	logger.Debug(3, "client registration: got response")
	jsonstream, err := ioutil.ReadAll(resp.Body)
	if err = json.Unmarshal(jsonstream, response); err != nil {
		err = errors.New("fail to unmashal response:" + string(jsonstream))
		return
	}
	if len(response.Errs) > 0 {
		err = fmt.Errorf("Server returned: %s", strings.Join(response.Errs, ","))
		return
	}

	client = &response.Data

	client.Init()
	core.SetClientProfile(client)

	logger.Debug(3, "Client registered")
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

func Set_Metadata(profile *core.Client) {
	// TODO create option --metadata=ec2 instead
	if len(conf.METADATA) > 0 {

		if conf.METADATA == "ec2" || conf.METADATA == "openstack" {

			metadata_url := "http://169.254.169.254/2009-04-04/meta-data"

			logger.Debug(1, fmt.Sprintf("Using metdata service %s with url %s, getting instance_id and instance_type...", conf.METADATA, metadata_url))

			// read all values: for i in `curl http://169.254.169.254/1.0/meta-data/` ; do echo ${i}: `curl -s http://169.254.169.254/1.0/meta-data/${i}` ; done
			instance_hostname, err := getMetaDataField(metadata_url, "hostname")
			if err == nil {
				instance_hostname = strings.TrimSuffix(instance_hostname, ".novalocal")
				profile.Name = instance_hostname
			}
			instance_id, err := getMetaDataField(metadata_url, "instance-id")
			if err == nil {
				profile.InstanceId = instance_id
			}
			instance_type, err := getMetaDataField(metadata_url, "instance-type")
			if err == nil {
				profile.InstanceType = instance_type
			}
			local_ipv4, err := getMetaDataField(metadata_url, "local-ipv4")
			if err == nil {
				profile.Host = local_ipv4
			}

		} else {
			logger.Error("Metdata service %s is unknown", conf.METADATA)
		}

	}

	// fall-back
	if profile.Host == "" {
		if addrs, err := net.InterfaceAddrs(); err == nil {
			for _, a := range addrs {
				if ipnet, ok := a.(*net.IPNet); ok && !ipnet.IP.IsLoopback() && len(strings.Split(ipnet.IP.String(), ".")) == 4 {
					profile.Host = ipnet.IP.String()
					break
				}
			}
		}
	}

}

// invoked only once on start of awe-worker
func ComposeProfile() (profile *core.Client, err error) {
	//profile = new(core.Client)
	profile = core.NewClient() // includes init

	profile.Name = conf.CLIENT_NAME
	profile.Host = conf.CLIENT_HOST
	profile.Group = conf.CLIENT_GROUP
	profile.CPUs = runtime.NumCPU()
	profile.Domain = conf.CLIENT_DOMAIN
	profile.Version = conf.VERSION
	profile.GitCommitHash = conf.GIT_COMMIT_HASH

	//app list
	//profile.Apps = []string{}
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

	Set_Metadata(profile)

	if core.Service == "proxy" {
		profile.Proxy = true
	}

	//profile.Init()

	return
}

func DiscardWorkunit(id string) (err error) {
	//fmt.Printf("try to discard workunit %s\n", id)
	logger.Info("trying to discard workunit %s", id)
	stage, ok, err := workmap.Get(id)
	if err != nil {
		return
	}
	if ok {
		if stage == ID_WORKER {
			chankill <- true
		}

		workmap.Set(id, ID_DISCARDED, "DiscardWorkunit")
		err = core.Self.Current_work_delete(id, true)
		if err != nil {
			logger.Error("(DiscardWorkunit) Could not remove workunit %s from client", id)
			err = nil
		}
	}
	return
}

func RestartClient() (err error) {
	//fmt.Printf("try to restart client\n")
	//to-do: implementation here
	return
}

func StopClient() (err error) {
	fmt.Printf("client deleted, exiting...\n")
	os.Exit(0)
	return
}

func CleanDisk() (err error) {
	//fmt.Printf("try to clean disk space\n")
	//to-do: implementation here
	return
}
func getMetaDataField(metadata_url string, field string) (result string, err error) {
	url := fmt.Sprintf("%s/%s", metadata_url, field) // TODO this is not OPENSTACK, this is EC2
	logger.Debug(1, fmt.Sprintf("url=%s", url))

	for i := 0; i < 3; i++ {
		//var res *http.Response
		error_chan := make(chan error)
		result_chan := make(chan string)
		go func() {
			res, xerr := http.Get(url)
			if xerr != nil {
				error_chan <- xerr //we are ending with error
				return
			}

			defer res.Body.Close()
			bodybytes, xerr := ioutil.ReadAll(res.Body)
			if xerr != nil {
				error_chan <- xerr //we are ending with error
				return
			}
			result = string(bodybytes[:])

			result_chan <- result
		}()
		select {
		case err = <-error_chan:
			//go ahead
		case result = <-result_chan:
			//go ahead
		case <-time.After(conf.INSTANCE_METADATA_TIMEOUT): //GET timeout
			err = errors.New("timeout: " + url)
		}

		if err != nil {
			logger.Error(fmt.Sprintf("warning: (iteration=%d) %s \"%s\"", i, url, err.Error()))
			continue
		} else if result == "" {
			logger.Error(fmt.Sprintf("warning: (iteration=%d) %s empty result", i, url))
			continue
		}

		break

	}

	if err != nil {
		return "", err
	}

	if result == "" {
		return "", errors.New(fmt.Sprintf("metadata result empty, %s", url))
	}

	logger.Debug(1, fmt.Sprintf("Intance Metadata %s => \"%s\"", url, result))
	return
}
