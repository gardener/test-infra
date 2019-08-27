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

package testmachinery

import (
	"github.com/gardener/test-infra/pkg/util"
	"github.com/hashicorp/go-multierror"
	"github.com/pkg/errors"
)

// ValidateS3Config validates a object store configuration
// and returns an error if the config is invalid.
// ValidateConfig validates a s3 configuration
func ValidateS3Config(config *S3Config) error {
	var result *multierror.Error
	if config == nil {
		return nil
	}
	if len(config.Endpoint) == 0 {
		result = multierror.Append(result, errors.New("no s3 endpoint is specified"))
	}
	if len(config.AccessKey) == 0 {
		result = multierror.Append(result, errors.New("no s3 access key is specified"))
	}
	if len(config.SecretKey) == 0 {
		result = multierror.Append(result, errors.New("no s3 secret key is specified"))
	}
	if len(config.BucketName) == 0 {
		result = multierror.Append(result, errors.New("no s3 bucket is specified"))
	}

	return util.ReturnMultiError(result)
}
