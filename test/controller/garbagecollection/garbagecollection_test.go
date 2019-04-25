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
	"os"
	"time"

	"github.com/gardener/test-infra/pkg/testmachinery"

	"github.com/gardener/gardener/pkg/client/kubernetes"
	"sigs.k8s.io/controller-runtime/pkg/client"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	minio "github.com/minio/minio-go"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"k8s.io/apimachinery/pkg/api/errors"

	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var (
	maxWaitTime int64 = 300
)

var _ = Describe("Garbage collection tests", func() {

	var (
		commitSha   string
		namespace   string
		tmClient    kubernetes.Interface
		minioClient *minio.Client
		minioBucket string
	)

	BeforeSuite(func() {
		var err error
		commitSha = os.Getenv("GIT_COMMIT_SHA")
		tmKubeconfig := os.Getenv("TM_KUBECONFIG_PATH")
		namespace = os.Getenv("TM_NAMESPACE")
		minioEndpoint := os.Getenv("S3_ENDPOINT")

		tmClient, err = kubernetes.NewClientFromFile("", tmKubeconfig, client.Options{
			Scheme: testmachinery.TestMachineryScheme,
		})
		Expect(err).ToNot(HaveOccurred())

		Expect(utils.WaitForClusterReadiness(tmClient, namespace, maxWaitTime)).ToNot(HaveOccurred())
		osConfig := utils.WaitForMinioService(tmClient, minioEndpoint, namespace, maxWaitTime)

		minioBucket = osConfig.BucketName
		minioClient, err = minio.New(osConfig.Endpoint, osConfig.AccessKey, osConfig.SecretKey, false)
		Expect(err).ToNot(HaveOccurred())
	})

	It("should cleanup all artifacts when a TestDef is deleted", func() {
		ctx := context.Background()
		defer ctx.Done()
		tr := resources.GetBasicTestrun(namespace, commitSha)

		tr, wf, err := utils.RunTestrun(ctx, tmClient, tr, argov1.NodeSucceeded, namespace, maxWaitTime)
		Expect(err).ToNot(HaveOccurred())
		utils.DeleteTestrun(tmClient, tr)

		startTime := time.Now()
		for {
			Expect(util.MaxTimeExceeded(startTime, maxWaitTime)).To(BeFalse(), "Max Wait time exceeded.")

			if err := tmClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: tr.Name}, tr); err != nil {
				Expect(errors.IsNotFound(err)).To(BeTrue(), "Testrun: %s", tr.Name)
				break
			}

			time.Sleep(5 * time.Second)
		}

		// check if artifacts are deleted
		ok, err := minioClient.BucketExists(minioBucket)
		Expect(err).ToNot(HaveOccurred(), "Testrun: %s", tr.Name)
		Expect(ok).To(BeTrue(), "Testrun: %s", tr.Name)

		_, err = minioClient.StatObject(minioBucket, fmt.Sprintf("testmachinery/%s", wf.Name), minio.StatObjectOptions{})
		Expect(err).To(HaveOccurred(), "Testrun: %s", tr.Name)
		Expect(minio.ToErrorResponse(err).StatusCode).To(Equal(http.StatusNotFound), "Testrun: %s", tr.Name)
	})
})
