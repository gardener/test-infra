// Copyright 2020 Copyright (c) 2020 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//      http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package s3

import (
	"github.com/gardener/test-infra/pkg/apis/config"
	"github.com/minio/minio-go"
	"github.com/pkg/errors"
	"io"
)

// Client is a interface to interact with a S3 object store
type Client interface {
	GetObject(bucketName, objectName string, opts minio.GetObjectOptions) (Object, error)
	RemoveObject(bucketName, key string) error
}

// Object represents an s3 object that can be retrieved from an object store
type Object interface {
	io.Reader
	Stat() (minio.ObjectInfo, error)
	Close() error
}

type client struct {
	minioClient       *minio.Client
	defaultBucketName string
}

// Config holds connection information for a s3 object storage
type Config struct {
	Endpoint   string
	SSL        bool
	BucketName string
	AccessKey  string
	SecretKey  string
}

// New creates a new s3 client which is a wrapper of the minio client
func New(config *Config) (Client, error) {
	minioClient, err := minio.New(config.Endpoint, config.AccessKey, config.SecretKey, config.SSL)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to create s3 client for %s", config.Endpoint)
	}

	ok, err := minioClient.BucketExists(config.BucketName)
	if err != nil {
		return nil, errors.Wrapf(err, "error getting bucket %s", config.BucketName)
	}
	if !ok {
		return nil, errors.Errorf("bucket %s does not exist", config.BucketName)
	}
	return &client{
		minioClient:       minioClient,
		defaultBucketName: config.BucketName,
	}, nil
}

func (c *client) GetObject(bucketName, objectName string, opts minio.GetObjectOptions) (Object, error) {
	if bucketName == "" {
		bucketName = c.defaultBucketName
	}
	return c.minioClient.GetObject(bucketName, objectName, opts)
}

func (c *client) RemoveObject(bucketName, key string) error {
	if bucketName == "" {
		bucketName = c.defaultBucketName
	}
	return c.minioClient.RemoveObject(bucketName, key)
}

func FromConfig(s3 *config.S3) *Config {
	return &Config{
		Endpoint:   s3.Server.Endpoint,
		SSL:        s3.Server.SSL,
		BucketName: s3.BucketName,
		AccessKey:  s3.AccessKey,
		SecretKey:  s3.SecretKey,
	}
}
