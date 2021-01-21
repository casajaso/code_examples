/*
AWS Okta -- https://gitlab.nordstrom.com/public-cloud/aws-okta
Maintained by Cloud Engineering <cloudengineering@nordstrom.com>
	Author: Jason Casas

Copyright 2020 @ Nordstrom, Inc. All rights reserved.
*/

package lib

import (
	"fmt"
	"io"

	"io/ioutil"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"runtime"
	"strings"

	"github.com/dustin/go-humanize"
	"github.com/tcnksm/go-latest"
)

var (
	//WarnDevBuild
	Development = func(u *UpdatePerams) string {
		return fmt.Sprintf("You are running a non-production build\nUpdate to the latest recommended production build by running: \n\t`%s update --install`",
			u.UpdtVers,
			u.ExecName)
	}
	//WarnOutdated
	Outdated = func(u *UpdatePerams) string {
		return fmt.Sprintf("Update availible: (%#v) \n\t\tto download the latest update (all OS): \t`%s update --download`\n\t\tto install the latest update (MacOS/Linux): \t`%s update --install`",
			u.UpdtVers,
			u.ExecName,
			u.ExecName,
		)
	}
	//Current
	Current = func(u *UpdatePerams) string {
		return fmt.Sprintf("You are running: (%v) which is the latest release\nIf you are experiancing issues See the FAQ: \n\t(%v)",
			u.ExecVers,
			u.FAQ.String())
	}
)

//WriteCounter object for displaying download status
type WriteCounter struct {
	Total uint64
}

//UpdatePerams update perams object
type UpdatePerams struct {
	UpdtVers string
	ExecVers string
	ExecMeta string
	ExecName string
	ExecPath string
	Temp     *os.File
	FAQ      *url.URL
	Latest   *url.URL
	Download *url.URL
	Platform string
}

//UpdateHandler - constructs update perams
func UpdateHandler(version string) (*UpdatePerams, error) {

	upd := &UpdatePerams{}
	upd.ExecVers = version
	upd.Platform = Platform
	upd.Latest = EPS["shared"]["latest"]
	upd.Download = EPS[runtime.GOOS]["download"]
	upd.FAQ = EPS["shared"]["faq"]

	ep, err := GetExecPath(os.Args[0])
	if err != nil {
		return nil, fmt.Errorf("Unable to determine path: (%s)", err)
	}

	upd.ExecPath = ep
	upd.ExecName = filepath.Base(ep)

	return upd, nil
}

//Check get/compare current...latest
func (u *UpdatePerams) Check() (string, bool, error) {
	upd := u.Latest.String()
	json := &latest.JSON{
		URL: upd,
	}

	res, err := latest.Check(json, u.ExecVers)
	if err != nil {
		return "", false, fmt.Errorf("Failed to get response from update server: %s (%s) ", u.Latest.String(), err)
	}

	u.UpdtVers = res.Meta.Message

	log.Infof("Version info - current: (%s), latest: (%s)", u.ExecVers, res.Current)

	if strings.Contains(u.ExecVers, "-dev") {
		resp := Development(u)
		return resp, true, nil
	}

	if res.Outdated {
		u.ExecVers = res.Current
		resp := (fmt.Sprintf(Outdated(u)))
		return resp, true, nil
	}

	resp := Current(u)

	return resp, false, nil
}

//Get downloads latest production release and stages temp file for update
func (u *UpdatePerams) Get() error {
	tf := u.ExecName + "-upd-staging-"
	t, err := ioutil.TempFile(os.TempDir(), tf)
	if err != nil {
		return fmt.Errorf("Unable to create: (%s)", err)
	}
	u.Temp = t
	fmt.Fprintln(os.Stdout, "Downloading: ", u.Download.String())
	defer u.Temp.Close()
	resp, err := http.Get(u.Download.String())
	if err != nil {
		return fmt.Errorf("Unable to retrieve update: (%s)", err)
	}
	defer resp.Body.Close()
	count := &WriteCounter{}
	_, err = io.Copy(u.Temp, io.TeeReader(resp.Body, count))
	if err != nil {
		return fmt.Errorf("Unable to retrieve update: (%s)", err)
	}
	fmt.Println()
	return nil
}

//Install updates(overwrites) binary "in-place"
func (u *UpdatePerams) Install() error {
	err := os.Rename(u.Temp.Name(), u.ExecPath)
	if err != nil {
		return fmt.Errorf("Unable to install update-in-place: (%s)", err)
	}
	err = os.Chmod(u.ExecPath, 0755)
	if err != nil {
		return fmt.Errorf("Failed to set permissions: [%v] (%v)", u.ExecPath, err)
	}
	return nil
}

// Save downloads update to temp file
func (u *UpdatePerams) Save() error {
	var fn string
	if Platform == "windows" {
		fn = fmt.Sprintf("%s.upd", u.ExecPath)
	} else {
		fn = fmt.Sprintf("%s.upd", u.ExecPath)
	}
	err := os.Rename(u.Temp.Name(), fn)
	if err != nil {
		return fmt.Errorf("Unable to complete update-in-place: (%s)", err)
	}
	err = os.Chmod(u.ExecPath, 0755)
	if err != nil {
		return fmt.Errorf("Failed to set permissions: [%v] (%v)", u.ExecPath, err)
	}
	fmt.Fprintf(os.Stdout, "Download saved as: (%s) move or delete: (%s) and remove `.upd` extention from (%s) to complete installation",
		fn,
		u.ExecPath,
		fn,
	)
	return nil
}

// PrintProgress prints download progress
func (wc *WriteCounter) PrintProgress() {
	fmt.Printf("\r%v", strings.Repeat(" ", 50))
	fmt.Printf("\rDownloading... %v complete", humanize.Bytes(wc.Total))
}

//Write io.Writer obj interface
func (wc *WriteCounter) Write(p []byte) (int, error) {
	n := len(p)
	wc.Total += uint64(n)
	wc.PrintProgress()
	return n, nil
}
