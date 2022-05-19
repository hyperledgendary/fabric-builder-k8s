// SPDX-License-Identifier: Apache-2.0

package builder

import (
	"fmt"
	"os"
	"path/filepath"

	"github.com/hyperledgendary/fabric-builder-k8s/internal/util"
	"github.com/otiai10/copy"
)

type Build struct {
	ChaincodeSourceDirectory   string
	ChaincodeMetadataDirectory string
	BuildOutputDirectory       string
	DevModeTag                 string
}

func (b *Build) Run() error {
	imageSrcPath := filepath.Join(b.ChaincodeSourceDirectory, "image.json")

	imageData, err := util.ReadImageJson(imageSrcPath)
	if err != nil {
		return fmt.Errorf("unable to read image.json: %w", err)
	}

	if b.DevModeTag != "" && imageData.Digest != "" {
		return fmt.Errorf("image digest not allowed in development mode: %s", imageData.Digest)
	}

	imageDestPath := filepath.Join(b.BuildOutputDirectory, "image.json")
	err = copy.Copy(imageSrcPath, imageDestPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error copying %s to %s: %s\n", imageSrcPath, imageDestPath, err)
		return err
	}

	// TODO copy any META-INF
	// metainfSrcPath := filepath.Join(b.ChaincodeSourceDirectory, "META-INF")
	// metainfDestPath := filepath.Join(b.BuildOutputDirectory, "META-INF")

	return nil
}
