package utils

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/oschwald/maxminddb-golang"
)

func FindMMDBAndMove(searchInPath string, moveToDir string, destFileName string) error {
	done := false
	_ = filepath.Walk(
		searchInPath,
		func(path string, info os.FileInfo, err error) error {
			if done {
				return nil
			}

			if path == searchInPath || info.Name() == destFileName {
				return nil
			}

			if filepath.Ext(info.Name()) == ".mmdb" {
				db, err := maxminddb.Open(path)
				if err != nil {
					return fmt.Errorf("Error opening DB: %s", err.Error())
				}

				if db.Metadata.RecordSize > 0 {
					err = os.Rename(path, moveToDir+"/"+destFileName)
					if err == nil {
						done = true
					}
					return nil
				}
			}

			return nil
		},
	)

	return nil
}
