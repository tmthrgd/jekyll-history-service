// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"net/http"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/golang/groupcache"
	"github.com/google/go-github/github"
	_ "github.com/joho/godotenv/autoload"
	"github.com/julienschmidt/httprouter"
	"github.com/keep94/weblogs"
	"github.com/mitchellh/goamz/s3"
)

func getRouter(httpPool http.Handler, poolOpts *groupcache.HTTPPoolOptions, githubClient *github.Client, highlightStyle string, buildJekyll *groupcache.Group, s3BucketNoGzip *s3.Bucket) http.Handler {
	baseRouter := httprouter.New()
	baseRouter.RedirectTrailingSlash = true
	baseRouter.RedirectFixedPath = true
	baseRouter.HandleMethodNotAllowed = true
	baseRouter.HandleOPTIONS = true

	baseRouter.Handler(http.MethodGet, poolOpts.BasePath, httpPool)

	baseRouter.HEAD("/", indexHandler)
	baseRouter.GET("/", indexHandler)
	baseRouter.GET("/goto/", getGotoHandler())
	user := getUserHandler(githubClient)
	baseRouter.GET("/u/:user/", user)
	baseRouter.GET("/u/:user/p/:page/", user)
	repo := getRepoHandler(githubClient)
	baseRouter.GET("/u/:user/r/:repo/", repo)
	baseRouter.GET("/u/:user/r/:repo/p/:page/", repo)
	baseRouter.GET("/u/:user/r/:repo/t/:tree/", repo)
	baseRouter.GET("/u/:user/r/:repo/t/:tree/p/:page/", repo)
	baseRouter.GET("/u/:user/r/:repo/c/:commit/", getCommitHandler(githubClient, highlightStyle))
	buildCommit := getBuildCommitHandler(buildJekyll)
	baseRouter.GET("/u/:user/r/:repo/c/:commit/b", buildCommit)
	baseRouter.GET("/u/:user/r/:repo/c/:commit/b/*path", buildCommit)

	assetsRouter := http.FileServer(&assetfs.AssetFS{
		Asset:     Asset,
		AssetDir:  AssetDir,
		AssetInfo: AssetInfo,

		Prefix: "assets",
	})
	baseRouter.Handler(http.MethodHead, "/favicon.ico", assetsRouter)
	baseRouter.Handler(http.MethodGet, "/favicon.ico", assetsRouter)
	baseRouter.Handler(http.MethodHead, "/robots.txt", assetsRouter)
	baseRouter.Handler(http.MethodGet, "/robots.txt", assetsRouter)

	baseRouter.ServeFiles("/assets/*filepath", &assetfs.AssetFS{
		Asset:     AssetFromNameHash,
		AssetDir:  AssetDir,
		AssetInfo: AssetInfoFromNameHash,

		Prefix: "assets",
	})

	hs := new(hostSwitch)
	hs.NotFound = &repoSwitch{
		S3Bucket: s3BucketNoGzip,
	}

	hs.Add("jekyllhistory.com", hostRedirector{
		Host: "jekyllhistory.org",
		Code: http.StatusFound,
	})
	hs.Add("jekyllhistory.org", errorHandler{
		Handler: baseRouter,
	})

	hs.Add("www.jekyllhistory.com", hostRedirector{
		Host: "jekyllhistory.com",
	})
	hs.Add("www.jekyllhistory.org", hostRedirector{
		Host: "jekyllhistory.org",
	})

	var router http.Handler = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Server", fullVersionStr)
		hs.ServeHTTP(w, r)
	})

	if verbose {
		router = weblogs.HandlerWithOptions(router, &weblogs.Options{
			Logger: debugLogger{},
		})
	}

	return router
}
