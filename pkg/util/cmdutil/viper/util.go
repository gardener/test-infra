// SPDX-FileCopyrightText: SAP SE or an SAP affiliate company and Gardener contributors
//
// SPDX-License-Identifier: Apache-2.0

package viper

import (
	"fmt"

	"github.com/spf13/cobra"
	"github.com/spf13/pflag"
)

func ListAllCommandsFlags(root *cobra.Command) []*pflag.FlagSet {
	fsList := []*pflag.FlagSet{root.Flags()}
	for _, cmd := range root.Commands() {
		fsList = append(fsList, ListAllCommandsFlags(cmd)...)
	}
	return fsList
}

// PrefixFlagSetConfigs adds a prefix to all flags of the flagset.
// config of "key" would then be "prefix.key"
func PrefixFlagSetConfigs(fs *pflag.FlagSet, prefix string) {
	fs.VisitAll(func(f *pflag.Flag) {
		key := f.Name
		if keyAnnotation, ok := f.Annotations[KeyAnnotation]; ok && len(keyAnnotation) != 0 {
			key = keyAnnotation[1]
		}
		AddCustomConfigForFlag(f, fmt.Sprintf("%s.%s", prefix, key))
	})
}

// PrefixConfigs adds a prefix to all flags with the given names.
// config of "val" would then be "prefix.val"
func PrefixConfigs(fs *pflag.FlagSet, prefix string, names ...string) {
	for _, name := range names {
		if f := fs.Lookup(name); f != nil {
			key := f.Name
			if keyAnnotation, ok := f.Annotations[KeyAnnotation]; ok && len(keyAnnotation) != 0 {
				key = keyAnnotation[1]
			}
			AddCustomConfigForFlag(f, fmt.Sprintf("%s.%s", prefix, key))
		}
	}
}
