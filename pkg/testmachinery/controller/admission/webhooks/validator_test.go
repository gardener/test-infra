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
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/testmachinery"
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

	var testRunValidator admission.Handler

	BeforeEach(func() {
		testRunValidator = webhooks.NewValidator(logr.Discard())

		decoder, err := admission.NewDecoder(testmachinery.TestMachineryScheme)
		Expect(err).NotTo(HaveOccurred())

		_, err = admission.InjectDecoderInto(decoder, testRunValidator)
		Expect(err).NotTo(HaveOccurred())
	})

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

			req, err := resources.GetCreateAdmissionRequest(tr)
			Expect(err).NotTo(HaveOccurred())

			resp := testRunValidator.Handle(ctx, *req)

			Expect(resp.Allowed).To(BeFalse())
			Expect(string(resp.Result.Reason)).To(ContainSubstring("name must not contain '.'"))

		})
	})

	Context("TestLocations", func() {
		It("should reject when no locations are defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.TestLocations = []tmv1beta1.TestLocation{}
			tr.Spec.LocationSets = nil

			req, err := resources.GetCreateAdmissionRequest(tr)
			Expect(err).ToNot(HaveOccurred())

			resp := testRunValidator.Handle(ctx, *req)

			Expect(resp.Allowed).To(BeFalse())
			Expect(string(resp.Result.Reason)).To(ContainSubstring("no location for TestDefinitions defined"))
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

			req, err := resources.GetCreateAdmissionRequest(tr)
			Expect(err).ToNot(HaveOccurred())

			resp := testRunValidator.Handle(ctx, *req)
			Expect(resp.Allowed).To(BeFalse())
			Expect(string(resp.Result.Reason)).To(ContainSubstring("Local testDefinition locations are only available in insecure mode"))
		})
	})

	Context("Kubeconfigs", func() {
		It("should reject when a invalid kubeconfig is provided", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)
			tr.Spec.Kubeconfigs.Gardener = strconf.FromString("dGVzdGluZwo=")

			req, err := resources.GetCreateAdmissionRequest(tr)
			Expect(err).ToNot(HaveOccurred())

			resp := testRunValidator.Handle(ctx, *req)
			Expect(resp.Allowed).To(BeFalse())
			Expect(string(resp.Result.Reason)).To(ContainSubstring("Cannot build config"))
		})
	})

	Context("OnExit", func() {
		It("should accept when no steps are defined", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)

			req, err := resources.GetCreateAdmissionRequest(tr)
			Expect(err).ToNot(HaveOccurred())

			resp := testRunValidator.Handle(ctx, *req)
			Expect(resp.Allowed).To(BeTrue())

		})
	})

	Context("ArgoAvailability", func() {
		It("should reject a Testrun when argo is not available", func() {
			ctx := context.Background()
			defer ctx.Done()
			tr := resources.GetBasicTestrun(namespace, commitSha)

			req, err := resources.GetCreateAdmissionRequest(tr)
			Expect(err).ToNot(HaveOccurred())

			mockReader := resources.MockReader{
				Error: errors.New("Argo deployment not found"),
			}
			duration := metav1.Duration{Duration: time.Second}
			webhooks.StartHealthCheck(ctx, mockReader, namespace, "argo", duration)
			time.Sleep(2 * time.Second)

			resp := testRunValidator.Handle(ctx, *req)
			Expect(resp.Allowed).To(BeFalse())
			Expect(resp.Result.Code).To(Equal(int32(424)))
			Expect(string(resp.Result.Message)).To(ContainSubstring("Argo deployment not found"))
		})
	})
})
