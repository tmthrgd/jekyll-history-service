// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"net"
	"net/http"
)

type hostRedirector struct {
	Host string
	Code int
}

func (h hostRedirector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := *r.URL

	if len(url.Scheme) == 0 {
		if r.TLS != nil {
			url.Scheme = "https"
		} else {
			url.Scheme = "http"
		}
	}

	if _, port, err := net.SplitHostPort(r.Host); err == nil {
		url.Host = net.JoinHostPort(h.Host, port)
	} else {
		url.Host = h.Host
	}

	code := http.StatusMovedPermanently

	if h.Code != 0 {
		code = h.Code
	}

	http.Redirect(w, r, url.String(), code)
}
