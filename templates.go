// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"html/template"
	"net/http"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"
)

var (
	templateFuncs = map[string]interface{}{
		"asset_path": assetPath,
		"html5_attr": html5Attr,
		"truncate":   truncate,
	}

	errorTemplate  = template.Must(template.New("error.tmpl").Funcs(templateFuncs).Parse(string(MustAsset("views/error.tmpl"))))
	indexTemplate  = template.Must(template.New("index.tmpl").Funcs(templateFuncs).Parse(string(MustAsset("views/index.tmpl"))))
	userTemplate   = template.Must(template.New("user.tmpl").Funcs(templateFuncs).Parse(string(MustAsset("views/user.tmpl"))))
	repoTemplate   = template.Must(template.New("repo.tmpl").Funcs(templateFuncs).Parse(string(MustAsset("views/repo.tmpl"))))
	commitTemplate = template.Must(template.New("commit.tmpl").Funcs(templateFuncs).Parse(string(MustAsset("views/commit.tmpl"))))
)

func assetPath(name string) string {
	if strings.HasPrefix(name, "http://") || strings.HasPrefix(name, "https://") || strings.HasPrefix(name, "//") {
		return name
	}

	return filepath.Join("/assets/", name)
}

var unquoteableRegexp = regexp.MustCompile("^[^ \t\n\f\r\"'`=<>]+$")

func html5Attr(value string) template.HTML {
	if unquoteableRegexp.MatchString(value) {
		return template.HTML(template.HTMLEscapeString(value))
	}

	return template.HTML(`"` + template.HTMLEscapeString(value) + `"`)
}

func truncate(value string, length int) string {
	numRunes := 0

	for index := range value {
		numRunes++

		if numRunes > length {
			return value[:index]
		}
	}

	return value
}

var indexModTime time.Time

func init() {
	if stat, err := AssetInfo("views/index.tmpl"); err == nil {
		indexModTime = stat.ModTime()
	} else {
		panic(err)
	}
}

func executeTemplate(tmpl *template.Template, data interface{}, w http.ResponseWriter) (didWrite bool, err error) {
	return executeTemplateWithCode(tmpl, data, w, http.StatusOK)
}

func executeTemplateWithCode(tmpl *template.Template, data interface{}, w http.ResponseWriter, code int) (didWrite bool, err error) {
	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	if err = tmpl.Execute(buf, data); err != nil {
		return false, err
	}

	h := w.Header()
	h.Set("Content-Length", strconv.FormatInt(int64(buf.Len()), 10))
	h.Set("Content-Type", "text/html; charset=utf-8")

	w.WriteHeader(code)

	_, err = buf.WriteTo(w)
	return true, err
}
