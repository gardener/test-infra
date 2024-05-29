package result

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/Masterminds/semver/v3"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v60/github"
	"github.com/pkg/errors"
	"sigs.k8s.io/controller-runtime/pkg/client"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
	"github.com/gardener/test-infra/pkg/common"
	trerrors "github.com/gardener/test-infra/pkg/common/error"
	"github.com/gardener/test-infra/pkg/logger"
	"github.com/gardener/test-infra/pkg/testmachinery/metadata"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
)

// uploads status results as assets to github component releases
func UploadStatusToGithub(log logr.Logger, runs testrunner.RunList, components []*componentdescriptor.Component, githubUser, githubPassword, assetPrefix string) error {
	var (
		prefix = assetPrefix
		dest   = "/tmp/"
	)

	log.V(3).Info(fmt.Sprintf("Storing asset files temporary to directory '%s'", dest))
	overviewFilepath := filepath.Join(dest, fmt.Sprintf("%s%s_overview.json", prefix, runs[0].Metadata.Landscape))
	extendedComponents, err := parseComponents(components, githubUser, githubPassword)
	if err != nil {
		return errors.Wrap(err, "failed to parse components")
	}

	for _, component := range extendedComponents {
		assetOverview, err := DownloadAssetOverview(log, component, overviewFilepath)
		if err != nil {
			return err
		}

		// remove previously failed items, to avoid that after component patch, failed items are kept forever
		removedTestrunItems := removeFailedItems(&assetOverview)
		if err := writeOverviewToFile(assetOverview, overviewFilepath); err != nil {
			return err
		}
		testrunsToUpload, err := identifyTestrunsToUpload(runs, assetOverview, prefix)
		if err != nil {
			return err
		}
		if len(testrunsToUpload) == 0 {
			log.Info("no testrun updates, therefore not assets to upload")
			continue
		}
		log.Info(fmt.Sprintf("identified %d testruns for github asset upload", len(testrunsToUpload)))

		const fileExtension = ".zip"
		archiveName := prefix + testrunsToUpload[0].Metadata.Landscape
		archiveFilename := archiveName + fileExtension

		archiveContentDir := path.Join(dest, archiveName)
		archiveFilepath := filepath.Join(dest, archiveFilename)
		if err := os.Remove(archiveFilepath); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.RemoveAll(archiveContentDir); err != nil && !os.IsNotExist(err) {
			return err
		}
		if err := os.MkdirAll(archiveContentDir, 0777); err != nil {
			return errors.Wrapf(err, "failed to create dir: %s", archiveContentDir)
		}
		remoteArchiveAssetID, err := getAssetIDByName(component, archiveFilename)
		if err != nil {
			log.Error(err, fmt.Sprintf("failed to get asset ID of %s in component %s/%s", archiveFilename, component.Owner, component.Name))
			continue
		}
		if remoteArchiveAssetID != 0 {
			// if status archive file exists, download and unzip it
			if err := downloadReleaseAssetByName(log, component, archiveFilename, archiveFilepath); err != nil {
				log.Error(err, fmt.Sprintf("failed to download release asset %s in component %s/%s", archiveFilename, component.Owner, component.Name))
				continue
			}
			log.Info(fmt.Sprintf("unzipping %s into %s", archiveFilename, archiveContentDir))
			if err := util.Unzip(archiveFilepath, filepath.Dir(archiveContentDir)); err != nil {
				return errors.Wrapf(err, "failed to unzip %s", archiveFilepath)
			}
			for _, fileToRemove := range removedTestrunItems {
				filepathToRemove := filepath.Join(archiveContentDir, fileToRemove)
				if err := os.Remove(filepathToRemove); err != nil {
					log.Info(fmt.Sprintf("failed to remove failed testrun file %s", filepathToRemove), err)
				}
			}
		}
		if err := storeRunsStatusAsFiles(log, testrunsToUpload, prefix, archiveContentDir); err != nil {
			log.Error(err, "Failed to store testrun status as files")
			continue
		}
		if err := util.Zipit(archiveContentDir, archiveFilepath); err != nil {
			log.Error(err, fmt.Sprintf("Failed to zip %s", archiveContentDir))
			continue
		}
		if err := createOrUpdateOverview(log, overviewFilepath, testrunsToUpload, prefix); err != nil {
			log.Error(err, fmt.Sprintf("Failed to create/update %s", overviewFilepath))
			continue
		}

		var filesToUpload []string
		filesToUpload = append(filesToUpload, overviewFilepath)
		filesToUpload = append(filesToUpload, archiveFilepath)

		if err := uploadFiles(log, component, filesToUpload); err != nil {
			return err
		}
	}

	return nil
}

// uploads files to github component releases as assets
func uploadFiles(log logr.Logger, c ComponentExtended, files []string) error {
	for _, filepathToUpload := range files {
		log.Info(fmt.Sprintf("uploading asset %s to %s/%s", filepath.Base(filepathToUpload), c.Owner, c.Name))
		file, err := os.Open(filepathToUpload)
		if err != nil {
			log.Error(err, fmt.Sprintf("Can't open file %s", filepathToUpload))
			return err
		}
		defer file.Close()
		filename := filepath.Base(filepathToUpload)
		uploadOptions := github.UploadOptions{Name: filename}

		// delete previous remote asset, since a new one will be uploaded
		if err := deleteAssetIfExists(c, filename); err != nil {
			log.Error(err, fmt.Sprintf("Can't open file %s", filepathToUpload))
			return err
		}

		_, response, err := c.GithubClient.Repositories.UploadReleaseAsset(context.Background(), c.Owner, c.Name, c.GithubReleaseID, &uploadOptions, file)
		if err != nil {
			log.Error(err, fmt.Sprintf("Was not able to upload %s release asset %s/%s", file.Name(), c.Owner, c.Name))
			return err
		} else if response.StatusCode != 201 {
			err := errors.New(fmt.Sprintf("Asset upload failed with status code %d", response.StatusCode))
			log.Error(err, "")
			return err
		}
	}
	return nil
}

func parseComponents(components []*componentdescriptor.Component, githubUser, githubPassword string) ([]ComponentExtended, error) {
	var extendedComponents []ComponentExtended
	for _, component := range components {
		extendedComponent, err := EnhanceComponent(component, githubUser, githubPassword)
		if err != nil {
			return nil, err
		}
		extendedComponents = append(extendedComponents, extendedComponent)
	}
	return extendedComponents, nil
}

// Either creates a new overview file and feeds it with current testrun results, or downloads the overview file from github and extends it
func createOrUpdateOverview(log logr.Logger, overviewFilepath string, testrunsToUpload testrunner.RunList, prefix string) error {
	assetOverview := AssetOverview{}
	_, err := os.Stat(overviewFilepath) // checks if file exists
	if err == nil {
		log.Info("assets already exist on remote")
		assetOverview, err = unmarshalOverview(overviewFilepath)
		if err != nil {
			return err
		}
	} else {
		log.Info("no assets exist on remote")
	}
	for _, run := range testrunsToUpload {
		assetItemName := generateTestrunAssetName(*run, prefix)
		isAssetItemSuccessful := run.Testrun.Status.Phase == tmv1beta1.RunPhaseSuccess
		assetOverviewItem := assetOverview.Get(assetItemName)
		if assetOverviewItem.Name != "" {
			assetOverview.Get(assetItemName).Successful = isAssetItemSuccessful
		}
		assetOverview.AssetOverviewItems = append(assetOverview.AssetOverviewItems, AssetOverviewItem{
			Name:       assetItemName,
			Successful: isAssetItemSuccessful,
			Dimension: metadata.Dimension{
				Description:       run.Metadata.FlavorDescription,
				Cloudprovider:     run.Metadata.CloudProvider,
				KubernetesVersion: run.Metadata.KubernetesVersion,
				OperatingSystem:   run.Metadata.OperatingSystem,
			},
		})
	}
	if err := writeOverviewToFile(assetOverview, overviewFilepath); err != nil {
		return err
	}
	return nil
}

func writeOverviewToFile(assetOverview AssetOverview, overviewFilepath string) error {
	overviewJSONBytes, err := json.MarshalIndent(assetOverview, "", "   ")
	if err != nil {
		return errors.Wrapf(err, "failed to marshal %s", overviewFilepath)
	}
	if err := os.WriteFile(overviewFilepath, overviewJSONBytes, 0644); err != nil {
		return errors.Wrapf(err, "failed to write file %s", overviewFilepath)
	}
	return nil
}

// renders testrun statuses and saves them as files
func storeRunsStatusAsFiles(log logr.Logger, runs testrunner.RunList, prefix, dest string) error {
	log.Info(fmt.Sprintf("storing testruns status as files in %s", dest))
	for _, run := range runs {
		tr := run.Testrun
		tableString := strings.Builder{}
		output.RenderStatusTable(&tableString, tr.Status.Steps)
		statusOutput := fmt.Sprintf("Testrun: %s\n\n%s\n%s", tr.Name, tableString.String(), util.PrettyPrintStruct(tr.Status))
		assetFilepath := filepath.Join(dest, generateTestrunAssetName(*run, prefix))
		if err := os.WriteFile(assetFilepath, []byte(statusOutput), 0644); err != nil {
			return errors.Wrapf(err, "failed to write file %s", assetFilepath)
		}
	}
	return nil
}

func generateTestrunAssetName(testrun testrunner.Run, prefix string) string {
	md := testrun.Metadata
	return fmt.Sprintf("%s%s-%s.txt", prefix, md.Landscape, md.GetDimensionFromMetadata("-"))
}

// compares overview file items with given testrun list to identify whether any testrun is missing or needs to be updated
func identifyTestrunsToUpload(runs testrunner.RunList, assetOverview AssetOverview, prefix string) (testrunner.RunList, error) {
	var testrunsToUpload testrunner.RunList
	for _, run := range runs {

		// do not consider testruns with a error that are not a timeout error
		if run.Error != nil && !trerrors.IsTimeout(run.Error) {
			continue
		}

		testrunAssetName := generateTestrunAssetName(*run, prefix)
		testrunSuccessful := run.Testrun.Status.Phase == tmv1beta1.RunPhaseSuccess
		if !assetOverview.Contains(testrunAssetName) || testrunSuccessful && !assetOverview.Get(testrunAssetName).Successful {
			testrunsToUpload = append(testrunsToUpload, run)
		}
	}
	return testrunsToUpload, nil
}

// DownloadAssetOverview downloads and parses the asset overview from a component from github
func DownloadAssetOverview(log logr.Logger, component ComponentExtended, overviewFilepath string) (AssetOverview, error) {
	_ = os.Remove(overviewFilepath) // try to remove previously downloaded file
	emptyOverview := AssetOverview{}
	remoteAssetID, err := getAssetIDByName(component, filepath.Base(overviewFilepath))
	if err != nil {
		return emptyOverview, err
	}
	if remoteAssetID == 0 {
		// if no status overview file exists upload results of all testruns
		return emptyOverview, nil
	}
	if err := downloadReleaseAssetByName(log, component, filepath.Base(overviewFilepath), overviewFilepath); err != nil {
		log.Error(err, "unable to download release asset")
		return emptyOverview, err
	}
	assetOverview, err := unmarshalOverview(overviewFilepath)
	if err != nil {
		return emptyOverview, err
	}
	return assetOverview, nil
}

func removeFailedItems(assetOverview *AssetOverview) (failedOverviewItems []string) {
	var successfulOverviewItems []AssetOverviewItem
	for _, item := range assetOverview.AssetOverviewItems {
		if item.Successful {
			successfulOverviewItems = append(successfulOverviewItems, item)
		} else {
			failedOverviewItems = append(failedOverviewItems, item.Name)
		}
	}
	assetOverview.AssetOverviewItems = successfulOverviewItems
	return failedOverviewItems
}

func unmarshalOverview(overviewFilepath string) (AssetOverview, error) {
	var assetOverview AssetOverview
	assetOverviewBytes, err := os.ReadFile(overviewFilepath)
	if err != nil {
		return AssetOverview{}, errors.Wrapf(err, "failed to read file %s", overviewFilepath)
	}
	if err := json.Unmarshal(assetOverviewBytes, &assetOverview); err != nil {
		return AssetOverview{}, errors.Wrapf(err, "failed to unmarshal %s", overviewFilepath)
	}
	return assetOverview, nil
}

func downloadReleaseAssetByName(log logr.Logger, component ComponentExtended, filename, targetPath string) error {
	log.Info(fmt.Sprintf("%s in %s exists, downloading...", filename, component.Name))
	remoteAssetID, err := getAssetIDByName(component, filename)
	if err != nil {
		return errors.Wrapf(err, "failed to get github asset ID of %s in %s", filename, component.Name)
	}
	assetReader, redirectURL, err := component.GithubClient.Repositories.DownloadReleaseAsset(context.Background(), component.Owner, component.Name, remoteAssetID, nil)
	if assetReader != nil {
		defer assetReader.Close()
	}
	if err != nil {
		return err
	}

	assetFile, err := os.Create(targetPath)
	if err != nil {
		return errors.Wrapf(err, "failed to create file %s", targetPath)
	}
	defer assetFile.Close()
	if redirectURL != "" {
		res, err := http.Get(redirectURL)
		if err != nil {
			err := errors.Wrap(err, "http.Get failed:")
			return errors.Wrapf(err, "failed to HTTP GET %s", redirectURL)
		}
		if _, err := io.Copy(assetFile, res.Body); err != nil {
			err := errors.Wrap(err, "http.Get failed:")
			return errors.Wrapf(err, "failed to write data to file %s", assetFile.Name())
		}
		res.Body.Close()

	} else {
		if _, err = io.Copy(assetFile, assetReader); err != nil {
			return errors.Wrapf(err, "failed to write data to file %s", assetFile.Name())
		}
	}

	return nil
}

func getGithubArtifacts(componentName, githubUser, githubPassword string) (githubClient *github.Client, repoOwner, repoName string, err error) {
	urlRaw := fmt.Sprintf("https://%s", componentName)
	repoURL, err := url.Parse(urlRaw)
	if err != nil {
		return nil, "", "", errors.Wrapf(err, "url parse failed for %s", urlRaw)
	}
	repoOwner, repoName, err = util.ParseRepoURL(repoURL)
	if err != nil {
		return nil, "", "", errors.Wrapf(err, "repoURL parse failed for %s", repoURL)
	}
	githubClient, err = getGithubClient(componentName, githubUser, githubPassword)
	if err != nil {
		return nil, "", "", err
	}
	return githubClient, repoOwner, repoName, nil
}

// deletes remote github asset if the asset exists
func deleteAssetIfExists(c ComponentExtended, filename string) error {
	remoteAssetID, err := getAssetIDByName(c, filename)
	if err != nil {
		return err
	}
	if remoteAssetID == 0 {
		// no remote asset exists, nothing to do
		return nil
	}
	response, err := c.GithubClient.Repositories.DeleteReleaseAsset(context.Background(), c.Owner, c.Name, remoteAssetID)
	if err != nil {
		return errors.New("delete github release asset failed")
	} else if response.StatusCode != 204 {
		return errors.Errorf("Delete github release asset failed with status code %d", response.StatusCode)
	}
	return nil
}

func getAssetIDByName(component ComponentExtended, filename string) (int64, error) {
	releaseAssets, response, err := component.GithubClient.Repositories.ListReleaseAssets(context.Background(), component.Owner, component.Name, component.GithubReleaseID, &github.ListOptions{})
	if err != nil {
		return 0, errors.Errorf("failed to list release assets of %s %s", component.Name, component.Version)
	} else if response.StatusCode != 200 {
		return 0, errors.Errorf("Get github release assets failed with status code %d", response.StatusCode)
	}
	for _, releaseAsset := range releaseAssets {
		if strings.Contains(*releaseAsset.Name, filename) {
			return *releaseAsset.ID, nil
		}
	}
	return 0, nil
}

// gets a GitHub release of a repo based on given version
func getRelease(githubClient *github.Client, repoOwner, repoName, componentVersion string) (*github.RepositoryRelease, error) {
	log := logger.Log.WithName("GetGithubReleases")

	version, err := semver.NewVersion(componentVersion)
	if err != nil {
		return nil, errors.Wrapf(err, "version parse failed of %s", componentVersion)
	}

	draft := version.Prerelease() != "" // assumption is that draft versions have always a prerelease e.g. 0.100.0-dev-s5d4f6sdf45s65df4sdf4s4sf
	if !draft {
		release, response, err := githubClient.Repositories.GetReleaseByTag(context.Background(), repoOwner, repoName, componentVersion)
		if err == nil {
			return release, nil
		}

		// At this point an error occurred. But instead of failing, we log error and try to find a release by its name.
		log.Info(errors.WithMessagef(err, "failed to get github release by tag %s in %s with status code %d", componentVersion, repoName, response.StatusCode).Error())
	}

	releaseName, err := version.SetPrerelease("")
	if err != nil {
		return nil, err
	}
	log.Info(fmt.Sprintf("continuing to search for a release in %s by name %s", repoName, releaseName))

	opt := &github.ListOptions{
		PerPage: 50,
	}
	for {
		releases, response, err := githubClient.Repositories.ListReleases(context.Background(), repoOwner, repoName, opt)
		if err != nil {
			return nil, errors.Wrapf(err, "component %s failed to list github releases at %s", componentVersion, repoName)
		} else if response.StatusCode != 200 {
			return nil, errors.Errorf("github releases GET failed with status code %d", response.StatusCode)
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

func getGithubClient(component, githubUser, githubPassword string) (*github.Client, error) {
	repoURL, err := url.Parse(fmt.Sprintf("https://%s", component))
	if err != nil {
		return nil, err
	}
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
		return nil, errors.Wrapf(err, "failed to get github client for %s", component)
	}
	return githubClient, nil
}

// MarkTestrunsAsIngested sets the ingest status of testruns to true
func MarkTestrunsAsUploadedToGithub(log logr.Logger, tmClient client.Client, runs testrunner.RunList) error {
	ctx := context.Background()
	defer ctx.Done()

	for _, run := range runs {
		tr := run.Testrun
		util.SetMetaDataLabel(&tr.ObjectMeta, common.LabelUploadedToGithub, "true")
		err := tmClient.Update(ctx, tr)
		if err != nil {
			return errors.Wrapf(err, "unable to update status of testrun %s in namespace: %s", tr.Name, tr.Namespace)
		}
	}
	log.Info("successfully updated status of testruns")
	return nil
}

type ComponentExtended struct {
	PlainURL        string
	Version         string
	GithubClient    *github.Client
	Owner           string
	Name            string
	GithubReleaseID int64
}

// EnhanceComponent wraps component struct with additional github properties: github client, repo owner, repo name, release ID
func EnhanceComponent(component *componentdescriptor.Component, githubUser string, githubPassword string) (ComponentExtended, error) {
	githubClient, repoOwner, repoName, err := getGithubArtifacts(component.Name, githubUser, githubPassword)
	if err != nil {
		return ComponentExtended{}, errors.Wrap(err, "failed to get github artifacts client, owner, name")
	}

	release, err := getRelease(githubClient, repoOwner, repoName, component.Version)
	if err != nil {
		return ComponentExtended{}, errors.Wrapf(err, "Failed to get repo release for %s %s", repoName, component.Version)
	}

	return ComponentExtended{
		PlainURL:        component.Name,
		Version:         component.Version,
		GithubClient:    githubClient,
		Owner:           repoOwner,
		Name:            repoName,
		GithubReleaseID: *release.ID,
	}, nil
}

type AssetOverview struct {
	AssetOverviewItems []AssetOverviewItem
}

func (overview AssetOverview) Get(assetName string) *AssetOverviewItem {
	for _, asset := range overview.AssetOverviewItems {
		if asset.Name == assetName {
			return &asset
		}
	}
	return &AssetOverviewItem{}
}

func (overview AssetOverview) Contains(searchAssetName string) bool {
	foundAsset := overview.Get(searchAssetName)
	if foundAsset.Name != "" {
		return true
	} else {
		return false
	}
}

type AssetOverviewItem struct {
	Name       string             `json:"name"`
	Successful bool               `json:"successful"`
	Dimension  metadata.Dimension `json:"dimension,omitempty"`
}
