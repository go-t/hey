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
	"regexp"
	"strings"
)

const (
	formRegexp   = `^([\w-]+)=(.+)`
	headerRegexp = `^([\w-]+):\s*(.+)`
	authRegexp   = `^(.+):([^\s].+)`
	boundary     = "--179CB67133D24A71A3DA3CC21F6F375F--"
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
)

func init() {
	flag.Var(&headerslice, "H", "")
	flag.Var(&queryslice, "d", "")
	flag.Var(&formslice, "F", "")
}

func makeRequest(url string) (*http.Request, []byte, error) {
	if bool2int(len(queryslice) != 0)+bool2int(len(formslice) != 0)+bool2int(len(*bodyFile) != 0) > 1 {
		return nil, nil, errors.New("conflict flag -F, -d, -D")
	}

	var (
		err    error
		body   []byte
		ctype  string
		method string
	)
	if len(queryslice) != 0 {
		ctype = "application/x-www-form-urlencoded"
		body, err = urlEncoded(queryslice, gourl.QueryEscape)
	} else if len(formslice) != 0 {
		ctype = "multipart/form-data; boundary=" + boundary
		body, err = formData(formslice)
	} else if *bodyFile != "" {
		ctype = *contentType
		body, err = ioutil.ReadFile(*bodyFile)
	}
	if err != nil {
		return nil, nil, err
	}
	header := make(http.Header)
	header.Set("Content-Type", ctype)
	if len(body) > 0 {
		header.Set("Content-Length", fmt.Sprintf("%d", len(body)))
	}

	if *m == "" {
		if len(body) > 0 {
			method = "POST"
		} else {
			method = "GET"
		}
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
	return req, body, nil
}

// -data value
// -data =value
// -data name=value
// -data name=@file
// -data name@file
func urlEncoded(data stringSlice, escape func(string) string) ([]byte, error) {
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
			data, err := ioutil.ReadFile(parts[1])
			if err != nil {
				return nil, fmt.Errorf("load file %s error:%s", parts[1], err.Error())
			}
			w.WriteString(escape(parts[0]))
			w.WriteString(escape(string(data)))
		} else {
			w.WriteString(escape(value))
		}
		w.WriteByte('&')
		w.Flush()
	}
	return b.Bytes(), nil
}

// -form name=value
// -form name=@file
func formData(form stringSlice) ([]byte, error) {
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
			data, err := ioutil.ReadFile(value[1:])
			if err != nil {
				return nil, fmt.Errorf("load file %s error:%s", value[1:], err.Error())
			}

			field, err := w.CreateFormFile(key, value[1:])
			if err != nil {
				return nil, fmt.Errorf("create form field %s error:%s", key, err.Error())
			}
			field.Write(data)
			w.Close()
		} else {
			err := w.WriteField(key, value)
			if err != nil {
				return nil, fmt.Errorf("create form field %s error:%s", key, err.Error())
			}
		}
	}
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
