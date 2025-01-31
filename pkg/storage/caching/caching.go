package caching

import (
	"context"
	"fmt"
	"time"

	"github.com/karlseguin/ccache/v3"
	"github.com/openfga/openfga/pkg/storage"
	openfgapb "go.buf.build/openfga/go/openfga/api/openfga/v1"
)

const ttl = time.Hour * 168

var _ storage.OpenFGADatastore = (*cachedOpenFGADatastore)(nil)

type cachedOpenFGADatastore struct {
	storage.OpenFGADatastore
	cache *ccache.Cache[*openfgapb.AuthorizationModel]
}

// NewCachedOpenFGADatastore returns a wrapper over a datastore that caches *openfgapb.AuthorizationModel
// on every call to storage.ReadAuthorizationModel.
func NewCachedOpenFGADatastore(inner storage.OpenFGADatastore, maxSize int) *cachedOpenFGADatastore {
	return &cachedOpenFGADatastore{
		OpenFGADatastore: inner,
		cache:            ccache.New(ccache.Configure[*openfgapb.AuthorizationModel]().MaxSize(int64(maxSize))),
	}
}

func (c *cachedOpenFGADatastore) ReadAuthorizationModel(ctx context.Context, storeID, modelID string) (*openfgapb.AuthorizationModel, error) {
	cacheKey := fmt.Sprintf("%s:%s", storeID, modelID)
	cachedEntry := c.cache.Get(cacheKey)

	if cachedEntry != nil {
		return cachedEntry.Value(), nil
	}

	model, err := c.OpenFGADatastore.ReadAuthorizationModel(ctx, storeID, modelID)
	if err != nil {
		return nil, err
	}

	c.cache.Set(cacheKey, model, ttl) // these are immutable, once created, there cannot be edits, therefore they can be cached without ttl

	return model, nil
}

func (c *cachedOpenFGADatastore) Close() {
	c.cache.Stop()
}
