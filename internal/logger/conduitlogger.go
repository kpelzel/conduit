// Copyright 2026. Triad National Security, LLC. All rights reserved.

package logger

import (
	"fmt"
	"io"
	"strings"

	"github.com/sirupsen/logrus"
)

type ConduitLogger struct {
	log    *logrus.Logger
	prefix *string
}

// Creates a ConduitLogger. This is just a simple wrapper around logrus
func NewConduitLogger(level logrus.Level, prefix string) *ConduitLogger {
	l := logrus.New()
	l.SetLevel(level)
	// add a single whitespace after prefix
	cleanPrefix := fmt.Sprintf("%s ", strings.TrimSpace(prefix))
	p := &cleanPrefix
	if prefix == "" {
		p = nil
	}

	return &ConduitLogger{
		log:    l,
		prefix: p,
	}
}

// GetLevel returns the logger level.
func (cl *ConduitLogger) GetLevel() logrus.Level {
	return cl.log.GetLevel()
}

// GetPrefix returns the prefix printed before every message from this logger
func (cl *ConduitLogger) GetPrefix() string {
	if cl.prefix == nil {
		return ""
	}

	return *cl.prefix
}

func (cl *ConduitLogger) Infof(format string, args ...interface{}) {
	clFormat := format
	if cl.prefix != nil {
		clFormat = fmt.Sprintf("%s%s", *cl.prefix, format)
	}
	cl.log.Logf(logrus.InfoLevel, clFormat, args...)
}

func (cl *ConduitLogger) Info(args ...interface{}) {
	if cl.prefix != nil {
		args = append([]interface{}{*cl.prefix}, args...)
	}
	cl.log.Log(logrus.InfoLevel, args...)
}

func (cl *ConduitLogger) Debugf(format string, args ...interface{}) {
	clFormat := format
	if cl.prefix != nil {
		clFormat = fmt.Sprintf("%s%s", *cl.prefix, format)
	}
	cl.log.Logf(logrus.DebugLevel, clFormat, args...)
}

func (cl *ConduitLogger) Debug(args ...interface{}) {
	if cl.prefix != nil {
		args = append([]interface{}{*cl.prefix}, args...)
	}
	cl.log.Log(logrus.DebugLevel, args...)
}

func (cl *ConduitLogger) Warnf(format string, args ...interface{}) {
	clFormat := format
	if cl.prefix != nil {
		clFormat = fmt.Sprintf("%s%s", *cl.prefix, format)
	}
	cl.log.Logf(logrus.WarnLevel, clFormat, args...)
}

func (cl *ConduitLogger) Warn(args ...interface{}) {
	if cl.prefix != nil {
		args = append([]interface{}{*cl.prefix}, args...)
	}
	cl.log.Log(logrus.WarnLevel, args...)
}

func (cl *ConduitLogger) Errorf(format string, args ...interface{}) {
	clFormat := format
	if cl.prefix != nil {
		clFormat = fmt.Sprintf("%s%s", *cl.prefix, format)
	}
	cl.log.Logf(logrus.ErrorLevel, clFormat, args...)
}

func (cl *ConduitLogger) Error(args ...interface{}) {
	if cl.prefix != nil {
		args = append([]interface{}{*cl.prefix}, args...)
	}
	cl.log.Log(logrus.ErrorLevel, args...)
}

func (cl *ConduitLogger) Panicf(format string, args ...interface{}) {
	clFormat := format
	if cl.prefix != nil {
		clFormat = fmt.Sprintf("%s%s", *cl.prefix, format)
	}
	cl.log.Logf(logrus.PanicLevel, clFormat, args...)
}

func (cl *ConduitLogger) Panic(args ...interface{}) {
	if cl.prefix != nil {
		args = append([]interface{}{*cl.prefix}, args...)
	}
	cl.log.Log(logrus.PanicLevel, args...)
}

func (cl *ConduitLogger) Fatalf(format string, args ...interface{}) {
	clFormat := format
	if cl.prefix != nil {
		clFormat = fmt.Sprintf("%s%s", *cl.prefix, format)
	}
	cl.log.Logf(logrus.FatalLevel, clFormat, args...)
	cl.log.Exit(1)
}

func (cl *ConduitLogger) Fatal(args ...interface{}) {
	if cl.prefix != nil {
		args = append([]interface{}{*cl.prefix}, args...)
	}
	cl.log.Log(logrus.FatalLevel, args...)
	cl.log.Exit(1)
}

// Writer at INFO level. See WriterLevel for details.
func (cl *ConduitLogger) Writer() *io.PipeWriter {
	return cl.log.Writer()
}

// SetOutput sets the logger output.
func (cl *ConduitLogger) SetOutput(output io.Writer) {
	cl.log.SetOutput(output)
}
