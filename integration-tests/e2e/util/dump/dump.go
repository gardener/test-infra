// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package dump

import (
	"context"
	"fmt"
	"sort"

	"github.com/hashicorp/go-multierror"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metricsv1beta1 "k8s.io/metrics/pkg/apis/metrics/v1beta1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	kutil "github.com/gardener/test-infra/pkg/util/kubernetes"
)

const (
	healthy   = "healthy"
	unhealthy = "unhealthy"
)

// Logger is simplified logging interface so that we can use the old logrus logger and the logr.Logger.
type Logger interface {
	Log(msg string)
}

// LoggerFunc is a logger func that implements the Logger interface.
type LoggerFunc func(msg string)

func (f LoggerFunc) Log(msg string) {
	f(msg)
}

// KubernetesDumper is helper struct that can print different information from a kubernetes cluster.
type KubernetesDumper struct {
	log       Logger
	k8sClient client.Client
}

//NewKubernetesDumper creates a new default k8s dumper.
func NewKubernetesDumper(log Logger, k8sClient client.Client) *KubernetesDumper {
	return &KubernetesDumper{
		log:       log,
		k8sClient: k8sClient,
	}
}

// DumpDefaultResourcesInAllNamespaces dumps all default k8s resources of a namespace
func (d *KubernetesDumper) DumpDefaultResourcesInAllNamespaces(ctx context.Context, ctxIdentifier string) error {
	namespaces := &corev1.NamespaceList{}
	if err := d.k8sClient.List(ctx, namespaces); err != nil {
		return err
	}

	var result error

	for _, ns := range namespaces.Items {
		if err := d.DumpDefaultResourcesInNamespace(ctx, ctxIdentifier, ns.Name); err != nil {
			result = multierror.Append(result, err)
		}
	}
	return result
}

// DumpDefaultResourcesInNamespace dumps all default K8s resources of a namespace.
func (d *KubernetesDumper) DumpDefaultResourcesInNamespace(ctx context.Context, ctxIdentifier string, namespace string) error {
	var result error
	if err := d.DumpEventsInNamespace(ctx, ctxIdentifier, namespace); err != nil {
		result = multierror.Append(result, fmt.Errorf("unable to fetch Events from namespace %s: %s", namespace, err.Error()))
	}
	if err := d.dumpPodInfoForNamespace(ctx, ctxIdentifier, namespace); err != nil {
		result = multierror.Append(result, fmt.Errorf("unable to fetch information of Pods from namespace %s: %s", namespace, err.Error()))
	}
	if err := d.dumpDeploymentInfoForNamespace(ctx, ctxIdentifier, namespace); err != nil {
		result = multierror.Append(result, fmt.Errorf("unable to fetch information of Deployments from namespace %s: %s", namespace, err.Error()))
	}
	if err := d.dumpStatefulSetInfoForNamespace(ctx, ctxIdentifier, namespace); err != nil {
		result = multierror.Append(result, fmt.Errorf("unable to fetch information of StatefulSets from namespace %s: %s", namespace, err.Error()))
	}
	if err := d.dumpDaemonSetInfoForNamespace(ctx, ctxIdentifier, namespace); err != nil {
		result = multierror.Append(result, fmt.Errorf("unable to fetch information of DaemonSets from namespace %s: %s", namespace, err.Error()))
	}
	if err := d.dumpServiceInfoForNamespace(ctx, ctxIdentifier, namespace); err != nil {
		result = multierror.Append(result, fmt.Errorf("unable to fetch information of Services from namespace %s: %s", namespace, err.Error()))
	}
	if err := d.dumpVolumeInfoForNamespace(ctx, ctxIdentifier, namespace); err != nil {
		result = multierror.Append(result, fmt.Errorf("unable to fetch information of Volumes from namespace %s: %s", namespace, err.Error()))
	}
	return result
}

// dumpDeploymentInfoForNamespace prints information about all Deployments of a namespace
func (d *KubernetesDumper) dumpDeploymentInfoForNamespace(ctx context.Context, ctxIdentifier string, namespace string) error {
	d.Logf("%s [NAMESPACE %s] [DEPLOYMENTS]", ctxIdentifier, namespace)
	deployments := &appsv1.DeploymentList{}
	if err := d.k8sClient.List(ctx, deployments, client.InNamespace(namespace)); err != nil {
		return err
	}
	for _, deployment := range deployments.Items {
		if err := kutil.CheckDeployment(&deployment); err != nil {
			d.Logf("Deployment %s is %s with %d/%d replicas - Error: %s - Conditions %v", deployment.Name, unhealthy, deployment.Status.AvailableReplicas, deployment.Status.Replicas, err.Error(), deployment.Status.Conditions)
			continue
		}
		d.Logf("Deployment %s is %s with %d/%d replicas", deployment.Name, healthy, deployment.Status.AvailableReplicas, deployment.Status.Replicas)
	}
	d.Logln()
	return nil
}

// dumpStatefulSetInfoForNamespace prints information about all StatefulSets of a namespace
func (d *KubernetesDumper) dumpStatefulSetInfoForNamespace(ctx context.Context, ctxIdentifier string, namespace string) error {
	d.Logf("%s [NAMESPACE %s] [STATEFULSETS]", ctxIdentifier, namespace)
	statefulSets := &appsv1.StatefulSetList{}
	if err := d.k8sClient.List(ctx, statefulSets, client.InNamespace(namespace)); err != nil {
		return err
	}
	for _, statefulSet := range statefulSets.Items {
		if err := kutil.CheckStatefulSet(&statefulSet); err != nil {
			d.Logf("StatefulSet %s is %s with %d/%d replicas - Error: %s - Conditions %v", statefulSet.Name, unhealthy, statefulSet.Status.ReadyReplicas, statefulSet.Status.Replicas, err.Error(), statefulSet.Status.Conditions)
			continue
		}
		d.Logf("StatefulSet %s is %s with %d/%d replicas", statefulSet.Name, healthy, statefulSet.Status.ReadyReplicas, statefulSet.Status.Replicas)
	}
	d.Logln()
	return nil
}

// dumpDaemonSetInfoForNamespace prints information about all DaemonSets of a namespace
func (d *KubernetesDumper) dumpDaemonSetInfoForNamespace(ctx context.Context, ctxIdentifier string, namespace string) error {
	d.Logf("%s [NAMESPACE %s] [DAEMONSETS]", ctxIdentifier, namespace)
	daemonSets := &appsv1.DaemonSetList{}
	if err := d.k8sClient.List(ctx, daemonSets, client.InNamespace(namespace)); err != nil {
		return err
	}
	for _, ds := range daemonSets.Items {
		if err := kutil.CheckDaemonSet(&ds); err != nil {
			d.Logf("DaemonSet %s is %s with %d/%d replicas - Error: %s - Conditions %v", ds.Name, unhealthy, ds.Status.CurrentNumberScheduled, ds.Status.DesiredNumberScheduled, err.Error(), ds.Status.Conditions)
			continue
		}
		d.Logf("DaemonSet %s is %s with %d/%d replicas", ds.Name, healthy, ds.Status.CurrentNumberScheduled, ds.Status.DesiredNumberScheduled)
	}
	d.Logln()
	return nil
}

// dumpNamespaceResource prints information about the Namespace itself
func (d *KubernetesDumper) dumpNamespaceResource(ctx context.Context, ctxIdentifier string, namespace string) error {
	d.Logf("%s [NAMESPACE RESOURCE %s]", ctxIdentifier, namespace)
	ns := &corev1.Namespace{}
	if err := d.k8sClient.Get(ctx, client.ObjectKey{Name: namespace}, ns); err != nil {
		return err
	}
	d.Logf("Namespace %s - Spec %+v - Status %+v", namespace, ns.Spec, ns.Status)
	d.Logln()
	return nil
}

// dumpServiceInfoForNamespace prints information about all Services of a namespace
func (d *KubernetesDumper) dumpServiceInfoForNamespace(ctx context.Context, ctxIdentifier string, namespace string) error {
	d.Logf("%s [NAMESPACE %s] [SERVICES]", ctxIdentifier, namespace)
	services := &corev1.ServiceList{}
	if err := d.k8sClient.List(ctx, services, client.InNamespace(namespace)); err != nil {
		return err
	}
	for _, service := range services.Items {
		d.Logf("Service %s - Spec %+v - Status %+v", service.Name, service.Spec, service.Status)
	}
	d.Logln()
	return nil
}

// dumpVolumeInfoForNamespace prints information about all PVs and PVCs of a namespace
func (d *KubernetesDumper) dumpVolumeInfoForNamespace(ctx context.Context, ctxIdentifier string, namespace string) error {
	d.Logf("%s [NAMESPACE %s] [PVC]", ctxIdentifier, namespace)
	pvcs := &corev1.PersistentVolumeClaimList{}
	if err := d.k8sClient.List(ctx, pvcs, client.InNamespace(namespace)); err != nil {
		return err
	}
	for _, pvc := range pvcs.Items {
		d.Logf("PVC %s - Spec %+v - Status %+v", pvc.Name, pvc.Spec, pvc.Status)
	}
	d.Logln()

	d.Logf("%s [NAMESPACE %s] [PV]", ctxIdentifier, namespace)
	pvs := &corev1.PersistentVolumeList{}
	if err := d.k8sClient.List(ctx, pvs, client.InNamespace(namespace)); err != nil {
		return err
	}
	for _, pv := range pvs.Items {
		d.Logf("PV %s - Spec %+v - Status %+v", pv.Name, pv.Spec, pv.Status)
	}
	d.Logln()
	return nil
}

// dumpNodes prints information about all nodes
func (d *KubernetesDumper) dumpNodes(ctx context.Context, ctxIdentifier string) error {
	d.Logf("%s [NODES]", ctxIdentifier)
	nodes := &corev1.NodeList{}
	if err := d.k8sClient.List(ctx, nodes); err != nil {
		return err
	}
	for _, node := range nodes.Items {
		if err := kutil.CheckNode(&node); err != nil {
			d.Logf("Node %s is %s with phase %s - Error: %s - Conditions %v", node.Name, unhealthy, node.Status.Phase, err.Error(), node.Status.Conditions)
		} else {
			d.Logf("Node %s is %s with phase %s", node.Name, healthy, node.Status.Phase)
		}
		d.Logf("Node %s has a capacity of %s cpu, %s memory", node.Name, node.Status.Capacity.Cpu().String(), node.Status.Capacity.Memory().String())

		nodeMetric := &metricsv1beta1.NodeMetrics{}
		if err := d.k8sClient.Get(ctx, client.ObjectKey{Name: node.Name}, nodeMetric); err != nil {
			d.Logf("unable to receive metrics for node %s: %s", node.Name, err.Error())
			continue
		}
		d.Logf("Node %s currently uses %s cpu, %s memory", node.Name, nodeMetric.Usage.Cpu().String(), nodeMetric.Usage.Memory().String())
	}
	d.Logln()
	return nil
}

// dumpPodInfoForNamespace prints node information of all pods in a namespace
func (d *KubernetesDumper) dumpPodInfoForNamespace(ctx context.Context, ctxIdentifier string, namespace string) error {
	d.Logf("%s [NAMESPACE %s] [PODS]", ctxIdentifier, namespace)
	pods := &corev1.PodList{}
	if err := d.k8sClient.List(ctx, pods, client.InNamespace(namespace)); err != nil {
		return err
	}
	for _, pod := range pods.Items {
		d.Logf("Pod %s is %s on Node %s", pod.Name, pod.Status.Phase, pod.Spec.NodeName)
	}
	d.Logln()
	return nil
}

// DumpEventsInNamespace prints all events of a namespace
func (d *KubernetesDumper) DumpEventsInNamespace(ctx context.Context, ctxIdentifier string, namespace string, filters ...EventFilterFunc) error {
	d.Logf("%s [NAMESPACE %s] [EVENTS]", ctxIdentifier, namespace)
	events := &corev1.EventList{}
	if err := d.k8sClient.List(ctx, events, client.InNamespace(namespace)); err != nil {
		return err
	}

	if len(events.Items) > 1 {
		sort.Sort(eventByFirstTimestamp(events.Items))
	}
	for _, event := range events.Items {
		if ApplyFilters(event, filters...) {
			d.Logf("At %v - event for %s: %v %v: %s", event.FirstTimestamp, event.InvolvedObject.Name, event.Source, event.Reason, event.Message)
		}
	}
	d.Logln()
	return nil
}

// Logf logs the messages and formats the message with the given args.
func (d *KubernetesDumper) Logf(msg string, a ...interface{}) {
	d.log.Log(fmt.Sprintf(msg, a...))
}

// Logln prints an empty line
func (d *KubernetesDumper) Logln() {
	d.log.Log("")
}

// EventFilterFunc is a function to filter events
type EventFilterFunc func(event corev1.Event) bool

// ApplyFilters checks if one of the EventFilters filters the current event
func ApplyFilters(event corev1.Event, filters ...EventFilterFunc) bool {
	for _, filter := range filters {
		if !filter(event) {
			return false
		}
	}
	return true
}

// eventByFirstTimestamp sorts a slice of events by first timestamp, using their involvedObject's name as a tie breaker.
type eventByFirstTimestamp []corev1.Event

func (o eventByFirstTimestamp) Len() int      { return len(o) }
func (o eventByFirstTimestamp) Swap(i, j int) { o[i], o[j] = o[j], o[i] }

func (o eventByFirstTimestamp) Less(i, j int) bool {
	if o[i].FirstTimestamp.Equal(&o[j].FirstTimestamp) {
		return o[i].InvolvedObject.Name < o[j].InvolvedObject.Name
	}
	return o[i].FirstTimestamp.Before(&o[j].FirstTimestamp)
}
