package event

import ()

const (
	CLIENT_REGISTRATION = "CR" //client registered (for the first time)
	CLIENT_AUTO_REREGI  = "CA" //client automatically re-registered
	CLIENT_UNREGISTER   = "CU" //client unregistered
	WORK_CHECKOUT       = "WC" //workunit checkout
	WORK_FAIL           = "WF" //workunit fails running
	WORK_FAILED         = "W!" //workunit fails running (not recoverable)
	//server only events
	SERVER_START         = "SS" //awe-server start
	SERVER_RECOVER       = "SR" //awe-server start with recover option  (-recover)
	DEBUG_LEVEL          = "DL" //debug level changed
	QUEUE_RESUME         = "QR" //awe-server queue resumed if suspended
	QUEUE_SUSPEND        = "QS" //awe-server queue suspended, not handing out work
	JOB_SUBMISSION       = "JQ" //job submitted
	JOB_IMPORT           = "JI" //job imported
	TASK_ENQUEUE         = "TQ" //task parsed and enqueue
	WORK_DONE            = "WD" //workunit received successful feedback from client
	WORK_REQUEUE         = "WR" //workunit requeue after receive failed feedback from client
	WORK_SUSPEND         = "WP" //workunit suspend after failing for conf.Max_Failure times
	TASK_DONE            = "TD" //task done (all the workunits in the task have finished)
	TASK_SKIPPED         = "TS" //task skipped (skip option > 0)
	JOB_DONE             = "JD" //job done (all the tasks in the job have finished)
	JOB_SUSPEND          = "JP" //job suspended
	JOB_DELETED          = "JL" //job deleted
	JOB_EXPIRED          = "JE" //job expired
	JOB_FULL_DELETE      = "JR" //job removed form mongodb (deleted fully)
	JOB_FAILED_PERMANENT = "JF" //job failed permanently
	//client only events
	WORK_START     = "WS" //workunit command start running
	WORK_END       = "WE" //workunit command finish running
	WORK_RETURN    = "WR" //send back failed workunit to server
	WORK_DISCARD   = "WI" //workunit discarded after receiving discard signal from server
	PRE_WORK_START = "PS" //workunit command start running
	PRE_WORK_END   = "PE" //workunit command finish running
	PRE_IN         = "PI" //start fetching predata file from url
	PRE_READY      = "PR" //predata file is available
	FILE_IN        = "FI" //start fetching input file from shock
	FILE_READY     = "FR" //finish fetching input file from shock
	FILE_OUT       = "FO" //start pushing output file to shock
	FILE_DONE      = "FD" //finish pushing output file to shock
	ATTR_IN        = "AI" //start fetching input attributes from shock
	ATTR_READY     = "AR" //finish fetching input attributes from shock
	//proxy only events
	WORK_QUEUED = "WQ" //workunit queued at proxy
)

var EventDiscription = map[string]map[string]string{
	"general": map[string]string{
		"CR": "client registered (for the first time)",
		"CA": "client automatically re-registered",
		"CU": "client unregistered",
		"WC": "workunit checkout",
		"WF": "workunit fails running",
		"W!": "workunit failed running (not recoverable)",
	},
	"server": map[string]string{
		"SS": "awe-server start",
		"SR": "awe-server start with recover option  (-recover)",
		"DL": "debug level changed",
		"QR": "awe-server queue resumed if suspended",
		"QS": "awe-server queue suspended, not handing out work",
		"JQ": "job submitted",
		"JI": "job imported",
		"TQ": "task parsed and enqueue",
		"WD": "workunit received successful feedback from client",
		"WR": "workunit requeue after receive failed feedback from client",
		"WP": "workunit suspend after failing for conf.Max_Failure times",
		"TD": "task done (all the workunits in the task have finished)",
		"TS": "task skipped (skip option > 0)",
		"JD": "job done (all the tasks in the job have finished)",
		"JP": "job suspended",
		"JL": "job deleted",
		"JE": "job expired",
		"JR": "job removed form mongodb (deleted fully)",
		"JF": "job failed permanently",
	},
	"client": map[string]string{
		"WS": "workunit command start running",
		"WE": "workunit command finish running",
		"WR": "send back failed workunit to server",
		"WI": "workunit discarded after receiving discard signal from server",
		"PS": "workunit command start running",
		"PE": "workunit command finish running",
		"PI": "start fetching predata file from url",
		"PR": "predata file is available",
		"FI": "start fetching input file from shock",
		"FR": "finish fetching input file from shock",
		"FO": "start pushing output file to shock",
		"FD": "finish pushing output file to shock",
		"AI": "start fetching input attributes from shock",
		"AR": "finish fetching input attributes from shock",
	},
	"proxy": map[string]string{
		"WQ": "workunit queued at proxy",
	},
}
