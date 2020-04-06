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

package validation

import (
	"github.com/gardener/test-infra/pkg/apis/config"
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// ValidateConfiguration validates the passed configuration instance
func ValidateConfiguration(config *config.Configuration) field.ErrorList {
	allErrs := field.ErrorList{}

	if !config.TestMachineryConfiguration.DisableCollector {
		if config.ElasticSearchConfiguration == nil {
			allErrs = append(allErrs, field.Required(field.NewPath("elasticsearchConfiguration"), "elastic search config is required if collector is enabled"))
		}
	}

	allErrs = append(allErrs, validateS3Config(config.S3Configuration, field.NewPath("s3Configuration"))...)

	return allErrs
}

// validateS3Config validates the passed s3 configuration instance
func validateS3Config(s3 *config.S3Configuration, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if s3.Server.Minio == nil && len(s3.Server.Endpoint) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("server.endpoint"), "endpoint or minio has to be defined"))
	}
	if len(s3.AccessKey) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("accessKey"), "no s3 access key is specified"))
	}
	if len(s3.SecretKey) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("secretKey"), "no s3 secret key is specified"))
	}
	if len(s3.BucketName) == 0 {
		allErrs = append(allErrs, field.Required(fldPath.Child("bucketName"), "no s3 bucket is specified"))
	}

	return allErrs
}
