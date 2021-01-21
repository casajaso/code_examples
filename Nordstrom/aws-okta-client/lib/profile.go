/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	"github.com/mitchellh/go-homedir"
	ini "gopkg.in/ini.v1"
)

var (
	prfPrefix string
	prf       *ini.Section
)

type File struct {
	file string
}

type Profile struct {
	ProfileData *ini.Section
	ProfileName string
}

type FileConfig struct {
	FileName string
	FilePath string
	FileData *ini.File
	Profile  *Profile
}

// NewFileFromEnv discovers/creates and loads aws config or credentials (determined by filename)
func NewProfileProvider(fileName string) (*FileConfig, error) {
	var file string
	if fileName == "config" {
		file = os.Getenv("AWS_CONFIG_FILE")
	} else if fileName == "credentials" {
		file = os.Getenv("AWS_SHARED_CREDENTIALS_FILE")
	}
	if file == "" {
		home, err := homedir.Dir()
		if err != nil {
			return nil, err
		}
		file = filepath.Join(home, "/.aws/", fileName)
		dir := filepath.Join(home, "/.aws")
		if _, err := os.Stat(file); os.IsNotExist(err) {
			err := os.MkdirAll(dir, os.ModePerm)
			if err != nil {
				return nil, fmt.Errorf("Unable to create: (%s)", err)
			}
			_, err = os.Create(file)
			if err != nil {
				return nil, fmt.Errorf("Unable to create: (%s)", err)
			}
		}
	}

	d, err := ini.Load(file) //load ini data from file
	if err != nil {
		fmt.Errorf("Error loading: (%s)", err)
	}
	d.BlockMode = false

	return &FileConfig{
		FileName: fileName,
		FileData: d,
		FilePath: file,
	}, nil
}

// Parse retrieves target profile ini data (and source profile if found)
func (c *FileConfig) Parse(profile string, reload bool) (*ini.Section, error) {
	if reload { //reload data-object from datasource (file)
		err := c.FileData.Reload()
		if err != nil {
			return nil, fmt.Errorf("Unable to reload: (%s)", err)
		}
	}

	if c.FileName == "config" { //add prefix `profile ` to profilename if config file
		prfPrefix = fmt.Sprintf("profile %s", profile)
	} else {
		prfPrefix = profile
	}

	if c.FileData.Section(prfPrefix) != nil { //load or create target or *source profile (*if present)
		p, err := c.FileData.GetSection(prfPrefix)
		if err != nil {
			return nil, fmt.Errorf("Unable to retrieve profile data: (%s)", err)
		}

		for _, key := range RemoveKeys {
			if p.HasKey(key) {
				p.DeleteKey(key)
			}
		}

		c.FileData.BlockMode = true
		err = c.FileData.SaveTo(c.FilePath)
		if err != nil {
			return nil, fmt.Errorf("Unable to commit changes for: (%s) (%s) ", c.Profile.ProfileData.Name(), err)
		}

		c.FileData.BlockMode = false

		prf = p

		//though aws-okta and aws-vault support similtanous use of credential_process and source_profile aws-cli does not -JCC
		//uncomment this section if/when that functionality is availible
		//s, err := SourceProfile(*p, *c.FileData)
		//if err != nil {
		//	return nil, fmt.Errorf("Unable to process source_profile: (%s)", err)
		//}
		//
		//if s != nil {
		//	prf = s
		//} else {
		//	prf = p
		//}

	} else {
		p, err := c.FileData.NewSection(prfPrefix)
		if err != nil {
			return nil, fmt.Errorf("Unable to create profile: (%s) ", err)
		}

		prf = p
	}

	for _, key := range RemoveKeys {
		if prf.HasKey(key) {
			prf.DeleteKey(key)
		}
	}

	c.Profile = &Profile{
		ProfileData: prf,
		ProfileName: prf.Name(),
	}

	c.FileData.BlockMode = true
	err := c.FileData.SaveTo(c.FilePath)
	if err != nil {
		return nil, fmt.Errorf("Unable to save to %s: (%s) ", c.Profile.ProfileData.Name(), err)
	}
	c.FileData.BlockMode = false

	return c.Profile.ProfileData, nil
}

// Save adds a single key/val pair to profile and commits objects to disk
func (c *FileConfig) Save(keyName string, keyValue string) error {
	if err := c.FileData.Reload(); err != nil {
		return fmt.Errorf("Unable to reload: (%s)", err)
	}

	c.FileData.Section(c.Profile.ProfileData.Name()).Key(keyName).SetValue(keyValue)

	c.FileData.BlockMode = true

	if err := c.FileData.SaveTo(c.FilePath); err != nil {
		return fmt.Errorf("Unable to save to %s: (%s) ", c.Profile.ProfileData.Name(), err)
	}

	c.FileData.BlockMode = false

	return nil
}

// Update adds an array of key/val pairs to profile and commits objects to disk
func (c *FileConfig) Update(keys map[string]string) error {

	sortedKeys := make([]string, 0, len(keys))

	for key := range keys {
		sortedKeys = append(sortedKeys, key)
	}

	sort.Strings(sortedKeys)

	for _, key := range sortedKeys {
		c.FileData.Section(c.Profile.ProfileData.Name()).Key(key).SetValue(keys[key])
		time.Sleep(time.Duration(1) * time.Second)
	}

	c.FileData.BlockMode = true

	if err := c.FileData.SaveTo(c.FilePath); err != nil {
		return fmt.Errorf("Unable to save to %s: (%s) ", c.Profile.ProfileData.Name(), err)
	}

	c.FileData.BlockMode = false

	return nil
}

// Clear deletes a profile section and recreates a blank one under the same name
func (c *FileConfig) Clear() error {
	profile := c.Profile.ProfileData.Name()
	path := c.FilePath
	err := c.FileData.Reload()
	if err != nil {
		return fmt.Errorf("Unable to reload: (%s)", err)
	}

	c.FileData.DeleteSection(profile)
	c.FileData.BlockMode = true
	err = c.FileData.SaveTo(path)
	if err != nil {
		return fmt.Errorf("Unable to clear current profile configuration: (%s) ", err)
	}
	c.FileData.BlockMode = true

	err = c.FileData.Reload()
	if err != nil {
		return fmt.Errorf("Unable to reload: (%s)", err)
	}

	prf, err := c.FileData.NewSection(profile)
	if err != nil {
		return fmt.Errorf("Unable to recreate profile: (%s) ", err)
	}

	c.FileData.BlockMode = true
	err = c.FileData.SaveTo(path)
	if err != nil {
		return fmt.Errorf("Unable to save to %s: (%s) ", c.Profile.ProfileData.Name(), err)
	}

	c.FileData.BlockMode = false

	c.Profile.ProfileData = prf

	return nil
}

// SourceProfile searches profile data for source_profile definition
func SourceProfile(profile ini.Section, data ini.File) (*ini.Section, error) {
	if profile.HasKey("source_profile") {
		s, err := profile.GetKey("source_profile")
		if err != nil {
			return nil, fmt.Errorf("Unable to retrieve source_profile name: (%s)", err)
		}

		if strings.Contains(profile.Name(), "profile") { //add prefix `profile ` to profilename if config file
			prfPrefix = fmt.Sprintf("profile %s", s.String())
		} else {
			prfPrefix = s.String()
		}

		if data.Section(prfPrefix) != nil {
			p, err := data.GetSection(prfPrefix)
			if err != nil {
				return nil, fmt.Errorf("Unable to reload: (%s)", err)
			}
			prf = p
		} else {
			p, err := data.NewSection(s.String())
			if err != nil {
				return nil, fmt.Errorf("Unable to create profile: (%s) ", err)
			}
			prf = p
		}

		return prf, nil
	}
	return nil, nil
}
