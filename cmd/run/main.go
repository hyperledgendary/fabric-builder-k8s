// SPDX-License-Identifier: Apache-2.0

package main

import (
	"fmt"
	"os"
	"sync"

	"github.com/hyperledgendary/fabric-builder-k8s/internal/builder"
	"github.com/hyperledgendary/fabric-builder-k8s/internal/util"
)

func main() {
	if len(os.Args) != 3 {
		fmt.Fprintln(os.Stderr, "Expected BUILD_OUTPUT_DIR and RUN_METADATA_DIR arguments")
		os.Exit(1)
	}

	peerID, err := util.GetRequiredEnv(util.PeerIdVariable)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Expected %s environment variable\n", util.PeerIdVariable)
		os.Exit(1)
	}

	kubeconfigPath := util.GetOptionalEnv(util.KubeconfigPathVariable, "")

	kubeNamespace := util.GetOptionalEnv(util.ChaincodeNamespaceVariable, "")
	if kubeNamespace == "" {
		kubeNamespace, err = util.GetKubeNamespace()
		if err != nil {
			kubeNamespace = "default"
		}
	}

	devModeTag := util.GetOptionalEnv(util.DevModeTag, "")

	run := &builder.Run{
		BuildOutputDirectory: os.Args[1],
		RunMetadataDirectory: os.Args[2],
		PeerID:               peerID,
		KubeconfigPath:       kubeconfigPath,
		KubeNamespace:        kubeNamespace,
		DevModeTag:           devModeTag,
	}

	if err := run.Run(); err != nil {
		// TODO better error handling?
		fmt.Fprintf(os.Stderr, "Error running chaincode.\nBuild dir: %s\nRun dir: %s\nError: %v\n", run.BuildOutputDirectory, run.RunMetadataDirectory, err)
		os.Exit(1)
	}

	// TODO nasty hack to keep chaincode running- peer assumes chaincode has terminated when builder run terminates!
	var m sync.Mutex
	m.Lock()
	m.Lock()
}
