// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

// GOMAXPROCS=10 go test

package main

import (
	"bytes"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"testing"
	"time"
)

func TestAsset(t *testing.T) {
	b, err := Asset(AssetNames()[0])

	if err != nil {
		t.Error(err)
	}

	if b == nil {
		t.Error("Asset returned nil")
	}

	b, err = Asset("non-existant/file-test.ext")

	if err == nil {
		t.Error("Asset did not return error")
	}

	if b != nil {
		t.Error("Asset returned non-existant file")
	}
}

func testAssetHasAllWalkFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info == nil {
		return &os.PathError{Op: "open", Path: path, Err: errors.New("failed to get file info")}
	}

	if info.IsDir() {
		return nil
	}

	_, err = Asset(path)
	return err
}

func TestAssetHasAll(t *testing.T) {
	if err := filepath.Walk("assets", testAssetHasAllWalkFunc); err != nil {
		t.Error(err)
	}

	if err := filepath.Walk("views", testAssetHasAllWalkFunc); err != nil {
		t.Error(err)
	}
}

func TestAssetHasNoExtra(t *testing.T) {
	for _, name := range AssetNames() {
		if _, err := os.Stat(name); os.IsNotExist(err) {
			t.Errorf("Asset has non existant file: %s", name)
		}
	}
}

func TestAssetInfo(t *testing.T) {
	for _, name := range AssetNames() {
		asset, err := AssetInfo(name)
		if err != nil {
			t.Error(err)
		}

		file, err := os.Stat(name)
		if err != nil {
			t.Error(err)
		}

		/* AssetInfo returns the relative path while os.Stat returns only the name */
		if _, name := path.Split(asset.Name()); name != file.Name() {
			t.Errorf("AssetInfo(%[1]s).Name() != os.Stat(%[1]s).Name(), (%s != %s)", name, name, file.Name())
		}

		if asset.Size() != file.Size() {
			t.Errorf("AssetInfo(%[1]s).Size() != os.Stat(%[1]s).Size(), (%d != %d)", name, asset.Size(), file.Size())
		}

		if asset.Mode() != file.Mode() {
			t.Errorf("AssetInfo(%[1]s).Mode() != os.Stat(%[1]s).Mode(), (%d != %d)", name, asset.Mode(), file.Mode())
		}

		/* AssetInfo only has second granularity */
		if file.ModTime().Sub(asset.ModTime()) > time.Second {
			t.Errorf("AssetInfo(%[1]s).ModTime() != os.Stat(%[1]s).ModTime(), (%s != %s)", name, asset.ModTime(), file.ModTime())
		}

		if asset.IsDir() {
			t.Errorf("AssetInfo(%s).IsDir()", name)
		}

		if file.IsDir() {
			t.Errorf("os.Stat(%s).IsDir()", name)
		}
	}
}

func testAssetContentWalkFunc(path string, info os.FileInfo, err error) error {
	if err != nil {
		return err
	}

	if info == nil {
		return &os.PathError{Op: "open", Path: path, Err: errors.New("failed to get file info")}
	}

	if info.IsDir() {
		return nil
	}

	file, err := ioutil.ReadFile(path)
	if err != nil {
		return err
	}

	asset, err := Asset(path)
	if err != nil {
		return err
	}

	if !bytes.Equal(file, asset) {
		return fmt.Errorf("Asset did not return same content for: %s", path)
	}

	return nil
}

func TestAssetContent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	if err := filepath.Walk("assets", testAssetContentWalkFunc); err != nil {
		t.Error(err)
	}

	if err := filepath.Walk("views", testAssetContentWalkFunc); err != nil {
		t.Error(err)
	}
}
