// SPDX-License-Identifier: BUSL-1.1

//go:build !windows

package authstate

import "os"

// SyncDirectoryForTest exposes the post-rename durability boundary.
func SyncDirectoryForTest(directory *os.File) error { return syncDirectory(directory) }

// WriteRecordForTest exposes write/sync failure handling without fault switches.
func WriteRecordForTest(file *os.File, record []byte) error { return writeRecord(file, record) }

// PublishWriteFailureForTest fails the actual publisher's private temporary file.
func PublishWriteFailureForTest(directory string, record []byte) error {
	return publish(directory, record, func(file *os.File, data []byte) error {
		if err := file.Close(); err != nil {
			return err
		}
		return writeRecord(file, data)
	})
}
