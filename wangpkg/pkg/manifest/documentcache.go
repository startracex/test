package manifest

import (
	"encoding/json/v2"
	"io"

	"github.com/dgraph-io/ristretto/v2"
	"github.com/startracex/wangpkg/pkg/auth"
	"github.com/startracex/wangpkg/pkg/concurrent"
	"github.com/startracex/wangpkg/pkg/npmini"
	"github.com/startracex/wangpkg/pkg/specifier"
)

type documentInflight struct {
	doc  *Document
	err  error
	done chan struct{}
}

type DocumentCache struct {
	docCache *ristretto.Cache[string, *Document]
	rawCache *ristretto.Cache[string, []byte]
	inflight concurrent.Map[string, *documentInflight]
}

func NewDocCache(docSize, rawSize int64) *DocumentCache {
	if docSize <= 0 || rawSize <= 0 {
		panic("negative size")
	}

	docCache, _ := ristretto.NewCache(&ristretto.Config[string, *Document]{
		NumCounters: docSize * 10,
		MaxCost:     docSize,
		BufferItems: 64,
	})
	rawCache, _ := ristretto.NewCache(&ristretto.Config[string, []byte]{
		NumCounters: rawSize * 10,
		MaxCost:     rawSize,
		BufferItems: 64,
	})

	return &DocumentCache{
		docCache: docCache,
		rawCache: rawCache,
	}
}

func (c *DocumentCache) Clear() {
	c.docCache.Clear()
	c.rawCache.Clear()
	c.inflight.Clear()
}

func (c *DocumentCache) cancelInflight(inf *documentInflight, key string) {
	close(inf.done)
	c.inflight.Delete(key)
}

func (c *DocumentCache) FetchDocument(spec *specifier.RegistrySpecifier, rc *npmini.NpmINI, dc auth.Client) (*Document, error) {
	key := spec.Namespace()

	// Check docCache
	if doc, ok := c.docCache.Get(key); ok {
		return doc, nil
	}

	doc := new(Document)

	// Check rawCache
	if raw, ok := c.rawCache.Get(key); ok {
		err := json.Unmarshal(raw, doc)
		if err == nil {
			c.docCache.Set(key, doc, 1)
			c.docCache.Wait()
			return doc, nil
		}
	}

	// Deduplicate concurrent requests
	if inf, ok := c.inflight.Get(key); ok {
		<-inf.done
		return inf.doc, inf.err
	}
	inf := &documentInflight{done: make(chan struct{})}
	c.inflight.Set(key, inf)

	// Fetch from network
	req, err := newDocumentRequest(spec, rc)
	if err != nil {
		inf.err = err
		c.cancelInflight(inf, key)
		return nil, err
	}
	resp, err := dc.Do(req)
	if err != nil {
		inf.err = err
		c.cancelInflight(inf, key)
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()
	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		inf.err = err
		c.cancelInflight(inf, key)
		return nil, err
	}

	if err = json.Unmarshal(raw, doc); err != nil {
		inf.err = err
		c.cancelInflight(inf, key)
		return nil, err
	}

	// Cache results
	c.rawCache.Set(key, raw, 1)
	c.docCache.Set(key, doc, 1)
	c.rawCache.Wait()
	c.docCache.Wait()

	// Signal waiters
	inf.doc = doc
	c.cancelInflight(inf, key)

	return doc, nil
}
