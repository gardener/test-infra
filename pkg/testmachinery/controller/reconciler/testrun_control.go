// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package reconciler

import (
	"context"
	"fmt"
	"time"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/go-logr/logr"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/controller/controllerutil"
	"sigs.k8s.io/controller-runtime/pkg/reconcile"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/testmachinery/collector"
	"github.com/gardener/test-infra/pkg/testmachinery/testrun"
	"github.com/gardener/test-infra/pkg/util/s3"
)

// New returns a new testmachinery reconciler
func New(log logr.Logger, kubeClient client.Client, scheme *runtime.Scheme, s3Client s3.Client, c collector.Interface) reconcile.Reconciler {
	return &TestmachineryReconciler{
		Client:    kubeClient,
		scheme:    scheme,
		Logger:    log,
		s3Client:  s3Client,
		collector: c,
		timers:    make(map[string]*time.Timer),
	}
}

// Reconcile handles various testrun events like crete, update and delete.
func (r *TestmachineryReconciler) Reconcile(ctx context.Context, request reconcile.Request) (reconcile.Result, error) {
	rCtx := &reconcileContext{
		tr: &tmv1beta1.Testrun{},
	}
	log := r.Logger.WithValues("testrun", request.NamespacedName)

	log.V(3).Info("start reconcile")

	err := r.Get(ctx, request.NamespacedName, rCtx.tr)
	if err != nil {
		log.Error(err, "unable to find testrun")
		if errors.IsNotFound(err) {
			return reconcile.Result{Requeue: false}, nil
		}
		return reconcile.Result{Requeue: true}, nil
	}

	if rCtx.tr.DeletionTimestamp != nil {
		log.Info("deletion caused by testrun")
		return r.deleteTestrun(ctx, rCtx)
	}

	///////////////
	// RECONCILE //
	///////////////

	if rCtx.tr.Status.Phase.Completed() {
		return reconcile.Result{}, nil
	}

	rCtx.wf = &argov1.Workflow{}
	err = r.Get(ctx, types.NamespacedName{Name: testmachinery.GetWorkflowName(rCtx.tr), Namespace: rCtx.tr.Namespace}, rCtx.wf)
	if err != nil {
		if !errors.IsNotFound(err) {
			log.Error(err, "unable to get workflow", "workflow", testmachinery.GetWorkflowName(rCtx.tr), "namespace", rCtx.tr.Namespace)
			return reconcile.Result{}, err
		}
		if rCtx.tr.Status.CompletionTime != nil {
			log.Error(err, "unable to get workflow but testrun is already finished", "workflow", testmachinery.GetWorkflowName(rCtx.tr), "namespace", rCtx.tr.Namespace)
			return reconcile.Result{}, err
		}

		if err, retry := testrun.Validate(log, rCtx.tr); err != nil {
			if !retry || RetryTimeoutExceeded(rCtx.tr) {
				rCtx.tr.Status.Phase = tmv1beta1.RunPhaseError
				t := metav1.Now()
				rCtx.tr.Status.CompletionTime = &t
			}
			rCtx.tr.Status.State = fmt.Sprintf("validation failed: %s", err.Error())
			if err := r.Status().Update(ctx, rCtx.tr); err != nil {
				log.Error(err, "unable to update testrun status")
				return reconcile.Result{}, err
			}
			return reconcile.Result{}, err
		}

		if res, err := r.createWorkflow(ctx, rCtx, log); err != nil {
			return res, err
		}
	}

	if rCtx.tr.Status.CompletionTime != nil {
		if rCtx.wf.DeletionTimestamp != nil {
			log.V(2).Info("Deletion: cause workflow")
			return r.deleteTestrun(ctx, rCtx)
		}
		return reconcile.Result{}, err
	}

	if err := r.handleActions(ctx, rCtx); err != nil {
		return reconcile.Result{}, err
	}

	return r.updateStatus(ctx, rCtx)
}

func (r *TestmachineryReconciler) createWorkflow(ctx context.Context, rCtx *reconcileContext, log logr.Logger) (reconcile.Result, error) {
	log.V(5).Info("generate workflow")
	var (
		k8sHelperResources []client.Object
		err                error
	)
	rCtx.wf, k8sHelperResources, err = r.generateWorkflow(ctx, rCtx.tr)
	if err != nil {
		log.Error(err, "unable to setup workflow")
		return reconcile.Result{}, err
	}

	// create additional kubernetes objects
	log.Info("creating helper resources")
	for _, obj := range k8sHelperResources {
		log.V(5).Info(fmt.Sprintf("creating helper resource %s/%s", obj.GetObjectKind().GroupVersionKind().Group, obj.GetObjectKind().GroupVersionKind().Kind))
		if err := r.Create(ctx, obj); err != nil {
			log.Error(err, "unable to create resource", "group", obj.GetObjectKind().GroupVersionKind().Group, "kind", obj.GetObjectKind().GroupVersionKind().Kind)
			return reconcile.Result{Requeue: true}, err
		}
	}

	log.Info("creating workflow", "workflow", rCtx.wf.Name, "namespace", rCtx.wf.Namespace)
	if err := r.Create(ctx, rCtx.wf); err != nil {
		log.Error(err, "unable to create workflow", "workflow", rCtx.wf.Name, "namespace", rCtx.wf.Namespace)
		return reconcile.Result{Requeue: true}, err
	}

	rCtx.tr.Status.Workflow = rCtx.wf.Name
	rCtx.tr.Status.Phase = tmv1beta1.RunPhaseRunning

	// update status first because otherwise it would be lost during the patch call
	rCtx.tr.Status.ObservedGeneration = rCtx.tr.Generation
	if err := r.Status().Update(ctx, rCtx.tr); err != nil {
		log.Error(err, "unable to update testrun status")
		return reconcile.Result{}, err
	}

	patch := client.MergeFrom(rCtx.tr.DeepCopy())

	// add finalizers for testrun
	trFinalizers := sets.New[string](rCtx.tr.Finalizers...)
	if !trFinalizers.Has(tmv1beta1.SchemeGroupVersion.Group) {
		trFinalizers.Insert(tmv1beta1.SchemeGroupVersion.Group)
	}
	if !trFinalizers.Has(metav1.FinalizerDeleteDependents) {
		trFinalizers.Insert(metav1.FinalizerDeleteDependents)
	}
	rCtx.tr.Finalizers = trFinalizers.UnsortedList()

	if err := r.Patch(ctx, rCtx.tr, patch); err != nil {
		return reconcile.Result{}, err
	}

	rCtx.updated = true
	return reconcile.Result{}, nil
}

func (r *TestmachineryReconciler) generateWorkflow(ctx context.Context, testrunDef *tmv1beta1.Testrun) (*argov1.Workflow, []client.Object, error) {
	tr, err := testrun.New(ctx, r.Logger.WithValues("testrun", types.NamespacedName{Name: testrunDef.Name, Namespace: testrunDef.Namespace}), testrunDef, r.Client)
	if err != nil {
		return nil, nil, fmt.Errorf("error parsing testrun: %s", err.Error())
	}

	wf, err := tr.GetWorkflow(testmachinery.GetWorkflowName(testrunDef), testrunDef.Namespace, r.getImagePullSecrets())
	if err != nil {
		return nil, nil, err
	}

	if err := controllerutil.SetControllerReference(testrunDef, wf, r.scheme); err != nil {
		return nil, nil, err
	}

	wfFinalizers := sets.New[string](wf.Finalizers...)
	if !wfFinalizers.Has(tmv1beta1.SchemeGroupVersion.Group) {
		wfFinalizers.Insert(tmv1beta1.SchemeGroupVersion.Group)
		wf.Finalizers = wfFinalizers.UnsortedList()
	}

	testrunDef.Status.Steps = tr.Testflow.Flow.GetStatuses()

	return wf, tr.HelperResources, nil
}
