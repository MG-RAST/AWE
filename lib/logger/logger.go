package logger

import (
	"fmt"
	"github.com/MG-RAST/AWE/lib/conf"
	l4g "github.com/MG-RAST/golib/log4go"
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

// Initialialize sets up package var Log for use in Info(), Error(), and Perf()
func Initialize(name string) {
	Log = NewLogger(name)
	go Log.Handle()
}

// Debug is a short cut function that uses package initialized logger and performance log
func Debug(level int, message string) {
	Log.Debug(level, message)
	return
}

// Event is a short cut function that uses package initialized logger and error log
func Event(evttype string, attributes ...string) {
	Log.Event(evttype, attributes)
	return
}

// Error is a short cut function that uses package initialized logger and error log
func Error(message string) {
	Log.Error(message)
	return
}

// Info is a short cut function that uses package initialized logger
func Info(log string, message string) {
	Log.Info(log, message)
	return
}

// Perf is a short cut function that uses package initialized logger and performance log
func Perf(message string) {
	Log.Perf(message)
	return
}

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

func (l *Logger) Event(evttype string, attributes []string) {
	msg := evttype
	for _, attr := range attributes {
		msg = msg + fmt.Sprintf(";%s", attr)
	}
	l.Log("event", l4g.INFO, msg)
}
