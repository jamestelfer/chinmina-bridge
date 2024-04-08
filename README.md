# bk-auth

## Configuration

- Github application private key
  - refinement: can this stay in KMS perhaps?

- Buildkite
  -  API token (what scopes?)
  - JWT verify
    - audience
    - issuer must be buildkite
    - JWKS url <- allows issuance from another source for testing
    - clock skew

## Purpose

Authenticated by Buildkite OIDC JWT, will issue a Github access token that has
`contents:read` permissions on the currently executing pipeline's repository.

Token is issued by Github application supplied by user. Token is issued for
application, not for Github user.

Returns the https repository url and the token for the pipeline. To use, likely
that ssh will need redirection.

## Functionality

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

