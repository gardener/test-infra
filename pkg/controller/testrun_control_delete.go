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

package controller

import (
	"context"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"time"

	argov1 "github.com/argoproj/argo/pkg/apis/workflow/v1alpha1"
	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/garbagecollection"
	log "github.com/sirupsen/logrus"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"
)

func (r *TestrunReconciler) deleteTestrun(ctx context.Context, tr *tmv1beta1.Testrun) (reconcile.Result, error) {

	if finalizers := sets.NewString(tr.GetFinalizers()...); !finalizers.Has(tmv1beta1.SchemeGroupVersion.Group) {
		return reconcile.Result{}, nil
	}

	foundWf := &argov1.Workflow{}
	if err := r.Get(ctx, types.NamespacedName{Name: testmachinery.GetWorkflowName(tr), Namespace: tr.Namespace}, foundWf); err == nil {
		log.Infof("Cleanup testrun %s", tr.Name)

		// garbage collect all outputs by traversing through nodes and collect outputs artifacts from minio
		os, err := garbagecollection.NewObjectStore()
		if err != nil {
			log.Debug(err.Error())
			return reconcile.Result{Requeue: true, RequeueAfter: 30 * time.Second}, err
		}
		for _, node := range foundWf.Status.Nodes {
			if node.Outputs == nil {
				continue
			}
			for _, artifact := range node.Outputs.Artifacts {
				log.Debugf("Processing artifact %s", artifact.Name)
				if artifact.S3 != nil {
					err := os.DeleteObject(artifact.S3.Key)
					if err != nil {
						log.Errorf("Cannot delete object %s from object storage: %s", artifact.S3.Key, err.Error())

						// do not retry deletion if the key does not not exist in s3 anymore
						// maybe use const from aws lib -> need to change to aws lib
						if err.Error() != "The specified key does not exist." {
							return reconcile.Result{Requeue: true, RequeueAfter: 30 * time.Second}, err
						}
					}
					log.Debugf("Deleted object %s from object store", artifact.S3.Key)
				}
			}
		}

		if removeFinalizer(foundWf, tmv1beta1.SchemeGroupVersion.Group) {
			err = r.Update(ctx, foundWf)
			if err != nil {
				log.Errorf("Error while removing finalizer from Workflow %s: %s", tr.Name, err.Error())
				return reconcile.Result{}, err
			}
		}

		log.Infof("Deleting testrun %s", foundWf.Name)
		if tr.DeletionTimestamp == nil {
			removeFinalizer(tr, tmv1beta1.SchemeGroupVersion.Group)
			err = r.Delete(ctx, tr)
			if err != nil {
				log.Errorf("Cannot delete testrun %s: %s", foundWf.Name, err.Error())
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}
	}

	//remove finalizers
	if removeFinalizer(tr, tmv1beta1.SchemeGroupVersion.Group) {
		err := r.Update(ctx, tr)
		if err != nil {
			log.Debugf("Cannot remove finalizer from Testrun %s: %s", tr.Name, err.Error())
			return reconcile.Result{}, err
		}
	}
	return reconcile.Result{}, nil
}

func removeFinalizer(obj metav1.Object, finalizer string) bool {
	finalizers := sets.NewString(obj.GetFinalizers()...)
	if !finalizers.Has(finalizer) {
		return false
	}
	finalizers.Delete(tmv1beta1.SchemeGroupVersion.Group)
	obj.SetFinalizers(finalizers.UnsortedList())
	return true
}
