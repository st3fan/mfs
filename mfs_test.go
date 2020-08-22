// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/

package mfs_test

import (
	"fmt"
	"os"
	"testing"

	"github.com/st3fan/diskcopy"
	"github.com/st3fan/mfs"
)

func volumeFromPath(path string) (*mfs.Volume, error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}

	image, err := diskcopy.NewImage(file)
	if err != nil {
		return nil, err
	}

	return mfs.NewVolume(image)
}

func Test_NewVolume(t *testing.T) {
	if _, err := volumeFromPath("testdata/VideoWorks Disk 1.image"); err != nil {
		t.Fail()
	}
}

func Test_VolumeName(t *testing.T) {
	volume, err := volumeFromPath("testdata/VideoWorks Disk 1.image")
	if err != nil {
		t.Fail()
	}

	if volume.Name != "VideoWorks Disk 1" {
		t.Fail()
	}
}

func Test_Files(t *testing.T) {
	volume, err := volumeFromPath("testdata/VideoWorks Disk 1.image")
	if err != nil {
		t.Fail()
	}

	if len(volume.Files) != 12 {
		t.Fail()
	}

	for _, file := range volume.Files {
		fmt.Printf("%s %s %s:%-32s %v\n", file.Type, file.Creator, volume.Name, file.Name, file.Created)
	}
}
