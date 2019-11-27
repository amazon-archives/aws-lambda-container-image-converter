// Copyright 2017 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package health

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
)

func TestNewHandler(t *testing.T) {
	s := httptest.NewServer(new(Handler))
	defer s.Close()
	code, err := check(s)
	if err != nil {
		t.Fatalf("GET %s: %v", s.URL, err)
	}
	if code != http.StatusOK {
		t.Errorf("got HTTP status %d; want %d", code, http.StatusOK)
	}
}

func TestChecker(t *testing.T) {
	c1 := &checker{err: errors.New("checker 1 down")}
	c2 := &checker{err: errors.New("checker 2 down")}
	h := new(Handler)
	h.Add(c1)
	h.Add(c2)
	s := httptest.NewServer(h)
	defer s.Close()

	t.Run("AllUnhealthy", func(t *testing.T) {
		code, err := check(s)
		if err != nil {
			t.Fatalf("GET %s: %v", s.URL, err)
		}
		if code != http.StatusInternalServerError {
			t.Errorf("got HTTP status %d; want %d", code, http.StatusInternalServerError)
		}
	})
	c1.set(nil)
	t.Run("PartialHealthy", func(t *testing.T) {
		code, err := check(s)
		if err != nil {
			t.Fatalf("GET %s: %v", s.URL, err)
		}
		if code != http.StatusInternalServerError {
			t.Errorf("got HTTP status %d; want %d", code, http.StatusInternalServerError)
		}
	})
	c2.set(nil)
	t.Run("AllHealthy", func(t *testing.T) {
		code, err := check(s)
		if err != nil {
			t.Fatalf("GET %s: %v", s.URL, err)
		}
		if code != http.StatusOK {
			t.Errorf("got HTTP status %d; want %d", code, http.StatusOK)
		}
	})
}

func check(s *httptest.Server) (code int, err error) {
	resp, err := http.Get(s.URL)
	if err != nil {
		return 0, err
	}
	resp.Body.Close()
	return resp.StatusCode, nil
}

type checker struct {
	mu  sync.Mutex
	err error
}

func (c *checker) CheckHealth() error {
	defer c.mu.Unlock()
	c.mu.Lock()
	return c.err
}

func (c *checker) set(e error) {
	defer c.mu.Unlock()
	c.mu.Lock()
	c.err = e
}
