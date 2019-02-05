// Copyright 2019 Copyright (c) 2019 SAP SE or an SAP affiliate company. All rights reserved. This file is licensed under the Apache Software License, v. 2 except as noted otherwise in the LICENSE file.
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

package garbagecollection

import (
	"fmt"
	"time"

	"github.com/gardener/test-infra/pkg/testmachinery"

	"github.com/minio/minio-go"
)

const (
	cleanupDays = 14
)

// NewObjectStore fetches endpoint and credentials, and creates a ObjectStorage object.
func NewObjectStore() (*ObjectStore, error) {

	cfg := testmachinery.GetConfig().ObjectStore

	minioClient, err := minio.New(cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, false)
	if err != nil {
		return nil, err
	}

	ok, err := minioClient.BucketExists(cfg.BucketName)
	if err != nil {
		return nil, fmt.Errorf("Error getting bucket name %s: %s", cfg.BucketName, err.Error())
	}
	if !ok {
		return nil, fmt.Errorf("Bucket %s does not exist", cfg.BucketName)
	}

	return &ObjectStore{cfg.Endpoint, cfg.AccessKey, cfg.SecretKey, cfg.BucketName, minioClient}, nil
}

// DeleteObject deletes an object from the object store
func (o *ObjectStore) DeleteObject(key string) error {
	return o.client.RemoveObject(o.bucketName, key)
}

// CleanTestrun deletes all data of a Testrun
func (o *ObjectStore) CleanTestrun(trName string) {
	// get all objects of the testrun
	doneCh := make(chan struct{})
	defer close(doneCh)
	for object := range o.client.ListObjectsV2(o.bucketName, "testmachinery/"+trName, true, doneCh) {
		o.client.RemoveObject(o.bucketName, object.Key)
	}
}

// CleanOldData deletes all data from object storage that is older than cleanupDays
func (o *ObjectStore) CleanOldData() {
	// get all objects of the testrun
	doneCh := make(chan struct{})
	defer close(doneCh)
	for object := range o.client.ListObjectsV2(o.bucketName, "testmachinery", true, doneCh) {
		cleanupDate := time.Now().Add(-(time.Hour * 24 * cleanupDays))
		if object.LastModified.Before(cleanupDate) {
			o.client.RemoveObject(o.bucketName, object.Key)
		}
	}
}
