package logger

import (
	"fmt"
	"github.com/MG-RAST/AWE/conf"
	l4g "github.com/jaredwilkening/log4go"
	"os"
	"strings"
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

type EventMsg struct {
	EventType string
	Attr      map[string]string
}

var (
	Log *Logger
)

const (
	EVENT_JOB_SUBMISSION      = "JQ"
	EVENT_TASK_ENQUEUE        = "TQ"
	EVENT_CLIENT_REGISTRATION = "CR"
	EVENT_WORK_CHECKOUT       = "WC"
	EVENT_WORK_DONE           = "WE"
	EVENT_TASK_DONE           = "TE"
	EVENT_JOB_DONE            = "JE"
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

func (l *Logger) Debug(log string, message string) {
	l.Log(log, l4g.DEBUG, message)
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

func (l *Logger) Critical(log string, message string) {
	l.Log(log, l4g.CRITICAL, message)
	return
}

func (l *Logger) Event(evt *EventMsg) {
	msg := evt.EventType
	for key, val := range evt.Attr {
		msg = msg + fmt.Sprintf(";%s=%s", key, val)
	}
	l.Log("event", l4g.INFO, msg)
}

func NewEventMsg(evttype string, attributes ...string) (event *EventMsg) {
	event = new(EventMsg)
	event.EventType = evttype
	event.Attr = map[string]string{}
	for _, attr := range attributes {
		parts := strings.Split(attr, "=")
		event.Attr[parts[0]] = parts[1]
	}
	return event
}
