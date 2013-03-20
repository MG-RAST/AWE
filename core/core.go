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
	var shocknode *ShockNode
	shocknode, err = GetShockNode(io.Host, io.Node)
	if err != nil {
		return
	}
	if _, ok := shocknode.Indexes[indextype]; ok {
		if shocknode.Indexes[indextype].TotalUnits > 0 {
			return shocknode.Indexes[indextype].TotalUnits, nil
		}
	}
	return 0, errors.New("invlid totalunits for shock node:" + io.Node)
}

func GetShockNode(host string, id string) (node *ShockNode, err error) {
	var res *http.Response
	shockurl := fmt.Sprintf("%s/node/%s", host, id)
	res, err = http.Get(shockurl)
	if err != nil {
		return
	}

	jsonstream, err := ioutil.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	res.Body.Close()
	response := new(ShockResponse)
	if err := json.Unmarshal(jsonstream, response); err != nil {
		return nil, err
	}
	if len(response.Errs) > 0 {
		return nil, errors.New(strings.Join(response.Errs, ","))
	}
	node = &response.Data
	if node == nil {
		err = errors.New("empty node got from Shock")
	}
	return
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

//get jobid from task id or workunit id
func getParentJobId(id string) (jobid string) {
	parts := strings.Split(id, "_")
	return parts[0]
}
