// Package logger implements async log web api messages
package logger

import (
	"fmt"
	"github.com/MG-RAST/Shock/shock-server/conf"
	l4g "github.com/MG-RAST/golib/log4go"
	"os"
)

var Log *Logger

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

// Initialialize sets up package var Log for use in Info(), Error(), and Perf()
func Initialize() {
	Log = New()
	go Log.Handle()
}

// Info is a short cut function that uses package initialized logger
func Info(log string, message string) {
	Log.Info(log, message)
	return
}

// Error is a short cut function that uses package initialized logger and error log
func Error(message string) {
	Log.Error(message)
	return
}

// Perf is a short cut function that uses package initialized logger and performance log
func Perf(message string) {
	Log.Perf(message)
	return
}

// New configures and returns a new logger. It also kicks off the goroutine that
// performs the log writing as messages queue.
func New() *Logger {
	l := &Logger{queue: make(chan m, 1024), logs: map[string]l4g.Logger{}}
	l.logs["access"] = make(l4g.Logger)
	accessf := l4g.NewFileLogWriter(conf.PATH_LOGS+"/access.log", false)
	if accessf == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating access log file")
		os.Exit(1)
	}
	if conf.LOG_ROTATE {
		l.logs["access"].AddFilter("access", l4g.FINEST, accessf.SetFormat("[%D %T] %M").SetRotate(true).SetRotateDaily(true))
	} else {
		l.logs["access"].AddFilter("access", l4g.FINEST, accessf.SetFormat("[%D %T] %M"))
	}

	l.logs["error"] = make(l4g.Logger)
	errorf := l4g.NewFileLogWriter(conf.PATH_LOGS+"/error.log", false)
	if errorf == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating error log file")
		os.Exit(1)
	}
	if conf.LOG_ROTATE {
		l.logs["error"].AddFilter("error", l4g.FINEST, errorf.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))
	} else {
		l.logs["error"].AddFilter("error", l4g.FINEST, errorf.SetFormat("[%D %T] [%L] %M"))
	}

	l.logs["perf"] = make(l4g.Logger)
	perff := l4g.NewFileLogWriter(conf.PATH_LOGS+"/perf.log", false)
	if perff == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating perf log file")
		os.Exit(1)
	}
	if conf.LOG_ROTATE {
		l.logs["perf"].AddFilter("perf", l4g.FINEST, perff.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))
	} else {
		l.logs["perf"].AddFilter("perf", l4g.FINEST, perff.SetFormat("[%D %T] [%L] %M"))
	}

	go func() {
		select {
		case m := <-l.queue:
			l.logs[m.log].Log(m.lvl, "", m.message)
		}
	}()

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

func (l *Logger) Critical(log string, message string) {
	l.Log(log, l4g.CRITICAL, message)
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
