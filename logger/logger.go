package logger

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	l4g "github.com/jaredwilkening/log4go"
	"os"
)

//type level int

type m struct {
	log     string
	lvl     l4g.Level
	message string
}

type Logger struct {
	queue chan m
	logs  map[string]l4g.Logger
}

var (
	Log *Logger
)

const (
	EVENT_CLIENT_REGISTRATION = "CR" //client registered (for the first time)
	EVENT_CLIENT_AUTO_REREGI  = "CA" //client automatically re-registered
	EVENT_CLIENT_UNREGISTER   = "CU" //client unregistered
	EVENT_WORK_CHECKOUT       = "WC" //workunit checkout
	EVENT_WORK_FAIL           = "WF" //workunit fails running
	//server only events
	EVENT_SERVER_START   = "SS" //awe-server start
	EVENT_SERVER_RECOVER = "SR" //awe-server start with recover option  (-recover)
	EVENT_JOB_SUBMISSION = "JQ" //job submitted
	EVENT_TASK_ENQUEUE   = "TQ" //task parsed and enqueue
	EVENT_WORK_DONE      = "WD" //workunit received successful feedback from client
	EVENT_WORK_REQUEUE   = "WR" //workunit requeue after receive failed feedback from client
	EVENT_WORK_SUSPEND   = "WP" //workunit suspend after failing for conf.Max_Failure times
	EVENT_TASK_DONE      = "TD" //task done (all the workunits in the task have finished)
	EVENT_JOB_DONE       = "JD" //job done (all the tasks in the job have finished)
	EVENT_JOB_SUSPEND    = "JP" //job suspended
	//client only events
	EVENT_WORK_START  = "WS" //workunit command start running
	EVENT_WORK_END    = "WE" //workunit command finish running
	EVENT_WORK_RETURN = "WR" //send back failed workunit to server
	EVENT_FILE_IN     = "FI" //start fetching input file from shock
	EVENT_FILE_READY  = "FR" //finish fetching input file from shock
	EVENT_FILE_OUT    = "FO" //start pushing output file to shock
	EVENT_FILE_DONE   = "FD" //finish pushing output file to shock
)

func NewLogger(name string) *Logger {
	l := &Logger{queue: make(chan m, 1024), logs: map[string]l4g.Logger{}}

	logdir := fmt.Sprintf("%s/%s", conf.LOGS_PATH, name)
	if err := os.MkdirAll(logdir, 0777); err != nil {
		fmt.Errorf("ERROR: error creating directory for logs: %s", logdir)
		os.Exit(1)
	}

	l.logs["access"] = make(l4g.Logger)
	accessf := l4g.NewFileLogWriter(logdir+"/access.log", false)
	if accessf == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating access log file")
		os.Exit(1)
	}
	l.logs["access"].AddFilter("access", l4g.FINEST, accessf.SetFormat("[%D %T] %M").SetRotate(true).SetRotateDaily(true))

	l.logs["error"] = make(l4g.Logger)
	errorf := l4g.NewFileLogWriter(logdir+"/error.log", false)
	if errorf == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating error log file")
		os.Exit(1)
	}
	l.logs["error"].AddFilter("error", l4g.FINEST, errorf.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))

	l.logs["event"] = make(l4g.Logger)
	eventf := l4g.NewFileLogWriter(logdir+"/event.log", false)
	if eventf == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating event log file")
		os.Exit(1)
	}
	l.logs["event"].AddFilter("event", l4g.FINEST, eventf.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))

	l.logs["debug"] = make(l4g.Logger)
	debugf := l4g.NewFileLogWriter(logdir+"/debug.log", false)
	if debugf == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating debug log file")
		os.Exit(1)
	}
	l.logs["debug"].AddFilter("debug", l4g.FINEST, debugf.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))

	l.logs["perf"] = make(l4g.Logger)
	perff := l4g.NewFileLogWriter(logdir+"/perf.log", false)
	if perff == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating perf log file")
		os.Exit(1)
	}
	l.logs["perf"].AddFilter("perf", l4g.FINEST, perff.SetFormat("[%D %T] %M").SetRotate(true).SetRotateDaily(true))

	return l
}

func (l *Logger) Handle() {
	for {
		m := <-l.queue
		l.logs[m.log].Log(m.lvl, "", m.message)
	}
}

func (l *Logger) Log(log string, lvl l4g.Level, message string) {
	l.queue <- m{log: log, lvl: lvl, message: message}
	return
}

func (l *Logger) Debug(level int, message string) {
	if level <= conf.DEBUG_LEVEL {
		l.Log("debug", l4g.DEBUG, message)
	}
	return
}

func (l *Logger) Warning(log string, message string) {
	l.Log(log, l4g.WARNING, message)
	return
}

func (l *Logger) Info(log string, message string) {
	l.Log(log, l4g.INFO, message)
	return
}

func (l *Logger) Error(message string) {
	l.Log("error", l4g.ERROR, message)
	return
}

func (l *Logger) Perf(message string) {
	l.Log("perf", l4g.INFO, message)
	return
}

func (l *Logger) Critical(log string, message string) {
	l.Log(log, l4g.CRITICAL, message)
	return
}

func (l *Logger) Event(evttype string, attributes ...string) {
	msg := evttype
	for _, attr := range attributes {
		msg = msg + fmt.Sprintf(";%s", attr)
	}
	l.Log("event", l4g.INFO, msg)
}
