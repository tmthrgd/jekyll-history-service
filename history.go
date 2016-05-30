// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"flag"
	"fmt"
	"hash/crc32"
	"io/ioutil"
	"log"
	"net/http"
	"os"

	"github.com/elazarl/go-bindata-assetfs"
	"github.com/golang/groupcache"
	"github.com/google/go-github/github"
	"github.com/gregjones/httpcache"
	_ "github.com/joho/godotenv/autoload"
	"github.com/julienschmidt/httprouter"
	"github.com/keep94/weblogs"
)

//go:generate go-bindata -nomemcopy -nocompress assets/... views/...

type httpError struct {
	Err  error
	Code int
}

func (h httpError) Error() string {
	return h.Err.Error()
}

var debug bool

func main() {
	flag.BoolVar(&debug, "debug", false, "do not delete temporary files")

	var highlightStyle string
	flag.StringVar(&highlightStyle, "highlight-style", "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/9.4.0/styles/github-gist.min.css", "the highlight.js stylesheet")

	flag.Parse()

	fmt.Println(fullVersionStr)

	tmp, err := ioutil.TempDir("", "jklhstry.")
	if err != nil {
		panic(err)
	}

	if debug {
		fmt.Printf("repositories will be dowloaded into '%s'\n", tmp)
	} else {
		defer os.RemoveAll(tmp)
	}

	dest, err := ioutil.TempDir("", "jklhstry.")
	if err != nil {
		panic(err)
	}

	if debug {
		fmt.Printf("site will be built into '%s'\n", dest)
	} else {
		defer os.RemoveAll(dest)
	}

	clientTr := httpcache.NewMemoryCacheTransport()

	id := os.Getenv("GITHUB_CLIENT_ID")
	if secret := os.Getenv("GITHUB_CLIENT_SECRET"); len(id) != 0 && len(secret) != 0 {
		clientTr.Transport = &githubAuth{
			ID:     id,
			Secret: secret,
		}
	} else if len(id) != 0 || len(secret) != 0 {
		panic("both GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET must be set")
	}

	githubClient := github.NewClient(clientTr.Client())
	githubClient.UserAgent = fullVersionStr

	buildJekyll := groupcache.NewGroup("build-jekyll", 1<<20, buildJekyllGetter{
		RepoBasePath: tmp,
		SiteBasePath: dest,

		GithubClient: githubClient,
	})
	builtFiles := groupcache.NewGroup("built-file", 1<<20, builtFileGetter{
		SiteBasePath: dest,
	})

	castagnoli := crc32.MakeTable(crc32.Castagnoli)

	poolOpts := &groupcache.HTTPPoolOptions{
		BasePath: "/_groupcache/",

		HashFn: func(data []byte) uint32 {
			if idx := bytes.IndexByte(data, 0x00); idx != -1 {
				return crc32.Checksum(data[:idx], castagnoli)
			} else {
				return crc32.Checksum(data, castagnoli)
			}
		},
	}
	httpPool := groupcache.NewHTTPPoolOpts("http://jekyllhistory.org:8080", poolOpts)

	baseRouter := httprouter.New()
	baseRouter.RedirectTrailingSlash = true
	baseRouter.RedirectFixedPath = true
	baseRouter.HandleMethodNotAllowed = true
	baseRouter.HandleOPTIONS = true

	baseRouter.Handler(http.MethodGet, poolOpts.BasePath, httpPool)

	baseRouter.HEAD("/", Index)
	baseRouter.GET("/", Index)
	baseRouter.GET("/goto/", Goto)
	user := GetUserHandler(githubClient)
	baseRouter.GET("/u/:user/", user)
	baseRouter.GET("/u/:user/p/:page/", user)
	repo := GetRepoHandler(githubClient)
	baseRouter.GET("/u/:user/r/:repo/", repo)
	baseRouter.GET("/u/:user/r/:repo/p/:page/", repo)
	baseRouter.GET("/u/:user/r/:repo/t/:tree/", repo)
	baseRouter.GET("/u/:user/r/:repo/t/:tree/p/:page/", repo)
	baseRouter.GET("/u/:user/r/:repo/c/:commit/", GetCommitHandler(githubClient, highlightStyle))
	buildCommit := GetBuildCommitHandler(buildJekyll)
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
	baseRouter.Handler(http.MethodHead, "/assets/*path", http.StripPrefix("/assets/", assetsRouter))
	baseRouter.Handler(http.MethodGet, "/assets/*path", http.StripPrefix("/assets/", assetsRouter))

	hs := new(hostSwitch)
	hs.NotFound = &repoSwitch{
		BuiltFiles: builtFiles,
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

	if debug {
		router = weblogs.HandlerWithOptions(router, &weblogs.Options{
			Logger: debugLogger{},
		})
	}

	fmt.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
