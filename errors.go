// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strconv"
	"strings"
)

type errorResponseWriter struct {
	http.ResponseWriter
	Request *http.Request

	wroteHeader bool
	didWrite    bool
	skipWrite   bool
}

func (w *errorResponseWriter) WriteHeader(code int) {
	w.wroteHeader = true

	if w.didWrite || w.skipWrite {
		w.ResponseWriter.WriteHeader(code)
		return
	}

	var name string
	var message string
	var description string

	switch code {
	case http.StatusBadRequest:
		name = "Bad Request"
		message = "Your user agent sent a request that this server could not understand."
	case http.StatusForbidden:
		name = "Forbidden"
		message = "You do not have permission to access this resource."
	case http.StatusNotFound:
		name = "File Not Found"
		message = "The link you followed may be broken, or the page may have been removed."
	case http.StatusMethodNotAllowed:
		name = "Method Not Allowed"
		message = "The specified HTTP method is not allowed for the requested resource."
		description = fmt.Sprintf("Request method '%s' is not supported for `%s`.", w.Request.Method, w.Request.URL.Path)

		if allow := w.Header().Get("Allow"); len(allow) != 0 {
			switch verbs := strings.Split(allow, ","); len(verbs) {
			case 1:
				allow = strings.TrimSpace(allow)
			default:
				for i := range verbs {
					verbs[i] = strings.TrimSpace(verbs[i])
				}

				allow = strings.Join(verbs[:len(verbs)-1], ", ") + " and " + verbs[len(verbs)-1]
			}

			description = fmt.Sprintf("%s Allowed verbs are %s.", description, allow)
		}
	case http.StatusInternalServerError:
		name = "Internal Server Error"
		message = "An internal server error has occurred."
	case http.StatusBadGateway:
		name = "Bad Gateway"
		message = "The upstream failed or was unreachable."
	default:
		w.ResponseWriter.WriteHeader(code)
		return
	}

	w.skipWrite = true

	h := w.Header()
	h.Del("Cache-Control")
	h.Del("Etag")
	h.Del("Last-Modified")
	h.Del("Content-Length")

	var padding template.HTML
	if code >= http.StatusBadRequest {
		if ua := w.Request.Header.Get("User-Agent"); len(ua) != 0 {
			if msie := strings.Index(ua, "MSIE "); msie != -1 && msie+7 < len(ua) && !strings.Contains(ua, "Opera") {
				const msieChromePadding = `
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->
<!-- a padding to disable MSIE and Chrome friendly error page -->`
				padding = template.HTML(msieChromePadding)
			}
		}
	}

	buf := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buf)
	buf.Reset()

	if err := errorTemplate.Execute(buf, struct {
		Code        int
		Name        string
		Message     string
		Description string
		Padding     template.HTML
	}{
		Code:        code,
		Name:        name,
		Message:     message,
		Description: description,
		Padding:     padding,
	}); err != nil {
		log.Printf("%[1]T %[1]v", err)

		http.Error(w.ResponseWriter, http.StatusText(code), code)
		return
	}

	h.Set("Content-Length", strconv.FormatInt(int64(buf.Len()), 10))
	h.Set("Content-Type", "text/html; charset=utf-8")

	w.ResponseWriter.WriteHeader(code)

	if _, err := buf.WriteTo(w.ResponseWriter); err != nil {
		log.Printf("%[1]T %[1]v", err)
	}
}

func (w *errorResponseWriter) Write(p []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if w.skipWrite {
		return len(p), nil
	}

	w.didWrite = true
	return w.ResponseWriter.Write(p)
}

type errorHandler struct {
	http.Handler
}

func (h errorHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	h.Handler.ServeHTTP(&errorResponseWriter{
		ResponseWriter: w,
		Request:        r,
	}, r)
}
