// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package config

// S3 holds information about the s3 endpoint
type S3 struct {
	Server     S3Server `json:"server"`
	BucketName string   `json:"bucketName,omitempty"`
	AccessKey  string   `json:"accessKey,omitempty"`
	SecretKey  string   `json:"secretKey,omitempty"`
}

// S3Server defines the used s3 server
type S3Server struct {
	Endpoint string `json:"endpoint,omitempty"`
	SSL      bool   `json:"ssl,omitempty"`
}
