// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"bytes"
	"errors"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"sort"
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
			t.Logf("AssetInfo(%[1]s).ModTime() != os.Stat(%[1]s).ModTime(), (%s != %s)", name, asset.ModTime(), file.ModTime())
		}

		if asset.IsDir() {
			t.Errorf("AssetInfo(%s).IsDir()", name)
		}

		if file.IsDir() {
			t.Errorf("os.Stat(%s).IsDir()", name)
		}
	}
}

func TestAssetContent(t *testing.T) {
	if testing.Short() {
		t.Skip("skipping test in short mode.")
	}

	for _, name := range AssetNames() {
		file, err := ioutil.ReadFile(name)
		if err != nil {
			t.Error(err)
		}

		asset, err := Asset(name)
		if err != nil {
			t.Error(err)
		}

		if !bytes.Equal(file, asset) {
			t.Errorf("Asset did not return same content for: %s", name)
		}
	}
}

func BenchmarkAssetAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, name := range AssetNames() {
			if _, err := Asset(name); err != nil {
				b.Error(err)
			}
		}
	}
}

func BenchmarkReadFileAll(b *testing.B) {
	for i := 0; i < b.N; i++ {
		for _, name := range AssetNames() {
			if _, err := ioutil.ReadFile(name); err != nil {
				b.Error(err)
			}
		}
	}
}

func benchmarkFindExtreme(large bool) string {
	var extreme string
	var size int64 = -1

	if !large {
		size = int64(^uint(0) >> 1)
	}

	for _, name := range AssetNames() {
		info, _ := AssetInfo(name)
		s := info.Size()

		if large {
			if s > size {
				extreme = name
				size = s
			}
		} else {
			if s < size {
				extreme = name
				size = s
			}
		}
	}

	return extreme
}

func BenchmarkAssetSmallest(b *testing.B) {
	smallest := benchmarkFindExtreme(false)
	if len(smallest) == 0 {
		b.Error("no files")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := Asset(smallest); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkReadFileSmallest(b *testing.B) {
	smallest := benchmarkFindExtreme(false)
	if len(smallest) == 0 {
		b.Error("no files")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := ioutil.ReadFile(smallest); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkAssetLargest(b *testing.B) {
	largest := benchmarkFindExtreme(true)
	if len(largest) == 0 {
		b.Error("no files")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := Asset(largest); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkReadFileLargest(b *testing.B) {
	largest := benchmarkFindExtreme(true)
	if len(largest) == 0 {
		b.Error("no files")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := ioutil.ReadFile(largest); err != nil {
			b.Error(err)
		}
	}
}

type bySizeSorter struct {
	items []struct {
		name string
		size int64
	}
}

func (s *bySizeSorter) Add(name string, size int64) {
	s.items = append(s.items, struct {
		name string
		size int64
	}{
		name: name,
		size: size,
	})
}

func (s *bySizeSorter) Get(i int) string {
	return s.items[i].name
}

func (s *bySizeSorter) Len() int {
	return len(s.items)
}

func (s *bySizeSorter) Swap(i, j int) {
	s.items[i], s.items[j] = s.items[j], s.items[i]
}

func (s *bySizeSorter) Less(i, j int) bool {
	return s.items[i].size < s.items[j].size
}

func benchmarkFindMedian() string {
	s := &bySizeSorter{}

	for _, name := range AssetNames() {
		info, _ := AssetInfo(name)
		s.Add(name, info.Size())
	}

	sort.Sort(s)

	return s.Get(s.Len() / 2)
}

func BenchmarkAssetMedian(b *testing.B) {
	median := benchmarkFindMedian()
	if len(median) == 0 {
		b.Error("no files")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := Asset(median); err != nil {
			b.Error(err)
		}
	}
}

func BenchmarkReadFileMedian(b *testing.B) {
	median := benchmarkFindMedian()
	if len(median) == 0 {
		b.Error("no files")
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if _, err := ioutil.ReadFile(median); err != nil {
			b.Error(err)
		}
	}
}
