// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"regexp"
	"testing"
)

func TestTemplateFuncs(t *testing.T) {
	if _, ok := templateFuncs["asset_path"]; !ok {
		t.Error("templateFuncs does not contain asset_path")
	}

	if _, ok := templateFuncs["truncate"]; !ok {
		t.Error("templateFuncs does not contain truncate")
	}
}

func TestAssetPath(t *testing.T) {
	// internal
	for name, expect := range map[string]string{
		"style.css": `^/assets/style-[0-9a-f]{64}.css$`,
		"commit.js": `^/assets/commit-[0-9a-f]{64}.js$`,
	} {
		expected := regexp.MustCompile(expect)

		path, err := assetPath(name)
		if err != nil {
			t.Error(err)
		}

		if !expected.MatchString(path) {
			t.Errorf("unexpected path for %s, expected %v, got %s", name, expect, path)
		}
	}

	// external
	for name, expect := range map[string]string{
		"http://example.com/test.ext":  "http://example.com/test.ext",
		"https://example.com/test.ext": "https://example.com/test.ext",
		"//example.com/test.ext":       "//example.com/test.ext",
	} {
		path, err := assetPath(name)
		if err != nil {
			t.Error(err)
		}

		if path != expect {
			t.Errorf("unexpected path for %s, expected %s, got %s", name, expect, path)
		}
	}
}

func TestTruncate(t *testing.T) {
	if got := truncate("abc", 2); got != "ab" {
		t.Errorf("unexpected truncation for abc at 2, expected ab, got %s", got)
	}

	if got := truncate("abc", 4); got != "abc" {
		t.Errorf("unexpected truncation for abc at 4, expected abc, got %s", got)
	}
}
