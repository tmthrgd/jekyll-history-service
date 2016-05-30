// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"net"
	"net/http"
)

type hostSwitch struct {
	handlers map[string]http.Handler
	NotFound http.Handler
}

func (hs *hostSwitch) Add(host string, handler http.Handler) {
	if hs.handlers == nil {
		hs.handlers = make(map[string]http.Handler)
	}

	if _, dup := hs.handlers[host]; dup {
		panic("a handle is already registered for host '" + host + "'")
	}

	hs.handlers[host] = handler
}

func (hs *hostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	if hs.handlers != nil {
		if handler := hs.handlers[host]; handler != nil {
			handler.ServeHTTP(w, r)
			return
		}
	}

	if hs.NotFound != nil {
		hs.NotFound.ServeHTTP(w, r)
	} else {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}
}
