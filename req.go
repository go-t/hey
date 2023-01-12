package main

import (
	"bufio"
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io/ioutil"
	"mime/multipart"
	"net/http"
	gourl "net/url"
	"os"
	"regexp"
	"strings"

	"github.com/go-T/hey/requester"
)

const (
	formRegexp   = `^([\w-]+)=(.+)`
	headerRegexp = `^([\w-]+):\s*(.+)`
	authRegexp   = `^(.+):([^\s].+)`
	boundary     = "--179CB67133D24A71A3DA3CC21F6F375F--"
	heyUA        = "hey/0.0.1"
)

type stringSlice []string

func (h *stringSlice) String() string {
	return fmt.Sprintf("%s", *h)
}

func (h *stringSlice) Set(value string) error {
	*h = append(*h, value)
	return nil
}

var (
	headerslice stringSlice
	queryslice  stringSlice
	formslice   stringSlice
	m           = flag.String("m", "", "")
	headers     = flag.String("h", "", "")
	bodyFile    = flag.String("D", "", "")
	accept      = flag.String("A", "", "")
	contentType = flag.String("T", "", "")
	authHeader  = flag.String("a", "", "")
	hostHeader  = flag.String("host", "", "")
	userAgent   = flag.String("U", "", "")
	traceFile   = flag.String("trace", "", "")
)

func init() {
	flag.Var(&headerslice, "H", "")
	flag.Var(&queryslice, "d", "")
	flag.Var(&formslice, "F", "")
}

func makeRequest(url string) (*http.Request, []byte, error) {
	if bool2int(len(queryslice) != 0)+
		bool2int(len(formslice) != 0)+
		bool2int(len(*bodyFile) != 0) > 1 {
		return nil, nil, errors.New("conflict flag -F, -d, -D")
	}

	var (
		err    error
		body   []byte
		ctype  string
		method string
		h      = helper{
			escape:   gourl.QueryEscape,
			readFile: ioutil.ReadFile,
		}
	)
	if len(queryslice) != 0 {
		ctype = "application/x-www-form-urlencoded"
		body, err = h.urlEncoded(queryslice)
	} else if len(formslice) != 0 {
		ctype = "multipart/form-data; boundary=" + boundary
		body, err = h.formData(formslice, boundary)
	} else if *bodyFile != "" {
		ctype = *contentType
		body, err = ioutil.ReadFile(*bodyFile)
	}
	if err != nil {
		return nil, nil, err
	}
	header := make(http.Header)
	header.Set("Content-Type", ctype)

	if *m == "" {
		if len(body) > 0 {
			method = "POST"
		} else {
			method = "GET"
		}
	} else {
		method = *m
	}

	// set any other additional headers
	if *headers != "" {
		return nil, nil, errors.New("Flag '-h' is deprecated, please use '-H' instead.")
	}
	// set any other additional repeatable headers
	for _, h := range headerslice {
		match, err := parseInputWithRegexp(h, headerRegexp)
		if err != nil {
			return nil, nil, err
		}
		header.Set(match[1], match[2])
	}

	if *accept != "" {
		header.Set("Accept", *accept)
	}

	// set basic auth if set
	var username, password string
	if *authHeader != "" {
		match, err := parseInputWithRegexp(*authHeader, authRegexp)
		if err != nil {
			return nil, nil, err
		}
		username, password = match[1], match[2]
	}

	ua := header.Get("User-Agent")
	if ua == "" {
		ua = heyUA
	} else {
		ua += " " + heyUA
	}
	header.Set("User-Agent", ua)

	// set userAgent header if set
	if *userAgent != "" {
		ua = *userAgent + " " + heyUA
		header.Set("User-Agent", ua)
	}

	req, err := http.NewRequest(method, url, nil)
	if err != nil {
		return nil, nil, err
	}
	req.Header = header
	if username != "" || password != "" {
		req.SetBasicAuth(username, password)
	}

	// set host header if set
	if *hostHeader != "" {
		req.Host = *hostHeader
	}

	if len(body) > 0 {
		req.ContentLength = int64(len(body))
	}

	if *traceFile != "" {
		out, err := os.Create(*traceFile)
		if err != nil {
			return nil, nil, err
		}
		defer out.Close()

		request := requester.CloneRequest(req, body)
		err = request.Write(out)
		if err != nil {
			return nil, nil, err
		}
	}
	return req, body, nil
}

type helper struct {
	escape   func(string) string
	readFile func(string) ([]byte, error)
}

// -data value
// -data =value
// -data name=value
// -data name=@file
// -data name@file
func (h helper) urlEncoded(data stringSlice) ([]byte, error) {

	var b bytes.Buffer
	w := bufio.NewWriter(&b)
	for _, item := range data {
		parts := strings.SplitN(item, "=", 2)
		var key, value string
		if len(parts) == 1 {
			value = parts[0]
		} else {
			key, value = parts[0], parts[1]
		}
		if key != "" {
			w.WriteString(key)
			w.WriteString("=")
		}
		parts = strings.SplitN(value, "@", 2)
		if len(parts) == 2 {
			data, err := h.readFile(parts[1])
			if err != nil {
				return nil, fmt.Errorf("load file %s error:%s", parts[1], err.Error())
			}
			w.WriteString(h.escape(parts[0]))
			w.WriteString(h.escape(string(data)))
		} else {
			w.WriteString(h.escape(value))
		}
		w.WriteByte('&')
		w.Flush()
	}
	return b.Bytes(), nil
}

// -form name=value
// -form name=@file
func (h helper) formData(form stringSlice, boundary string) ([]byte, error) {

	var b bytes.Buffer
	var w = multipart.NewWriter(&b)
	w.SetBoundary(boundary)

	for _, item := range form {
		match, err := parseInputWithRegexp(strings.SplitN(item, ";", 2)[0], formRegexp)
		if err != nil {
			return nil, err
		}
		key, value := match[1], match[2]
		if len(value) > 0 && value[:1] == "@" {
			data, err := h.readFile(value[1:])
			if err != nil {
				return nil, fmt.Errorf("load file %s error:%s", value[1:], err.Error())
			}

			field, err := w.CreateFormFile(key, value[1:])
			if err != nil {
				return nil, fmt.Errorf("create form field %s error:%s", key, err.Error())
			}
			field.Write(data)
		} else {
			err := w.WriteField(key, value)
			if err != nil {
				return nil, fmt.Errorf("create form field %s error:%s", key, err.Error())
			}
		}
	}
	w.Close()
	return b.Bytes(), nil
}

func parseInputWithRegexp(input, regx string) ([]string, error) {
	re := regexp.MustCompile(regx)
	matches := re.FindStringSubmatch(input)
	if len(matches) < 1 {
		return nil, fmt.Errorf("could not parse the provided input; input = %v", input)
	}
	return matches, nil
}

func bool2int(b bool) int {
	if b {
		return 1
	}
	return 0
}
