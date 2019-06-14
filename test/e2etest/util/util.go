package util

import (
	"bytes"
	"fmt"
	"github.com/gardener/test-infra/pkg/util"
	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
	"io/ioutil"
	"net/http"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
)

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

func UniqueStrings(stringSlice []string) []string {
	keys := make(map[string]bool)
	list := []string{}
	for _, entry := range stringSlice {
		if _, value := keys[entry]; !value {
			keys[entry] = true
			list = append(list, entry)
		}
	}
	return list
}

func GetEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok {
		return value
	}
	return fallback
}

func RunCmd(command, execPath string) error {
	separator := " "
	parts := strings.Split(command, separator)

	head := parts[0]
	args := parts[1:len(parts)]
	cmd := exec.Command(head, args...)
	if execPath != "" {
		cmd.Dir = execPath
	}

	//	Sanity check -- capture stdout and stderr:
	var out bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &stderr

	err := cmd.Run()

	//	Output our results
	if out.String() != "" {
		log.Info(out.String())
	}
	if stderr.String() != "" {
		log.Error(fmt.Println(stderr.String()))
	}

	return err
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

func SilentStrToInt(s string) int {
	num, err := strconv.Atoi(s)
	if err != nil {
		log.Error(err)
	}
	return num
}

func GetGroupMapOfRegexMatches(re *regexp.Regexp, input string) (groupToValue map[string]string, matched bool) {
	n1 := re.SubexpNames()
	r2 := re.FindAllStringSubmatch(input, -1)[0]

	md := map[string]string{}
	for i, n := range r2 {
		md[n1[i]] = n
	}
	return md, len(md) != 0
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
