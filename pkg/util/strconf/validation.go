package strconf

import (
	"k8s.io/apimachinery/pkg/util/validation/field"
)

// Validate validates a testrun config element.
func Validate(fldPath *field.Path, source *ConfigSource) field.ErrorList {
	var allErrs field.ErrorList
	if source.ConfigMapKeyRef == nil && source.SecretKeyRef == nil {
		allErrs = append(allErrs, field.Required(fldPath.Child("configMapKeyRef/secretMapKeyRef"), "Either a configmap ref or a secretmap ref have to be defined"))
		return allErrs
	}
	if source.ConfigMapKeyRef != nil {
		cmPath := fldPath.Child("configMapKeyRef")
		if source.ConfigMapKeyRef.Key == "" {
			allErrs = append(allErrs, field.Required(cmPath.Child("key"), "has to be defined"))
		}
		if source.ConfigMapKeyRef.Name == "" {
			allErrs = append(allErrs, field.Required(cmPath.Child("name"), "has to be defined"))
		}
	}
	if source.SecretKeyRef != nil {
		skPath := fldPath.Child("secretKeyRef")
		if source.SecretKeyRef.Key == "" {
			allErrs = append(allErrs, field.Required(skPath.Child("key"), "has to be defined"))
		}
		if source.SecretKeyRef.Name == "" {
			allErrs = append(allErrs, field.Required(skPath.Child("name"), "has to be defined"))
		}
	}
	return allErrs
}
