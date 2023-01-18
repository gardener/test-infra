package resources

import (
	"context"
	"encoding/json"

	"sigs.k8s.io/controller-runtime/pkg/client"

	admissionv1 "k8s.io/api/admission/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"sigs.k8s.io/controller-runtime/pkg/webhook/admission"

	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

func GetCreateAdmissionRequest(tr *v1beta1.Testrun) (*admission.Request, error) {

	trRaw, err := json.Marshal(tr)
	if err != nil {
		return nil, err
	}
	return &admission.Request{
		AdmissionRequest: admissionv1.AdmissionRequest{
			Operation: admissionv1.Create,
			Object: runtime.RawExtension{
				Raw: trRaw,
			},
		},
	}, nil
}

type MockReader struct {
	Error error
}

func (m MockReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return m.Error
}

func (m MockReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return m.Error
}
