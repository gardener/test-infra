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

package framework

import (
	"context"
	"fmt"
	"strings"

	"github.com/hashicorp/go-multierror"

	rbacv1 "k8s.io/api/rbac/v1"

	"github.com/onsi/gomega"
	"github.com/pkg/errors"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"

	"github.com/gardener/test-infra/pkg/util"
)

// EnsureTestNamespace creates the namespace specified in the config it does not exist.
// if o.config.TestNamespace is empty a random generated namespace will be created.
// Additionally necessary resources were created and added to the operation state.
func (o *Operation) EnsureTestNamespace(ctx context.Context) error {

	if err := o.createNewTestNamespace(ctx); err != nil {
		return errors.Wrapf(err, "unable to create new test namespace")
	}

	if err := o.copyDefaultSecretsToNamespace(ctx, o.testConfig.Namespace); err != nil {
		return errors.Wrapf(err, "unable to copy secrets to new namespace")
	}
	if err := o.setupNamespace(ctx, o.testConfig.Namespace); err != nil {
		return errors.Wrapf(err, "unable to setup new namespace")
	}

	o.log.WithName("framework").Info(fmt.Sprintf("using namespace %s", o.TestNamespace()))
	return nil
}

func (o *Operation) createNewTestNamespace(ctx context.Context) error {
	var ns *corev1.Namespace
	if len(o.testConfig.Namespace) != 0 {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: o.testConfig.Namespace,
			},
		}
	} else {
		ns = &corev1.Namespace{
			ObjectMeta: metav1.ObjectMeta{
				Name: fmt.Sprintf("%s-%s", TestNamespacePrefix, util.RandomString(3)),
			},
		}
	}

	if _, err := controllerutil.CreateOrUpdate(ctx, o.Client(), ns, func() error { return nil }); err != nil {
		return errors.Wrapf(err, "unable to create new test namespace")
	}
	o.State.AppendObject(ns)
	o.testConfig.Namespace = ns.Name
	return nil
}

func (o *Operation) setupNamespace(ctx context.Context, namespace string) error {
	// Create rbac binding for default ServiceAccount in new namespace
	rb := &rbacv1.ClusterRoleBinding{
		ObjectMeta: metav1.ObjectMeta{
			Name: namespace,
		},
		RoleRef: rbacv1.RoleRef{
			APIGroup: "rbac.authorization.k8s.io",
			Kind:     "ClusterRole",
			Name:     "argo-workflow-role",
		},
		Subjects: []rbacv1.Subject{
			rbacv1.Subject{
				Kind:      "ServiceAccount",
				Name:      "default",
				Namespace: namespace,
			},
		},
	}
	if _, err := controllerutil.CreateOrUpdate(ctx, o.Client(), rb, func() error { return nil }); err != nil {
		return errors.Wrapf(err, "unable to create cluster rolebinding %s", namespace)
	}
	o.State.AppendObject(rb)
	return nil
}

func (o *Operation) copyDefaultSecretsToNamespace(ctx context.Context, namespace string) error {
	for _, secretName := range CoreSecrets {
		secret := &corev1.Secret{}
		if err := o.Client().Get(ctx, client.ObjectKey{Name: secretName, Namespace: o.testConfig.TmNamespace}, secret); err != nil {
			return errors.Wrapf(err, "unable to fetch secret %s from namespace %s", secretName, o.testConfig.TmNamespace)
		}

		newSecret := &corev1.Secret{
			ObjectMeta: metav1.ObjectMeta{
				Name:      secretName,
				Namespace: namespace,
			},
			Data: secret.Data,
		}
		if _, err := controllerutil.CreateOrUpdate(ctx, o.Client(), newSecret, func() error { return nil }); err != nil {
			return errors.Wrapf(err, "unable to create new secret %s in namespace %s", secretName, namespace)
		}
		o.State.AppendObject(newSecret)
	}

	return nil
}

// AfterSuite should be registered as ginkgo's after suite.
// It cleans up all previously created resources that are in the operation state.
func (o *Operation) AfterSuite() {
	ctx := context.Background()
	defer ctx.Done()
	o.log.Info("deleting namespace", "namespace", o.TestNamespace())
	if !strings.HasPrefix(o.TestNamespace(), TestNamespacePrefix) {
		return
	}

	var res *multierror.Error
	for _, obj := range o.State.Objects {
		if err := o.Client().Delete(ctx, obj); err != nil {
			res = multierror.Append(res, err)
		}
	}

	gomega.Expect(res.ErrorOrNil()).ToNot(gomega.HaveOccurred())
}
