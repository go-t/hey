package main

import (
	"bytes"
	"mime/multipart"
	gourl "net/url"
	"testing"

	"github.com/stretchr/testify/assert"
)

var testHelper = helper{
	escape: gourl.QueryEscape,
	readFile: func(filename string) ([]byte, error) {
		return []byte("{" + filename + "}"), nil
	},
}

func TestFormData(t *testing.T) {
	var buf bytes.Buffer

	{
		var w = multipart.NewWriter(&buf)
		w.SetBoundary(boundary)
		w.WriteField("a", "123")
		fw, _ := w.CreateFormFile("b", "a.txt")
		fw.Write([]byte("{a.txt}"))
		fw, _ = w.CreateFormFile("c", "img.png")
		fw.Write([]byte("{img.png}"))
		w.Close()
	}

	forms := []string{"a=123", "b=@a.txt", "c=@img.png;image/png"}
	body, err := testHelper.formData(forms, boundary)
	assert.NoError(t, err)
	assert.Equal(t, buf.String(), string(body))
}

func TestUrlEncoding(t *testing.T) {
	fields := []string{
		"b&c",
		"=b+c",
		"a=b=c",
		"b=@b",
		// "c@c",
		// "c@x", // TODO: append to c
	}

	body, err := testHelper.urlEncoded(fields)
	assert.NoError(t, err)
	assert.Equal(t, "b%26c&b%2Bc&a=b%3Dc&b=%7Bb%7D&", string(body))
}
