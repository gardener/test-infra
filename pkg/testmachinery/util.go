package testmachinery

import (
	"github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/pkg/errors"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
)

var decoder runtime.Decoder

// ParseTestrunFromFile reads a testrun.yaml file from filePath and parses the yaml.
func ParseTestrunFromFile(filePath string) (*v1beta1.Testrun, error) {
	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		return nil, err
	}
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, err
	}

	return ParseTestrun(data)
}

// ParseTestrun parses testrun.
func ParseTestrun(data []byte) (*v1beta1.Testrun, error) {
	if len(data) == 0 {
		return nil, errors.New("empty data")
	}

	tr := &v1beta1.Testrun{}
	_, _, err := decoder.Decode(data, nil, tr)
	if err != nil {
		return nil, err
	}
	return tr, nil
}
