package perf_disk_attach_test

import (
	"bytes"
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"github.com/gardener/gardener/pkg/utils/kubernetes/health"
	"github.com/gardener/test-infra/pkg/util"
	"io/ioutil"
	"k8s.io/apimachinery/pkg/util/wait"
	"os"
	"path/filepath"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sync"
	"text/template"
	"time"

	gardenv1beta1 "github.com/gardener/gardener/pkg/apis/garden/v1beta1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/pkg/logger"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
	"github.com/sirupsen/logrus"

	. "github.com/gardener/gardener/test/integration/framework"
	. "github.com/gardener/gardener/test/integration/shoots"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
)

var (
	InitializationTimeout = 1800 * time.Second
	DiskAttachTestTimeout = 1800 * time.Second

	StatefulSetNum = 10

	kubeconfig     = flag.String("kubeconfig", "", "the path to the kubeconfig  of the garden cluster that will be used for integration tests")
	shootName      = flag.String("shootName", "", "the name of the shoot we want to test")
	shootNamespace = flag.String("shootNamespace", "", "the namespace name that the shoot resides in")
	outputDirPath  = flag.String("output-dir-path", "", "Path to the directory where the results should be written to.")
)

func validateFlags() {
	if !StringSet(*kubeconfig) {
		Fail("you need to specify the correct path for the kubeconfig")
	}

	if !FileExists(*kubeconfig) {
		Fail("kubeconfig path does not exist")
	}
}

var _ = Describe("Shoot vm disk attach testing", func() {
	var (
		shootGardenerTest   *ShootGardenerTest
		shootTestOperations *GardenerTestOperation
		shootAppTestLogger  *logrus.Logger
		//cloudProvider       gardenv1beta1.CloudProvider

		resourcesDir = filepath.Join("..", "..", "resources")
		templateDir  = filepath.Join(resourcesDir, "templates")

		statefulsetTplName = "disk-attach-statefulset.yaml.tpl"
		statefulsetTpl     *template.Template

		namespace = "default"
	)

	CBeforeSuite(func(ctx context.Context) {
		validateFlags()
		shootAppTestLogger = logger.AddWriter(logger.NewLogger("debug"), GinkgoWriter)

		var err error
		shootGardenerTest, err = NewShootGardenerTest(*kubeconfig, nil, shootAppTestLogger)
		Expect(err).NotTo(HaveOccurred())

		shoot := &gardenv1beta1.Shoot{ObjectMeta: metav1.ObjectMeta{Namespace: *shootNamespace, Name: *shootName}}
		shootTestOperations, err = NewGardenTestOperation(ctx, shootGardenerTest.GardenClient, shootAppTestLogger, shoot)
		Expect(err).NotTo(HaveOccurred())

		//cloudProvider, err = shootTestOperations.GetCloudProvider()
		//Expect(err).NotTo(HaveOccurred())

		statefulsetTpl = template.Must(template.ParseFiles(filepath.Join(templateDir, statefulsetTplName)))
	}, InitializationTimeout)

	CAfterSuite(func(ctx context.Context) {
		By("Deleting statefulsets")
		for i := 0; i < StatefulSetNum; i++ {
			name := Name(i)

			By(fmt.Sprintf("Delete statefulset %s", name))
			sts := &appsv1.StatefulSet{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}
			err := shootTestOperations.ShootClient.Client().Delete(ctx, sts)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Delete svc %s", name))
			svc := &corev1.Service{
				ObjectMeta: metav1.ObjectMeta{
					Name:      name,
					Namespace: namespace,
				},
			}
			err = shootTestOperations.ShootClient.Client().Delete(ctx, svc)
			Expect(err).NotTo(HaveOccurred())

			By(fmt.Sprintf("Delete pvc %s", name))
			pvc := &corev1.PersistentVolumeClaim{
				ObjectMeta: metav1.ObjectMeta{
					Name:      fmt.Sprintf("www-%s-0", name),
					Namespace: namespace,
				},
			}
			err = shootTestOperations.ShootClient.Client().Delete(ctx, pvc)
			Expect(err).NotTo(HaveOccurred())
		}

	}, InitializationTimeout)

	CIt("should deploy multiple statefulsets and evict pods to another node", func(ctx context.Context) {
		ctx = context.WithValue(ctx, "name", "vm attach test")

		machineList, err := shootTestOperations.SeedClient.Machine().MachineV1alpha1().Machines(shootTestOperations.ShootSeedNamespace()).List(metav1.ListOptions{})
		Expect(err).ToNot(HaveOccurred())
		Expect(len(machineList.Items)).To(Equal(1))

		shootAppTestLogger.Debugf("Found machine %s", machineList.Items[0].Name)

		err = runParallelNTimes(StatefulSetNum, func(i int) {
			tplParams := struct {
				Name       string
				Namespace  string
				PVCStorage string
			}{
				Name(i),
				namespace,
				"1Gi",
			}

			By(fmt.Sprintf("Deploy statefulset %s", tplParams.Name))
			var writer bytes.Buffer
			err = statefulsetTpl.Execute(&writer, tplParams)
			Expect(err).NotTo(HaveOccurred())

			manifestReader := kubernetes.NewManifestReader(writer.Bytes())
			err = shootTestOperations.ShootClient.Applier().ApplyManifest(ctx, manifestReader, kubernetes.DefaultApplierOptions)
			Expect(err).NotTo(HaveOccurred())

			err = shootTestOperations.WaitUntilStatefulSetIsRunning(ctx, tplParams.Name, tplParams.Namespace, shootTestOperations.ShootClient)
			Expect(err).NotTo(HaveOccurred())
		})
		Expect(err).NotTo(HaveOccurred())

		// trigger eviction of pods by deleting the machine
		err = shootTestOperations.SeedClient.Machine().MachineV1alpha1().Machines(shootTestOperations.ShootSeedNamespace()).Delete(machineList.Items[0].Name, &metav1.DeleteOptions{})
		Expect(err).ToNot(HaveOccurred())

		results := make([]*Result, 0)
		mutex := &sync.Mutex{}

		err = runParallelNTimes(StatefulSetNum, func(i int) {
			err := WaitUntilStatefulSetIsUnhealthy(ctx, shootTestOperations, Name(i), namespace, shootTestOperations.ShootClient)
			Expect(err).NotTo(HaveOccurred())
			startTime := time.Now()

			err = shootTestOperations.WaitUntilStatefulSetIsRunning(ctx, Name(i), namespace, shootTestOperations.ShootClient)
			Expect(err).NotTo(HaveOccurred())

			completionTime := time.Now().Sub(startTime)
			mutex.Lock()
			results = append(results, &Result{Name: Name(i), CompletionTime: completionTime.Nanoseconds(), duration: completionTime})
			mutex.Unlock()
			shootTestOperations.Logger.Infof("Total time to drain pods for sts %s: %s", Name(i), completionTime.String())
		})
		Expect(err).ToNot(HaveOccurred())

		// Print the results:
		for _, result := range results {
			shootTestOperations.Logger.Infof("%s took %s", result.Name, result.duration.String())
			err := writeTestResults(result, *outputDirPath)
			Expect(err).NotTo(HaveOccurred())
		}

	}, DiskAttachTestTimeout)
})

func runParallelNTimes(n int, f func(i int)) error {
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			f(i)
		}(i)
	}
	wg.Wait()
	return nil
}

func writeTestResults(r *Result, path string) error {
	if path == "" {
		return nil
	}
	err := os.MkdirAll(path, 0777)
	if err != nil {
		return err
	}

	dat, err := json.Marshal(r)
	if err != nil {
		return err
	}
	err = ioutil.WriteFile(filepath.Join(path, util.RandomString(6)), dat, 0777)
	if err != nil {
		return err
	}

	return nil
}

func setNodesToOne(ctx context.Context, operation *GardenerTestOperation) error {
	if operation.Shoot.Spec.Cloud.Azure.Workers[0].AutoScalerMax != 1 {
		operation.Shoot.Spec.Cloud.Azure.Workers[0].AutoScalerMin = 1
		operation.Shoot.Spec.Cloud.Azure.Workers[0].AutoScalerMax = 1
	}

	return operation.ShootClient.Client().Update(ctx, operation.Shoot)
}

// WaitUntilStatefulSetIsUnhealthy waits until the stateful set with <statefulSetName> is not running
func WaitUntilStatefulSetIsUnhealthy(ctx context.Context, operation *GardenerTestOperation, statefulSetName, statefulSetNamespace string, c kubernetes.Interface) error {
	return wait.PollImmediateUntil(5*time.Second, func() (bool, error) {
		statefulSet := &appsv1.StatefulSet{}
		if err := c.Client().Get(ctx, client.ObjectKey{Namespace: statefulSetNamespace, Name: statefulSetName}, statefulSet); err != nil {
			return false, err
		}

		if err := health.CheckStatefulSet(statefulSet); err == nil {
			operation.Logger.Infof("Waiting for %s to be unhealthy!!", statefulSetName)
			return false, nil
		}
		operation.Logger.Infof("%s is now unhealthy!!", statefulSetName)
		return true, nil

	}, ctx.Done())
}
