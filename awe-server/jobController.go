package main

import (
	"github.com/MG-RAST/AWE/core"
	e "github.com/MG-RAST/Shock/errors"
	"github.com/jaredwilkening/goweb"
	"net/http"
)

type JobController struct{}

func handleAuthError(err error, cx *goweb.Context) {
	switch err.Error() {
	case e.MongoDocNotFound:
		cx.RespondWithErrorMessage("Invalid username or password", http.StatusBadRequest)
		return
	case e.InvalidAuth:
		cx.RespondWithErrorMessage("Invalid Authorization header", http.StatusBadRequest)
		return
	}
	log.Error("Error at Auth: " + err.Error())
	cx.RespondWithError(http.StatusInternalServerError)
	return
}

// POST: /job
func (cr *JobController) Create(cx *goweb.Context) {
	// Log Request and check for Auth
	LogRequest(cx.Request)

	// Parse uploaded form 
	params, files, err := ParseMultipartForm(cx.Request)

	if err != nil {
		// If not multipart/form-data it will create an empty node. 
		if err.Error() == "request Content-Type isn't multipart/form-data" {
			cx.RespondWithErrorMessage("No job script is found in submission", http.StatusBadRequest)
		} else {
			// Some error other than request encoding. Theoretically 
			// could be a lost db connection between user lookup and parsing.
			// Blame the user, Its probaby their fault anyway.
			log.Error("Error parsing form: " + err.Error())
			cx.RespondWithError(http.StatusBadRequest)
			return
		}
	}
	// Create job	
	job, err := core.CreateJobUpload(params, files)

	if err != nil {
		log.Error("err " + err.Error())
		cx.RespondWithErrorMessage(err.Error(), http.StatusBadRequest)
		return
	}
	cx.RespondWithData(job)
	return
}
