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

	"github.com/golang/groupcache"
	_ "github.com/joho/godotenv/autoload"
)

//go:generate go-bindata -nomemcopy -nocompress assets/... views/...
//go:generate ./asset-hashes assets/* views/*
//go:generate protoc --go_out=. groupcache.proto

var debug bool
var verbose bool

func main() {
	flag.BoolVar(&debug, "debug", false, "do not delete temporary files")
	flag.BoolVar(&verbose, "verbose", false, "log more information than normal")

	var addr string
	flag.StringVar(&addr, "addr", ":8080", "the address to listen on")

	var work string
	flag.StringVar(&work, "work", "/tmp/jklhstry.${random}", "the working directory")

	var jekyll string
	flag.StringVar(&jekyll, "jekyll", "shell", "the method to run jekyll (shell, docker)")

	var jekyllOpts string
	flag.StringVar(&jekyllOpts, "jekyll-opts", "", "option string to use when running jekyll")

	var highlightStyle string
	flag.StringVar(&highlightStyle, "highlight-style", "https://cdnjs.cloudflare.com/ajax/libs/highlight.js/9.4.0/styles/github-gist.min.css", "the highlight.js stylesheet")

	flag.Parse()

	fmt.Println(fullVersionStr)

	hasWork := false
	flag.Visit(func(f *flag.Flag) {
		if f.Name == "work" {
			hasWork = true
		}
	})

	if !hasWork {
		var err error
		if work, err = ioutil.TempDir("", "jklhstry."); err != nil {
			panic(err)
		}

		if !debug {
			defer os.RemoveAll(work)
		}
	}

	if verbose {
		fmt.Printf("using work directory '%s'\n", work)
	}

	s3Bucket, s3BucketNoGzip, err := getS3Buckets()
	if err != nil {
		panic(err)
	}

	githubClient, err := getGithubClient()
	if err != nil {
		panic(err)
	}

	var executeJekyll func(src, dst string) error

	switch jekyll {
	case "shell":
		executeJekyll, err = getExecuteShellJekyll(jekyllOpts)
	case "docker":
		executeJekyll, err = getExecuteDockerJekyll(jekyllOpts)
	default:
		panic(fmt.Errorf("invalid -jekyll flag value of '%s'", jekyll))
	}

	if err != nil {
		panic(err)
	}

	buildJekyll := groupcache.NewGroup("build-jekyll", 1<<20, buildJekyllGetter{
		WorkingDirectory: work,

		ExecuteJekyll: executeJekyll,

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

	router := getRouter(httpPool, poolOpts, githubClient, highlightStyle, buildJekyll, s3BucketNoGzip)

	fmt.Printf("Listening on %s\n", addr)
	log.Fatal(http.ListenAndServe(addr, router))
}
