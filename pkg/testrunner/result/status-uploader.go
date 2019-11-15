package result

import (
	"context"
	"encoding/json"
	"fmt"
	"github.com/Masterminds/semver"
	"github.com/gardener/test-infra/pkg/testrunner"
	"github.com/gardener/test-infra/pkg/testrunner/componentdescriptor"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/gardener/test-infra/pkg/util/output"
	"github.com/go-logr/logr"
	"github.com/google/go-github/v27/github"
	"github.com/pkg/errors"
	"io"
	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"strings"

	"github.com/gardener/gardener/pkg/client/kubernetes"

	tmv1beta1 "github.com/gardener/test-infra/pkg/apis/testmachinery/v1beta1"
)

var prefix string
var l logr.Logger

// uploads status results as assets to github component releases
func UploadStatusToGithub(loggerInstance logr.Logger, runs testrunner.RunList, components []*componentdescriptor.Component, githubUser, githubPassword, assetPrefix string) error {
	prefix = assetPrefix
	l = loggerInstance
	dest := "/tmp/"
	l.Info(fmt.Sprintf("Storing asset files temporary to directory '%s'", dest))
	overviewFilepath := filepath.Join(dest, fmt.Sprintf("%s%s_overview.json", prefix, (runs)[0].Metadata.Landscape))
	extendedComponents, err := parseComponents(components, githubUser, githubPassword)
	if err != nil {
		l.Error(err, "failed to parse components")
		return err
	}
	// need one component to check whether status assets have been uploaded before to corresponding release
	randomComponent := extendedComponents[0]

	testrunsToUpload, err := identifyTestrunsToUpload(runs, randomComponent, overviewFilepath)
	if testrunsToUpload == nil || len(testrunsToUpload) == 0 {
		l.Info("no testrun updates, therefore not assets to upload")
		return nil
	}
	l.Info(fmt.Sprintf("identified %d testruns for github asset upload", len(testrunsToUpload)))

	archiveFilename := fmt.Sprintf("%s%s.zip", prefix, (runs)[0].Metadata.Landscape)
	extension := filepath.Ext(archiveFilename)
	archiveFilenameWithoutExtension := archiveFilename[0 : len(archiveFilename)-len(extension)]
	archiveContentDir := path.Join(dest, archiveFilenameWithoutExtension)
	archiveFilepath := filepath.Join(dest, archiveFilename)
	_ = os.Remove(archiveFilepath)
	_ = os.RemoveAll(archiveContentDir)
	if err := os.MkdirAll(archiveContentDir, 0777); err != nil {
		l.Error(err, fmt.Sprintf("failed to create dir: %s", archiveContentDir))
		return err
	}
	remoteArchiveAssetID, err := getAssetIDByName(randomComponent, archiveFilename)
	if err != nil {
		l.Error(err, fmt.Sprintf("failed to get asset ID of %s in component %s/%s", archiveFilename, randomComponent.Owner, randomComponent.Name))
		return err
	}
	if remoteArchiveAssetID != 0 {
		// if status archive file exists, download and unzip it
		if err := downloadReleaseAssetByName(randomComponent, archiveFilename, archiveFilepath); err != nil {
			l.Error(err, fmt.Sprintf("failed to download release asset %s in component %s/%s", archiveFilename, randomComponent.Owner, randomComponent.Name))
			return err
		}
		l.Info(fmt.Sprintf("unzipping %s into %s", archiveFilename, archiveContentDir))
		if err := util.Unzip(archiveFilepath, archiveContentDir); err != nil {
			l.Error(err, fmt.Sprintf("failed to unzip %s", archiveFilepath))
			return err
		}
	}
	if err := storeRunsStatusAsFiles(testrunsToUpload, archiveContentDir); err != nil {
		l.Error(err, "Failed to store testrun status as files")
		return err
	}
	if err := util.Zipit(archiveContentDir, archiveFilepath); err != nil {
		l.Error(err, fmt.Sprintf("Failed to zip %s", archiveContentDir))
		return err
	}
	if err := createOrUpdateOverview(overviewFilepath, testrunsToUpload); err != nil {
		l.Error(err, fmt.Sprintf("Failed to create/update %s", overviewFilepath))
		return err
	}

	var filesToUpload []string
	filesToUpload = append(filesToUpload, overviewFilepath)
	filesToUpload = append(filesToUpload, archiveFilepath)

	if err := uploadFiles(extendedComponents, filesToUpload); err != nil {
		return err
	}

	return nil
}

// uploads files to github component releases as assets
func uploadFiles(components []ComponentExtended, files []string) error {
	for _, c := range components {
		for _, filepathToUpload := range files {
			l.Info(fmt.Sprintf("uploading asset %s to %s/%s", filepath.Base(filepathToUpload), c.Owner, c.Name))
			file, err := os.Open(filepathToUpload)
			if err != nil {
				l.Error(err, fmt.Sprintf("Can't open file %s", filepathToUpload))
				return err
			}
			defer file.Close()
			filename := filepath.Base(filepathToUpload)
			uploadOptions := github.UploadOptions{Name: filename}

			// delete previous remote asset, since a new one will be uploaded
			if err := deleteAssetIfExists(c, filename); err != nil {
				l.Error(err, fmt.Sprintf("Can't open file %s", filepathToUpload))
				return err
			}

			_, response, err := c.GithubClient.Repositories.UploadReleaseAsset(context.Background(), c.Owner, c.Name, c.GithubReleaseID, &uploadOptions, file)
			if err != nil {
				l.Error(err, fmt.Sprintf("Was not able to upload %s release asset %s/%s", file.Name(), c.Owner, c.Name))
				return err
			} else if response.StatusCode != 201 {
				err := errors.New(fmt.Sprintf("Asset upload failed with status code %d", response.StatusCode))
				l.Error(err, "")
				return err
			}
		}
	}
	return nil
}

func parseComponents(components []*componentdescriptor.Component, githubUser, githubPassword string) ([]ComponentExtended, error) {
	var extendedComponents []ComponentExtended
	for _, component := range components {
		extendedComponent, err := enhanceComponent(component, githubUser, githubPassword)
		if err != nil {
			return nil, err
		}
		extendedComponents = append(extendedComponents, extendedComponent)
	}
	return extendedComponents, nil
}

// Either creates a new overview file and feeds it with current testrun results, or downloads the overview file from github and extends it
func createOrUpdateOverview(overviewFilepath string, testrunsToUpload testrunner.RunList) error {
	assetOverview := AssetOverview{}
	_, err := os.Stat(overviewFilepath) // checks if file exists
	if err == nil {
		l.Info("assets already exist on remote")
		assetOverview, err = unmarshalOverview(overviewFilepath)
		if err != nil {
			return err
		}
	} else {
		l.Info("no assets exist on remote")
	}
	for _, run := range testrunsToUpload {
		assetItemName := generateTestrunAssetName(*run)
		isAssetItemSuccessful := run.Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess
		assetOverviewItem := assetOverview.Get(assetItemName)
		if assetOverviewItem.Name != "" {
			assetOverview.Get(assetItemName).Successful = isAssetItemSuccessful
		}
		assetOverview.AssetOverviewItems = append(assetOverview.AssetOverviewItems, AssetOverviewItem{Name: assetItemName, Successful: isAssetItemSuccessful})
	}
	overviewJSONBytes, err := json.MarshalIndent(assetOverview, "", "   ")
	if err != nil {
		l.Error(err, fmt.Sprintf("failed to marshal %s", overviewFilepath))
		return err
	}
	if err := ioutil.WriteFile(overviewFilepath, overviewJSONBytes, 0644); err != nil {
		l.Error(err, fmt.Sprintf("failed to write file %s", overviewFilepath))
		return err
	}
	return nil
}

// renders testrun statuses and saves them as files
func storeRunsStatusAsFiles(runs testrunner.RunList, dest string) error {
	l.Info(fmt.Sprintf("storing testruns status as files in %s", dest))
	for _, run := range runs {
		tr := run.Testrun
		tableString := strings.Builder{}
		output.RenderStatusTable(&tableString, tr.Status.Steps)
		statusOutput := fmt.Sprintf("Testrun: %s\n\n%s\n%s", tr.Name, tableString.String(), util.PrettyPrintStruct(tr.Status))
		assetFilepath := filepath.Join(dest, generateTestrunAssetName(*run))
		if err := ioutil.WriteFile(assetFilepath, []byte(statusOutput), 0644); err != nil {
			l.Error(err, fmt.Sprintf("failed to write file %s", assetFilepath))
			return err
		}
	}
	return nil
}

func generateTestrunAssetName(testrun testrunner.Run) string {
	md := testrun.Metadata
	return fmt.Sprintf("%s%s-%s-%s.txt", prefix, md.Landscape, md.CloudProvider, md.KubernetesVersion)
}

// compares overview file items with given testrun list to identify whether any testrun is missing or needs to be updated
func identifyTestrunsToUpload(runs testrunner.RunList, component ComponentExtended, overviewFilepath string) (testrunner.RunList, error) {
	_ = os.Remove(overviewFilepath) // try to remove previously downloaded file
	remoteAssetID, err := getAssetIDByName(component, filepath.Base(overviewFilepath))
	if err != nil {
		return nil, err
	}
	if remoteAssetID == 0 {
		// if no status overview file exists upload results of all testruns
		return runs, nil
	}
	if err := downloadReleaseAssetByName(component, filepath.Base(overviewFilepath), overviewFilepath); err != nil {
		return nil, err
	}
	var testrunsToUpload testrunner.RunList
	assetOverview, err := unmarshalOverview(overviewFilepath)
	if err != nil {
		return nil, err
	}

	for _, run := range runs {
		testrunAssetName := generateTestrunAssetName(*run)
		testrunSuccessful := run.Testrun.Status.Phase == tmv1beta1.PhaseStatusSuccess
		if !assetOverview.Contains(testrunAssetName) || testrunSuccessful && !assetOverview.Get(testrunAssetName).Successful {
			testrunsToUpload = append(testrunsToUpload, run)
		}
	}
	return testrunsToUpload, nil
}

func unmarshalOverview(overviewFilepath string) (AssetOverview, error) {
	var assetOverview AssetOverview
	assetOverviewBytes, err := ioutil.ReadFile(overviewFilepath)
	if err != nil {
		l.Error(err, fmt.Sprintf("failed to read file %s", overviewFilepath))
		return AssetOverview{}, err
	}
	if err := json.Unmarshal(assetOverviewBytes, &assetOverview); err != nil {
		l.Error(err, fmt.Sprintf("failed to unmarshal %s", overviewFilepath))
		return AssetOverview{}, err
	}
	return assetOverview, nil
}

func downloadReleaseAssetByName(component ComponentExtended, filename, targetPath string) error {
	l.Info(fmt.Sprintf("%s in %s exists, downloading...", filename, component.Name))
	remoteAssetID, err := getAssetIDByName(component, filename)
	if err != nil {
		l.Error(err, fmt.Sprintf("failed to get github asset ID of %s in %s", filename, component.Name))
		return err
	}
	assetReader, redirectURL, err := component.GithubClient.Repositories.DownloadReleaseAsset(context.Background(), component.Owner, component.Name, remoteAssetID)
	if assetReader != nil {
		defer assetReader.Close()
	}
	if err != nil {
		return err
	}

	assetFile, err := os.Create(targetPath)
	if err != nil {
		l.Error(err, fmt.Sprintf("failed to create file %s", targetPath))
		return err
	}
	defer assetFile.Close()
	if redirectURL != "" {
		res, err := http.Get(redirectURL)
		if err != nil {
			err := errors.Wrap(err, "http.Get failed:")
			l.Error(err, fmt.Sprintf("failed to HTTP GET %s", redirectURL))
			return err
		}
		if _, err := io.Copy(assetFile, res.Body); err != nil {
			err := errors.Wrap(err, "http.Get failed:")
			l.Error(err, fmt.Sprintf("failed to write data to file %s", assetFile.Name()))
			return err
		}
		res.Body.Close()

	} else {
		if _, err = io.Copy(assetFile, assetReader); err != nil {
			l.Error(err, fmt.Sprintf("failed to write data to file %s", assetFile.Name()))
			return err
		}
	}

	return nil
}

func getGithubArtifacts(componentName, githubUser, githubPassword string) (githubClient *github.Client, repoOwner, repoName string, err error) {
	urlRaw := fmt.Sprintf("https://%s", componentName)
	repoURL, err := url.Parse(urlRaw)
	if err != nil {
		l.Error(err, fmt.Sprintf("url parse failed for %s", urlRaw))
		return nil, "", "", err
	}
	repoOwner, repoName = util.ParseRepoURL(repoURL)
	githubClient = getGithubClient(componentName, githubUser, githubPassword)
	return githubClient, repoOwner, repoName, nil
}

// deletes remote github asset if the asset exists
func deleteAssetIfExists(c ComponentExtended, filename string) error {
	remoteAssetID, err := getAssetIDByName(c, filename)
	if remoteAssetID == 0 {
		// no remote asset exists, nothing to do
		return nil
	}
	response, err := c.GithubClient.Repositories.DeleteReleaseAsset(context.Background(), c.Owner, c.Name, remoteAssetID)
	if err != nil {
		l.Error(errors.New("delete github release asset failed"), "")
		return err
	} else if response.StatusCode != 204 {
		l.Error(errors.New(fmt.Sprintf("Delete github release asset failed with status code %d", response.StatusCode)), "")
		return err
	}
	return nil
}

func getAssetIDByName(component ComponentExtended, filename string) (int64, error) {
	releaseAssets, response, err := component.GithubClient.Repositories.ListReleaseAssets(context.Background(), component.Owner, component.Name, component.GithubReleaseID, &github.ListOptions{})
	if err != nil {
		l.Error(errors.New(fmt.Sprintf("failed to list release assets of %s %s", component.Name, component.Version)), "")
		return 0, err
	} else if response.StatusCode != 200 {
		err := errors.New(fmt.Sprintf("Get github release assets failed with status code %d", response.StatusCode))
		l.Error(err, "")
		return 0, err
	}
	for _, releaseAsset := range releaseAssets {
		if strings.Contains(*releaseAsset.Name, filename) {
			return *releaseAsset.ID, nil
		}
	}
	return 0, nil
}

// gets a github release of a repo based on given version
func getRelease(githubClient *github.Client, repoOwner, repoName, componentVersion string) (*github.RepositoryRelease, error) {
	version, err := semver.NewVersion(componentVersion)
	if err != nil {
		l.Error(err, fmt.Sprintf("version parse failed of %s", componentVersion))
		return nil, err
	}

	draft := version.Prerelease() != "" // assumption is that draft versions have always a prerelease e.g. 0.100.0-dev-s5d4f6sdf45s65df4sdf4s4sf
	if !draft {
		release, response, err := githubClient.Repositories.GetReleaseByTag(context.Background(), repoOwner, repoName, componentVersion)
		if err != nil {
			l.Error(err, fmt.Sprintf("failed to get github release by tag %s in %s", componentVersion, repoName))
			return nil, err
		} else if response.StatusCode != 200 {
			err := errors.New(fmt.Sprintf("release retrival failed with status code %d", response.StatusCode))
			l.Error(err, "")
			return nil, err
		}
		return release, nil
	}

	releaseName, err := version.SetPrerelease("")
	if err != nil {
		l.Error(err, "")
		return nil, err
	}

	opt := &github.ListOptions{
		PerPage: 50,
	}
	for {
		releases, response, err := githubClient.Repositories.ListReleases(context.Background(), repoOwner, repoName, opt)
		if err != nil {
			l.Error(err, fmt.Sprintf("failed to get github release by tag %s in %s", componentVersion, repoName))
			return nil, err
		} else if response.StatusCode != 200 {
			err := errors.New(fmt.Sprintf("github releases GET failed with status code %d", response.StatusCode))
			l.Error(err, "")
			return nil, err
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
	err = errors.New("no releases found")
	l.Error(err, "")
	return nil, err
}

func getGithubClient(component, githubUser, githubPassword string) *github.Client {
	repoURL, err := url.Parse(fmt.Sprintf("https://%s", component))
	if err != nil {
		return nil
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
		l.Error(err, fmt.Sprintf("failed to get github client for %s", component))
		return nil
	}
	return githubClient
}

// MarkTestrunsAsIngested sets the ingest status of testruns to true
func MarkTestrunsAsUploadedToGithub(log logr.Logger, tmClient kubernetes.Interface, runs testrunner.RunList) error {
	ctx := context.Background()
	defer ctx.Done()

	for _, run := range runs {
		tr := run.Testrun
		enabled := true
		tr.Status.UploadedToGithub = &enabled
		err := tmClient.Client().Update(ctx, tr)
		if err != nil {
			log.Error(err, fmt.Sprintf("unable to update status of testrun %s in namespace: %s", tr.Name, tr.Namespace))
			return err
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

// wraps component struct with additional github properties: github client, repo owner, repo name, release ID
func enhanceComponent(component *componentdescriptor.Component, githubUser string, githubPassword string) (ComponentExtended, error) {
	githubClient, repoOwner, repoName, err := getGithubArtifacts(component.Name, githubUser, githubPassword)
	if err != nil {
		l.Error(err, "Failed to get github artifacts client, owner, name")
		return ComponentExtended{}, err
	}

	release, err := getRelease(githubClient, repoOwner, repoName, component.Version)
	if err != nil {
		l.Error(err, fmt.Sprintf("Failed to get repo release for %s %s", repoName, component.Version))
		return ComponentExtended{}, err
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
	Name       string `json:"name"`
	Successful bool   `json:"successful"`
}
