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

	"github.com/julienschmidt/httprouter"
)

var githubURLRegex = regexp.MustCompile(`^(?:https?://)?github.com/([^/]+)(?:/([^/]+)(?:/commit/([a-fA-F0-9]+)|/tree/([^/]+))?)?/?$`)

func gotoHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	w.Header().Set("Cache-Control", "max-age=0")

	if err := r.ParseForm(); err != nil {
		log.Printf("%[1]T %[1]v", err)
		http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		return
	}

	newURL := *r.URL

	if githubURL := r.Form.Get("url"); len(githubURL) != 0 {
		m := githubURLRegex.FindStringSubmatch(githubURL)
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
	} else {
		newURL.Path = "/"
	}

	newURL.RawQuery = ""
	http.Redirect(w, r, newURL.String(), http.StatusFound)
}
