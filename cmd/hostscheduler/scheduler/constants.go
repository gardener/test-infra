package scheduler

import (
	"fmt"
	"os"
	"path/filepath"
)

const (
	ShootLabel = "testmachinery.sapcloud.io/host"
)

func ShootKubeconfigSecretName(shootName string) string {
	return fmt.Sprintf("%s.kubeconfig", shootName)
}

func HostKubeconfigPath() string {
	return filepath.Join(os.Getenv("TM_KUBECONFIG_PATH"), "host.config")
}

func HostConfigFilePath() string {
	return filepath.Join(os.Getenv("TM_SHARED_PATH"), "host", "config.json")
}
