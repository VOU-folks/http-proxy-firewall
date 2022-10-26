package utils

import (
	"archive/tar"
	"compress/gzip"
	"io"
	"log"
	"os"
	"path/filepath"
)

func ExtractMMDBFromTarGz(gzipStream io.Reader, destDir string) {
	uncompressedStream, err := gzip.NewReader(gzipStream)
	if err != nil {
		log.Println("ExtractTarGz: NewReader failed")
	}

	tarReader := tar.NewReader(uncompressedStream)

	for {
		header, err := tarReader.Next()

		if err == io.EOF {
			break
		}

		if err != nil {
			log.Println("ExtractTarGz: Next() failed: %s", err.Error())
		}

		switch header.Typeflag {
		case tar.TypeReg:
			if filepath.Ext(header.Name) == ".mmdb" {
				outFile, err := os.Create(destDir + "/" + filepath.Base(header.Name))
				if err != nil {
					log.Println("ExtractTarGz: Create() failed: %s", err.Error())
				}
				if _, err := io.Copy(outFile, tarReader); err != nil {
					log.Println("ExtractTarGz: Copy() failed: %s", err.Error())
				}
				outFile.Close()
			}
		}
	}
}
