// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
)

func getUserHandler(githubClient *github.Client) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var cacheControl = fmt.Sprintf("public, max-age=%d", time.Minute/time.Second)

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		h := w.Header()
		h.Set("Cache-Control", cacheControl)

		if checkLastModified(w, r, time.Now(), time.Minute) {
			return
		}

		page, redirect, err := parsePageString(ps.ByName("page"))
		if err != nil {
			h.Del("Cache-Control")

			http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
			return
		} else if redirect {
			localRedirect(w, r, "../../")
			return
		}

		user := ps.ByName("user")

		repos, resp, err := githubClient.Repositories.List(user, &github.RepositoryListOptions{
			Sort: "updated",

			ListOptions: github.ListOptions{
				Page: page,

				PerPage: 50,
			},
		})
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

		if wrote, err := executeTemplate(userTemplate, struct {
			User  string
			Repos []github.Repository
			Resp  *github.Response
		}{
			User:  user,
			Repos: repos,
			Resp:  resp,
		}, w); err != nil {
			log.Printf("%[1]T %[1]v", err)

			if !wrote {
				h.Del("Cache-Control")
				http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
			}
		}
	}
}
