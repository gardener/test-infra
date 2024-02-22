// SPDX-FileCopyrightText: 2024 SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package testmachinery

type TestLabel string

const (
	// TestLabelShoot are tests that are meant to test a shoot
	TestLabelShoot TestLabel = "shoot"

	// TestLabelGardener are tests that are meant to test gardener and do not rely on a shoot.
	TestLabelGardener TestLabel = "gardener"

	// TestLabelDefault are tests that are graduated to GA but are not a release blocker
	TestLabelDefault TestLabel = "default"

	// TestLabelRelease are tests that are graduated GA and release blocker
	TestLabelRelease TestLabel = "release"

	// TestLabelBeta are tests that are in beta which means that they are currently unstable
	// but will be promoted to GA someday
	TestLabelBeta TestLabel = "beta"
)
