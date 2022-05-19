// SPDX-License-Identifier: Apache-2.0

package util

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
)

// ChaincodeJson represents the chaincode.json file that is supplied by Fabric in
// the RUN_METADATA_DIR
type ChaincodeJson struct {
	ChaincodeID string `json:"chaincode_id"`
	PeerAddress string `json:"peer_address"`
	ClientCert  string `json:"client_cert"`
	ClientKey   string `json:"client_key"`
	RootCert    string `json:"root_cert"`
	MspID       string `json:"mspid"`
}

// ImageJson represents the image.json file in the k8s chaincode package
type ImageJson struct {
	Name   string `json:"name"`
	Digest string `json:"digest"`
}

func ReadImageJson(imageJsonPath string) (*ImageJson, error) {
	fmt.Println("Reading image.json...")
	_, err := os.Stat(imageJsonPath)
	if err != nil {
		return nil, fmt.Errorf("unable to access image.json: %w", err)
	}

	imageJsonContents, err := ioutil.ReadFile(imageJsonPath)
	if err != nil {
		return nil, fmt.Errorf("unable to read image.json: %w", err)
	}

	var imageData ImageJson
	if err := json.Unmarshal(imageJsonContents, &imageData); err != nil {
		return nil, fmt.Errorf("unable to process image.json: %w", err)
	}

	fmt.Fprintf(os.Stdout, "Image name: %s\nImage digest: %s\n", imageData.Name, imageData.Digest)

	return &imageData, nil
}
