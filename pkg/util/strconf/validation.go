package strconf

import "fmt"

// Validate validates a testrun config element.
func Validate(identifier string, source *ConfigSource) error {
	if source.ConfigMapKeyRef == nil && source.SecretKeyRef == nil {
		return fmt.Errorf("%s.(configMapKeyRef or secretMapKeyRef): Required configMapKeyRef or secretMapKeyRef: Either a configmap ref or a secretmap ref have to be defined", identifier)
	}
	if source.ConfigMapKeyRef != nil {
		if source.ConfigMapKeyRef.Key == "" {
			return fmt.Errorf("%s.configMapKeyRef.key: Required value", identifier)
		}
		if source.ConfigMapKeyRef.Name == "" {
			return fmt.Errorf("%s.configMapKeyRef.name: Required value", identifier)
		}
	}
	if source.SecretKeyRef != nil {
		if source.SecretKeyRef.Key == "" {
			return fmt.Errorf("%s.secretKeyRef.key: Required value", identifier)
		}
		if source.SecretKeyRef.Name == "" {
			return fmt.Errorf("%s.secretKeyRef.name: Required value", identifier)
		}
	}
	return nil
}
