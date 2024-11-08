package webhooks_test

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/go-logr/logr"
	. "github.com/onsi/ginkgo/v2"
	. "github.com/onsi/gomega"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery/controller/admission/webhooks"
	"github.com/gardener/test-infra/pkg/util/strconf"
	"github.com/gardener/test-infra/test/resources"
)

func TestValidationWebhook(t *testing.T) {
	RegisterFailHandler(Fail)
	RunSpecs(t, "Validation webhook Integration Test Suite")
}

const (
	namespace = "default"
	commitSha = "10e4414c42d0761765ab47034a870a308c0ae91e"
)

var _ = Describe("Testrun validation tests", func() {

	var testRunValidator = webhooks.TestRunCustomValidator{Log: logr.Discard()}

	Context("Metadata", func() {
		It("should reject when name contains '.'", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestFlow = tmv1beta1.TestFlow{
				&tmv1beta1.DAGStep{
					Name: "int-test",
					Definition: tmv1beta1.StepDefinition{
						Name: "unit.testdef",
					},
				},
			}

			warnings, err := testRunValidator.ValidateCreate(context.TODO(), runtime.Object(tr))
			Expect(warnings).To(BeNil())
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("name must not contain '.'"))

		})
	})

	Context("TestLocations", func() {
		It("should reject when no locations are defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestLocations = []tmv1beta1.TestLocation{}
			tr.Spec.LocationSets = nil

			warnings, err := testRunValidator.ValidateCreate(context.TODO(), runtime.Object(tr))
			Expect(warnings).To(BeNil())
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("no location for TestDefinitions defined"))
		})

		It("should reject when a local location is defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.LocationSets = nil
			tr.Spec.TestLocations = []tmv1beta1.TestLocation{}
			tr.Spec.TestLocations = append(tr.Spec.TestLocations, tmv1beta1.TestLocation{
				Type: tmv1beta1.LocationTypeLocal,
			})

			warnings, err := testRunValidator.ValidateCreate(context.TODO(), runtime.Object(tr))
			Expect(warnings).To(BeNil())
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("Local testDefinition locations are only available in insecure mode"))
		})
	})

	Context("Kubeconfigs", func() {
		It("should reject when a invalid kubeconfig is provided", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.Kubeconfigs.Gardener = strconf.FromString("dGVzdGluZwo=")

			warnings, err := testRunValidator.ValidateCreate(context.TODO(), runtime.Object(tr))
			Expect(warnings).To(BeNil())
			Expect(err).To(HaveOccurred())

			Expect(err.Error()).To(ContainSubstring("Cannot build config"))
		})
	})

	Context("OnExit", func() {
		It("should accept when no steps are defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)

			warnings, err := testRunValidator.ValidateCreate(context.TODO(), runtime.Object(tr))
			Expect(warnings).To(BeNil())
			Expect(err).ToNot(HaveOccurred())

		})
	})

	Context("ArgoAvailability", func() {
		It("should reject a Testrun when argo is not available", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)

			mockReader := resources.MockReader{
				Error: errors.New("Argo deployment not found"),
			}
			duration := metav1.Duration{Duration: time.Second}
			webhooks.StartHealthCheck(ctx, mockReader, namespace, "argo", duration)
			time.Sleep(2 * time.Second)

			warnings, err := testRunValidator.ValidateCreate(context.TODO(), runtime.Object(tr))
			Expect(warnings).To(BeNil())
			Expect(err).To(HaveOccurred())
			Expect(err.Error()).To(ContainSubstring("Argo deployment not found"))
		})
	})
})
