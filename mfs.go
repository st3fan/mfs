// This Source Code Form is subject to the terms of the Mozilla Public
// License, v. 2.0. If a copy of the MPL was not distributed with this
// file, You can obtain one at http://mozilla.org/MPL/2.0/

package mfs

import (
	"bytes"
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

// Volume represents an MFS volume.
type Volume struct {
	r                io.ReadSeeker
	allocationBlocks []uint16
	Name             string
	Files            []File
	vi               volumeInformation
}

// File represents a file on the disk
type File struct {
	Name               string
	Type               string
	Creator            string
	Created            time.Time
	Modified           time.Time
	DataForkLength     int64
	ResourceForkLength int64
	directoryEntry     fileDirectoryEntry
}

func pascalString(data []byte) string {
	length := int(data[0])
	if length == 0 {
		return ""
	}
	return string(data[1 : length+1])
}

func readByte(r io.Reader) (byte, error) {
	buf := make([]byte, 1)
	if _, err := r.Read(buf); err != nil {
		return 0, err
	}
	return buf[0], nil
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

	// Read the volume allocation block map

	var t byte = 0

	allocationBlocks := make([]uint16, vi.NumberOfAllocationBlocks)
	for i := 0; i < int(vi.NumberOfAllocationBlocks); i++ {
		if i%2 == 0 {
			a, err := readByte(r)
			if err != nil {
				return nil, err
			}

			b, err := readByte(r)
			if err != nil {
				return nil, err
			}

			t = b

			allocationBlocks[i] = (uint16(a) << 4) | (uint16(b) >> 4)
		} else {
			a, err := readByte(r)
			if err != nil {
				return nil, err
			}

			allocationBlocks[i] = (uint16(t&0x0f) << 8) | uint16(a)
		}
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
			Name:               pascalString(fde.Nam[:]),
			Created:            time.Unix(int64(fde.CrDat)-2082844800, 0),
			Modified:           time.Unix(int64(fde.MdDat)-2082844800, 0),
			Type:               string(fde.UsrWds[0:4]),
			Creator:            string(fde.UsrWds[4:8]),
			DataForkLength:     int64(fde.LgLen),
			ResourceForkLength: int64(fde.RLgLen),
			directoryEntry:     fde,
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
		vi:               vi,
	}, nil
}

func (volume *Volume) readAllocationBlock(allocationBlockIndex uint16) ([]byte, error) {
	//log.Printf("Reading allocation block %v", allocationBlockIndex)

	buffer := make([]byte, volume.vi.SizeOfAllocationBlocks)

	offset := int64(volume.vi.DirSt+volume.vi.BlLen) * logicalBlockSize
	offset += int64(allocationBlockIndex) * int64(volume.vi.SizeOfAllocationBlocks)

	if _, err := volume.r.Seek(offset, io.SeekStart); err != nil {
		return nil, err
	}

	if _, err := volume.r.Read(buffer); err != nil {
		return nil, err
	}

	return buffer, nil
}

func (volume *Volume) bytesReader(allocationBlockIndex uint16, length uint32) (io.Reader, error) {
	data := []byte{}

	if length != 0 {
		allocationBlockData, err := volume.readAllocationBlock(allocationBlockIndex)
		if err != nil {
			return nil, err
		}

		data = append(data, allocationBlockData...)
		allocationBlockIndex = volume.allocationBlocks[allocationBlockIndex-2]

		for allocationBlockIndex != 1 {
			allocationBlockData, err := volume.readAllocationBlock(allocationBlockIndex)
			if err != nil {
				return nil, err
			}

			data = append(data, allocationBlockData...)
			allocationBlockIndex = volume.allocationBlocks[allocationBlockIndex-2]
		}
	}

	return bytes.NewReader(data[0:length]), nil
}

// OpenDataFork returns a io.Reader for the file with the given index
func (volume *Volume) OpenDataFork(fileIndex int) (io.Reader, error) {
	file := volume.Files[fileIndex]
	return volume.bytesReader(file.directoryEntry.StBlk, file.directoryEntry.LgLen)
}

// OpenResourceFork returns a io.Reader for the file with the given index
func (volume *Volume) OpenResourceFork(fileIndex int) (io.Reader, error) {
	file := volume.Files[fileIndex]
	return volume.bytesReader(file.directoryEntry.RStBlk, file.directoryEntry.RLgLen)
}
