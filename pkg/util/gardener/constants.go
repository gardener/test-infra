// SPDX-FileCopyrightText: 2021 SAP SE or an SAP affiliate company and Gardener contributors.
//
// SPDX-License-Identifier: Apache-2.0

package gardener

const (
	// ConfirmationDeletion is an annotation on a Shoot and Project resources whose value must be set to "true" in order to
	// allow deleting the resource (if the annotation is not set any DELETE request will be denied).
	ConfirmationDeletion = "confirmation.gardener.cloud/deletion"
	// ShootIgnore is a constant for an annotation on a Shoot which may be used to tell the Gardener that the Shoot with this name should be
	// ignored completely. That means that the Shoot will never reach the reconciliation flow (independent of the operation (create/update/
	// delete)).
	ShootIgnore = "shoot.gardener.cloud/ignore"
)
