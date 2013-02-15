package core

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"labix.org/v2/mgo/bson"
	"net/http"
	"os/exec"
	"path/filepath"
	"strings"
)

type ShockResponse struct {
	Code int       `bson:"S" json:"S"`
	Data ShockNode `bson:"D" json:"D"`
	Errs []string  `bson:"E" json:"E"`
}

type ShockNode struct {
	Id         string             `bson:"id" json:"id"`
	Version    string             `bson:"version" json:"version"`
	File       shockfile          `bson:"file" json:"file"`
	Attributes interface{}        `bson:"attributes" json:"attributes"`
	Indexes    map[string]IdxInfo `bson:"indexes" json:"indexes"`
	Type       []string           `bson:"type" json:"type"`
}

type shockfile struct {
	Name         string            `bson:"name" json:"name"`
	Size         int64             `bson:"size" json:"size"`
	Checksum     map[string]string `bson:"checksum" json:"checksum"`
	Format       string            `bson:"format" json:"format"`
	Virtual      bool              `bson:"virtual" json:"virtual"`
	VirtualParts []string          `bson:"virtual_parts" json:"virtual_parts"`
}

type IdxInfo struct {
	Type        string `bson: "index_type" json:"index_type"`
	TotalUnits  int    `bson: "total_units" json:"total_units"`
	AvgUnitSize int    `bson: "avg_unitsize" json:"avg_unitsize"`
}

type FormFiles map[string]FormFile

type FormFile struct {
	Name     string
	Path     string
	Checksum map[string]string
}

func CreateJobUpload(params map[string]string, files FormFiles, jid string) (job *Job, err error) {

	job, err = ParseJobTasks(files["upload"].Path, jid)
	if err != nil {
		return
	}
	err = job.Mkdir()
	if err != nil {
		return
	}
	err = job.UpdateFile(params, files)
	if err != nil {
		return
	}

	err = job.Save()
	return
}

func LoadJob(id string) (job *Job, err error) {
	if db, err := DBConnect(); err == nil {
		defer db.Close()
		job = new(Job)
		if err = db.FindById(id, job); err == nil {
			return job, nil
		} else {
			return nil, err
		}
	}
	return nil, err
}

func LoadJobs(ids []string) (jobs []*Job, err error) {
	if db, err := DBConnect(); err == nil {
		defer db.Close()
		if err = db.FindJobs(ids, &jobs); err == nil {
			return jobs, err
		} else {
			return nil, err
		}
	}
	return nil, err
}

func ReloadFromDisk(path string) (err error) {
	id := filepath.Base(path)
	jobbson, err := ioutil.ReadFile(path + "/" + id + ".bson")
	if err != nil {
		return
	}
	job := new(Job)
	err = bson.Unmarshal(jobbson, &job)
	if err == nil {
		db, er := DBConnect()
		if er != nil {
			err = er
		}
		defer db.Close()
		err = db.Upsert(job)
		if err != nil {
			err = er
		}
	}
	return
}

//create a shock node for output
func PostNode(io *IO, numParts int) (nodeid string, err error) {
	var res *http.Response
	shockurl := fmt.Sprintf("%s/node", io.Host)
	res, err = http.Post(shockurl, "", strings.NewReader(""))

	//fmt.Printf("shockurl=%s\n", shockurl)
	if err != nil {
		return "", err
	}

	jsonstream, err := ioutil.ReadAll(res.Body)
	res.Body.Close()

	response := new(ShockResponse)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return "", errors.New(fmt.Sprintf("failed to marshal post response:\"%s\"", jsonstream))
	}
	if len(response.Errs) > 0 {
		return "", errors.New(strings.Join(response.Errs, ","))
	}

	shocknode := &response.Data
	nodeid = shocknode.Id

	if numParts > 1 {
		putParts(io.Host, nodeid, numParts)
	}
	return
}

func GetIndexUnits(indextype string, io *IO) (totalunits int, err error) {
	var res *http.Response

	shockurl := fmt.Sprintf("%s/node/%s", io.Host, io.Node)
	res, err = http.Get(shockurl)
	if err != nil {
		return 0, err
	}
	jsonstream, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return 0, err
	}
	res.Body.Close()

	response := new(ShockResponse)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return 0, err
	}
	if len(response.Errs) > 0 {
		return 0, errors.New(strings.Join(response.Errs, ","))
	}

	shocknode := &response.Data
	if _, ok := shocknode.Indexes[indextype]; ok {
		if shocknode.Indexes[indextype].TotalUnits > 0 {
			return shocknode.Indexes[indextype].TotalUnits, nil
		}
	}
	return 0, errors.New("invlid totalunits for shock node:" + io.Node)
}

//create parts
func putParts(host string, nodeid string, numParts int) (err error) {
	argv := []string{}
	argv = append(argv, "-X")
	argv = append(argv, "PUT")
	argv = append(argv, "-F")
	argv = append(argv, fmt.Sprintf("parts=%d", numParts))
	target_url := fmt.Sprintf("%s/node/%s", host, nodeid)
	argv = append(argv, target_url)

	cmd := exec.Command("curl", argv...)
	err = cmd.Run()
	if err != nil {
		return
	}
	return
}
