package main

import (
	"errors"
	"fmt"
	"os"
	"path"
	"slices"
	"time"

	libcopy "github.com/otiai10/copy"
)

// inspiration taken from `time.RFC3339Nano`
const _FolderNameTimeFormat string = "2006-01-02T15_04_05.999999999Z07_00"

func takeSnapshotIfNeeded(sourceFolder string, snapshotFolder string, minSnapshotIntervalDays float64, maxSnapshots int64) error {
	now := time.Now()
	// fmt.Printf("now=%v\n", now)

	lastSnapshot, err := getLastSnapshot(snapshotFolder)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			lastSnapshot = time.Time{} // 0001-01-01 00:00:00 +0000 UTC
		} else {
			return err
		}
	}
	// fmt.Printf("lastSnapshot=%v\n", lastSnapshot)

	nextSnapshot := lastSnapshot.Add(time.Duration(minSnapshotIntervalDays * float64(24*time.Hour)))
	// fmt.Printf("nextSnapshot=%v\n", nextSnapshot)

	if now.Before(nextSnapshot) {
		fmt.Printf("Not yet time to take a snapshot\n")
		return nil
	} else {
		fmt.Printf("Time to take a snapshot\n")
	}

	err = takeSnapshot(sourceFolder, snapshotFolder, now)
	if err != nil {
		return err
	}

	err = collectGarbage(snapshotFolder, maxSnapshots)
	if err != nil {
		return err
	}

	fmt.Printf("All done\n")

	return nil
}

func getLastSnapshot(snapshotFolder string) (time.Time, error) {
	entries, err := os.ReadDir(snapshotFolder)
	if err != nil {
		return time.Time{}, err
	}

	latest := time.Time{}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		folder, err := time.Parse(_FolderNameTimeFormat, entry.Name())
		if err != nil {
			fmt.Printf("Could not parse date based on folder name: %v\n", err)
		}

		if latest.Before(folder) {
			latest = folder
		}
	}

	return latest, nil
}

func takeSnapshot(sourceFolder string, baseSnapshotFolder string, now time.Time) error {
	folder := now.Format(_FolderNameTimeFormat)
	folder = path.Join(baseSnapshotFolder, folder)

	err := os.Mkdir(folder, 0755)
	if err != nil {
		return err
	}

	// Even if this does fail, we won't delete the "corrupted" files
	return libcopy.Copy(
		sourceFolder,
		folder,
		libcopy.Options{
			OnSymlink: func(src string) libcopy.SymlinkAction {
				return libcopy.Shallow
			},
			OnDirExists: func(src, dest string) libcopy.DirExistsAction {
				fmt.Printf("Warning: Destination already exists, copy will not be performed: %v\n", dest)
				return libcopy.Untouchable
			},
			OnError: func(src, dest string, err error) error {
				if err != nil {
					fmt.Printf("Warning: Error detected: %v\n", err)
				}
				return nil
			},
			Sync:          true, // obviously this will decrease performance
			PreserveTimes: true,
			PreserveOwner: true,
			// NumOfWorkers
		},
	)
}

type _FolderName struct {
	name string
	date time.Time
}

func collectGarbage(snapshotFolder string, maxSnapshots int64) error {
	entries, err := os.ReadDir(snapshotFolder)
	if err != nil {
		return err
	}

	if int64(len(entries)) <= maxSnapshots {
		return nil
	}

	names := make([]_FolderName, 0, len(entries))

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		date, err := time.Parse(_FolderNameTimeFormat, entry.Name())
		if err != nil {
			fmt.Printf("Could not parse date based on folder name: %v\n", err)
			continue
		}

		names = append(names, _FolderName{name: entry.Name(), date: date})
	}

	snapshotsToDelete := int64(len(names)) - maxSnapshots

	if snapshotsToDelete <= 0 {
		return nil
	}

	fmt.Printf("Max number of snapshots reached\n")

	slices.SortFunc(
		names,
		func(a _FolderName, b _FolderName) int {
			if a.date.Before(b.date) {
				return -1
			}
			if a.date.After(b.date) {
				return 1
			}
			return 0
		},
	)

	atLeastOnceDeleteionFailure := false

	for idx := range snapshotsToDelete {
		name := names[idx].name
		fullPath := path.Join(snapshotFolder, name)

		fmt.Printf("Deleting: %v\n", fullPath)

		err = os.RemoveAll(fullPath)
		if err != nil {
			atLeastOnceDeleteionFailure = true
			fmt.Printf("Warning: Could not delete folder: %v\n", err)
		}
	}

	if atLeastOnceDeleteionFailure {
		return fmt.Errorf("Deletion failure")
	}

	return nil
}
