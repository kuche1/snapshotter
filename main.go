package main

import (
	"flag"
	"fmt"
	"os"
)

func main() {
	sourceFolder := flag.String("source-folder", "", "Folder to take snapshots of")
	snapshotFolder := flag.String("snapshot-folder", "", "Folder to keep snapshots in")
	minSnapshotIntervalDays := flag.Float64("min-snapshot-interval-days", 0.9, "How often at most would snapshots get taken") // Need to check `flag.Duration`
	maxSnapshots := flag.Int64("max-snapshots", 30, "Maximum number of snapshots to keep")
	flag.Parse()

	if *sourceFolder == "" {
		fmt.Printf("You need to specify the source folder\n")
		os.Exit(1)
	}

	if *snapshotFolder == "" {
		fmt.Printf("You need to specify the snapshot folder\n")
		os.Exit(1)
	}

	err := takeSnapshotIfNeeded(
		*sourceFolder,
		*snapshotFolder,
		*minSnapshotIntervalDays,
		*maxSnapshots,
	)

	if err != nil {
		fmt.Printf("Error: %v\n", err)
		os.Exit(1)
	}
}
