/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"

	"github.com/sirupsen/logrus"
	prefixed "github.com/x-cray/logrus-prefixed-formatter"
	"github.com/zput/zxcTool/ztLog/zt_formatter"
)

type Logger struct {
	logrus.Logger
}

type LogConfig struct {
	Format string
	Level  string
	Fields map[string]interface{}
}
type trace struct {
	StackTrace []string
}

var (
	ok          bool
	pc          uintptr
	file        string
	line        int
	funcAbsPath string
	fileName    string
	funcName    string
)

var stackTrace interface{}

func NewLogProvidor(config *LogConfig) (*Logger, error) {
	format := strings.ToLower(config.Format)
	level := strings.ToLower(config.Level)
	l := logrus.New()

	switch config.Format {
	case "zt":
		l.SetFormatter(&zt_formatter.ZtFormatter{})
	case "prefixed":
		l.SetFormatter(&prefixed.TextFormatter{})
	case "json":
		l.SetFormatter(&logrus.JSONFormatter{})
	case "text":
		l.SetFormatter(&logrus.TextFormatter{})
	case "":
		l.SetFormatter(&logrus.TextFormatter{})
	default:
		return nil, fmt.Errorf("Logger: Invalid log-formatter: (`%s`) - must be one of [json, string]", format)
	}

	switch config.Level {
	case "trace":
		l.SetLevel(logrus.TraceLevel)
	case "debug":
		l.SetLevel(logrus.DebugLevel)
	case "info":
		l.SetLevel(logrus.InfoLevel)
	case "warn":
		l.SetLevel(logrus.WarnLevel)
	case "error":
		l.SetLevel(logrus.ErrorLevel)
	case "fatal":
		l.SetLevel(logrus.FatalLevel)
	case "panic":
		l.SetLevel(logrus.PanicLevel)
	case "":
		l.SetLevel(logrus.WarnLevel)
	default:
		return nil, fmt.Errorf("Logger: Invalid log-level: (`%s`) - must be one of [trace, debug, info, warn, error, fatal, panic]", level)
	}
	return &Logger{*l}, nil
}

//interface{} input
func (l *Logger) Trace(msg interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}
	stackTrace := fmt.Sprintf("%s", (debug.Stack()))
	l.Logger.Debug(fmt.Sprintf("%v", stackTrace))
	l.Logger.WithFields(logrus.Fields{
		"FILE": fileName,
		"LINE": line,
		"FUNC": funcName,
	}).Tracef(fmt.Sprint(msg))
}

func (l *Logger) Debug(msg interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}

	l.Logger.WithFields(logrus.Fields{}).Debugf(fmt.Sprint(msg))
}

func (l *Logger) Info(msg interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}

	l.Logger.WithFields(logrus.Fields{}).Infof(fmt.Sprint(msg))
}

func (l *Logger) Warn(msg interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}

	l.Logger.WithFields(logrus.Fields{}).Warnf(fmt.Sprint(msg))
}

func (l *Logger) Error(msg interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}
	l.Logger.WithFields(logrus.Fields{}).Errorf(fmt.Sprint(msg))
	os.Exit(1)
}

func (l *Logger) Fatal(msg interface{}) {
	defer func() {
		if r := recover(); r != nil {
			if pc, file, line, ok = runtime.Caller(1); ok {
				funcAbsPath = runtime.FuncForPC(pc).Name()
				fileName = filepath.Base(file)
				funcName = filepath.Base(funcAbsPath)
			}
		}
	}()
	stackTrace := fmt.Sprintf("%s", (debug.Stack()))
	l.Logger.Debug(fmt.Sprintf("%v", stackTrace))
	l.Logger.WithFields(logrus.Fields{
		"FILE": fileName,
		"LINE": line,
		"FUNC": funcName,
	}).Fatalf(fmt.Sprint(msg))
}

func (l *Logger) Panic(msg interface{}) {
	defer func() {
		if r := recover(); r != nil {
			if pc, file, line, ok = runtime.Caller(1); ok {
				funcAbsPath = runtime.FuncForPC(pc).Name()
				fileName = filepath.Base(file)
				funcName = filepath.Base(funcAbsPath)
			}
		}
	}()
	stackTrace := fmt.Sprintf("%s", (debug.Stack()))
	l.Logger.Debug(fmt.Sprintf("%v", stackTrace))
	l.Logger.WithFields(logrus.Fields{
		"FILE": fileName,
		"LINE": line,
		"FUNC": funcName,
	}).Fatalf(fmt.Sprint(msg))
}

//string, interface{} input
func (l *Logger) Tracef(msg string, args ...interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}
	stackTrace := fmt.Sprintf("%s", (debug.Stack()))
	l.Logger.Debug(fmt.Sprintf("%v", stackTrace))
	l.Logger.WithFields(logrus.Fields{
		"FILE": fileName,
		"LINE": line,
		"FUNC": funcName,
	}).Tracef(fmt.Sprintf(msg, args...))
}

func (l *Logger) Debugf(msg string, args ...interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}
	l.Logger.WithFields(logrus.Fields{}).Debugf(fmt.Sprintf(msg, args...))
}

func (l *Logger) Infof(msg string, args ...interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}
	l.Logger.WithFields(logrus.Fields{}).Infof(fmt.Sprintf(msg, args...))
}

func (l *Logger) Warnf(msg string, args ...interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}
	l.Logger.WithFields(logrus.Fields{}).Warnf(fmt.Sprintf(msg, args...))
}

func (l *Logger) Errorf(msg string, args ...interface{}) {
	if pc, file, line, ok = runtime.Caller(1); ok {
		funcAbsPath = runtime.FuncForPC(pc).Name()
		fileName = filepath.Base(file)
		funcName = filepath.Base(funcAbsPath)
	}
	l.Logger.WithFields(logrus.Fields{}).Errorf(fmt.Sprintf(msg, args...))
	os.Exit(1)
}

func (l *Logger) Fatalf(msg string, args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			if pc, file, line, ok = runtime.Caller(1); ok {
				funcAbsPath = runtime.FuncForPC(pc).Name()
				fileName = filepath.Base(file)
				funcName = filepath.Base(funcAbsPath)
			}
		}
	}()
	stackTrace := fmt.Sprintf("%s", (debug.Stack()))
	l.Logger.Debug(fmt.Sprintf("%v", stackTrace))
	l.Logger.WithFields(logrus.Fields{
		"FILE": fileName,
		"LINE": line,
		"FUNC": funcName,
	}).Fatalf(fmt.Sprintf(msg, args...))
}

func (l *Logger) Panicf(msg string, args ...interface{}) {
	defer func() {
		if r := recover(); r != nil {
			if pc, file, line, ok = runtime.Caller(1); ok {
				funcAbsPath = runtime.FuncForPC(pc).Name()
				fileName = filepath.Base(file)
				funcName = filepath.Base(funcAbsPath)
			}
		}
	}()
	stackTrace := fmt.Sprintf("%s", (debug.Stack()))
	l.Logger.Debug(fmt.Sprintf("%v", stackTrace))
	l.Logger.WithFields(logrus.Fields{
		"FILE": fileName,
		"LINE": line,
		"FUNC": funcName,
	}).Fatalf(fmt.Sprintf(msg, args...))
}
