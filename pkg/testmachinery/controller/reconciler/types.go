// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"time"

	"github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/collector"
	"github.com/gardener/test-infra/pkg/util/s3"
)

type TestmachineryReconciler struct {
	client.Client
	scheme    *runtime.Scheme
	Logger    logr.Logger
	collector collector.Interface
	s3Client  s3.Client

	timers map[string]*time.Timer
}

type reconcileContext struct {
	tr      *v1beta1.Testrun
	wf      *v1alpha1.Workflow
	updated bool
}
