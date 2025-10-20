package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oschwald/maxminddb-golang"
)

var errStopWalk = fmt.Errorf("stop walk")

func FindMMDBAndMove(searchInPath string, moveToDir string, destFileName string) error {
	err := filepath.Walk(
		searchInPath,
		func(path string, info os.FileInfo, err error) error {
			if err != nil {
				return err
			}

			if path == searchInPath || info.Name() == destFileName {
				return nil
			}

			if filepath.Ext(info.Name()) == ".mmdb" {
				db, err := maxminddb.Open(path)
				if err != nil {
					return fmt.Errorf("error opening DB: %w", err)
				}
				defer db.Close()

				if db.Metadata.RecordSize > 0 {
					destPath := filepath.Join(moveToDir, destFileName)
					if err := os.Rename(path, destPath); err != nil {
						return fmt.Errorf("error moving file: %w", err)
					}
					return errStopWalk
				}
			}

			return nil
		},
	)

	if err == errStopWalk {
		return nil
	}

	return err
}
