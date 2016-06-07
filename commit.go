// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"time"

	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
)

func getCommitHandler(githubClient *github.Client, highlightStyle string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var cacheControl = fmt.Sprintf("public, max-age=%d", time.Minute/time.Second)

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		h := w.Header()
		h.Set("Cache-Control", cacheControl)

		if checkLastModified(w, r, time.Now(), time.Minute) {
			return
		}

		user, repo, commit := ps.ByName("user"), ps.ByName("repo"), ps.ByName("commit")

		repoCommit, resp, err := githubClient.Repositories.GetCommit(user, repo, commit)
		if err != nil {
			h.Del("Cache-Control")

			if gerr, ok := err.(*github.ErrorResponse); ok && gerr.Response.StatusCode == http.StatusNotFound {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			} else {
				log.Printf("%[1]T %[1]v", err)
				http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			}

			return
		}

		if verbose {
			log.Printf("GitHub API Rate Limit is %d remaining of %d, to be reset at %s\n", resp.Remaining, resp.Limit, resp.Reset)
		}

		base := url.URL{
			Scheme: "http",
			Host:   r.Host,
		}

		if r.TLS != nil {
			base.Scheme = "https"
		}

		if wrote, err := executeTemplate(commitTemplate, struct {
			User   string
			Repo   string
			Commit *github.RepositoryCommit

			URLBase string

			HighlightStyle string
		}{
			User:   user,
			Repo:   repo,
			Commit: repoCommit,

			URLBase: base.String(),

			HighlightStyle: highlightStyle,
		}, w); err != nil {
			log.Printf("%[1]T %[1]v", err)

			if !wrote {
				h.Del("Cache-Control")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}
	}
}

func getLocalCommitHandler(repoDir string, highlightStyle string) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var cacheControl = fmt.Sprintf("public, max-age=%d", time.Minute/time.Second)

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		h := w.Header()
		h.Set("Cache-Control", cacheControl)

		if checkLastModified(w, r, time.Now(), time.Minute) {
			return
		}

		user, repo, commit := ps.ByName("user"), ps.ByName("repo"), ps.ByName("commit")

		repoCommit, resp, err := githubClient.Repositories.GetCommit(user, repo, commit)
		if err != nil {
			h.Del("Cache-Control")

			if gerr, ok := err.(*github.ErrorResponse); ok && gerr.Response.StatusCode == http.StatusNotFound {
				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			} else {
				log.Printf("%[1]T %[1]v", err)
				http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			}

			return
		}

		if verbose {
			log.Printf("GitHub API Rate Limit is %d remaining of %d, to be reset at %s\n", resp.Remaining, resp.Limit, resp.Reset)
		}

		base := url.URL{
			Scheme: "http",
			Host:   r.Host,
		}

		if r.TLS != nil {
			base.Scheme = "https"
		}

		if wrote, err := executeTemplate(commitTemplate, struct {
			User   string
			Repo   string
			Commit *github.RepositoryCommit

			URLBase string

			HighlightStyle string
		}{
			User:   user,
			Repo:   repo,
			Commit: repoCommit,

			URLBase: base.String(),

			HighlightStyle: highlightStyle,
		}, w); err != nil {
			log.Printf("%[1]T %[1]v", err)

			if !wrote {
				h.Del("Cache-Control")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}
	}
}
