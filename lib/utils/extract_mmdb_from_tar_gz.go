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
		return fmt.Errorf("ExtractTarGz: NewReader failed: %w", err)
	}
	defer uncompressedStream.Close()

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			return fmt.Errorf("ExtractTarGz: Next() failed: %w", err)
		}

		if header.Typeflag == tar.TypeReg && filepath.Ext(header.Name) == ".mmdb" {
			outPath := filepath.Join(destDir, filepath.Base(header.Name))
			outFile, err := os.Create(outPath)
			if err != nil {
				return fmt.Errorf("ExtractTarGz: Create() failed: %w", err)
			}

			if _, err := io.Copy(outFile, tarReader); err != nil {
				outFile.Close()
				return fmt.Errorf("ExtractTarGz: Copy() failed: %w", err)
			}

			if err := outFile.Close(); err != nil {
				return fmt.Errorf("ExtractTarGz: Close() failed: %w", err)
			}
		}
	}

	return nil
}
