// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import "testing"

func TestTemplateFuncs(t *testing.T) {
	if _, ok := templateFuncs["asset_path"]; !ok {
		t.Error("templateFuncs does not contain asset_path")
	}

	if _, ok := templateFuncs["truncate"]; !ok {
		t.Error("templateFuncs does not contain truncate")
	}
}

func TestAssetPath(t *testing.T) {
	for name, expect := range map[string]string{
		// internal
		"a":        "/assets/a",
		"test.ext": "/assets/test.ext",

		// external
		"http://example.com/test.ext":  "http://example.com/test.ext",
		"https://example.com/test.ext": "https://example.com/test.ext",
		"//example.com/test.ext":       "//example.com/test.ext",
	} {
		if path := assetPath(name); path != expect {
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
