package event

import ()

const (
	CLIENT_REGISTRATION = "CR" //client registered (for the first time)
	CLIENT_AUTO_REREGI  = "CA" //client automatically re-registered
	CLIENT_UNREGISTER   = "CU" //client unregistered
	WORK_CHECKOUT       = "WC" //workunit checkout
	WORK_FAIL           = "WF" //workunit fails running
	//server only events
	SERVER_START   = "SS" //awe-server start
	SERVER_RECOVER = "SR" //awe-server start with recover option  (-recover)
	JOB_SUBMISSION = "JQ" //job submitted
	TASK_ENQUEUE   = "TQ" //task parsed and enqueue
	WORK_DONE      = "WD" //workunit received successful feedback from client
	WORK_REQUEUE   = "WR" //workunit requeue after receive failed feedback from client
	WORK_SUSPEND   = "WP" //workunit suspend after failing for conf.Max_Failure times
	TASK_DONE      = "TD" //task done (all the workunits in the task have finished)
	TASK_SKIPPED   = "TS" //task skipped (skip option > 0)
	JOB_DONE       = "JD" //job done (all the tasks in the job have finished)
	JOB_SUSPEND    = "JP" //job suspended
	//client only events
	WORK_START  = "WS" //workunit command start running
	WORK_END    = "WE" //workunit command finish running
	WORK_RETURN = "WR" //send back failed workunit to server
	FILE_IN     = "FI" //start fetching input file from shock
	FILE_READY  = "FR" //finish fetching input file from shock
	FILE_OUT    = "FO" //start pushing output file to shock
	FILE_DONE   = "FD" //finish pushing output file to shock
	//proxy only events
	WORK_QUEUED = "WQ" //workunit queued at proxy
)
