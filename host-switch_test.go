// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"net/http"
	"testing"
)

func TestHostSwitchAdd(t *testing.T) {
	hs := &hostSwitch{}
	hs.Add("example.com", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
	hs.Add("example.org", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))

	defer func() {
		if err := recover(); err != nil {
			if err != `a handle is already registered for host 'example.com'` {
				panic(err)
			}
		} else {
			t.Error("(*hostSwitch).Add did not panic on duplicate")
		}
	}()
	hs.Add("example.com", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {}))
}

func TestHostSwitchNotFound(t *testing.T) {
	calledNotFound := false
	hs := &hostSwitch{
		NotFound: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			calledNotFound = true
		}),
	}

	hs.ServeHTTP(new(fakeResponseWriter), &http.Request{Host: "example.com"})

	if !calledNotFound {
		t.Error("hostSwitch did not call NotFound")
	}
}

func TestHostSwitch(t *testing.T) {
	calledNotFound := false
	hs := &hostSwitch{
		NotFound: http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
			calledNotFound = true
		}),
	}

	calledExampleCom := false
	hs.Add("example.com", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calledExampleCom = true
	}))

	calledExampleOrg := false
	hs.Add("example.org", http.HandlerFunc(func(http.ResponseWriter, *http.Request) {
		calledExampleOrg = true
	}))

	hs.ServeHTTP(new(fakeResponseWriter), &http.Request{Host: "example.com"})

	if calledNotFound || !calledExampleCom || calledExampleOrg {
		t.Error("hostSwitch did not call correct handler")
	}
}
