package result

import (
	"context"
	"fmt"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v27/github"
	"github.com/pkg/errors"
	"io/ioutil"
	"net/url"
	"os"
	"strings"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

// uploads status results as asset to the component
func UploadStatusToGithub(run *testrunner.Run, component *componentdescriptor.Component, githubUser, githubPassword string) error {
	tr := run.Testrun
	md := run.Metadata
	tableString := strings.Builder{}
	util.RenderStatusTable(&tableString, tr.Status.Steps)
	statusOutput := fmt.Sprintf("Testrun: %s\n%s\n%s", tr.GenerateName, tableString.String(), util.PrettyPrintStruct(tr.Status))
	filename := fmt.Sprintf("%s-%s-%s.txt", md.Landscape, md.CloudProvider, md.KubernetesVersion)
	if err := ioutil.WriteFile(filename, []byte(statusOutput), 0644); err != nil {
		return err
	}
	file, err := os.Open(filename)
	if err != nil {
		return err
	}

	repoURL, err := url.Parse(fmt.Sprintf("https://%s", component.Name))
	if err != nil {
		return err
	}
	repoOwner, repoName := util.ParseRepoURL(repoURL)
	githubClient, err := getGithubClient(repoURL, githubUser, githubPassword)
	if err != nil {
		return err
	}

	// get github release
	release, response, err := githubClient.Repositories.GetReleaseByTag(context.Background(), repoOwner, repoName, component.Version)
	if err != nil {
		return err
	} else if response.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Github release GET failed with status code %d", response.StatusCode))
	}

	// check if asset exists and delete if so
	releaseAssets, response, err := githubClient.Repositories.ListReleaseAssets(context.Background(), repoOwner, repoName, *release.ID, &github.ListOptions{Page: 1, PerPage: 200})
	if err != nil {
		return err
	} else if response.StatusCode != 200 {
		return errors.New(fmt.Sprintf("Get github release assets failed with status code %d", response.StatusCode))
	}
	for _, releaseAsset := range releaseAssets {
		if *releaseAsset.Name == filename {
			response, err = githubClient.Repositories.DeleteReleaseAsset(context.Background(), repoOwner, repoName, *releaseAsset.ID)
			if err != nil {
				return err
			} else if response.StatusCode != 204 {
				return errors.New(fmt.Sprintf("Delete of github release asset %s failed with status code %d", *releaseAsset.Name, response.StatusCode))
			}
		}
	}

	// upload new asset
	_, response, err = githubClient.Repositories.UploadReleaseAsset(context.Background(), repoOwner, repoName, *release.ID, &github.UploadOptions{Name: filename,}, file)
	if err != nil {
		return err
	} else if response.StatusCode != 201 {
		return errors.New(fmt.Sprintf("Failed to create a github asset with status code %d", response.StatusCode))
	}
	return nil
}

func getGithubClient(repoURL *url.URL, githubUser, githubPassword string) (*github.Client, error) {
	var apiURL, uploadURL string
	if repoURL.Hostname() == "github.com" {
		apiURL = "https://api." + repoURL.Hostname()
		uploadURL = "https://uploads." + repoURL.Hostname()
	} else {
		apiURL = "https://" + repoURL.Hostname() + "/api/v3"
		uploadURL = "https://uploads." + repoURL.Hostname() + "/api/uploads"
	}
	githubClient, err := util.GetGitHubClient(apiURL, githubUser, githubPassword, uploadURL, true)
	if err != nil {
		return nil, err
	}
	return githubClient, nil
}

// MarkTestrunsAsIngested sets the ingest status of testruns to true
func MarkTestrunsAsUploadedToGithub(log logr.Logger, tmClient kubernetes.Interface, tr *tmv1beta1.Testrun) error {
	ctx := context.Background()
	defer ctx.Done()

	tr.Status.UploadedToGithub = true
	err := tmClient.Client().Update(ctx, tr)
	if err != nil {
		return fmt.Errorf("unable to update status of testrun %s in namespace %s: %s", tr.Name, tr.Namespace, err.Error())
	}
	log.V(3).Info("Successfully updated status of testrun")

	return nil
}