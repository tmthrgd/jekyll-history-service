// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/keep94/weblogs"
	"github.com/keep94/weblogs/loggers"
)

type debugLoggerSnapshot struct {
	*loggers.Snapshot

	Host string
}

type debugLogger struct{}

func (l debugLogger) NewSnapshot(r *http.Request) weblogs.Snapshot {
	return debugLoggerSnapshot{
		loggers.NewSnapshot(r),

		r.Host,
	}
}

func (l debugLogger) NewCapture(w http.ResponseWriter) weblogs.Capture {
	return &loggers.Capture{
		ResponseWriter: w,
	}
}

func (l debugLogger) Log(w io.Writer, log *weblogs.LogRecord) {
	s := log.R.(debugLoggerSnapshot)
	c := log.W.(*loggers.Capture)
	fmt.Fprintf(w, "%s %s %s %s %s %d %d%s\n",
		log.T.Format("02/01/2006 15:04:05"),
		loggers.StripPort(s.RemoteAddr),
		s.Method,
		s.Host,
		s.URL,
		c.Status(),
		log.Duration/time.Millisecond,
		log.Extra)
}
