package result

import (
	"context"
	"fmt"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/test-infra/pkg/testmachinery"
	"github.com/gardener/test-infra/pkg/util"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"

	corev1 "k8s.io/api/core/v1"
)

func writeBulks(path string, bufs [][]byte) error {
	// check if directory exists and create of not
	if _, err := os.Stat(path); os.IsNotExist(err) {
		err := os.MkdirAll(path, os.ModePerm)
		if err != nil {
			return err
		}
	}
	for _, buf := range bufs {
		err := writeToFile(filepath.Join(path, fmt.Sprintf("res-%s", util.RandomString(5))), buf)
		if err != nil {
			return err
		}
	}
	return nil
}

func writeToFile(filePath string, data []byte) error {
	log.Debugf("write to file %s", filePath)
	err := ioutil.WriteFile(filePath, data, 0644)
	if err != nil {
		return fmt.Errorf("cannot write to '%s': %s", filePath, err.Error())
	}

	return nil
}

func getOSConfig(tmClient kubernetes.Interface, namespace, minioEndpoint string, ssl bool) (*testmachinery.ObjectStoreConfig, error) {
	ctx := context.Background()
	defer ctx.Done()

	minioConfig := &corev1.ConfigMap{}
	err := tmClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: "tm-config"}, minioConfig)
	if err != nil {
		return nil, fmt.Errorf("cannot get ConfigMap 'tm-config': %s", err.Error())
	}
	minioSecrets := &corev1.Secret{}
	err = tmClient.Client().Get(ctx, client.ObjectKey{Namespace: namespace, Name: minioConfig.Data["objectstore.secretName"]}, minioSecrets)
	if err != nil {
		return nil, fmt.Errorf("cannot get Secret '%s': %s", minioConfig.Data["objectstore.secretName"], err.Error())
	}

	return &testmachinery.ObjectStoreConfig{
		Endpoint:   minioEndpoint,
		SSL:        ssl,
		AccessKey:  string(minioSecrets.Data["accessKey"]),
		SecretKey:  string(minioSecrets.Data["secretKey"]),
		BucketName: minioConfig.Data["objectstore.bucketName"],
	}, nil
}
