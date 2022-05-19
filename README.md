# fabric-builder-k8s

Proof of concept Fabric builder for Kubernetes

Advantages:
- prepublished chaincode images avoids compile issues at deploy time
- standard CI/CD pipelines can be used to publish chaincode images
- traceability of installed chaincode's implementation (demo uses Git commit hash as image tag)

The aim is for the builder to work as closely as possible with the [Fabric chaincode lifecycle](https://hyperledger-fabric.readthedocs.io/en/latest/chaincode_lifecycle.html) first, and then make sensible choices for deploying chaincode workloads using Kubernetes within those Fabric constraints.
The assumption being that there are more people with Kubernetes skills than are familiar with the inner workings of Fabric!

Status: it _should_ just about work now but there are a few issues to iron out (and tests to write) before it's properly usable!

## Usage

The k8s builder can be run in cluster using the `KUBERNETES_SERVICE_HOST` and `KUBERNETES_SERVICE_PORT` environment variables, or it can connect using a `KUBECONFIG_PATH` environment variable.

An optional `FABRIC_K8S_BUILDER_NAMESPACE` can be used to specify the namespace to deploy chaincode to.

If you are using the builder in a development environment and want to deploy chaincode images which have not been pushed to a registry, you will need to configure the builder to run in development mode.
Set the `FABRIC_K8S_BUILDER_DEV_MODE_TAG` environment variable to an image tag which the builder will use instead of the `digest` value specified in chaincode packages.
For example, use an `unstable` tag for local chaincode images.
**Warning:** Use this option with care since it causes the k8s builder to ignore the `digest` value in all chaincode packages!

A `CORE_PEER_ID` environment variable is also currently required.

External builders are configured in the `core.yaml` file, for example:

```
  externalBuilders:
    - name: k8s_builder
      path: /opt/hyperledger/k8s_builder
      propagateEnvironment:
        - CORE_PEER_ID
        - KUBERNETES_SERVICE_HOST
        - KUBERNETES_SERVICE_PORT
        - FABRIC_K8S_BUILDER_NAMESPACE
        - FABRIC_K8S_BUILDER_DEV_MODE_TAG
```

See [External Builders and Launchers](https://hyperledger-fabric.readthedocs.io/en/latest/cc_launcher.html) for details of Hyperledger Fabric builders.

There are addition docs with more detailed usage instructions for specific Fabric network deployments:

- [Kubernetes Test Network](docs/TEST_NETWORK_K8S.md)
- [Nano Test Network](docs/TEST_NETWORK_NANO.md)

## Chaincode Docker image

Unlike the traditional chaincode language support for Go, Java, and Node.js, the k8s builder *does not* build a chaincode Docker image using Docker-in-Docker.
Instead, a chaincode Docker image must be built and published before it can be used with the k8s builder.

The chaincode will have access to the following environment variables:

- CORE_CHAINCODE_ID_NAME
- CORE_PEER_ADDRESS
- CORE_PEER_TLS_ENABLED
- CORE_PEER_TLS_ROOTCERT_FILE
- CORE_TLS_CLIENT_KEY_PATH
- CORE_TLS_CLIENT_CERT_PATH
- CORE_TLS_CLIENT_KEY_FILE
- CORE_TLS_CLIENT_CERT_FILE
- CORE_PEER_LOCALMSPID

See [conga-nft-contract](https://github.com/hyperledgendary/conga-nft-contract) for an example project which publishes a chaincode image using GitHub Actions.

## Chaincode package

The k8s chaincode package file, which is installed by the `peer lifecycle chaincode install` command, must contain the Docker image name and digest of the chaincode being deployed.

[Fabric chaincode packages](https://hyperledger-fabric.readthedocs.io/en/latest/cc_launcher.html#chaincode-packages) are `.tgz` files which contain two files:

- metadata.json - the chaincode label and type
- code.tar.gz - source artifacts for the chaincode

To create a k8s chaincode package file, start by creating an `image.json` file.
For example,

```shell
cat << IMAGEJSON-EOF > image.json
{
  "name": "ghcr.io/hyperledgendary/conga-nft-contract",
  "digest": "sha256:b35962f000d26ad046d4102f22d70a1351692fc69a9ddead89dfa13aefb942a7"
}
IMAGEJSON-EOF
```

**Note:** the k8s chaincode package file uses digests because these are immutable, unlike tags.
The docker inspect command can be used to find the digest if required.

```
docker inspect --format='{{index .RepoDigests 0}}' ghcr.io/hyperledgendary/conga-nft-contract:0bee560018ea932ec4c7ec252134e2506ec6e797 | cut -d'@' -f2
```

If you are using the the k8s builder in a development environment by setting the `FABRIC_K8S_BUILDER_DEV_MODE_TAG` environment variable, you must set the `digest` to an empty string.

```shell
cat << IMAGEJSON-EOF > image.json
{
  "name": "conga-nft-contract",
  "digest": ""
}
IMAGEJSON-EOF
```

Next, create a `code.tar.gz` archive containing the `image.json` file.

```shell
tar -czf code.tar.gz image.json
```

Create a `metadata.json` file for the chaincode package.
For example,

```shell
cat << METADATAJSON-EOF > metadata.json
{
    "type": "k8s",
    "label": "conga-nft-contract"
}
METADATAJSON-EOF
```

Create the final chaincode package archive.

```shell
tar -czf conga-nft-contract.tgz metadata.json code.tar.gz
```

Ideally the chaincode package should be created in the same CI/CD pipeline which builds the docker image.
There is an example [package-k8s-chaincode-action](https://github.com/hyperledgendary/package-k8s-chaincode-action) GitHub Action which can create the required k8s chaincode package.

The GitHub Action repository includes a basic shell script which can also be used for automating the process above outside GitHub workflows.
For example, to create a basic k8s chaincode package using the `pkgk8scc.sh` helper script.

```shell
curl -fsSL https://raw.githubusercontent.com/hyperledgendary/package-k8s-chaincode-action/main/pkgk8scc.sh -o pkgk8scc.sh && chmod u+x pkgk8scc.sh
./pkgk8scc.sh -l conga-nft-contract -n ghcr.io/hyperledgendary/conga-nft-contract -d sha256:b39eb624e9cc7ed3fa70bf7ea27721e266ae56b48992a916165af3a6b2a4f6eb
```

## Chaincode deploy

Deploy the chaincode package as usual, starting by installing the k8s chaincode package.

```shell
peer lifecycle chaincode install conga-nft-contract.tgz
```

You can also user the `peer` command to get the chaincode package ID.

```shell
export PACKAGE_ID=$(peer lifecycle chaincode calculatepackageid conga-nft-contract.tgz) && echo $PACKAGE_ID
```
