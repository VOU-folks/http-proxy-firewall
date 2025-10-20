package utils

import (
	"archive/tar"
	"compress/gzip"
	"fmt"
	"io"
	"os"
	"path/filepath"
)

func ExtractMMDBFromTarGz(gzipStream io.Reader, destDir string) error {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		return fmt.Errorf("ExtractTarGz: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("ExtractTarGz: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if filepath.Ext(header.Name) == ".mmdb" {
				outFile, err := os.Create(destDir + "/" + filepath.Base(header.Name))
				if err != nil {
					return fmt.Errorf("ExtractTarGz: Create() failed: %s", err.Error())
				}
				if _, err := io.Copy(outFile, tarReader); err != nil {
					return fmt.Errorf("ExtractTarGz: Copy() failed: %s", err.Error())
				}

				if outFile != nil {
					_ = outFile.Close()
				}
			}
		}
	}

	return nil
}
