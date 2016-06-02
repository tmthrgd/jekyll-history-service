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

func gotoNotFoundHandler(w http.ResponseWriter, r *http.Request) {
	newURL := *r.URL
	newURL.Path = "/"
	newURL.RawQuery = ""

	http.Redirect(w, r, newURL.String(), http.StatusFound)
}

func gotoUserHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user := ps.ByName("user")

	if len(user) == 0 {
		gotoNotFoundHandler(w, r)
		return
	}

	newURL := *r.URL
	newURL.Path = "/u/" + url.QueryEscape(user) + "/"
	newURL.RawQuery = ""

	http.Redirect(w, r, newURL.String(), http.StatusFound)
}

func gotoRepoHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user, repo := ps.ByName("user"), ps.ByName("repo")

	if len(user) == 0 || len(repo) == 0 {
		gotoNotFoundHandler(w, r)
		return
	}

	newURL := *r.URL
	newURL.Path = "/u/" + url.QueryEscape(user) + "/r/" + url.QueryEscape(repo) + "/"
	newURL.RawQuery = ""

	http.Redirect(w, r, newURL.String(), http.StatusFound)
}

func gotoCommitHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user, repo, commit := ps.ByName("user"), ps.ByName("repo"), ps.ByName("commit")

	if len(user) == 0 || len(repo) == 0 || len(commit) == 0 {
		gotoNotFoundHandler(w, r)
		return
	}

	newURL := *r.URL
	newURL.Path = "/u/" + url.QueryEscape(user) + "/r/" + url.QueryEscape(repo) + "/c/" + url.QueryEscape(commit) + "/"
	newURL.RawQuery = ""

	http.Redirect(w, r, newURL.String(), http.StatusFound)
}

func gotoTreeHandler(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user, repo, tree := ps.ByName("user"), ps.ByName("repo"), ps.ByName("tree")

	if len(user) == 0 || len(repo) == 0 || len(tree) == 0 {
		gotoNotFoundHandler(w, r)
		return
	}

	newURL := *r.URL
	newURL.Path = "/u/" + url.QueryEscape(user) + "/r/" + url.QueryEscape(repo) + "/t/" + url.QueryEscape(tree) + "/"
	newURL.RawQuery = ""

	http.Redirect(w, r, newURL.String(), http.StatusFound)
}

func getGotoHandler() func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	router := new(httprouter.Router)

	router.NotFound = http.HandlerFunc(gotoNotFoundHandler)
	router.GET("/:user", gotoUserHandler)
	router.GET("/:user/", gotoUserHandler)
	router.GET("/:user/:repo", gotoRepoHandler)
	router.GET("/:user/:repo/", gotoRepoHandler)
	router.GET("/:user/:repo/commit/:commit", gotoCommitHandler)
	router.GET("/:user/:repo/commit/:commit/", gotoCommitHandler)
	router.GET("/:user/:repo/tree/:tree", gotoTreeHandler)
	router.GET("/:user/:repo/tree/:tree/", gotoTreeHandler)

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Cache-Control", "max-age=0")

		if err := r.ParseForm(); err != nil {
			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			return
		}

		urlField := r.Form.Get("url")

		if len(urlField) == 0 {
			gotoNotFoundHandler(w, r)
			return
		}

		if !strings.HasPrefix(urlField, "http:") && !strings.HasPrefix(urlField, "https:") {
			urlField = "https://" + urlField
		}

		parsedURL, err := url.Parse(urlField)
		if err != nil {
			gotoNotFoundHandler(w, r)
			return
		}

		if host := strings.ToLower(parsedURL.Host); host != "github.com" && host != "www.github.com" {
			gotoNotFoundHandler(w, r)
			return
		}

		req := *r

		reqURL := *r.URL
		reqURL.Path = parsedURL.Path
		req.URL = &reqURL

		router.ServeHTTP(w, &req)
	}
}
