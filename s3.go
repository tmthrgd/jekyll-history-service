// Copyright 2016 Tom Thorogood. All rights reserved.
// Use of this source code is governed by a
// Modified BSD License license that can be found in
// the LICENSE file.

package main

import (
	"fmt"
	"net/http"
	"os"

	"github.com/gregjones/httpcache"
	_ "github.com/joho/godotenv/autoload"
	"github.com/mitchellh/goamz/aws"
	"github.com/mitchellh/goamz/s3"
)

func getS3Buckets() (s3Bucket *s3.Bucket, s3BucketNoGzip *s3.Bucket, err error) {
	endpoint := os.Getenv("S3_ENDPOINT")
	if len(endpoint) == 0 {
		endpoint = "us-east-1"
	}

	bucket := os.Getenv("S3_BUCKET")
	if len(bucket) == 0 {
		return nil, nil, fmt.Errorf("S3_BUCKET must be set")
	}

	region, ok := aws.Regions[endpoint]
	if !ok {
		return nil, nil, fmt.Errorf("invalid S3_ENDPOINT value of %s", endpoint)
	}

	auth, err := aws.EnvAuth()
	if err != nil {
		return
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

	return
}
