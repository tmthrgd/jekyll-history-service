// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"log"
	"net/http"
	"net/url"
	"regexp"
	"strings"

	"github.com/julienschmidt/httprouter"
)

var githubPathRegex = regexp.MustCompile(`(?i)^/([^/]+)(?:/([^/]+)(?:/commit/([a-fA-F0-9]+)|/tree/([^/]+))?)?/?$`)

func gotoHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Cache-Control", "max-age=0")

	if err := r.ParseForm(); err != nil {
		log.Printf("%[1]T %[1]v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	newURL := *r.URL
	newURL.Path = "/"
	newURL.RawQuery = ""

	urlField := r.Form.Get("url")

	if len(urlField) == 0 {
		http.Redirect(w, r, newURL.String(), http.StatusFound)
		return
	}

	if !strings.HasPrefix(urlField, "http:") && !strings.HasPrefix(urlField, "https:") {
		urlField = "https://" + urlField
	}

	parsedURL, err := url.Parse(urlField)
	if err != nil {
		http.Redirect(w, r, newURL.String(), http.StatusFound)
		return
	}

	if host := strings.ToLower(parsedURL.Host); host != "github.com" && host != "www.github.com" {
		http.Redirect(w, r, newURL.String(), http.StatusFound)
		return
	}

	m := githubPathRegex.FindStringSubmatch(parsedURL.Path)
	switch {
	case m == nil:
		newURL.Path = "/"
	case len(m[4]) != 0:
		newURL.Path = "/u/" + url.QueryEscape(m[1]) + "/r/" + url.QueryEscape(m[2]) + "/t/" + url.QueryEscape(m[4]) + "/"
	case len(m[3]) != 0:
		newURL.Path = "/u/" + url.QueryEscape(m[1]) + "/r/" + url.QueryEscape(m[2]) + "/c/" + url.QueryEscape(m[3]) + "/"
	case len(m[2]) != 0:
		newURL.Path = "/u/" + url.QueryEscape(m[1]) + "/r/" + url.QueryEscape(m[2]) + "/"
	default:
		newURL.Path = "/u/" + url.QueryEscape(m[1]) + "/"
	}

	http.Redirect(w, r, newURL.String(), http.StatusFound)
}
