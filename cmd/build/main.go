// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"

	"github.com/hyperledgendary/fabric-builder-k8s/internal/builder"
	"github.com/hyperledgendary/fabric-builder-k8s/internal/util"
)

func main() {
	if len(os.Args) != 4 {
		fmt.Fprintln(os.Stderr, "Expected CHAINCODE_SOURCE_DIR, CHAINCODE_METADATA_DIR and BUILD_OUTPUT_DIR arguments")
		os.Exit(1)
	}

	devModeTag := util.GetOptionalEnv(util.DevModeTag, "")

	build := &builder.Build{
		ChaincodeSourceDirectory:   os.Args[1],
		ChaincodeMetadataDirectory: os.Args[2],
		BuildOutputDirectory:       os.Args[3],
		DevModeTag:                 devModeTag,
	}

	if err := build.Run(); err != nil {
		// TODO better error handling?
		fmt.Fprintf(os.Stderr, "Error building chaincode.\nSource dir: %s\nMetadata dir: %s\nOutput dir: %s\nError: %v\n", build.ChaincodeSourceDirectory, build.ChaincodeMetadataDirectory, build.BuildOutputDirectory, err)
		os.Exit(1)
	}
}
