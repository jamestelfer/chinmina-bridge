# Verifying releases with `cosign`

Releases [are signed](./releases.md) with `cosign` as part of the release
process. The build produces additional attestation bundles during this process,
which can be used to verify both binaries and Docker images.

For binaries, bundles are present in the `tar.gz` archive created by the
release. For images, bundles are stored in the OCI registry alongside the image
itself.

## Obtaining `cosign`

Download from the [`sigstore/cosign` project on GitHub][cosign-download], and
[verify the release][cosign-verify] as you prefer.

[cosign-download]: https://github.com/sigstore/cosign?tab=readme-ov-file#installation
[cosign-verify]: https://docs.sigstore.dev/cosign/system_config/installation/#verifying-cosign-releases

## Release identity

The certificates issued by the release are issued for the GitHub Actions OIDC
provider, and the identity is the executed workflow, referenced by the Git tag
being built.

| Field | Format |
|-|-|
| Issuer | `https://token.actions.githubusercontent.com` |
| Identity | `https://github.com/jamestelfer/chinmina-bridge/.github/workflows/release.yaml@refs/tags/<tag name>` |

> [!IMPORTANT]
> **Git tags are not static:** they can be updated to point to a different
> commit SHA. Examine the recorded claims for the exact commit.
>
> There are claims recorded for the exact commit of both the workflow that
> produced the artifact and the commit that the artifact source was built from.

## Verifying an image release

Images are published to Docker Hub in the `chinmina` repository. The images are
named `chinmina-bridge` and are labelled with their release tag (`vX.Y.Z`).

An image can be verified with the following `cosign` command:

```shell
TAG=vX.Y.Z \
cosign verify "chinmina/chinmina-bridge:$TAG" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  --certificate-identity "https://github.com/jamestelfer/chinmina-bridge/.github/workflows/release.yaml@refs/tags/$TAG" \
  --output text

# more details are available if you use JSON output:
TAG=vX.Y.Z \
cosign verify "chinmina/chinmina-bridge:$TAG" \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  --certificate-identity "https://github.com/jamestelfer/chinmina-bridge/.github/workflows/release.yaml@refs/tags/$TAG" \
  --output json | jq
```

The path `.[].optional.Bundle.Payload.logIndex` is the index entry in the public
transparency log, recording the details of the signing event. The details of the
event can be found at: https://search.sigstore.dev/.

## Verifying the binary releases

Download and extract the `tar.gz` of the binary you're interested in. The
artifacts present include both the binary itself (named `chinmina-bridge`) and
the signing bundle (`chinmina-bridge.cosign.bundle`).

```shell
# declare the release details for download
TAG=vX.Y.Z
ARCH=arm64

# download the binary
curl -L -o chinmina-bridge_linux_${ARCH}.tar.gz \
  https://github.com/jamestelfer/chinmina-bridge/releases/download/${TAG}/chinmina-bridge_linux_${ARCH}.tar.gz

# extract to the current directory
tar xvf chinmina-bridge_linux_${ARCH}.tar.gz

# verify
cosign verify-blob \
  chinmina-bridge \
  --bundle chinmina-bridge.cosign.bundle \
  --certificate-oidc-issuer 'https://token.actions.githubusercontent.com' \
  --certificate-identity "https://github.com/jamestelfer/chinmina-bridge/.github/workflows/release.yaml@refs/tags/$TAG"

# peek the details
jq -r '.rekorBundle.Payload.logIndex | "https://search.sigstore.dev/?logIndex=\(.)"' < chinmina-bridge.cosign.bundle

# open the URL that is shown
```
