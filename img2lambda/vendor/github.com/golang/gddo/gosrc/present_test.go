package gosrc

import (
	"bytes"
	"io/ioutil"
	"os"
	"testing"

	"golang.org/x/tools/present"
)

// Verify that the output of presBuilder is still a valid presentation.
func TestPresentationBuilderValidOutput(t *testing.T) {
	os.Chdir("testdata")
	defer os.Chdir("..")

	filename := "sample.slide"
	data, err := ioutil.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}

	// Confirm the slide is valid to begin with.
	_, err = present.Parse(bytes.NewReader(data), "testing", 0)
	if err != nil {
		t.Fatal(err)
	}

	// Transform the presentation.
	b := presBuilder{
		filename:   filename,
		data:       data,
		resolveURL: func(fname string) string { return fname },
		fetch:      func(fnames []string) ([]*File, error) { return nil, nil },
	}

	p, err := b.build()
	if err != nil {
		t.Fatal(err)
	}

	output := p.Files[filename]
	if len(output) == 0 {
		t.Fatal("presentation builder produced no output")
	}

	// Confirm the output is still valid.
	_, err = present.Parse(bytes.NewReader(output), "testing", 0)
	if err != nil {
		t.Fatal(err)
	}
}

func TestPresentationBuilderTransforms(t *testing.T) {
	resolveURL := func(fname string) string {
		return "https://resolved.com/" + fname
	}

	fetch := func(fnames []string) ([]*File, error) {
		var files []*File
		for _, fname := range fnames {
			files = append(files, &File{
				Name: fname,
				Data: []byte("data"),
			})
		}
		return files, nil
	}

	cases := []struct {
		Name    string
		Input   string
		Expect  string
		Fetched []string
	}{
		{
			Name:   "image",
			Input:  ".image blah.jpg _ 42",
			Expect: ".image https://resolved.com/blah.jpg _ 42",
		},
		{
			Name:   "background",
			Input:  ".background blah.jpg",
			Expect: ".background https://resolved.com/blah.jpg",
		},
		{
			Name:   "iframe",
			Input:  ".iframe iframe.html 200 300",
			Expect: ".iframe https://resolved.com/iframe.html 200 300",
		},
		{
			Name:   "html",
			Input:  ".html embed.html",
			Expect: "\nERROR: .html not supported\n",
		},
		{
			Name:    "code",
			Input:   ".code hello.go /start/,/end/",
			Expect:  ".code hello.go /start/,/end/",
			Fetched: []string{"hello.go"},
		},
		{
			Name:    "play",
			Input:   ".play hello.go",
			Expect:  ".play hello.go",
			Fetched: []string{"hello.go"},
		},
	}

	for _, c := range cases {
		t.Run(c.Name, func(t *testing.T) {
			b := presBuilder{
				filename:   "snippet.slide",
				data:       []byte(c.Input),
				resolveURL: resolveURL,
				fetch:      fetch,
			}

			p, err := b.build()
			if err != nil {
				t.Fatal(err)
			}

			output := p.Files["snippet.slide"]
			if !bytes.Equal([]byte(c.Expect), output) {
				t.Fatalf("bad output: got '%s' expect '%s", string(output), c.Expect)
			}

			for _, fname := range c.Fetched {
				if _, ok := p.Files[fname]; !ok {
					t.Fatalf("file %s not fetched", fname)
				}
			}
		})
	}
}
