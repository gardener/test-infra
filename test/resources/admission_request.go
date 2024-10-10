package resources

import (
	"context"

	"sigs.k8s.io/controller-runtime/pkg/client"
)

type MockReader struct {
	Error error
}

func (m MockReader) Get(ctx context.Context, key client.ObjectKey, obj client.Object, opts ...client.GetOption) error {
	return m.Error
}

func (m MockReader) List(ctx context.Context, list client.ObjectList, opts ...client.ListOption) error {
	return m.Error
}
