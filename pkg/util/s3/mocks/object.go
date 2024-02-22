// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

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
