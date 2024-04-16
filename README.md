# Buildkite/Github OIDC bridge 

Exposes an endpoint that allows Buildkite Agents to use an OIDC token to request
a Github token in return. The GitHub token is created for the agent using a
Github application that is created for the bridge and configured to allow access
to the implementor's GitHub organization.

The token is created with `contents:read` permissions on the repository
associated with the executing pipeline.

## Why?

Using a GitHub application to authenticate GitHub actions allows:

1. The use of ephemeral API tokens (they expire after 1 hour).
2. Tokens can enable a wider set of actions than simple Git operations (e.g. PR
   comments).
3. Supplied tokens are scoped to just the resources and actions requested, not
   to the whole set of repositories and actions allowed for the app.
4. Additional configuration per repo not required. If the app has access, the
   agent can request a token for it. No need to create PATs or generate keypairs,
   and no need to upload them in multiple places.

Also, since the OIDC agent uses Buildkite's OIDC tokens to authorize requests,
the claims associated with the token can be used to further refine access to a token.

There are two options generally used to authenticate Buildkite agents to GitHub:

1. Via a PAT (owned by a GitHub user) that is saved in the agent S3 secrets bucket
2. Via a deploy key (registered to a single repository) that is likewise saved to
   S3.

Each of these have some downsides:

| **PAT**                                 | **Deploy keys**                                           |
|-----------------------------------------|-----------------------------------------------------------|
| optional expiry                         | no expiry                                                 |
| access governed by associated user [^1] | access to single repo                                     |
| manual creation (generally)             | read or read/write                                        |
|                                         | may be user-associated [^2]                               |

[^1]: if the user is decommissioned, the PAT is deactivated. The PAT has access to all repos that the
      issuing user can access.
[^2]: a registered deploy key can be associated with a user, and deactivated if that user is deactivated.
      This isn't good if the key is used to authenticate automation that is still required.

## Overview

The OIDC bridge (this project) is used by jobs running on a Buildkite agent to
request tokens from Github. These can be used to communicate with the GitHub API
or (via Git) to enable authenticated Git actions.

Git authentication is facilitated by a [Git credential
helper](https://github.com/jamestelfer/github-app-auth-buildkite-plugin), which
communicates with the bridge and supplies the result to Git in the appropriate
format.

The following sequence illustrates a Git authentication flow facilitated by the
OIDC bridge.

```mermaid
sequenceDiagram
    box Buildkite Agent
        participant Buildkite Job
        participant Git
        participant Credential Helper
    end
    box Self hosted
        participant OIDC Bridge
    end
    Buildkite Job->>+Git: clone
    Git ->>+ Credential Helper: get credentials
    Credential Helper->>+Buildkite API: Request Buildkite OIDC token
    Buildkite API->>-Credential Helper: bk-oidc
    Credential Helper->>+OIDC Bridge: Request GH token (auth bk-oidc)
    OIDC Bridge->>+Buildkite API: Get Pipeline Repo
    Buildkite API-->>-OIDC Bridge: 
    OIDC Bridge->>+GitHub: Create Token (auth app JWT)
    GitHub-->>-OIDC Bridge: 
    OIDC Bridge->>-Credential Helper: bk-oidc
    Credential Helper->>-Git: "x-access-token"/app-token
    Git-->>-Buildkite Job: 
```



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

## Generating test keys

Use [https://mkjwk.org], save private and public to `.development/keys`. Good enough for test credentials.
