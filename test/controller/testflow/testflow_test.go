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

package testflow_test

import (
	"context"
	"fmt"
	"strings"
	"time"

	argov1 "github.com/argoproj/argo-workflows/v3/pkg/apis/workflow/v1alpha1"
	"github.com/gardener/gardener/pkg/utils/retry"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/strconf"
	"github.com/gardener/test-infra/test/resources"
	"github.com/gardener/test-infra/test/utils"
)

var _ = Describe("Testflow execution tests", func() {

	Context("testflow", func() {
		It("should run a test with TestDefs defined by name", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())

			tr, wf, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") && node.Phase == argov1.NodeSucceeded {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(1), "Testrun: %s", tr.Name)
		})

		It("should run a test with TestDefs defined by label and name", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = append(tr.Spec.TestFlow, &tmv1beta1.DAGStep{
				Name: "integration-testdef",
				Definition: tmv1beta1.StepDefinition{
					Label: "tm-integration",
				},
			})

			tr, wf, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") && node.Phase == argov1.NodeSucceeded {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(2), "Testrun: %s", tr.Name)
		})

		It("should execute all tests in right order when no testdefs for a label can be found", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = append(tr.Spec.TestFlow,
				&tmv1beta1.DAGStep{
					Name:      "B",
					DependsOn: []string{"A"},
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
				},
				&tmv1beta1.DAGStep{
					Name:      "D",
					DependsOn: []string{"A"},
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
				})

			tr, wf, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") && node.Phase == argov1.NodeSucceeded {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(3), "Testrun: %s", tr.Name)
		})

		It("should execute serial steps after parallel steps", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Label: "tm-integration",
					},
				},
			}

			tr, wf, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") {
					Expect(node.Phase).To(Equal(argov1.NodeSucceeded))

					Expect(len(node.Children)).To(Equal(1))

					nextNode := wf.Status.Nodes[node.Children[0]]
					Expect(strings.HasPrefix(nextNode.TemplateName, "serial-testdef"))
				}
			}

		})

		It("should execute the testflow with a step that has outputs and is serial", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "A",
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
				},
				&tmv1beta1.DAGStep{
					Name:      "B",
					DependsOn: []string{"A"},
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
				},
				&tmv1beta1.DAGStep{
					Name:      "C",
					DependsOn: []string{"B"},
					Definition: tmv1beta1.StepDefinition{
						Name: "integration-testdef",
					},
				},
			}

			tr, wf, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") && node.Phase == argov1.NodeSucceeded {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(3), "Testrun: %s", tr.Name)
		})

		It("should execute one serial step successfully", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "serial-testdef",
					},
				},
			}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})

		It("should run a test with a paused step that finishes after it is resumed", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = append(tr.Spec.TestFlow, &tmv1beta1.DAGStep{
				Name: "pause-testdef",
				Definition: tmv1beta1.StepDefinition{
					Name: "integration-testdef",
				},
				Pause: &tmv1beta1.Pause{Enabled: true},
			})

			err := operation.Client().Create(ctx, tr)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			err = utils.WatchTestrun(ctx, operation.Client(), tr, 2*time.Minute, func(tr *tmv1beta1.Testrun) (bool, error) {
				testrun := &tmv1beta1.Testrun{}
				if err := operation.Client().Get(ctx, client.ObjectKey{Name: tr.GetName(), Namespace: tr.GetNamespace()}, testrun); err != nil {
					operation.Log().Error(err, "unable to get testrun")
					return false, nil
				}
				tr = testrun
				wf := &argov1.Workflow{}
				if err := operation.Client().Get(ctx, client.ObjectKey{Name: tr.Status.Workflow, Namespace: tr.GetNamespace()}, wf); err != nil {
					operation.Log().Error(err, "unable to get workflow")
					return false, nil
				}

				for _, node := range wf.Status.Nodes {
					if node.TemplateName == testmachinery.PauseTemplateName && node.Phase == tmv1beta1.StepPhaseRunning {
						return retry.Ok()
					}
				}

				return retry.NotOk()
			})
			Expect(err).ToNot(HaveOccurred())

			err = util.ResumeTestrun(ctx, operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			tr, err = utils.WatchTestrunUntilCompleted(ctx, operation.Client(), tr, 2*time.Minute)
			Expect(err).ToNot(HaveOccurred())

			wf := &argov1.Workflow{}
			err = operation.Client().Get(ctx, client.ObjectKey{Name: tr.Status.Workflow, Namespace: tr.GetNamespace()}, wf)
			Expect(err).ToNot(HaveOccurred())

			numExecutedTestDefs := 0
			for _, node := range wf.Status.Nodes {
				if strings.HasPrefix(node.TemplateName, "integration-testdef") && node.Phase == argov1.NodeSucceeded {
					numExecutedTestDefs++
				}
			}

			Expect(numExecutedTestDefs).To(Equal(2), "Testrun: %s", tr.Name)
		})
	})

	Context("File created in shared folder", func() {
		sharedFilePath := fmt.Sprintf("%s/%s", testmachinery.TM_SHARED_PATH, "test")
		It("should be visible from withing another step", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "create-artifact",
					Definition: tmv1beta1.StepDefinition{
						Name: "check-file-content-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: sharedFilePath,
							},
							{
								Type:  tmv1beta1.ConfigTypeFile,
								Name:  "TEST_NAME",
								Value: "dGVzdAo=", // base64 encoded 'test' string
								Path:  sharedFilePath,
							},
						},
					},
				},
				&tmv1beta1.DAGStep{
					Name:      "check-artifact",
					DependsOn: []string{"create-artifact"},
					Definition: tmv1beta1.StepDefinition{
						Name: "check-file-content-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: sharedFilePath,
							},
						},
					},
				},
			}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})
		It("should be visible from within another step that uses artifactsFrom feature", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "create-artifact",
					Definition: tmv1beta1.StepDefinition{
						Name: "check-file-content-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: sharedFilePath,
							},
							{
								Type:  tmv1beta1.ConfigTypeFile,
								Name:  "TEST_NAME",
								Value: "dGVzdAo=", // base64 encoded 'test' string
								Path:  sharedFilePath,
							},
						},
					},
				},
				&tmv1beta1.DAGStep{
					Name:      "check-artifact",
					DependsOn: []string{"create-artifact"},
					Definition: tmv1beta1.StepDefinition{
						Name: "check-file-content-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: sharedFilePath,
							},
						},
					},
				},
				&tmv1beta1.DAGStep{
					Name:          "check-artifact2",
					DependsOn:     []string{"check-artifact"},
					ArtifactsFrom: "create-artifact",
					Definition: tmv1beta1.StepDefinition{
						Name: "check-file-content-testdef",
						Config: []tmv1beta1.ConfigElement{
							{
								Type:  tmv1beta1.ConfigTypeEnv,
								Name:  "FILE",
								Value: sharedFilePath,
							},
						},
					},
				},
			}

			tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())
		})
	})

	Context("config", func() {
		Context("type environment variable", func() {
			It("should mount a config as environment variable", func() {
				ctx := context.Background()
				defer ctx.Done()
				tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
				tr.Spec.TestFlow = tmv1beta1.TestFlow{
					&tmv1beta1.DAGStep{
						Name: "int-test",
						Definition: tmv1beta1.StepDefinition{
							Name: "check-envvar-testdef",
							Config: []tmv1beta1.ConfigElement{
								{
									Type:  tmv1beta1.ConfigTypeEnv,
									Name:  "TEST_NAME",
									Value: "test",
								},
							},
						},
					},
				}

				tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
				defer utils.DeleteTestrun(operation.Client(), tr)
				Expect(err).ToNot(HaveOccurred())

			})

			It("should mount a global config as environment variable", func() {
				ctx := context.Background()
				defer ctx.Done()
				tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
				tr.Spec.TestFlow = tmv1beta1.TestFlow{
					&tmv1beta1.DAGStep{
						Name: "int-test",
						Definition: tmv1beta1.StepDefinition{
							Name: "check-envvar-testdef",
						},
					},
				}
				tr.Spec.Config = []tmv1beta1.ConfigElement{
					{
						Type:  tmv1beta1.ConfigTypeEnv,
						Name:  "TEST_NAME",
						Value: "test",
					},
				}

				tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
				defer utils.DeleteTestrun(operation.Client(), tr)
				Expect(err).ToNot(HaveOccurred())
			})
		})

		Context("type file", func() {
			It("should mount a config with value as file to a specific path", func() {
				ctx := context.Background()
				defer ctx.Done()
				tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
				tr.Spec.TestFlow = tmv1beta1.TestFlow{
					&tmv1beta1.DAGStep{
						Name: "int-test",
						Definition: tmv1beta1.StepDefinition{
							Name: "check-file-content-testdef",
							Config: []tmv1beta1.ConfigElement{
								{
									Type:  tmv1beta1.ConfigTypeEnv,
									Name:  "FILE",
									Value: "/tmp/test",
								},
								{
									Type:  tmv1beta1.ConfigTypeFile,
									Name:  "TEST_NAME",
									Value: "dGVzdAo=", // base64 encoded 'test' string
									Path:  "/tmp/test",
								},
							},
						},
					},
				}

				tr, _, err := operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
				defer utils.DeleteTestrun(operation.Client(), tr)
				Expect(err).ToNot(HaveOccurred())
			})

			It("should mount a config from a secret as file to a specific path", func() {
				ctx := context.Background()
				defer ctx.Done()
				secret := &corev1.Secret{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "test-secret-",
						Namespace:    operation.TestNamespace(),
					},
					Type: corev1.SecretTypeOpaque,
					Data: map[string][]byte{
						"test": []byte("test"),
					},
				}
				err := operation.Client().Create(ctx, secret)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err := operation.Client().Delete(ctx, secret)
					Expect(err).ToNot(HaveOccurred(), "Cannot delete secret")
				}()

				tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
				tr.Spec.TestFlow = tmv1beta1.TestFlow{
					&tmv1beta1.DAGStep{
						Name: "int-test",
						Definition: tmv1beta1.StepDefinition{
							Name: "check-file-content-testdef",
							Config: []tmv1beta1.ConfigElement{
								{
									Type:  tmv1beta1.ConfigTypeEnv,
									Name:  "FILE",
									Value: "/tmp/test/test.txt",
								},
								{
									Type: tmv1beta1.ConfigTypeFile,
									Name: "TEST_NAME",
									Path: "/tmp/test/test.txt",
									ValueFrom: &strconf.ConfigSource{
										SecretKeyRef: &corev1.SecretKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: secret.Name,
											},
											Key: "test",
										},
									},
								},
							},
						},
					},
				}

				tr, _, err = operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
				defer utils.DeleteTestrun(operation.Client(), tr)
				Expect(err).ToNot(HaveOccurred())

			})

			It("should mount a config from a configmap as file to a specific path", func() {
				ctx := context.Background()
				defer ctx.Done()
				configmap := &corev1.ConfigMap{
					ObjectMeta: metav1.ObjectMeta{
						GenerateName: "test-configmap-",
						Namespace:    operation.TestNamespace(),
					},
					Data: map[string]string{
						"test": "test",
					},
				}
				err := operation.Client().Create(ctx, configmap)
				Expect(err).ToNot(HaveOccurred())
				defer func() {
					err := operation.Client().Delete(ctx, configmap)
					Expect(err).ToNot(HaveOccurred(), "Cannot delete configmap %s", configmap.Name)
				}()

				tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
				tr.Spec.TestFlow = tmv1beta1.TestFlow{
					&tmv1beta1.DAGStep{
						Name: "int-test",
						Definition: tmv1beta1.StepDefinition{
							Name: "check-file-content-testdef",
							Config: []tmv1beta1.ConfigElement{
								{
									Type:  tmv1beta1.ConfigTypeEnv,
									Name:  "FILE",
									Value: "/tmp/test/test.txt",
								},
								{
									Type: tmv1beta1.ConfigTypeFile,
									Name: "TEST_NAME",
									Path: "/tmp/test/test.txt",
									ValueFrom: &strconf.ConfigSource{
										ConfigMapKeyRef: &corev1.ConfigMapKeySelector{
											LocalObjectReference: corev1.LocalObjectReference{
												Name: configmap.Name,
											},
											Key: "test",
										},
									},
								},
							},
						},
					},
				}

				tr, _, err = operation.RunTestrunUntilCompleted(ctx, tr, argov1.WorkflowSucceeded, TestrunDurationTimeout)
				defer utils.DeleteTestrun(operation.Client(), tr)
				Expect(err).ToNot(HaveOccurred())

			})
		})

	})

	Context("TTL", func() {
		It("should delete the testrun after ttl has finished", func() {
			ctx := context.Background()
			defer ctx.Done()

			var ttl int32 = 90
			var InitializationTimeout int64 = 600
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TTLSecondsAfterFinished = &ttl

			err := operation.Client().Create(ctx, tr)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			startTime := time.Now()
			for !util.MaxTimeExceeded(startTime, InitializationTimeout) {
				err = operation.Client().Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, tr)
				if errors.IsNotFound(err) {
					return
				}
				time.Sleep(5 * time.Second)
			}

			err = operation.Client().Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, tr)
			Expect(errors.IsNotFound(err)).To(BeTrue(), "Testrun %s was not deleted in %d seconds", tr.Name, InitializationTimeout)
		})

		It("should delete the testrun after workflow has finished", func() {
			ctx := context.Background()
			defer ctx.Done()

			var ttl int32 = 1
			var InitializationTimeout int64 = 600
			tr := resources.GetBasicTestrun(operation.TestNamespace(), operation.Commit())
			tr.Spec.TTLSecondsAfterFinished = &ttl

			err := operation.Client().Create(ctx, tr)
			defer utils.DeleteTestrun(operation.Client(), tr)
			Expect(err).ToNot(HaveOccurred())

			startTime := time.Now()
			for !util.MaxTimeExceeded(startTime, InitializationTimeout) {
				err = operation.Client().Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, tr)
				if errors.IsNotFound(err) {
					return
				}
				time.Sleep(5 * time.Second)
			}
			err = operation.Client().Get(ctx, client.ObjectKey{Namespace: tr.Namespace, Name: tr.Name}, tr)
			Expect(errors.IsNotFound(err)).To(BeTrue(), "Testrun %s was not deleted in %d seconds", tr.Name, InitializationTimeout)
		})
	})
})
