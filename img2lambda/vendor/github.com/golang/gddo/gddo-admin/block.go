// Copyright 2013 The Go Authors. All rights reserved.
//
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file or at
// https://developers.google.com/open-source/licenses/bsd.

package main

import (
	"log"
	"os"

	"github.com/golang/gddo/database"
)

var blockCommand = &command{
	name:  "block",
	run:   block,
	usage: "block path",
}

func block(c *command) {
	if len(c.flag.Args()) != 1 {
		c.printUsage()
		os.Exit(1)
	}
	db, err := database.New(*redisServer, *dbIdleTimeout, false, gaeEndpoint)
	if err != nil {
		log.Fatal(err)
	}
	if err := db.Block(c.flag.Args()[0]); err != nil {
		log.Fatal(err)
	}
}
