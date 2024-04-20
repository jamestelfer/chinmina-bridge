package vendor

import (
	"context"
	"time"

	"github.com/jamestelfer/ghauth/internal/jwt"
	"github.com/maypok86/otter"
	"github.com/rs/zerolog/log"
)

// Cached supplies a vendor that caches the results of the wrapped vendor. The
// cache is non-locking, and so concurrent requests for the same pipeline could
// cause multiple token requests, In this case, the last one returned wins. In
// this use case, given that concurrent calls are likely to be less common, the
// additional tokens issued are worth gains made skipping locking.
func Cached() (func(PipelineTokenVendor) PipelineTokenVendor, error) {
	cache, err := otter.
		MustBuilder[string, PipelineRepositoryToken](10_000).
		CollectStats().
		WithTTL(45 * time.Minute).
		Build()
	if err != nil {
		return nil, err
	}

	return func(v PipelineTokenVendor) PipelineTokenVendor {
		return func(ctx context.Context, claims jwt.BuildkiteClaims, repo string) (*PipelineRepositoryToken, error) {
			// Cache by pipeline: this allows token reuse across multiple
			// builds. It's likely that this will change if rules allow for
			// builds to be given different permissions based on the branch or
			// tag.
			key := claims.PipelineID

			if response, ok := cache.Get(key); ok {
				// The expected repo may be unknown, but if it's supplied it needs to
				// match the cached repo. An "unknown" means "give me the token for the
				// pipeline's repository"; when supplied, a token is request for a given
				// repo (if possible).
				if repo == "" || response.RepositoryURL == repo {
					log.Info().Time("expiry", response.Expiry).
						Str("key", key).
						Msg("hit: existing token found for pipeline")

					return &response, nil
				} else {
					// Tsoken invalid: remove from cache and fall through to reissue and
					// re-cache likely to happen if the pipeline's repository is changed.
					log.Info().
						Str("key", key).Str("expected", repo).
						Str("actual", response.RepositoryURL).
						Msg("invalid: cached token issued for different repository")

					// the delete is required as "set" is not guaranteed to write to the cache
					cache.Delete(key)
				}
			}

			// cache miss: request and cache
			token, err := v(ctx, claims, repo)
			if err != nil {
				return nil, err
			}

			cache.Set(key, *token)

			return token, nil
		}
	}, nil
}
