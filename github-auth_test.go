// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"net/http"
	"testing"
)

type fakeTransport struct{}

func (fakeTransport) RoundTrip(req *http.Request) (*http.Response, error) {
	return &http.Response{
		Request: req,
	}, nil
}

func TestGithubAuth(t *testing.T) {
	auth := &githubAuth{
		ID:     "test-auth-id",
		Secret: "test-auth-secret",

		Transport: &fakeTransport{},
	}

	req, err := http.NewRequest(http.MethodGet, "http://example.com/test-url?test=test&client_id=invalid&client_secret=invalid", nil)
	if err != nil {
		t.Error(err)
	}

	resp, err := auth.RoundTrip(req)
	if err != nil {
		t.Error(err)
	}

	if resp.Request.URL.String() != "http://example.com/test-url?client_id=test-auth-id&client_secret=test-auth-secret&test=test" {
		t.Errorf("githubAuth create invalid request url of: %s", resp.Request.URL)
	}

	if req.URL.String() != "http://example.com/test-url?test=test&client_id=invalid&client_secret=invalid" {
		t.Error("githubAuth modified the original request")
	}
}
