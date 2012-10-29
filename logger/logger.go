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

func New() *Logger {
	l := &Logger{queue: make(chan m, 1024), logs: map[string]l4g.Logger{}}
	l.logs["access"] = make(l4g.Logger)
	accessf := l4g.NewFileLogWriter(conf.LOGS_PATH+"/access.log", false)
	if accessf == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating access log file")
		os.Exit(1)
	}
	l.logs["access"].AddFilter("access", l4g.FINEST, accessf.SetFormat("[%D %T] %M").SetRotate(true).SetRotateDaily(true))

	l.logs["error"] = make(l4g.Logger)
	errorf := l4g.NewFileLogWriter(conf.LOGS_PATH+"/error.log", false)
	if errorf == nil {
		fmt.Fprintln(os.Stderr, "ERROR: error creating error log file")
		os.Exit(1)
	}
	l.logs["error"].AddFilter("error", l4g.FINEST, errorf.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))
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
