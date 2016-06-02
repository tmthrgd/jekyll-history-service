// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"crypto/sha256"
	"encoding/hex"
	"testing"
)

func TestAssetHash(t *testing.T) {
	h, err := AssetHash(AssetNames()[0])

	if err != nil {
		t.Error(err)
	}

	if h == "" {
		t.Error("AssetHash returned nil")
	}

	h, err = AssetHash("non-existant/file-test.ext")

	if err == nil {
		t.Error("AssetHash did not return error")
	}

	if h != "" {
		t.Error("AssetHash returned hash for non-existant file")
	}
}

func TestAssetHashHasAll(t *testing.T) {
	for _, name := range AssetNames() {
		if _, err := AssetHash(name); err != nil {
			t.Error(err)
		}
	}
}

func TestAssetHashAddStrip(t *testing.T) {
	for _, name := range AssetNames() {
		a, err := AssetHashName(name)
		if err != nil {
			t.Error(err)
		}

		b, err := AssetNameStripHash(a)
		if err != nil {
			t.Error(err)
		}

		if name != b {
			t.Errorf("AssetNameStripHash(AssetHashName(%q)) = %q", name, b)
		}
	}
}

func TestAssetHashes(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	h := sha256.New()

	for _, name := range AssetNames() {
		asset, err := Asset(name)
		if err != nil {
			t.Error(err)
		}

		if _, err := h.Write(asset); err != nil {
			t.Error(err)
		}

		a := hex.EncodeToString(h.Sum(nil))
		h.Reset()

		b, err := AssetHash(name)
		if err != nil {
			t.Error(err)
		}

		if a != b {
			t.Errorf("AssetHash did not return correct hash for: %s", name)
		}
	}
}
