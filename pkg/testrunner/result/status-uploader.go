package result

import (
	"context"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v27/github"
	"github.com/pkg/errors"
	"net/url"
	"strings"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

const (
	success = "SUCCESS"
	fail    = "FAIL"
)

// uploads status results as asset to the component
func UploadStatusToGithub(log logr.Logger, run *testrunner.Run, component *componentdescriptor.Component, githubUser, githubPassword, assetPrefix string) error {
	tr := run.Testrun
	md := run.Metadata
	testrunStatusTag := fail
	if run.Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess {
		testrunStatusTag = success
	}
	tableString := strings.Builder{}
	output.RenderStatusTable(&tableString, tr.Status.Steps)
	statusOutput := fmt.Sprintf("Testrun: %s\n\n%s\n%s", tr.Name, tableString.String(), util.PrettyPrintStruct(tr.Status))
	filenameWithoutStatus := fmt.Sprintf("%s%s-%s-%s", assetPrefix, md.Landscape, md.CloudProvider, md.KubernetesVersion)
	filename := fmt.Sprintf("%s-%s.txt", filenameWithoutStatus, testrunStatusTag)

	repoURL, err := url.Parse(fmt.Sprintf("https://%s", component.Name))
	if err != nil {
		return err
	}
	repoOwner, repoName := util.ParseRepoURL(repoURL)
	githubClient, err := getGithubClient(repoURL, githubUser, githubPassword)
	if err != nil {
		return err
	}

	release, err := getRelease(githubClient, repoOwner, repoName, component.Version)
	if err != nil {
		return err
	}

	remoteAssetID, isRemoteAssetSuccessful, err := getAssetIDByName(githubClient, *release.ID, repoOwner, repoName, filenameWithoutStatus)
	if err != nil {
		return err
	}
	if remoteAssetID != 0 {
		if isRemoteAssetSuccessful {
			// do not overwrite existing successful asset to ensure consistent reporting
			log.V(3).Info(fmt.Sprintf("Skip asset upload, since asset of a a succeeded testrun '%s*' already exists", filenameWithoutStatus))
			return nil
		} else {
			// delete failed remote asset, since a new one will be uploaded
			log.V(3).Info(fmt.Sprintf("Delete remote asset '%s*', so that a newer version can be uploaded", filenameWithoutStatus))
			if err := deleteAsset(githubClient, repoOwner, repoName, remoteAssetID); err != nil {
				return err
			}
		}
	}

	if err = uploadAsset(githubClient, *release.ID, repoOwner, repoName, filename, statusOutput); err != nil {
		return err
	}
	return nil
}

func deleteAsset(githubClient *github.Client, repoOwner string, repoName string, assetID int64) error {
	response, err := githubClient.Repositories.DeleteReleaseAsset(context.Background(), repoOwner, repoName, assetID)
	if err != nil {
		return err
	} else if response.StatusCode != 204 {
		return errors.New(fmt.Sprintf("Delete github release asset failed with status code %d", response.StatusCode))
	}
	return nil
}

func getAssetIDByName(githubClient *github.Client, releaseID int64, repoOwner, repoName, filename string) (int64, bool, error) {
	releaseAssets, response, err := githubClient.Repositories.ListReleaseAssets(context.Background(), repoOwner, repoName, releaseID, &github.ListOptions{})
	if err != nil {
		return 0, false, err
	} else if response.StatusCode != 200 {
		return 0, false, errors.New(fmt.Sprintf("Get github release assets failed with status code %d", response.StatusCode))
	}
	for _, releaseAsset := range releaseAssets {
		if strings.Contains(*releaseAsset.Name, filename) {
			if strings.Contains(*releaseAsset.Name, fail) {
				return *releaseAsset.ID, false, nil
			} else if strings.Contains(*releaseAsset.Name, success) {
				return *releaseAsset.ID, true, nil
			}
		}
	}
	return 0, false, nil
}

func uploadAsset(githubClient *github.Client, releaseID int64, repoOwner, repoName, filename, statusOutput string) error {
	uploadUrl := fmt.Sprintf("repos/%s/%s/releases/%d/assets?name=%s", repoOwner, repoName, releaseID, filename)
	mediaType := "text/plain; charset=utf-8"
	request, err := githubClient.NewUploadRequest(uploadUrl, strings.NewReader(statusOutput), int64(len(statusOutput)), mediaType)
	if err != nil {
		return err
	}
	asset := new(github.ReleaseAsset)
	response, err := githubClient.Do(context.Background(), request, asset)
	if err != nil {
		return err
	} else if response.StatusCode != 201 {
		return errors.New(fmt.Sprintf("Asset upload failed with status code %d", response.StatusCode))
	}
	return nil
}

func getRelease(githubClient *github.Client, repoOwner, repoName, componentVersion string) (*github.RepositoryRelease, error) {
	version, err := semver.NewVersion(componentVersion)
	if err != nil {
		return nil, err
	}

	draft := version.Prerelease() != "" // assumption is that draft versions have always a prerelease e.g. 0.100.0-dev-s5d4f6sdf45s65df4sdf4s4sf
	if !draft {
		release, response, err := githubClient.Repositories.GetReleaseByTag(context.Background(), repoOwner, repoName, componentVersion)
		if err != nil {
			return nil, err
		} else if response.StatusCode != 200 {
			return nil, errors.New(fmt.Sprintf("Github release GET failed with status code %d", response.StatusCode))
		}
		return release, nil
	}

	releaseName, err := version.SetPrerelease("")
	if err != nil {
		return nil, err
	}

	opt := &github.ListOptions{
		PerPage: 50,
	}
	for {
		releases, response, err := githubClient.Repositories.ListReleases(context.Background(), repoOwner, repoName, opt)
		if err != nil {
			return nil, err
		} else if response.StatusCode != 200 {
			return nil, errors.New(fmt.Sprintf("Github releases GET failed with status code %d", response.StatusCode))
		}
		for _, release := range releases {
			if *release.Draft && strings.Contains(*release.Name, releaseName.String()) {
				return release, nil
			}
		}
		if response.NextPage == 0 {
			break
		}
		opt.Page = response.NextPage
	}
	return nil, errors.New("no releases found")
}

func getGithubClient(repoURL *url.URL, githubUser, githubPassword string) (*github.Client, error) {
	var apiURL, uploadURL string
	if repoURL.Hostname() == "github.com" {
		apiURL = "https://api." + repoURL.Hostname()
		uploadURL = "https://uploads." + repoURL.Hostname()
	} else {
		apiURL = "https://" + repoURL.Hostname() + "/api/v3"
		uploadURL = "https://" + repoURL.Hostname() + "/api/uploads"
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

	enabled := true
	tr.Status.UploadedToGithub = &enabled
	err := tmClient.Client().Update(ctx, tr)
	if err != nil {
		return fmt.Errorf("unable to update status of testrun %s in namespace %s: %s", tr.Name, tr.Namespace, err.Error())
	}
	log.V(3).Info("Successfully updated status of testrun")

	return nil
}
