# Observability

The system produces traces and metrics via Open Telemetry, and logs via zerolog.
There are minimal informational logs, as well as per-request audit logs that are
written to the process's `stdout`.

## Audit logs

Audit logs provide a level of non-repudiation for the system. These logs are
written to the container's stdout, and cannot be disabled.

> [!TIP]
> Requests to non-existent routes do not form part of the audit log. Access logs
> or firewall logs are better sources for this information.

Each authenticated endpoint (both `/token` and `/git-credentials`) will record
details about the request, the authorization process, and the GitHub token
created. If an error occurs, this is also written out.

At a technical level, logs are written to stdout using zerolog at the "audit"
log level. Initial data is collected by request middleware and the partial entry
is then accessible via the context. The context entry is further enriched with
details by other components, including both the JWT middleware and the vendor.
such that the log is fully completed by the end of the request.

> [!IMPORTANT]
> A panic in the request chain will still result in the audit log being written,
> and the panic details will also be included.

### Audit log fields

1. Request data
    - `Method`: the request method. This will currently be `GET` for all standard requests.
    - `Path`: the requested path.
    - `Status` is the HTTP response status of the request
    - `SourceIP` is the client IP of the requestor
    - `UserAgent` is the user agent reported by the client
    - `Error` is the error produced by the request. This may come from internal
      errors or panics, as well as the JWT validation and token creation
      components.
2. Authorization data
    - `Authorized` is a boolean that is `true` when the request JWT is
      successfully authorized by the service.
    - `AuthSubject` is the contents of the `sub` field from the JWT
    - `AuthIssuer` is the JWT `iss` field
    - `AuthAudience` is the (possibly multiple) reported `aud` field values from
      the JWT
    - `AuthExpirySecs` is the JWT expiry time in seconds after the Unix epoch
3. Token data
    - `Repositories` is the set of repositories that the token allows access to
    - `Permissions` is the set of GitHub token permissions assigned to the token
    - `ExpirySecs` is the GitHub token expiry time in seconds after the Unix
      epoch

## Open Telemetry

This section is a stub. For now, refer to the [`.envrc`](../.envrc) file for
details of all Open Telemetry related configuration that's currently possible.
