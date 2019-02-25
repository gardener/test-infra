package main

import (
	"os"

	"github.com/joho/godotenv"

	log "github.com/sirupsen/logrus"
)

func init() {
	err := godotenv.Load()
	if err == nil {
		log.Info(".env file loaded")
	} else {
		log.Warnf("Error loading .env file: %s", err.Error())
	}

	formatter := &log.TextFormatter{
		FullTimestamp: true,
	}
	log.SetFormatter(formatter)
	log.SetOutput(os.Stderr)

	if os.Getenv("LOG_LEVEL") == "debug" {
		log.SetLevel(log.DebugLevel)
		log.Warn("Set debug log level")
	}

	// Set commandline flags

	// configuration flags
	rootCmd.Flags().StringVar(&tmKubeconfigPath, "tm-kubeconfig-path", "", "Path to the testmachinery cluster kubeconfig")
	rootCmd.MarkFlagRequired("tm-kubeconfig-path")
	rootCmd.MarkFlagFilename("tm-kubeconfig-path")
	rootCmd.Flags().StringVar(&testrunChartPath, "testruns-chart-path", "", "Path to the testruns chart.")
	rootCmd.MarkFlagRequired("testruns-chart-path")
	rootCmd.MarkFlagFilename("testruns-chart-path")
	rootCmd.Flags().StringVar(&testrunNamePrefix, "testrun-prefix", "default-", "Testrun name prefix which is used to generate a unique testrun name.")
	rootCmd.MarkFlagRequired("testrun-prefix")

	rootCmd.Flags().Int64Var(&timeout, "timeout", -1, "timout of the testrunner to wait for the complete testrun to finish.")
	rootCmd.Flags().StringVar(&outputFilePath, "output-file-path", "./testout", "The filepath where the summary should be written to.")
	rootCmd.Flags().StringVar(&elasticSearchConfigName, "es-config-name", "sap_internal", "The elasticsearch secret-server config name.")
	rootCmd.Flags().StringVar(&s3Endpoint, "s3-endpoint", os.Getenv("S3_ENDPOINT"), "S3 endpoint of the testmachinery cluster.")
	// TODO: refactor: apply code style: concourse-on-error-dir
	rootCmd.Flags().StringVar(&concourseOnErrorDir, "concourseOnErrorDir", os.Getenv("ON_ERROR_DIR"), "On error dir which is used by Concourse.")

	// parameter flags
	rootCmd.Flags().StringVar(&gardenKubeconfigPath, "gardener-kubeconfig-path", "", "Path to the gardener kubeconfig.")
	rootCmd.MarkFlagRequired("gardener-kubeconfig-path")
	rootCmd.MarkFlagFilename("gardener-kubeconfig-path")
	rootCmd.Flags().StringVar(&projectName, "project-name", "", "Gardener project name of the shoot")
	rootCmd.MarkFlagRequired("gardener-kubeconfig-path")
	rootCmd.Flags().StringVar(&shootName, "shoot-name", "", "Shoot name which is used to run tests.")
	rootCmd.MarkFlagRequired("gardener-kubeconfig-path")
	rootCmd.Flags().StringVar(&cloudprovider, "cloudprovider", "", "Cloudprovider where the shoot is created.")
	rootCmd.MarkFlagRequired("gardener-kubeconfig-path")
	rootCmd.Flags().StringVar(&cloudprofile, "cloudprofile", "", "Cloudprofile of shoot.")
	rootCmd.MarkFlagRequired("gardener-kubeconfig-path")
	rootCmd.Flags().StringVar(&secretBinding, "secret-binding", "", "SecretBinding that should be used to create the shoot.")
	rootCmd.MarkFlagRequired("gardener-kubeconfig-path")
	rootCmd.Flags().StringVar(&region, "region", "", "Region where the shoot is created.")
	rootCmd.MarkFlagRequired("gardener-kubeconfig-path")
	rootCmd.Flags().StringVar(&zone, "zone", "", "Zone of the shoot worker nodes. Not required for azure shoots.")
	rootCmd.MarkFlagRequired("gardener-kubeconfig-path")

	rootCmd.Flags().StringVar(&k8sVersion, "k8s-version", "", "Kubernetes version of the shoot.")
	rootCmd.Flags().StringVar(&machineType, "machinetype", "", "Machinetype of the shoot's worker nodes.")
	rootCmd.Flags().StringVar(&autoscalerMin, "autoscaler-min", "", "Min number of worker nodes.")
	rootCmd.Flags().StringVar(&autoscalerMax, "autoscaler-max", "", "Max number of worker nodes.")
	rootCmd.Flags().StringVar(&componenetDescriptorPath, "component-descriptor-path", "", "Path to the component descriptor (BOM) of the current landscape.")
	rootCmd.Flags().StringVar(&landscape, "landscape", "", "Current gardener landscape.")
}
