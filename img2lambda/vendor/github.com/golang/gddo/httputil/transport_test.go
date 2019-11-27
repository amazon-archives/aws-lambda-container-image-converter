// Copyright 2017 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package httputil

import (
	"bytes"
	"io/ioutil"
	"net/http"
	"net/url"
	"testing"
)

func TestTransportGithubAuth(t *testing.T) {
	tests := []struct {
		name string

		url          string
		token        string
		clientID     string
		clientSecret string

		queryClientID     string
		queryClientSecret string
		authorization     string
	}{
		{
			name:          "Github token",
			url:           "https://api.github.com/",
			token:         "xyzzy",
			authorization: "token xyzzy",
		},
		{
			name:              "Github client ID/secret",
			url:               "https://api.github.com/",
			clientID:          "12345",
			clientSecret:      "xyzzy",
			queryClientID:     "12345",
			queryClientSecret: "xyzzy",
		},
		{
			name:  "non-Github site does not have token headers",
			url:   "http://www.example.com/",
			token: "xyzzy",
		},
		{
			name:         "non-Github site does not have client ID/secret headers",
			url:          "http://www.example.com/",
			clientID:     "12345",
			clientSecret: "xyzzy",
		},
		{
			name:  "Github token not sent over HTTP",
			url:   "http://api.github.com/",
			token: "xyzzy",
		},
		{
			name:         "Github client ID/secret not sent over HTTP",
			url:          "http://api.github.com/",
			clientID:     "12345",
			clientSecret: "xyzzy",
		},
		{
			name:  "Github token not sent over schemeless",
			url:   "//api.github.com/",
			token: "xyzzy",
		},
		{
			name:         "Github client ID/secret not sent over schemeless",
			url:          "//api.github.com/",
			clientID:     "12345",
			clientSecret: "xyzzy",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			var (
				query      url.Values
				authHeader string
			)
			client := &http.Client{
				Transport: &AuthTransport{
					Base: roundTripFunc(func(r *http.Request) {
						query = r.URL.Query()
						authHeader = r.Header.Get("Authorization")
					}),
					GithubToken:        test.token,
					GithubClientID:     test.clientID,
					GithubClientSecret: test.clientSecret,
				},
			}
			_, err := client.Get(test.url)
			if err != nil {
				t.Fatal(err)
			}
			if got := query.Get("client_id"); got != test.queryClientID {
				t.Errorf("url query client_id = %q; want %q", got, test.queryClientID)
			}
			if got := query.Get("client_secret"); got != test.queryClientSecret {
				t.Errorf("url query client_secret = %q; want %q", got, test.queryClientSecret)
			}
			if authHeader != test.authorization {
				t.Errorf("header Authorization = %q; want %q", authHeader, test.authorization)
			}
		})
	}
}

type roundTripFunc func(r *http.Request)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	f(r)
	return &http.Response{
		Status:        "200 OK",
		StatusCode:    http.StatusOK,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Body:          ioutil.NopCloser(bytes.NewReader(nil)),
		ContentLength: 0,
		Request:       r,
	}, nil
}
