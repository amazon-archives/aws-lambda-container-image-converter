package otbuiltin

import (
	"io/ioutil"
	"os"
	"path"
	"testing"
)

func TestInitSuccess(t *testing.T) {
	// Make a base directory in which all of our test data resides
	baseDir, err := ioutil.TempDir("", "otbuiltin-test-")
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	defer os.RemoveAll(baseDir)
	// Make a directory in which the repo should exist
	repoDir := path.Join(baseDir, "repo")
	err = os.Mkdir(repoDir, 0777)
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	// Initialize the repo
	inited, err := Init(repoDir, NewInitOptions())
	if err != nil {
		t.Errorf("%s", err)
		return
	} else if !inited {
		t.Errorf("Cannot test commit: failed to initialize repo")
		return
	}
}

func TestInitBareUser(t *testing.T) {
	// Make a base directory in which all of our test data resides
	baseDir, err := ioutil.TempDir("", "otbuiltin-test-")
	if err != nil {
		t.Errorf("%s", err)
		return
	}
	defer os.RemoveAll(baseDir)
	// Make a directory in which the repo should exist
	repoDir := path.Join(baseDir, "repo")
	err = os.Mkdir(repoDir, 0777)
	if err != nil {
		t.Errorf("%s", err)
		return
	}

	// Initialize the repo
	initOpts := NewInitOptions()
	initOpts.Mode = "bare-user"
	inited, err := Init(repoDir, initOpts)
	if err != nil {
		t.Errorf("%s", err)
		return
	} else if !inited {
		t.Errorf("Cannot test commit: failed to initialize repo")
		return
	}
}

func TestParseRepoMode(t *testing.T) {
	tests := []struct {
		in    string
		isErr bool
	}{
		{
			"archive-z2",
			false,
		},
		{
			"bare",
			false,
		},
		{
			"bare-user",
			false,
		},
		{
			"fooMode",
			true,
		},
	}

	for _, tt := range tests {
		_, err := parseRepoMode(tt.in)
		if tt.isErr && err == nil {
			t.Fatal("got unexpected nil error")
		} else if !tt.isErr && err != nil {
			t.Fatalf("got unexpected error %q", err)
		}
	}
}
