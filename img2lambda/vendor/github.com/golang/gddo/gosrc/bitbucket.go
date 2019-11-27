// Copyright 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package gosrc

import (
	"context"
	"log"
	"net/http"
	"path"
	"regexp"
	"time"
)

func init() {
	addService(&service{
		pattern: regexp.MustCompile(`^bitbucket\.org/(?P<owner>[a-z0-9A-Z_.\-]+)/(?P<repo>[a-z0-9A-Z_.\-]+)(?P<dir>/[a-z0-9A-Z_.\-/]*)?$`),
		prefix:  "bitbucket.org/",
		get:     getBitbucketDir,
	})
}

var bitbucketEtagRe = regexp.MustCompile(`^(hg|git)-`)

type bitbucketRepo struct {
	Scm       string      `json:"scm"`
	CreatedOn string      `json:"created_on"`
	UpdatedOn string      `json:"updated_on"`
	Parent    interface{} `json:"parent"`
}

type bitbucketRefs struct {
	Values []struct {
		Name   string `json:"name"`
		Target struct {
			Date string `json:"date"`
			Hash string `json:"hash"`
		} `json:"target"`
	} `json:"values"`
	bitbucketPage
}

type bitbucketSrc struct {
	Values []struct {
		Path string `json:"path"`
		Type string `json:"type"`
	} `json:"values"`
	bitbucketPage
}

type bitbucketPage struct {
	Next string `json:"next",omitempty`
}

func getBitbucketDir(ctx context.Context, client *http.Client, match map[string]string, savedEtag string) (*Directory, error) {
	var repo *bitbucketRepo
	c := &httpClient{client: client}

	if m := bitbucketEtagRe.FindStringSubmatch(savedEtag); m != nil {
		match["vcs"] = m[1]
	} else {
		repo, err := getBitbucketRepo(ctx, c, match)
		if err != nil {
			return nil, err
		}

		match["vcs"] = repo.Scm
	}

	tags := make(map[string]string)
	timestamps := make(map[string]time.Time)

	url := expand("https://api.bitbucket.org/2.0/repositories/{owner}/{repo}/refs?pagelen=100", match)
	for {
		var refs bitbucketRefs
		if _, err := c.getJSON(ctx, url, &refs); err != nil {
			return nil, err
		}
		for _, v := range refs.Values {
			tags[v.Name] = v.Target.Hash
			committed, err := time.Parse(time.RFC3339, v.Target.Date)
			if err != nil {
				log.Println("error parsing timestamp:", v.Target.Date)
				continue
			}
			timestamps[v.Name] = committed
		}
		if refs.Next == "" {
			break
		}
		url = refs.Next
	}

	var err error
	tag, commit, err := bestTag(tags, defaultTags[match["vcs"]])
	if err != nil {
		return nil, err
	}
	match["tag"] = tag
	match["commit"] = commit
	etag := expand("{vcs}-{commit}", match)
	if etag == savedEtag {
		return nil, NotModifiedError{Since: timestamps[tag]}
	}

	if repo == nil {
		repo, err = getBitbucketRepo(ctx, c, match)
		if err != nil {
			return nil, err
		}
	}

	var dirs []string
	var files []*File
	var dataURLs []string

	url = expand("https://api.bitbucket.org/2.0/repositories/{owner}/{repo}/src/{tag}{dir}/?pagelen=100", match)
	for {
		var contents bitbucketSrc
		if _, err := c.getJSON(ctx, url, &contents); err != nil {
			return nil, err
		}

		for _, v := range contents.Values {
			switch v.Type {
			case "commit_file":
				_, name := path.Split(v.Path)
				if isDocFile(name) {
					files = append(files, &File{Name: name, BrowseURL: expand("https://bitbucket.org/{owner}/{repo}/src/{tag}/{0}", match, v.Path)})
					dataURLs = append(dataURLs, expand("https://api.bitbucket.org/2.0/repositories/{owner}/{repo}/src/{tag}/{0}", match, v.Path))
				}
			case "commit_directory":
				dirs = append(dirs, v.Path)
			}
		}
		if contents.Next == "" {
			break
		}
		url = contents.Next
	}

	if err := c.getFiles(ctx, dataURLs, files); err != nil {
		return nil, err
	}

	status := Active
	if isBitbucketDeadEndFork(repo) {
		status = DeadEndFork
	}

	return &Directory{
		BrowseURL:      expand("https://bitbucket.org/{owner}/{repo}/src/{tag}{dir}", match),
		Etag:           etag,
		Files:          files,
		LineFmt:        "%s#cl-%d",
		ProjectName:    match["repo"],
		ProjectRoot:    expand("bitbucket.org/{owner}/{repo}", match),
		ProjectURL:     expand("https://bitbucket.org/{owner}/{repo}/", match),
		Subdirectories: dirs,
		VCS:            match["vcs"],
		Status:         status,
		Fork:           repo.Parent != nil,
	}, nil
}

func getBitbucketRepo(ctx context.Context, c *httpClient, match map[string]string) (*bitbucketRepo, error) {
	var repo bitbucketRepo
	if _, err := c.getJSON(ctx, expand("https://api.bitbucket.org/2.0/repositories/{owner}/{repo}", match), &repo); err != nil {
		return nil, err
	}

	return &repo, nil
}

func isBitbucketDeadEndFork(repo *bitbucketRepo) bool {
	created, err := time.Parse(time.RFC3339, repo.CreatedOn)
	if err != nil {
		return false
	}

	updated, err := time.Parse(time.RFC3339, repo.UpdatedOn)
	if err != nil {
		return false
	}

	isDeadEndFork := false
	if repo.Parent != nil && created.Unix() >= updated.Unix() {
		isDeadEndFork = true
	}

	return isDeadEndFork
}
