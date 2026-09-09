// Command history creates an isolated run-history fixture for browser acceptance.
package main

import (
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"

	"github.com/MustardSeedNetworks/niac-go/internal/storage"
)

func main() {
	const argumentCount = 2
	if len(os.Args) != argumentCount {
		log.Fatal("usage: history <new database path>")
	}
	path := filepath.Clean(os.Args[1])
	if _, err := os.Stat(path); !os.IsNotExist(err) {
		log.Fatal("history fixture requires a new database")
	}
	if err := seedHistory(path); err != nil {
		log.Fatal(err)
	}
}

func seedHistory(path string) error {
	store, err := storage.Open(path, storage.RetainAll)
	if err != nil {
		return fmt.Errorf("open fixture: %w", err)
	}
	defer store.Close()
	for index := range 40 {
		record := storage.RunRecord{
			ConfigName: fmt.Sprintf("history-%02d", index+1),
			StartedAt:  time.Now(), Duration: time.Second, Interface: "e2e-dry-run0",
		}
		if err = store.AddRun(record); err != nil {
			return fmt.Errorf("seed fixture: %w", err)
		}
	}
	return nil
}
