package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/bitrise-io/go-steputils/stepconf"
	"github.com/bitrise-io/go-utils/log"
	"github.com/bitrise-io/go-utils/pathutil"
)

// Configs stores the step's inputs
type Configs struct {
	JSONKeyPath       stepconf.Secret `env:"service_account_json_key_path,required"`
	PackageName       string          `env:"package_name,required"`
	AppPath           string          `env:"app_path,required"`
	ExpansionfilePath string          `env:"expansionfile_path"`
	Track             string          `env:"track,required"`
	UserFraction      float64         `env:"user_fraction,range]0.0..1.0["`
	UpdatePriority    int             `env:"update_priority,range[0..5]"`
	WhatsnewsDir      string          `env:"whatsnews_dir"`
	MappingFile       string          `env:"mapping_file"`
}

// validate validates the Configs.
func (c Configs) validate() error {
	if err := c.validateJSONKeyPath(); err != nil {
		return err
	}

	if err := c.validateWhatsnewsDir(); err != nil {
		return err
	}

	if err := c.validateMappingFile(); err != nil {
		return err
	}

	return c.validateApps()
}

// validateJSONKeyPath validates if service_account_json_key_path input value exists if defined and has file:// URL scheme.
func (c Configs) validateJSONKeyPath() error {
	if !strings.HasPrefix(string(c.JSONKeyPath), "file://") {
		return nil
	}

	pth := strings.TrimPrefix(string(c.JSONKeyPath), "file://")
	if exist, err := pathutil.IsPathExists(pth); err != nil {
		return fmt.Errorf("failed to check if json key path exist at: %s, error: %s", pth, err)
	} else if !exist {
		return errors.New("json key path not exist at: " + pth)
	}
	return nil
}

// validateWhatsnewsDir validates if whatsnews_dir input value exists if provided.
func (c Configs) validateWhatsnewsDir() error {
	if c.WhatsnewsDir == "" {
		return nil
	}

	if exist, err := pathutil.IsDirExists(c.WhatsnewsDir); err != nil {
		return fmt.Errorf("failed to check if what's new directory exist at: %s, error: %s", c.WhatsnewsDir, err)
	} else if !exist {
		return errors.New("what's new directory not exist at: " + c.WhatsnewsDir)
	}
	return nil
}

// validateMappingFile validates if mapping_file input value exists if provided.
func (c Configs) validateMappingFile() error {
	if c.MappingFile == "" {
		return nil
	}

	if exist, err := pathutil.IsPathExists(c.MappingFile); err != nil {
		return fmt.Errorf("failed to check if mapping file exist at: %s, error: %s", c.MappingFile, err)
	} else if !exist {
		return errors.New("mapping file not exist at: " + c.MappingFile)
	}
	return nil
}

func splitElements(list []string, sep string) (s []string) {
	for _, e := range list {
		s = append(s, strings.Split(e, sep)...)
	}
	return
}

func parseAppList(list string) (apps []string) {
	log.Debugf("Parsing app list: '%v'", list)
	list = strings.TrimSpace(list)
	if len(list) == 0 {
		return nil
	}

	s := []string{list}
	for _, sep := range []string{"\n", `\n`, "|"} {
		s = splitElements(s, sep)
	}

	for _, app := range s {
		app = strings.TrimSpace(app)
		if len(app) > 0 {
			apps = append(apps, app)
			log.Debugf("Found app: %v", app)
		}
	}
	return
}

// appPaths returns the app to deploy, by preferring .aab files.
func (c Configs) appPaths() ([]string, []string) {
	var apks, aabs, warnings []string
	for _, pth := range parseAppList(c.AppPath) {
		pth = strings.TrimSpace(pth)
		ext := strings.ToLower(filepath.Ext(pth))
		if ext == ".aab" {
			log.Infof("Found .aab file: %v", pth)
			aabs = append(aabs, pth)
		} else if ext == ".apk" {
			log.Infof("Found .apk file: %v", pth)
			apks = append(apks, pth)
		} else {
			warnings = append(warnings, fmt.Sprintf("unknown app path extension in path: %s, supported extensions: .apk, .aab", pth))
		}
	}

	if len(aabs) > 0 && len(apks) > 0 {
		warnings = append(warnings, fmt.Sprintf("Both .aab and .apk files provided, using the .aab file(s): %s", strings.Join(aabs, ",")))
	}

	if len(aabs) > 1 {
		warnings = append(warnings, fmt.Sprintf("More than 1 .aab files provided, using the first: %s", aabs[0]))
	}

	if len(aabs) > 0 {
		return aabs[:1], warnings
	}

	return apks, warnings
}

// validateApps validates if files provided via app_path are existing files,
// if app_path is empty it validates if files provided via app_path input are existing .apk or .aab files.
func (c Configs) validateApps() error {
	apps, warnings := c.appPaths()
	for _, warn := range warnings {
		log.Warnf(warn)
	}

	if len(apps) == 0 {
		return fmt.Errorf("no app provided")
	}

	for _, pth := range apps {
		if exist, err := pathutil.IsPathExists(pth); err != nil {
			return fmt.Errorf("failed to check if app exist at: %s, error: %s", pth, err)
		} else if !exist {
			return errors.New("app not exist at: " + pth)
		}
	}

	return nil
}

// expansionFiles gets the expansion files from the received configuration. Returns true and the entries (type and
// path) of them when any found, false or error otherwise.
func expansionFiles(appPaths []string, expansionFilePathConfig string) ([]string, error) {
	// "main:/file/path/1.obb|patch:/file/path/2.obb"
	var expansionFileEntries = []string{}
	if strings.TrimSpace(expansionFilePathConfig) != "" {
		expansionFileEntries = strings.Split(expansionFilePathConfig, "|")

		if len(appPaths) != len(expansionFileEntries) {
			return []string{}, fmt.Errorf("mismatching number of APKs(%d) and Expansionfiles(%d)", len(appPaths), len(expansionFileEntries))
		}

		log.Infof("Found %v expansion file(s) to upload.", len(expansionFileEntries))
		for i, expansionFile := range expansionFileEntries {
			log.Debugf("%v - %v", i+1, expansionFile)
		}
	}
	return expansionFileEntries, nil
}
