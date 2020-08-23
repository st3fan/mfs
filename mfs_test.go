// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/

package mfs_test

import (
	"io/ioutil"
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
}

func Test_OpenDataFork(t *testing.T) {
	volume, err := volumeFromPath("testdata/VideoWorks Disk 1.image")
	if err != nil {
		t.Error("Could not open volume:", err)
	}

	r, err := volume.OpenDataFork(3)
	if err != nil {
		t.Error("Could not open data fork:", err)
	}

	contents, err := ioutil.ReadAll(r)
	if err != nil {
		t.Error("Could not read all:", err)
	}

	if len(contents) != int(volume.Files[3].DataForkLength) {
		t.Errorf("Did not read all: expected %d got %d", volume.Files[3].DataForkLength, len(contents))
	}
}

func Test_OpenResourceFork(t *testing.T) {
	volume, err := volumeFromPath("testdata/VideoWorks Disk 1.image")
	if err != nil {
		t.Error("Could not open volume:", err)
	}

	r, err := volume.OpenResourceFork(3)
	if err != nil {
		t.Error("Could not open data fork:", err)
	}

	contents, err := ioutil.ReadAll(r)
	if err != nil {
		t.Error("Could not read all:", err)
	}

	if len(contents) != int(volume.Files[3].ResourceForkLength) {
		t.Errorf("Did not read all: expected %d got %d", volume.Files[3].ResourceForkLength, len(contents))
	}
}

func TestReadAllFiles(t *testing.T) {
	volume, err := volumeFromPath("testdata/VideoWorks Disk 1.image")
	if err != nil {
		t.Error("Could not open volume:", err)
	}

	for fileIndex := range volume.Files {
		r, err := volume.OpenDataFork(fileIndex)
		if err != nil {
			t.Error("Could not open data fork:", err)
		}

		contents, err := ioutil.ReadAll(r)
		if err != nil {
			t.Error("Could not read all:", err)
		}

		if len(contents) != int(volume.Files[fileIndex].DataForkLength) {
			t.Errorf("Did not read all: expected %d got %d", volume.Files[fileIndex].DataForkLength, len(contents))
		}
	}

	for fileIndex := range volume.Files {
		r, err := volume.OpenResourceFork(fileIndex)
		if err != nil {
			t.Error("Could not open data fork:", err)
		}

		contents, err := ioutil.ReadAll(r)
		if err != nil {
			t.Error("Could not read all:", err)
		}

		if len(contents) != int(volume.Files[fileIndex].ResourceForkLength) {
			t.Errorf("Did not read all: expected %d got %d", volume.Files[fileIndex].ResourceForkLength, len(contents))
		}
	}
}
