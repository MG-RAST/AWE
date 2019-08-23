package logger

import (
	"fmt"
	"os"

	"github.com/MG-RAST/AWE/lib/conf"
	l4g "github.com/MG-RAST/golib/log4go"
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
func Debug(level int, format string, a ...interface{}) {
	Log.Debug(level, format, a...)
	return
}

// Event is a short cut function that uses package initialized logger and error log
func Event(evttype string, attributes ...string) {
	Log.Event(evttype, attributes)
	return
}

// Error is a short cut function that uses package initialized logger and error log
func Error(format string, a ...interface{}) {
	Log.Error(format, a...)
	return
}

func Warning(format string, a ...interface{}) {
	Log.Warning(format, a...)
	return
}

func Info(format string, a ...interface{}) {
	Log.Info(format, a...)
	return
}

func Access(format string, a ...interface{}) {
	Log.Access(format, a...)
	return
}

// Perf is a short cut function that uses package initialized logger and performance log
func Perf(message string) {
	Log.Perf(message)
	return
}

func NewLogger(name string) *Logger {

	bufferSize := 1024
	if conf.DEBUG_LEVEL > 0 {
		bufferSize = 1 // this makes sure we do not miss debug messages if AWE crashes
	}

	l := &Logger{queue: make(chan m, bufferSize), logs: map[string]l4g.Logger{}}

	// log file dir
	var logdir string
	if (conf.LOG_OUTPUT == "file") || (conf.LOG_OUTPUT == "both") {
		logdir = fmt.Sprintf("%s/%s", conf.LOGS_PATH, name)
		if err := os.MkdirAll(logdir, 0777); err != nil {
			fmt.Errorf("ERROR: error creating directory for logs: %s", logdir)
			os.Exit(1)
		}
	}

	// access log
	l.logs["access"] = make(l4g.Logger)
	if (conf.LOG_OUTPUT == "file") || (conf.LOG_OUTPUT == "both") {
		accessf := l4g.NewFileLogWriter(logdir+"/access.log", false)
		if accessf == nil {
			fmt.Fprintln(os.Stderr, "ERROR: error creating access log file")
			os.Exit(1)
		}
		l.logs["access"].AddFilter("access", l4g.FINEST, accessf.SetFormat("[%D %T] %M").SetRotate(true).SetRotateDaily(true))
	}
	if (conf.LOG_OUTPUT == "console") || (conf.LOG_OUTPUT == "both") {
		l.logs["access"].AddFilter("stdout", l4g.FINEST, l4g.NewConsoleLogWriter())
	}

	// error log
	l.logs["error"] = make(l4g.Logger)
	if (conf.LOG_OUTPUT == "file") || (conf.LOG_OUTPUT == "both") {
		errorf := l4g.NewFileLogWriter(logdir+"/error.log", false)
		if errorf == nil {
			fmt.Fprintln(os.Stderr, "ERROR: error creating error log file")
			os.Exit(1)
		}
		l.logs["error"].AddFilter("error", l4g.FINEST, errorf.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))
	}
	if (conf.LOG_OUTPUT == "console") || (conf.LOG_OUTPUT == "both") {
		l.logs["error"].AddFilter("stderr", l4g.FINEST, l4g.NewConsoleLogWriter())
	}

	// warning log
	l.logs["warning"] = make(l4g.Logger)
	if (conf.LOG_OUTPUT == "file") || (conf.LOG_OUTPUT == "both") {
		errorf := l4g.NewFileLogWriter(logdir+"/warning.log", false)
		if errorf == nil {
			fmt.Fprintln(os.Stderr, "ERROR: error creating warning log file")
			os.Exit(1)
		}
		l.logs["warning"].AddFilter("error", l4g.FINEST, errorf.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))
	}
	if (conf.LOG_OUTPUT == "console") || (conf.LOG_OUTPUT == "both") {
		l.logs["warning"].AddFilter("stderr", l4g.FINEST, l4g.NewConsoleLogWriter())
	}

	// event log
	l.logs["event"] = make(l4g.Logger)
	if (conf.LOG_OUTPUT == "file") || (conf.LOG_OUTPUT == "both") {
		eventf := l4g.NewFileLogWriter(logdir+"/event.log", false)
		if eventf == nil {
			fmt.Fprintln(os.Stderr, "ERROR: error creating event log file")
			os.Exit(1)
		}
		l.logs["event"].AddFilter("event", l4g.FINEST, eventf.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))
	}
	if (conf.LOG_OUTPUT == "console") || (conf.LOG_OUTPUT == "both") {
		l.logs["event"].AddFilter("stdout", l4g.FINEST, l4g.NewConsoleLogWriter())
	}

	// debug log
	l.logs["debug"] = make(l4g.Logger)
	if (conf.LOG_OUTPUT == "file") || (conf.LOG_OUTPUT == "both") {
		debugf := l4g.NewFileLogWriter(logdir+"/debug.log", false)
		if debugf == nil {
			fmt.Fprintln(os.Stderr, "ERROR: error creating debug log file")
			os.Exit(1)
		}
		l.logs["debug"].AddFilter("debug", l4g.FINEST, debugf.SetFormat("[%D %T] [%L] %M").SetRotate(true).SetRotateDaily(true))
	}
	if (conf.LOG_OUTPUT == "console") || (conf.LOG_OUTPUT == "both") {
		l.logs["debug"].AddFilter("stdout", l4g.FINEST, l4g.NewConsoleLogWriter())
	}

	// perf log
	l.logs["perf"] = make(l4g.Logger)
	if (conf.LOG_OUTPUT == "file") || (conf.LOG_OUTPUT == "both") {
		perff := l4g.NewFileLogWriter(logdir+"/perf.log", false)
		if perff == nil {
			fmt.Fprintln(os.Stderr, "ERROR: error creating perf log file")
			os.Exit(1)
		}
		l.logs["perf"].AddFilter("perf", l4g.FINEST, perff.SetFormat("[%D %T] %M").SetRotate(true).SetRotateDaily(true))
	}
	if (conf.LOG_OUTPUT == "console") || (conf.LOG_OUTPUT == "both") {
		l.logs["perf"].AddFilter("stdout", l4g.FINEST, l4g.NewConsoleLogWriter())
	}

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

//func (l *Logger) Debug(level int, message string) {
//	if level <= conf.DEBUG_LEVEL {
//		l.Log("debug", l4g.DEBUG, message)
//	}
//	return
//}

func (l *Logger) Debug(level int, format string, a ...interface{}) {
	if level <= conf.DEBUG_LEVEL {
		l.Log("debug", l4g.DEBUG, fmt.Sprintf(format, a...))
	}
	return
}

func (l *Logger) Warning(format string, a ...interface{}) {
	l.Log("warning", l4g.WARNING, fmt.Sprintf(format, a...))
	return
}

func (l *Logger) Info(format string, a ...interface{}) {
	l.Log("debug", l4g.INFO, fmt.Sprintf(format, a...))
	return
}

func (l *Logger) Access(format string, a ...interface{}) {
	l.Log("access", l4g.INFO, fmt.Sprintf(format, a...))
	return
}

func (l *Logger) Error(format string, a ...interface{}) {
	l.Log("error", l4g.ERROR, fmt.Sprintf(format, a...))
	return
}

func (l *Logger) Perf(message string) {
	l.Log("perf", l4g.INFO, message)
	return
}

func (l *Logger) Critical(log string, format string, a ...interface{}) {
	l.Log(log, l4g.CRITICAL, fmt.Sprintf(format, a...))
	return
}

func (l *Logger) Event(evttype string, attributes []string) {
	msg := evttype
	for _, attr := range attributes {
		msg = msg + fmt.Sprintf(";%s", attr)
	}
	l.Log("event", l4g.INFO, msg)
}
