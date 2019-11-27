// Copyright 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package main

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"regexp"
	"strings"
	"time"

	"cloud.google.com/go/pubsub"

	"github.com/golang/gddo/doc"
	"github.com/golang/gddo/gosrc"
)

var (
	testdataPat = regexp.MustCompile(`/testdata(?:/|$)`)
)

// crawlNote is a message sent to Pub/Sub when a crawl occurs.
// It is encoded as JSON, so changes should match its
// compatibility requirements.
type crawlNote struct {
	ImportPath string
}

func (s *server) publishCrawl(ctx context.Context, importPath string) {
	if s.crawlTopic == nil {
		return
	}

	note := &crawlNote{ImportPath: importPath}
	b, err := json.Marshal(note)
	if err != nil {
		log.Printf("Encoding crawlNote: %v", err)
		return
	}
	s.crawlTopic.Publish(ctx, &pubsub.Message{Data: b})
}

// crawlDoc fetches the package documentation from the VCS and updates the database.
func (s *server) crawlDoc(ctx context.Context, source string, importPath string, pdoc *doc.Package, hasSubdirs bool, nextCrawl time.Time) (*doc.Package, error) {
	message := []interface{}{source}
	defer func() {
		message = append(message, importPath)
		log.Println(message...)
	}()

	if !nextCrawl.IsZero() {
		d := time.Since(nextCrawl) / time.Hour
		if d > 0 {
			message = append(message, "late:", int64(d))
		}
	}

	etag := ""
	if pdoc != nil {
		etag = pdoc.Etag
		message = append(message, "etag:", etag)
	}

	start := time.Now()
	var err error
	if strings.HasPrefix(importPath, "code.google.com/p/go.") {
		// Old import path for Go sub-repository.
		pdoc = nil
		err = gosrc.NotFoundError{Message: "old Go sub-repo", Redirect: "golang.org/x/" + importPath[len("code.google.com/p/go."):]}
	} else if blocked, e := s.db.IsBlocked(importPath); blocked && e == nil {
		pdoc = nil
		err = gosrc.NotFoundError{Message: "blocked."}
	} else if testdataPat.MatchString(importPath) {
		pdoc = nil
		err = gosrc.NotFoundError{Message: "testdata."}
	} else {
		var pdocNew *doc.Package
		pdocNew, err = doc.Get(ctx, s.httpClient, importPath, etag)
		message = append(message, "fetch:", int64(time.Since(start)/time.Millisecond))
		if err == nil && pdocNew.Name == "" && !hasSubdirs {
			for _, e := range pdocNew.Errors {
				message = append(message, "err:", e)
			}
			pdoc = nil
			err = gosrc.NotFoundError{Message: "no Go files or subdirs"}
		} else if _, ok := err.(gosrc.NotModifiedError); !ok {
			pdoc = pdocNew
		}
	}

	maxAge := s.v.GetDuration(ConfigMaxAge)
	nextCrawl = start.Add(maxAge)
	switch {
	case strings.HasPrefix(importPath, "github.com/") || (pdoc != nil && len(pdoc.Errors) > 0):
		nextCrawl = start.Add(maxAge * 7)
	case strings.HasPrefix(importPath, "gist.github.com/"):
		// Don't spend time on gists. It's silly thing to do.
		nextCrawl = start.Add(maxAge * 30)
	}

	if err == nil {
		message = append(message, "put:", pdoc.Etag)
		if err := s.put(ctx, pdoc, nextCrawl); err != nil {
			log.Println(err)
		}
		s.publishCrawl(ctx, importPath)
		return pdoc, nil
	} else if e, ok := err.(gosrc.NotModifiedError); ok {
		if pdoc.Status == gosrc.Active && !s.isActivePkg(importPath, e.Status) {
			if e.Status == gosrc.NoRecentCommits {
				e.Status = gosrc.Inactive
			}
			message = append(message, "archive", e)
			pdoc.Status = e.Status
			if err := s.db.Put(ctx, pdoc, nextCrawl, false); err != nil {
				log.Printf("ERROR db.Put(%q): %v", importPath, err)
			}
		} else {
			// Touch the package without updating and move on to next one.
			message = append(message, "touch")
			if err := s.db.SetNextCrawl(importPath, nextCrawl); err != nil {
				log.Printf("ERROR db.SetNextCrawl(%q): %v", importPath, err)
			}
		}
		s.publishCrawl(ctx, importPath)
		return pdoc, nil
	} else if e, ok := err.(gosrc.NotFoundError); ok {
		message = append(message, "notfound:", e)
		if err := s.db.Delete(ctx, importPath); err != nil {
			log.Printf("ERROR db.Delete(%q): %v", importPath, err)
		}
		return nil, e
	} else {
		message = append(message, "ERROR:", err)
		return nil, err
	}
}

func (s *server) put(ctx context.Context, pdoc *doc.Package, nextCrawl time.Time) error {
	if pdoc.Status == gosrc.NoRecentCommits &&
		s.isActivePkg(pdoc.ImportPath, gosrc.NoRecentCommits) {
		pdoc.Status = gosrc.Active
	}
	if err := s.db.Put(ctx, pdoc, nextCrawl, false); err != nil {
		return fmt.Errorf("ERROR db.Put(%q): %v", pdoc.ImportPath, err)
	}
	return nil
}

// isActivePkg reports whether a package is considered active,
// either because its directory is active or because it is imported by another package.
func (s *server) isActivePkg(pkg string, status gosrc.DirectoryStatus) bool {
	switch status {
	case gosrc.Active:
		return true
	case gosrc.NoRecentCommits:
		// It should be inactive only if it has no imports as well.
		n, err := s.db.ImporterCount(pkg)
		if err != nil {
			log.Printf("ERROR db.ImporterCount(%q): %v", pkg, err)
		}
		return n > 0
	}
	return false
}
