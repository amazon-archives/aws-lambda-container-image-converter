// Copyright 2019 Amazon.com, Inc. or its affiliates. All Rights Reserved.
// SPDX-License-Identifier: MIT-0
package version

// Version indicates which version of the binary is running.
var Version = "mainline"

// GitCommitSHA indicates which git shorthash the binary was built off of
var GitCommitSHA string

func VersionString() string {
	return Version + " (" + GitCommitSHA + ")"
}
