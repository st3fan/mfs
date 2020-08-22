// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/

package mfs

import (
	"encoding/binary"
	"errors"
	"io"
	"time"
)

const (
	logicalBlockSize    = 512
	maxFileNameLength   = 31 // Excluding the length byte
	maxVolumeNameLength = 27 // Exluding the length byte
)

// Volume represents an MFS volume.
type Volume struct {
	r                io.ReadSeeker
	allocationBlocks []int
	Name             string
	Files            []File
}

type volumeInformation struct {
	Signature                      uint16
	CreateDate                     uint32
	LastBackup                     uint32
	Attributes                     uint16
	NumberOfFiles                  uint16
	DirSt                          uint16
	BlLen                          uint16
	NumberOfAllocationBlocks       uint16
	SizeOfAllocationBlocks         uint32
	ClpSize                        uint32
	FirstAllocationBlockInBlockMap uint16
	NextUnusedFileNumber           uint32
	FreeBlocks                     uint16
	VolumeName                     [maxVolumeNameLength + 1]byte
}

type fileDirectoryEntry struct {
	Flags   byte
	Version byte
	UsrWds  [16]byte
	FlNum   uint32
	StBlk   uint16
	LgLen   uint32
	PyLen   uint32
	RStBlk  uint16
	RLgLen  uint32
	RPyLen  uint32
	CrDat   uint32
	MdDat   uint32
	Nam     [maxFileNameLength + 1]byte
}

// File represents a file on the disk
type File struct {
	Name           string
	Type           string
	Creator        string
	Created        time.Time
	Modified       time.Time
	directoryEntry fileDirectoryEntry
}

func readFileDirectoryEntry(r io.Reader) (fileDirectoryEntry, error) {
	return fileDirectoryEntry{}, nil // TODO
}

func pascalString(data []byte) string {
	length := int(data[0])
	if length == 0 {
		return ""
	}
	return string(data[1 : length+1])
}

// NewVolume creates a new volume.
func NewVolume(r io.ReadSeeker) (*Volume, error) {
	if _, err := r.Seek(1024, io.SeekStart); err != nil {
		return nil, err
	}

	vi := volumeInformation{}
	if err := binary.Read(r, binary.BigEndian, &vi); err != nil {
		return nil, err
	}

	if vi.Signature != 0xd2d7 {
		return nil, errors.New("Invalid volume signature")
	}

	//fmt.Printf("The disk <VideoWorks Disk 1.image> was created at <%v>", time.Unix(int64(vi.CreateDate)-2082844800, 0))

	// Read the volume allocation block map

	allocationBlocks := make([]int, vi.NumberOfAllocationBlocks)
	for i := 0; i < int(vi.NumberOfAllocationBlocks); i++ {
		allocationBlocks[i] = 0 // readAllocationBlock(i)
	}

	// Read the file directory

	if _, err := r.Seek(int64(vi.DirSt)*logicalBlockSize, io.SeekStart); err != nil {
		return nil, err
	}

	var files []File

	for i := 0; i < int(vi.NumberOfFiles); i++ {
		var fde fileDirectoryEntry
		if err := binary.Read(r, binary.BigEndian, &fde); err != nil {
			return nil, err
		}

		files = append(files, File{
			Name:           pascalString(fde.Nam[:]),
			Created:        time.Unix(int64(fde.CrDat)-2082844800, 0),
			Modified:       time.Unix(int64(fde.MdDat)-2082844800, 0),
			Type:           string(fde.UsrWds[0:4]),
			Creator:        string(fde.UsrWds[4:8]),
			directoryEntry: fde,
		})

		// If we do not have enough room left in the current logical block, jump to the next one.

		current, err := r.Seek(0, io.SeekCurrent)
		if err != nil {
			return nil, err
		}

		if (current+52)%logicalBlockSize < 52 {
			if _, err := r.Seek(logicalBlockSize-(current%logicalBlockSize), io.SeekCurrent); err != nil {
				return nil, err
			}
			continue
		}

		// To make parsing the directory easier, we cheat a bit and just read the max size
		// for the filename. So we have to jump back a bit based on the actual name length.

		// DeskTop = 7 -> We skipped 32 but should have skipped 8 -> move back 24 (32 - 1 - length)

		offset := int64(maxFileNameLength + 1 - 1 - fde.Nam[0])
		if fde.Nam[0]%2 == 0 {
			offset = offset - 1
		}

		if _, err := r.Seek(-offset, io.SeekCurrent); err != nil {
			return nil, err
		}
	}

	return &Volume{
		r:                r,
		allocationBlocks: allocationBlocks,
		Name:             pascalString(vi.VolumeName[:]),
		Files:            files,
	}, nil
}
