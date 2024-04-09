# Buildkite/Github OIDC bridge 

Exposes an endpoint that allows Buildkite Agents to use an OIDC token to request
a Github token in return. The GitHub token is created for the agent using a
Github application that is created for the bridge and configured to allow access
to the implementor's GitHub organization.

The token is created with `contents:read` permissions on the repository
associated with the executing pipeline.

## Configuration

- Github application private key
  - refinement: can this stay in KMS perhaps?

- Buildkite
  - API token (what scopes?) choose organization, REST `read_pipelines`, no GraphQL
  - JWT verify
    - audience
    - issuer must be buildkite
    - JWKS url <- allows issuance from another source for testing
    - clock skew

## Required functionality

* stdout audit log:
  * JSON: repo, permissions, generated_at, issued_at, pipeline_slug,build_id, step_id

* caching of token: sliding window, perhaps configure minimum lifetime in minutes?
* cache lookups from buildkite of repo for pipeline, doesn't have to be long lived
* cache doesn't have to be distributed: don't want hassle of persisting sensitive credentials
* going to want to have metrics
  * token cache hit rate (by repo?)
  * token generation time?

* traces:
  * requesting pipeline,build,step
  * cached?
  * request status

