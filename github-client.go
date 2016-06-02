// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"errors"
	"net/http"
	"os"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
)

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

func getGithubClient() (*github.Client, error) {
	transport := httpcache.NewMemoryCacheTransport()
	transport.MarkCachedResponses = true

	id := os.Getenv("GITHUB_CLIENT_ID")
	if secret := os.Getenv("GITHUB_CLIENT_SECRET"); len(id) != 0 && len(secret) != 0 {
		transport.Transport = &githubAuth{
			ID:     id,
			Secret: secret,
		}
	} else if len(id) != 0 || len(secret) != 0 {
		return nil, errors.New("both GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET must be set")
	}

	client := github.NewClient(transport.Client())
	client.UserAgent = fullVersionStr

	return client, nil
}
