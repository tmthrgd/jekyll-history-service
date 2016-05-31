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
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

//go:generate go-bindata -nomemcopy -nocompress assets/... views/...
//go:generate protoc --go_out=. groupcache.proto

var debug bool

func main() {
	var addr string
	flag.StringVar(&addr, "addr", ":8080", "the address to listen on")

	flag.BoolVar(&debug, "debug", false, "do not delete temporary files")

	var highlightStyle string
	flag.StringVar(&highlightStyle, "highlight-style", "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/9.4.0/styles/github-gist.min.css", "the highlight.js stylesheet")

	flag.Parse()

	fmt.Println(fullVersionStr)

	tmpSrc, err := ioutil.TempDir("", "jklhstry.")
	if err != nil {
		panic(err)
	}

	if debug {
		fmt.Printf("repositories will be dowloaded into '%s'\n", tmpSrc)
	} else {
		defer os.RemoveAll(tmpSrc)
	}

	tmpDest, err := ioutil.TempDir("", "jklhstry.")
	if err != nil {
		panic(err)
	}

	if debug {
		fmt.Printf("site will be built into '%s'\n", tmpDest)
	} else {
		defer os.RemoveAll(tmpDest)
	}

	var s3Bucket *s3.Bucket
	var s3BucketNoGzip *s3.Bucket

	bucket := os.Getenv("S3_BUCKET")
	endpoint := os.Getenv("S3_ENDPOINT")
	if len(bucket) != 0 && len(endpoint) != 0 {
		region, ok := aws.Regions[endpoint]
		if !ok {
			panic(fmt.Errorf("invalid S3_ENDPOINT value of %s", endpoint))
		}

		auth, err := aws.EnvAuth()
		if err != nil {
			panic(err)
		}

		s3Bucket = s3.New(auth, region).Bucket(bucket)
		s3BucketNoGzip = s3.New(auth, region).Bucket(bucket)

		clientTr := httpcache.NewMemoryCacheTransport()
		clientTr.MarkCachedResponses = true

		client := clientTr.Client()
		s3Bucket.S3.HTTPClient = func() *http.Client {
			return client
		}

		noGzipClientTr := httpcache.NewMemoryCacheTransport()
		noGzipClientTr.MarkCachedResponses = true

		noGzipTransport := *http.DefaultTransport.(*http.Transport)
		noGzipTransport.DisableCompression = true
		noGzipClientTr.Transport = &noGzipTransport

		noGzipClient := noGzipClientTr.Client()
		s3BucketNoGzip.S3.HTTPClient = func() *http.Client {
			return noGzipClient
		}
	} else {
		panic("both S3_BUCKET and S3_ENDPOINT must be set")
	}

	githubClientTr := httpcache.NewMemoryCacheTransport()
	githubClientTr.MarkCachedResponses = true

	id := os.Getenv("GITHUB_CLIENT_ID")
	if secret := os.Getenv("GITHUB_CLIENT_SECRET"); len(id) != 0 && len(secret) != 0 {
		githubClientTr.Transport = &githubAuth{
			ID:     id,
			Secret: secret,
		}
	} else if len(id) != 0 || len(secret) != 0 {
		panic("both GITHUB_CLIENT_ID and GITHUB_CLIENT_SECRET must be set")
	}

	githubClient := github.NewClient(githubClientTr.Client())
	githubClient.UserAgent = fullVersionStr

	buildJekyll := groupcache.NewGroup("build-jekyll", 1<<20, buildJekyllGetter{
		RepoBasePath: tmpSrc,
		SiteBasePath: tmpDest,

		S3Bucket: s3Bucket,

		GithubClient: githubClient,
	})

	castagnoli := crc32.MakeTable(crc32.Castagnoli)

	poolOpts := &groupcache.HTTPPoolOptions{
		BasePath: "/_groupcache/",

		HashFn: func(data []byte) uint32 {
			if idx := bytes.IndexByte(data, 0x00); idx != -1 {
				return crc32.Checksum(data[:idx], castagnoli)
			}

			return crc32.Checksum(data, castagnoli)
		},
	}
	httpPool := groupcache.NewHTTPPoolOpts("http://jekyllhistory.org:8080", poolOpts)

	baseRouter := httprouter.New()
	baseRouter.RedirectTrailingSlash = true
	baseRouter.RedirectFixedPath = true
	baseRouter.HandleMethodNotAllowed = true
	baseRouter.HandleOPTIONS = true

	baseRouter.Handler(http.MethodGet, poolOpts.BasePath, httpPool)

	baseRouter.HEAD("/", indexHandler)
	baseRouter.GET("/", indexHandler)
	baseRouter.GET("/goto/", gotoHandler)
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
	baseRouter.Handler(http.MethodHead, "/assets/*path", http.StripPrefix("/assets/", assetsRouter))
	baseRouter.Handler(http.MethodGet, "/assets/*path", http.StripPrefix("/assets/", assetsRouter))

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

	if debug {
		router = weblogs.HandlerWithOptions(router, &weblogs.Options{
			Logger: debugLogger{},
		})
	}

	fmt.Printf("Listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
