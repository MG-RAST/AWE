package main

import (
	"encoding/json"
	"errors"
	"fmt"
	. "github.com/MG-RAST/AWE/core"
	"io/ioutil"
	"net/http"
)

type Response struct {
	Code int      `bson:"S" json:"S"`
	Data Workunit `bson:"D" json:"D"`
	Errs []string `bson:"E" json:"E"`
}

func CheckoutWorkunit() (workunit *Workunit, err error) {
	job := NewJob()
	task := NewTask(job, 0)
	workunit = NewWorkunit(task, 0)
	return workunit, nil
}

func CheckoutWorkunitRemote(url string) (workunit *Workunit, err error) {

	response := new(Response)

	res, err := http.Get(url)

	if err != nil {
		fmt.Printf("err=%#v", err)
		return
	}

	jsonstream, err := ioutil.ReadAll(res.Body)

	fmt.Printf("jsonstream=%s\n", jsonstream)

	if err != nil {
		return
	}

	json.Unmarshal(jsonstream, response)

	if response.Code == 200 {
		workunit = &response.Data
		return workunit, nil
	}
	return workunit, errors.New("empty workunit queue")
}
