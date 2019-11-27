// Copyright 2017 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

// This file implements an http.Client with request timeouts set by command
// line flags.

package main

import (
	"net"
	"net/http"

	"cloud.google.com/go/trace"
	"github.com/gregjones/httpcache"
	"github.com/gregjones/httpcache/memcache"
	"github.com/spf13/viper"

	"github.com/golang/gddo/httputil"
)

func newHTTPClient(v *viper.Viper) *http.Client {
	requestTimeout := v.GetDuration(ConfigRequestTimeout)
	var t http.RoundTripper = &http.Transport{
		Proxy: http.ProxyFromEnvironment,
		Dial: (&net.Dialer{
			Timeout:   v.GetDuration(ConfigDialTimeout),
			KeepAlive: requestTimeout / 2,
		}).Dial,
		ResponseHeaderTimeout: requestTimeout / 2,
		TLSHandshakeTimeout:   requestTimeout / 2,
	}
	if addr := v.GetString(ConfigMemcacheAddr); addr != "" {
		ct := httpcache.NewTransport(memcache.New(addr))
		ct.Transport = t
		t = ct
	}
	t = &httputil.AuthTransport{
		Base: t,

		UserAgent:          v.GetString(ConfigUserAgent),
		GithubToken:        v.GetString(ConfigGithubToken),
		GithubClientID:     v.GetString(ConfigGithubClientID),
		GithubClientSecret: v.GetString(ConfigGithubClientSecret),
	}
	t = trace.Transport{Base: t}
	return &http.Client{
		Transport: t,
		Timeout:   requestTimeout,
	}
}
