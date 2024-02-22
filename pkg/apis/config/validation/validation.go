// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package validation

import (
	"k8s.io/apimachinery/pkg/util/validation/field"

	"github.com/gardener/test-infra/pkg/apis/config"
)

// ValidateConfiguration validates the passed configuration instance
func ValidateConfiguration(config *config.Configuration) field.ErrorList {
	allErrs := field.ErrorList{}

	if !config.TestMachinery.DisableCollector {
		if config.ElasticSearch == nil {
			allErrs = append(allErrs, field.Required(field.NewPath("elasticsearchConfiguration"), "elastic search config is required if collector is enabled"))
		}
	}

	allErrs = append(allErrs, validateS3Config(config.S3, field.NewPath("s3Configuration"))...)

	return allErrs
}

// validateS3Config validates the passed s3 configuration instance
func validateS3Config(s3 *config.S3, fldPath *field.Path) field.ErrorList {
	allErrs := field.ErrorList{}

	if len(s3.Server.Endpoint) == 0 {
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
