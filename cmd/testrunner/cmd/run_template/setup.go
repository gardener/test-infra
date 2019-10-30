package run_template

import (
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/common"
	"github.com/gardener/test-infra/pkg/shootflavors"
	"github.com/pkg/errors"
	"io/ioutil"
	"sigs.k8s.io/yaml"
)

func GetShootFlavors(cfgPath string, k8sClient kubernetes.Interface) (*shootflavors.ExtendedFlavors, error) {
	// read and parse test shoot configuration
	dat, err := ioutil.ReadFile(cfgPath)
	if err != nil {
		return nil, errors.Wrapf(err, "unable to read test shoot configuration file from %s", testShootConfigPath)
	}

	flavors := common.ExtendedShootFlavors{}
	if err := yaml.Unmarshal(dat, &flavors); err != nil {
		return nil, err
	}

	return shootflavors.NewExtended(k8sClient, flavors.Flavors)
}
