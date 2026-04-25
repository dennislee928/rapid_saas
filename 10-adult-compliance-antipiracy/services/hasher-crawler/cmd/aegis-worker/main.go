package main

import (
	"crypto/sha256"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/rapid-saas/aegis-adult/hasher-crawler/internal/crawl"
	"github.com/rapid-saas/aegis-adult/hasher-crawler/internal/hash"
	"github.com/rapid-saas/aegis-adult/hasher-crawler/internal/store"
)

func main() {
	mode := flag.String("mode", "once", "run mode: once, hasher, crawler")
	flag.Parse()

	repo := store.NewMemoryStore()
	repo.AddAsset(store.Asset{
		ID:         "asset_demo",
		TenantID:   "tenant_demo",
		Kind:       "image",
		Label:      "demo reference",
		PHash:      hash.ImagePHash([]byte("demo reference image")),
		Uploaded:   time.Now().UTC(),
		SourceGone: true,
	})

	switch *mode {
	case "once", "hasher":
		runHasherDemo(repo)
	case "crawler":
		runCrawlerDemo(repo)
	default:
		log.Fatalf("unknown mode %q", *mode)
	}
}

func runHasherDemo(repo *store.MemoryStore) {
	data := []byte("local-only sample content")
	sum := sha256.Sum256(data)
	fmt.Fprintf(os.Stdout, "queued asset hash=%s phash=%016x\n", hex.EncodeToString(sum[:]), hash.ImagePHash(data))
	fmt.Fprintf(os.Stdout, "assets=%d\n", len(repo.Assets()))
}

func runCrawlerDemo(repo *store.MemoryStore) {
	crawler := crawl.NewCrawler(crawl.Policy{RateLimitPerSecond: 1, RespectRobots: true})
	candidates := []crawl.Candidate{
		{URL: "https://example.invalid/leak-one", PHash: hash.ImagePHash([]byte("demo reference image"))},
		{URL: "https://example.invalid/no-match", PHash: hash.ImagePHash([]byte("unrelated"))},
	}
	matches := crawler.MatchCandidates(repo.Assets(), candidates)
	fmt.Fprintf(os.Stdout, "matches=%d\n", len(matches))
}
