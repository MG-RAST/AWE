package main

import (
	. "github.com/MG-RAST/AWE/core"
)

func CheckoutWorkunit() (workunit *Workunit, err error) {
	job := NewJob()
	task := NewTask(job, 0)
	workunit = NewWorkunit(task, 0)
	return workunit, nil
}
