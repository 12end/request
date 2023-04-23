package raw

import (
	"compress/gzip"
	"io"
	"net/http"
	"strings"

	"github.com/12end/request/raw/client"
)

// StatusError is a HTTP status error object
type StatusError struct {
	client.Status
}

func (s *StatusError) Error() string {
	return s.Status.String()
}

type readCloser struct {
	io.Reader
	io.Closer
}

func toRequest(method string, path string, query []string, headers map[string][]string, body io.Reader, options *Options) *client.Request {
	if len(options.CustomRawBytes) > 0 {
		return &client.Request{RawBytes: options.CustomRawBytes}
	}
	reqHeaders := toHeaders(headers)
	if len(options.CustomHeaders) > 0 {
		reqHeaders = options.CustomHeaders
	}

	return &client.Request{
		Method:  method,
		Path:    path,
		Query:   query,
		Version: client.HTTP_1_1,
		Headers: reqHeaders,
		Body:    body,
	}
}
func toHTTPResponse(conn Conn, resp *client.Response) (*http.Response, error) {
	rheaders := fromHeaders(resp.Headers)
	r := http.Response{
		ProtoMinor:    resp.Version.Minor,
		ProtoMajor:    resp.Version.Major,
		Status:        resp.Status.String(),
		StatusCode:    resp.Status.Code,
		Header:        rheaders,
		ContentLength: resp.ContentLength(),
	}

	var err error
	rbody := resp.Body
	if headerValue(rheaders, "Content-Encoding") == "gzip" {
		rbody, err = gzip.NewReader(rbody)
		if err != nil {
			return nil, err
		}
	}
	rc := &readCloser{rbody, conn}

	r.Body = rc

	return &r, nil
}

func toHeaders(h map[string][]string) []client.Header {
	var r []client.Header
	for k, v := range h {
		for _, v := range v {
			r = append(r, client.Header{Key: k, Value: v})
		}
	}
	return r
}

func fromHeaders(h []client.Header) map[string][]string {
	if h == nil {
		return nil
	}
	var r = make(map[string][]string)
	for _, hh := range h {
		r[hh.Key] = append(r[hh.Key], hh.Value)
	}
	return r
}

func headerValue(headers map[string][]string, key string) string {
	return strings.Join(headers[key], " ")
}

func firstErr(err1, err2 error) error {
	if err1 != nil {
		return err1
	}
	if err2 != nil {
		return err2
	}
	return nil
}

func HasPrefixAny(s string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}
