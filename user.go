// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"time"

	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
)

func GetUserHandler(githubClient *github.Client) func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	var cacheControl = fmt.Sprintf("public, max-age=%d", time.Minute/time.Second)

	return func(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
		w.Header().Set("Cache-Control", cacheControl)

		if checkLastModified(w, r, time.Now(), time.Minute) {
			return
		}

		var page int = 1

		if pageStr := ps.ByName("page"); len(pageStr) != 0 {
			var err error
			if page, err = strconv.Atoi(pageStr); err != nil || page <= 0 {
				w.Header().Del("Cache-Control")

				http.Error(w, http.StatusText(http.StatusNotFound), http.StatusNotFound)
				return
			}

			if page == 1 {
				localRedirect(w, r, "../../")
				return
			}
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
			w.Header().Del("Cache-Control")

			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
			return
		}

		if debug {
			log.Printf("GitHub API Rate Limit is %d remaining of %d, to be reset at %s\n", resp.Remaining, resp.Limit, resp.Reset)
		}

		if err := userTemplate.Execute(w, struct {
			User  string
			Repos []github.Repository
			Resp  *github.Response
		}{
			User:  user,
			Repos: repos,
			Resp:  resp,
		}); err != nil {
			w.Header().Del("Cache-Control")

			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}
	}
}
