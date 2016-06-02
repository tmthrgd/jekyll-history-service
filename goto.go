// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"log"
	"net/http"
	"net/url"
	"strings"

	"github.com/julienschmidt/httprouter"
)

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

	switch path := strings.Split(strings.TrimSuffix(parsedURL.Path[1:], "/"), "/"); len(path) {
	case 1:
		if len(path[0]) == 0 {
			break
		}

		newURL.Path = "/u/" + url.QueryEscape(path[0]) + "/"
	case 2:
		newURL.Path = "/u/" + url.QueryEscape(path[0]) + "/r/" + url.QueryEscape(path[1]) + "/"
	case 4:
		switch strings.ToLower(path[2]) {
		case "commit":
			newURL.Path = "/u/" + url.QueryEscape(path[0]) + "/r/" + url.QueryEscape(path[1]) + "/c/" + url.QueryEscape(path[3]) + "/"
		case "tree":
			newURL.Path = "/u/" + url.QueryEscape(path[0]) + "/r/" + url.QueryEscape(path[1]) + "/t/" + url.QueryEscape(path[3]) + "/"
		}
	}

	http.Redirect(w, r, newURL.String(), http.StatusFound)
}
