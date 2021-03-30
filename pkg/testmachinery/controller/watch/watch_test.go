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

package watch_test

import (
	"context"
	"testing"

	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apiextensions-apiserver/pkg/apis/apiextensions/v1beta1"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/client-go/rest"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/envtest"

	"github.com/gardener/test-infra/pkg/apis/testmachinery"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	testmachineryapi "github.com/gardener/test-infra/pkg/testmachinery"
)

func TestConfig(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Testmachinery Watch Suite")
}

var (
	testenv    *envtest.Environment
	restConfig *rest.Config
	tr         *tmv1beta1.Testrun
	fakeClient client.Client
)

var _ = BeforeSuite(func() {
	ctx := context.Background()
	defer ctx.Done()
	var err error
	crd := &v1beta1.CustomResourceDefinition{}
	crd.Name = "testruns.testmachinery.sapcloud.io"
	crd.Spec.Group = testmachinery.GroupName
	crd.Spec.Scope = v1beta1.NamespaceScoped
	crd.Spec.Names = v1beta1.CustomResourceDefinitionNames{
		Kind:     "Testrun",
		Plural:   "testruns",
		Singular: "testrun",
	}
	crd.Spec.Versions = []v1beta1.CustomResourceDefinitionVersion{{
		Name:    "v1beta1",
		Served:  true,
		Storage: true,
	}}
	crd.Spec.Subresources = &v1beta1.CustomResourceSubresources{
		Status: &v1beta1.CustomResourceSubresourceStatus{},
	}
	testenv = &envtest.Environment{
		CRDs: []client.Object{crd},
	}

	restConfig, err = testenv.Start()
	Expect(err).ToNot(HaveOccurred())

	fakeClient, err = client.New(restConfig, client.Options{Scheme: testmachineryapi.TestMachineryScheme})
	Expect(err).ToNot(HaveOccurred())

	ns := &corev1.Namespace{}
	ns.Name = "test"
	if err := fakeClient.Create(ctx, ns); err != nil {
		if !apierrors.IsAlreadyExists(err) {
			Expect(err).ToNot(HaveOccurred())
		}
	}
})

var _ = AfterSuite(func() {
	Expect(testenv.Stop()).ToNot(HaveOccurred())
})
