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

package mock_s3

import (
	"io"
	"os"

	"github.com/minio/minio-go"

	"github.com/gardener/test-infra/pkg/util/s3"
)

type mockObject struct {
	io.Reader
	fileInfo os.FileInfo
}

func (o *mockObject) Stat() (minio.ObjectInfo, error) {
	return minio.ObjectInfo{
		Size: o.fileInfo.Size(),
	}, nil
}

func (o *mockObject) Close() error { return nil }

// CreateS3ObjectFromFile creates a mock S§ Object from a file
func CreateS3ObjectFromFile(file string) (s3.Object, error) {
	fileInfo, err := os.Stat(file)
	if os.IsExist(err) {
		return nil, err
	}

	f, err := os.Open(file)
	if err != nil {
		return nil, err
	}

	return &mockObject{
		Reader:   f,
		fileInfo: fileInfo,
	}, nil
}
