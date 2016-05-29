// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"flag"
	"fmt"
	"hash/crc32"
	"html"
	"io"
	"io/ioutil"
	"log"
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"time"

	"github.com/golang/groupcache"
	"github.com/google/go-github/github"
	"github.com/julienschmidt/httprouter"
	"github.com/keep94/weblogs"
	"github.com/keep94/weblogs/loggers"
)

type hostRedirector struct {
	Host string
	Code int
}

func (h hostRedirector) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	url := *r.URL

	if _, port, err := net.SplitHostPort(r.Host); err == nil {
		url.Host = net.JoinHostPort(h.Host, port)
	} else {
		url.Host = h.Host
	}

	code := http.StatusMovedPermanently

	if h.Code != 0 {
		code = h.Code
	}

	http.Redirect(w, r, url.String(), code)
}

type hostSwitch struct {
	handlers map[string]http.Handler
	NotFound http.Handler
}

func (hs *hostSwitch) Add(host string, handler http.Handler) {
	if hs.handlers == nil {
		hs.handlers = make(map[string]http.Handler)
	}

	if _, dup := hs.handlers[host]; dup {
		panic("a handle is already registered for host '" + host + "'")
	}

	hs.handlers[host] = handler
}

func (hs *hostSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	if hs.handlers != nil {
		if handler := hs.handlers[host]; handler != nil {
			handler.ServeHTTP(w, r)
			return
		}
	}

	if hs.NotFound != nil {
		hs.NotFound.ServeHTTP(w, r)
	} else {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
	}
}

var repoListOpts = &github.RepositoryListOptions{
	Sort: "updated",

	ListOptions: github.ListOptions{
		PerPage: 100,
	},
}

func User(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user := ps.ByName("user")

	repos, _, err := client.Repositories.List(user, repoListOpts)
	if err != nil {
		log.Printf("%[1]T %[1]v", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	htmlUser := html.EscapeString(user)

	fmt.Fprintf(w, "<!doctype html>\n<title>%s</title>\n<p>%d repositories:</p>\n<ul>\n", htmlUser, len(repos))

	for _, repo := range repos {
		var desc string

		if repo.Description != nil {
			desc = *repo.Description
		}

		fmt.Fprintf(w, "<li><a href=\"/u/%s/%s\">%s</a>: %s</li>\n", htmlUser, html.EscapeString(*repo.Name), html.EscapeString(*repo.FullName), html.EscapeString(desc))
	}

	fmt.Fprint(w, "</ul>")
}

var commitListOpts = &github.CommitsListOptions{
	ListOptions: github.ListOptions{
		PerPage: 100,
	},
}

func Repo(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user := ps.ByName("user")
	repo := ps.ByName("repo")

	commits, _, err := client.Repositories.ListCommits(user, repo, commitListOpts)
	if err != nil {
		log.Printf("%[1]T %[1]v", err)
		http.Error(w, http.StatusText(http.StatusBadGateway), http.StatusBadGateway)
		return
	}

	htmlUser := html.EscapeString(user)
	htmlRepo := html.EscapeString(repo)

	fmt.Fprintf(w, "<!doctype html>\n<title>%s/%s</title>\n<p>%d commits:</p>\n<ul>\n", htmlUser, htmlRepo, len(commits))

	for _, commit := range commits {
		var msg string

		if commit.Message != nil {
			msg = *commit.Message
		} else if commit.Commit != nil && commit.Commit.Message != nil {
			msg = *commit.Commit.Message
		}

		fmt.Fprintf(w, "<li><a href=\"/u/%[1]s/%[2]s/%[3]s\">%[1]s/%[2]s@<code>%[3]s</code></a>: %[4]s</li>\n", htmlUser, htmlRepo, *commit.SHA, html.EscapeString(msg))
	}

	fmt.Fprint(w, "</ul>")
}

func Commit(w http.ResponseWriter, r *http.Request, ps httprouter.Params) {
	user := ps.ByName("user")
	repo := ps.ByName("repo")
	commit := ps.ByName("commit")

	data := user + "\x00" + repo + "\x00" + commit

	rawTag := sha256.Sum256([]byte(data))
	tag := hex.EncodeToString(rawTag[:16])

	var res []byte

	if err := buildFiles.Get(nil, tag+"\x00"+data, groupcache.TruncatingByteSliceSink(&res)); err != nil {
		if herr, ok := err.(*httpError); ok {
			log.Printf("%[1]T %[1]v", herr.Err)
			http.Error(w, http.StatusText(herr.Code), herr.Code)
		} else if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	url := *r.URL
	url.Host = tag + ".jekyllhistory.org"

	if _, port, err := net.SplitHostPort(r.Host); err == nil {
		url.Host = net.JoinHostPort(url.Host, port)
	}

	url.Path = ps.ByName("path")
	http.Redirect(w, r, url.String(), http.StatusFound)
}

var (
	cacheControl = fmt.Sprintf("public, max-age=%d", (10*365*24*time.Hour)/time.Second)

	timeZero time.Time

	hostRegex = regexp.MustCompile(`^([0-9a-fA-F]{32}).jekyllhistory.org$`)
)

type repoSwitch struct{}

func (rs repoSwitch) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	host, _, err := net.SplitHostPort(r.Host)
	if err != nil {
		host = r.Host
	}

	m := hostRegex.FindStringSubmatch(host)
	if m == nil {
		http.Error(w, http.StatusText(http.StatusForbidden), http.StatusForbidden)
		return
	}

	tag := strings.ToLower(m[1])

	switch r.Method {
	case http.MethodGet:
	case http.MethodHead:
	case http.MethodOptions:
		w.Header().Set("Allow", http.MethodGet+", "+http.MethodHead+", "+http.MethodOptions)
		w.WriteHeader(http.StatusOK)
		return
	default:
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	if strings.HasSuffix(r.URL.Path, "/index.html") {
		localRedirect(w, r, "./")
		return
	}

	h := w.Header()
	h.Set("Cache-Control", cacheControl)
	h.Set("Etag", `"`+tag+`"`)

	if checkETag(w, r) {
		return
	}

	var dst []byte

	if err := builtFiles.Get(nil, tag+"\x00"+r.URL.Path, groupcache.AllocatingByteSliceSink(&dst)); err != nil {
		h.Del("Cache-Control")
		h.Del("Etag")

		if herr, ok := err.(*httpError); ok {
			log.Printf("%[1]T %[1]v", herr.Err)
			http.Error(w, http.StatusText(herr.Code), herr.Code)
		} else if os.IsNotExist(err) {
			http.NotFound(w, r)
		} else {
			log.Printf("%[1]T %[1]v", err)
			http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
		}

		return
	}

	http.ServeContent(w, r, r.URL.Path, timeZero, bytes.NewReader(dst))
}

type httpError struct {
	Err  error
	Code int
}

func (h httpError) Error() string {
	return h.Err.Error()
}

type buildFileGetter struct {
	RepoBasePath string
	SiteBasePath string

	Client *http.Client
}

func (bf buildFileGetter) Get(_ groupcache.Context, key string, dest groupcache.Sink) error {
	parts := strings.Split(key, "\x00")
	if len(parts) != 4 {
		return &httpError{errors.New("invalid key"), http.StatusBadRequest}
	}

	tag, user, repo, commit := parts[0], parts[1], parts[2], parts[3]

	repoPath := filepath.Join(bf.RepoBasePath, tag[0:1], tag[1:2], tag[2:])
	sitePath := filepath.Join(bf.SiteBasePath, tag[0:1], tag[1:2], tag[2:])

	if _, err := os.Stat(sitePath); err == nil {
		return nil
	}

	if _, err := os.Stat(repoPath); err != nil {
		u, _, err := client.Repositories.GetArchiveLink(user, repo, github.Tarball, &github.RepositoryContentGetOptions{
			Ref: commit,
		})
		if err != nil {
			return &httpError{err, http.StatusBadGateway}
		}

		if u == nil {
			return os.ErrNotExist
		}

		client := bf.Client
		if client == nil {
			client = http.DefaultClient
		}

		resp, err := client.Do(&http.Request{
			URL:  u,
			Host: u.Host,
		})
		if err != nil {
			return &httpError{err, http.StatusBadGateway}
		}

		if resp.Body == nil {
			return errors.New("(*http.Client).Do did not return body")
		}

		defer resp.Body.Close()

		reader, err := gzip.NewReader(resp.Body)
		if err != nil {
			return err
		}

		defer reader.Close()

		if !debug {
			defer os.RemoveAll(repoPath)
		}

		tarReader := tar.NewReader(reader)

		for {
			header, err := tarReader.Next()
			if err == io.EOF {
				break
			} else if err != nil {
				return err
			}

			idx := strings.IndexRune(header.Name, filepath.Separator)
			if idx == -1 {
				continue
			}

			path := filepath.Join(repoPath, header.Name[idx+1:])

			info := header.FileInfo()
			mode := info.Mode()

			if info.IsDir() {
				if err = os.MkdirAll(path, mode); err != nil {
					return err
				}

				continue
			}

			if mode&(os.ModeSymlink|os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
				log.Printf("tar file '%s' has invalid mode: %d", header.Name, mode)
				continue
			}

			file, err := os.OpenFile(path, os.O_CREATE|os.O_TRUNC|os.O_WRONLY, mode)
			if err != nil {
				return err
			}

			_, err = io.Copy(file, tarReader)
			file.Close()

			if err != nil {
				return err
			}
		}
	}

	cmd := exec.Command("jekyll", "build", "--no-watch", "--quiet", "--safe", "-s", repoPath, "-d", sitePath)
	cmd.Dir = repoPath
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		return err
	}

	return nil
}

type builtFileGetter struct {
	SiteBasePath string
}

func (bf builtFileGetter) Get(_ groupcache.Context, key string, dest groupcache.Sink) error {
	parts := strings.Split(key, "\x00")
	if len(parts) != 2 {
		return &httpError{errors.New("invalid key"), http.StatusBadRequest}
	}

	tag, file := parts[0], parts[1]

	dir := http.Dir(filepath.Join(bf.SiteBasePath, tag[0:1], tag[1:2], tag[2:]))

	f, err := dir.Open(file)
	if err != nil {
		return err
	}

	stat, err := f.Stat()
	if err != nil {
		f.Close()
		return err
	}

	if stat.Mode()&(os.ModeSymlink|os.ModeNamedPipe|os.ModeSocket|os.ModeDevice) != 0 {
		f.Close()
		return errors.New("not a regular file")
	}

	if stat.IsDir() {
		f.Close()

		if f, err = dir.Open(strings.TrimSuffix(file, "/") + "/index.html"); err != nil {
			return err
		}
	}

	b, err := ioutil.ReadAll(f)

	f.Close()

	if err != nil {
		return err
	}

	return dest.SetBytes(b)
}

type debugLoggerSnapshot struct {
	*loggers.Snapshot

	Host string
}

type debugLogger struct{}

func (l debugLogger) NewSnapshot(r *http.Request) weblogs.Snapshot {
	return debugLoggerSnapshot{
		loggers.NewSnapshot(r),

		r.Host,
	}
}

func (l debugLogger) NewCapture(w http.ResponseWriter) weblogs.Capture {
	return &loggers.Capture{
		ResponseWriter: w,
	}
}

func (l debugLogger) Log(w io.Writer, log *weblogs.LogRecord) {
	s := log.R.(debugLoggerSnapshot)
	c := log.W.(*loggers.Capture)
	fmt.Fprintf(w, "%s %s %s %s %s %d %d%s\n",
		log.T.Format("01/02/2006 15:04:05"),
		loggers.StripPort(s.RemoteAddr),
		s.Method,
		s.Host,
		s.URL,
		c.Status(),
		log.Duration/time.Millisecond,
		log.Extra)
}

var (
	debug bool

	client *github.Client

	dest string
	tmp  string

	buildFiles *groupcache.Group
	builtFiles *groupcache.Group
)

func init() {
	client = github.NewClient(nil)
}

func main() {
	flag.BoolVar(&debug, "debug", false, "do not delete temporary files")
	flag.Parse()

	var err error

	if tmp, err = ioutil.TempDir("", "jklhstry."); err != nil {
		panic(err)
	}

	if debug {
		fmt.Printf("repositories will be dowloaded into '%s'\n", tmp)
	} else {
		defer os.RemoveAll(tmp)
	}

	if dest, err = ioutil.TempDir("", "jklhstry."); err != nil {
		panic(err)
	}

	if debug {
		fmt.Printf("site will be built into '%s'\n", dest)
	} else {
		defer os.RemoveAll(dest)
	}

	buildFiles = groupcache.NewGroup("build-file", 1<<20, buildFileGetter{
		RepoBasePath: tmp,
		SiteBasePath: dest,
	})
	builtFiles = groupcache.NewGroup("built-file", 1<<20, builtFileGetter{
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

	baseRouter.GET("/u/:user/", User)
	baseRouter.GET("/u/:user/:repo/", Repo)
	baseRouter.GET("/u/:user/:repo/:commit/*path", Commit)

	hs := new(hostSwitch)
	hs.NotFound = new(repoSwitch)

	hs.Add("jekyllhistory.com", hostRedirector{
		Host: "jekyllhistory.org",
		Code: http.StatusFound,
	})
	hs.Add("jekyllhistory.org", baseRouter)

	hs.Add("www.jekyllhistory.com", hostRedirector{
		Host: "jekyllhistory.com",
	})
	hs.Add("www.jekyllhistory.org", hostRedirector{
		Host: "jekyllhistory.org",
	})

	var router http.Handler

	if debug {
		router = weblogs.HandlerWithOptions(hs, &weblogs.Options{
			Logger: debugLogger{},
		})
	} else {
		router = http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			defer func() {
				if err := recover(); err != nil {
					if er := err.(error); er != nil {
						log.Printf("%[1]T %[1]v", er)
					} else {
						log.Printf("unkown panic: %v", err)
					}

					http.Error(w, http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
				}
			}()

			hs.ServeHTTP(w, r)
		})
	}

	fmt.Println("Listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", router))
}
