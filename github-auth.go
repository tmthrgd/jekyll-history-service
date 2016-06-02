// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import "net/http"

type githubAuth struct {
	ID     string
	Secret string

	Transport http.RoundTripper
}

func (ga githubAuth) RoundTrip(req *http.Request) (*http.Response, error) {
	newReq := *req
	url := *req.URL
	q := req.URL.Query()

	q.Set("client_id", ga.ID)
	q.Set("client_secret", ga.Secret)

	url.RawQuery = q.Encode()
	newReq.URL = &url

	if ga.Transport != nil {
		return ga.Transport.RoundTrip(&newReq)
	}

	return http.DefaultTransport.RoundTrip(&newReq)
}
