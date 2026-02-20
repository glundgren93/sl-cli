package api

import (
	"context"
	"sync"
	"time"

	"github.com/glundgren93/sl-cli/internal/model"
)

const siteCacheTTL = 5 * time.Minute

// SiteCache caches the full sites list to avoid repeated API calls.
type SiteCache struct {
	mu       sync.Mutex
	sites    []model.Site
	fetchedAt time.Time
}

var globalSiteCache = &SiteCache{}

// GetSitesCached returns sites from cache if fresh, otherwise fetches from API.
func (c *Client) GetSitesCached(ctx context.Context) ([]model.Site, error) {
	globalSiteCache.mu.Lock()
	defer globalSiteCache.mu.Unlock()

	if len(globalSiteCache.sites) > 0 && time.Since(globalSiteCache.fetchedAt) < siteCacheTTL {
		return globalSiteCache.sites, nil
	}

	sites, err := c.GetSites(ctx)
	if err != nil {
		return nil, err
	}

	globalSiteCache.sites = sites
	globalSiteCache.fetchedAt = time.Now()
	return sites, nil
}
