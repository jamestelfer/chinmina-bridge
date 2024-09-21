# Release process

In short:

1. Releases are triggered by creating a release tag from `main`. This is currently manual.
2. Release tags conform to semantic versioning
3. Commits use conventional commit messages to aid in the changelog creation process
4. A GoReleaser pipeline is used to create the artifacts
5. All artifacts (binaries and images) are signed by the build process using `cosign`

## When is a release ready?

Releases are created on an as-needed basis. We prefer multiple, smaller releases over releases that have a greater number of changes.

A release is ready when:

- there are committed changes on `main`, and
- there is confidence in its stability.

Stability is a pre-requisite for merging, so there should not be significant questions about the appropriateness of a `main` release.

## Triggering a release

Releases are triggered via the creation of a semantic-versioned tag, in the format `vX.Y.Z`. Creation of a tag in this format triggers the automated release process.

Only repository administrators may create a tag in this format.

## Release signing

The [Sigstore][sigstore] ecosystem is leveraged for signing executable release outputs. ([Docs][sigstore-docs].)

- [`cosign`][cosign] is used as the signing CLI tool
- The [`fulcio`][fulcio] public-good instance is used for ephemeral signing certificates
- The [`rekor`][rekor] [public-good instance][rekor-search] is used for Certificate Transparency record publishing.

The signing process allows some useful attributes of the binaries to be verified:

- the provider of the identity for the build process (i.e. GitHub Actions)
- the build process that was used to generate them (both scripts and compute)
- the Git reference of the code that was used to build the binary

Releases are signed with `cosign`, with transparency records published to the [public-good Rekor instance].

[sigstore]: https://www.sigstore.dev/
[sigstore-docs]: https://docs.sigstore.dev/
[cosign]: https://github.com/sigstore/cosign?tab=readme-ov-file
[fulcio]: https://github.com/sigstore/fulcio?tab=readme-ov-file
[rekor]: https://github.com/sigstore/rekor?tab=readme-ov-file
[rekor-search]: https://search.sigstore.dev/

## Testing the release process

It is possible to run GoReleaser locally to test some of the release proceses.
(`goreleaser` must be available.)

```shell
# from the root of the local working copy
goreleaser release --clean --verbose --skip "announce,validate"
```

This will run the binary and image builds, and publish a temporary image to
[`ttl.sh`](https://ttl.sh/). Temporary images can be used in local testing with
`docker compose`.

Some processes are skipped when doing this:

- binary signing
- image signing
- changelog generation
- GitHub release creation

Thus release testing verifies a proportion of the GoReleaser configuration, and allows the image/binary builds to be integration tested.
