package util

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	gardencorev1alpha1 "github.com/gardener/gardener/pkg/apis/core/v1alpha1"
	"github.com/gardener/gardener/pkg/client/kubernetes"
	"github.com/gardener/gardener/test/integration/framework"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"strings"
)

// use max log line length, because kubetest commands can be very long
const logMaxLength = 300

func DownloadFile(url, dir string) (filePath string, err error) {
	request, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return "", err
	}
	fileName := path.Base(request.URL.Path)
	filePath = filepath.Join(dir, fileName)
	fileData, err := util.DownloadFile(&http.Client{}, url)
	if err != nil {
		return "", err
	}
	if err := ioutil.WriteFile(filePath, fileData, 0777); err != nil {
		return "", err
	}
	return filePath, nil
}

func RunCmd(command, execPath string) (output CmdOutput, err error) {
	separator := " "
	parts := strings.Split(command, separator)

	head := parts[0]
	args := parts[1:len(parts)]
	cmd := exec.Command(head, args...)
	if execPath != "" {
		cmd.Dir = execPath
	}

	//	Sanity check -- capture outPipe and stderr:
	var out bytes.Buffer
	var stderr bytes.Buffer

	outWriter := io.MultiWriter(os.Stdout, &out)
	errWriter := io.MultiWriter(os.Stderr, &stderr)
	cmd.Stdout = outWriter
	cmd.Stderr = errWriter

	if len(command) > logMaxLength {
		log.Infof("%s...", command[:logMaxLength])
	} else {
		log.Info(command)
	}
	err = cmd.Run()

	//	Output our results
	if out.String() != "" {
		stdoutString := out.String()
		output.StdOut = stdoutString
	}
	if stderr.Len() != 0 {
		stderrString := stderr.String()
		output.StdErr = stderrString
	}

	return output, err
}

// Contains tells whether a contains x.
func Contains(a []string, x string) bool {
	for _, n := range a {
		if x == n {
			return true
		}
	}
	return false
}

func GetGroupMapOfRegexMatches(re *regexp.Regexp, input string) map[string]string {
	n1 := re.SubexpNames()
	r2 := re.FindAllStringSubmatch(input, -1)[0]

	md := map[string]string{}
	for i, n := range r2 {
		md[n1[i]] = n
	}
	return md
}

func GetFilesByPattern(rootDir, filenamePattern string) []string {
	var files []string

	err := filepath.Walk(rootDir, func(path string, info os.FileInfo, err error) error {
		matched, err := regexp.MatchString(filenamePattern, path)
		if matched {
			files = append(files, path)
		}
		return nil
	})
	if err != nil {
		log.Fatal(errors.Wrapf(err, "Couldn't walk path %s", rootDir))
	}
	return files
}

type CmdOutput struct {
	StdOut string
	StdErr string
}

/*
   GoLang: os.Rename() give error "invalid cross-device link" for Docker container with Volumes.
   MoveFile(source, destination) will work moving file between folders
*/
func MoveFile(sourcePath, destPath string) error {
	inputFile, err := os.Open(sourcePath)
	if err != nil {
		return fmt.Errorf("couldn't open source file: %s", err)
	}
	outputFile, err := os.Create(destPath)
	if err != nil {
		_ = inputFile.Close()
		return fmt.Errorf("couldn't open dest file: %s", err)
	}
	defer outputFile.Close()
	_, err = io.Copy(outputFile, inputFile)
	_ = inputFile.Close()
	if err != nil {
		return fmt.Errorf("writing to output file failed: %s", err)
	}
	// The copy was successful, so now delete the original file
	err = os.Remove(sourcePath)
	if err != nil {
		return fmt.Errorf("failed removing original file: %s", err)
	}
	return nil
}

/*
	CommandExists checks whether the given command executable exists and returns a boolean result value
*/
func CommandExists(cmd string) bool {
	_, err := exec.LookPath(cmd)
	return err == nil
}

func ReadLinesFromFile(file io.Reader) <-chan []byte {
	c := make(chan []byte)
	go func() {
		reader := bufio.NewReader(file)
		doc := make([]byte, 0)
		for {
			line, isPrefix, err := reader.ReadLine()
			if err == io.EOF {
				break
			}
			if err != nil {
				return
			}
			doc = append(doc, line...)
			if isPrefix {
				continue
			}
			c <- doc
			doc = make([]byte, 0)
		}

		close(c)
	}()
	return c
}

func Copy(src, dst string) (int64, error) {
	sourceFileStat, err := os.Stat(src)
	if err != nil {
		return 0, err
	}

	if !sourceFileStat.Mode().IsRegular() {
		return 0, fmt.Errorf("%s is not a regular file", src)
	}

	source, err := os.Open(src)
	if err != nil {
		return 0, err
	}
	defer source.Close()

	destination, err := os.Create(dst)
	if err != nil {
		return 0, err
	}
	defer destination.Close()
	nBytes, err := io.Copy(destination, source)
	return nBytes, err
}

func DumpShootLogs(gardenKubeconfigPath, projectNamespace, shootName string) error {
	logger := log.New()
	if gardenKubeconfigPath == "" || projectNamespace == "" || shootName == "" {
		logger.Warn("cannot dump shoot cluster events because of missing parameters gardener kubconfig / project namespace / shoot name")
		return nil
	}
	gardenerClient, err := kubernetes.NewClientFromFile("", gardenKubeconfigPath, kubernetes.WithClientOptions(
		client.Options{
			Scheme: kubernetes.GardenScheme,
		}))
	if err != nil {
		return err
	}
	gardenerTestOperations, err := framework.NewGardenTestOperation(gardenerClient, logger)
	if err != nil {
		return err
	}

	ctx := context.Background()
	shoot := &gardencorev1alpha1.Shoot{ObjectMeta: metav1.ObjectMeta{Namespace: projectNamespace, Name: shootName}}
	if err := gardenerTestOperations.AddShoot(ctx, shoot); err != nil {
		logger.Error(err.Error())
		gardenerTestOperations.Shoot = shoot
	}
	gardenerTestOperations.DumpState(ctx)
	return nil
}
