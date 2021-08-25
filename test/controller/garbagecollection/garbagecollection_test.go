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

package garbagecollection_test

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"github.com/gardener/gardener/pkg/utils/retry"

	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/minio/minio-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var _ = Describe("Garbage collection tests", func() {

	It("should cleanup all artifacts when a TestDef is deleted", func() {
		ctx := context.Background()
		defer ctx.Done()
		tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())

		tr, wf, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.NodeSucceeded, TestrunDurationTimeout)
		Expect(err).ToNot(HaveOccurred())
		utils.DeleteTestrun(operation.Client(), tr)

		err = retry.UntilTimeout(ctx, 5*time.Second, InitializationTimeout, func(ctx context.Context) (bool, error) {
			if err := operation.Client().Get(ctx, client.ObjectKey{Namespace: operation.TestNamespace(), Name: tr.Name}, tr); err != nil {
				if errors.IsNotFound(err) {
					return retry.Ok()
				}
				return retry.MinorError(err)
			}
			return retry.NotOk()
		})
		Expect(err).ToNot(HaveOccurred())

		// check if artifacts are deleted
		ok, err := minioClient.BucketExists(minioBucket)
		Expect(err).ToNot(HaveOccurred(), "Testrun: %s", tr.Name)
		Expect(ok).To(BeTrue(), "Testrun: %s", tr.Name)

		_, err = minioClient.StatObject(minioBucket, fmt.Sprintf("testmachinery/%s", wf.Name), minio.StatObjectOptions{})
		Expect(err).To(HaveOccurred(), "Testrun: %s", tr.Name)
		Expect(minio.ToErrorResponse(err).StatusCode).To(Equal(http.StatusNotFound), "Testrun: %s", tr.Name)
	})
})
