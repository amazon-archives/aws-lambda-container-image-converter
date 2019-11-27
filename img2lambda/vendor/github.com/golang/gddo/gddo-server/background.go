// Copyright 2017 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package main

import (
	"context"
	"log"
	"time"

	"cloud.google.com/go/trace"

	"github.com/golang/gddo/gosrc"
)

func (s *server) doCrawl(ctx context.Context) error {
	span := s.traceClient.NewSpan("Crawl")
	defer span.Finish()
	ctx = trace.NewContext(ctx, span)

	// Look for new package to crawl.
	importPath, hasSubdirs, err := s.db.PopNewCrawl()
	if err != nil {
		log.Printf("db.PopNewCrawl() returned error %v", err)
		return nil
	}
	if importPath != "" {
		if pdoc, err := s.crawlDoc(ctx, "new", importPath, nil, hasSubdirs, time.Time{}); pdoc == nil && err == nil {
			if err := s.db.AddBadCrawl(importPath); err != nil {
				log.Printf("ERROR db.AddBadCrawl(%q): %v", importPath, err)
			}
		}
		return nil
	}

	// Crawl existing doc.
	pdoc, pkgs, nextCrawl, err := s.db.Get(ctx, "-")
	if err != nil {
		log.Printf("db.Get(\"-\") returned error %v", err)
		return nil
	}
	if pdoc == nil || nextCrawl.After(time.Now()) {
		return nil
	}
	if _, err = s.crawlDoc(ctx, "crawl", pdoc.ImportPath, pdoc, len(pkgs) > 0, nextCrawl); err != nil {
		// Touch package so that crawl advances to next package.
		if err := s.db.SetNextCrawl(pdoc.ImportPath, time.Now().Add(s.v.GetDuration(ConfigMaxAge)/3)); err != nil {
			log.Printf("ERROR db.SetNextCrawl(%q): %v", pdoc.ImportPath, err)
		}
	}
	return nil
}

func (s *server) readGitHubUpdates(ctx context.Context) error {
	span := s.traceClient.NewSpan("GitHubUpdates")
	defer span.Finish()
	ctx = trace.NewContext(ctx, span)

	const key = "gitHubUpdates"
	var last string
	if err := s.db.GetGob(key, &last); err != nil {
		return err
	}
	last, names, err := gosrc.GetGitHubUpdates(ctx, s.httpClient, last)
	if err != nil {
		return err
	}

	for _, name := range names {
		log.Printf("bump crawl github.com/%s", name)
		if err := s.db.BumpCrawl("github.com/" + name); err != nil {
			log.Println("ERROR force crawl:", err)
		}
	}

	if err := s.db.PutGob(key, last); err != nil {
		return err
	}
	return nil
}
