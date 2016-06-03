// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"errors"
	"os"

	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
)

func getGithubClient() (*github.Client, error) {
	transport := httpcache.NewMemoryCacheTransport()
	transport.MarkCachedResponses = true

	id := os.Getenv("GITHUB_CLIENT_ID")
	if secret := os.Getenv("GITHUB_CLIENT_SECRET"); len(id) != 0 && len(secret) != 0 {
		transport.Transport = &github.UnauthenticatedRateLimitedTransport{
			ClientID:     id,
			ClientSecret: secret,
		}
	} else if len(id) != 0 || len(secret) != 0 {
		return nil, errors.New("both GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET must be set")
	}

	client := github.NewClient(transport.Client())
	client.UserAgent = fullVersionStr

	return client, nil
}
