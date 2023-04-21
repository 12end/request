package request

import (
	"bufio"
	"bytes"
	"fmt"
	"github.com/12end/tls"
	"github.com/valyala/fasthttp"
	"io"
	"mime/multipart"
	"net/textproto"
	"strings"
	"sync"
	"time"
)

var requestPool sync.Pool

type Params map[string]string
type Data map[string]string // for post form
type Files map[string]File  // name ,file-content
type File struct {
	FileName    string
	ContentType string
	Content     []byte
}

var defaultClient = fasthttp.Client{
	TLSConfig: &tls.Config{InsecureSkipVerify: true, MinVersion: tls.VersionSSL30},
}

// AcquireRequest returns an empty Request instance from request pool.
//
// The returned Request instance may be passed to ReleaseRequest when it is
// no longer needed. This allows Request recycling, reduces GC pressure
// and usually improves performance.
func AcquireRequest() *Request {
	v := requestPool.Get()
	if v == nil {
		return &Request{
			Request: fasthttp.AcquireRequest(),
		}
	}
	return v.(*Request)
}

// ReleaseRequest returns req acquired via AcquireRequest to request pool.
//
// It is forbidden accessing req and/or its' members after returning
// it to request pool.
func ReleaseRequest(req *Request) {
	req.Reset()
	req.Trace = nil
	requestPool.Put(req)
}

type TraceInfo struct {
	Request  string
	Response string
	Duration time.Duration
}

type Request struct {
	*fasthttp.Request
	Trace        *[]TraceInfo
	maxRedirects int
}

func (r *Request) Reset() {
	r.Trace = nil
	r.maxRedirects = 0
	fasthttp.ReleaseRequest(r.Request)
}

func (r *Request) SetMaxRedirects(t int) *Request {
	r.maxRedirects = t
	return r
}

func (r *Request) String() string {
	return r.Request.String()
}

func (r *Request) Method(method string) *Request {
	r.Request.Header.SetMethod(method)
	return r
}

func (r *Request) URL(url string) *Request {
	r.Request.SetRequestURIBytes([]byte(url))
	return r
}

func (r *Request) WithTrace(t *[]TraceInfo) *Request {
	r.Trace = t
	return r
}

func (r *Request) SetParams(p Params) *Request {
	for k, v := range p {
		r.Request.URI().QueryArgs().Set(k, v)
	}
	return r
}

func (r *Request) SetTimeout(t time.Duration) *Request {
	r.Request.SetTimeout(t)
	return r
}

func (r *Request) SetData(p Data) *Request {
	for k, v := range p {
		r.Request.PostArgs().Set(k, v)
	}
	return r
}

func (r *Request) DisableNormalizing() *Request {
	r.Request.Header.DisableNormalizing()
	r.Request.URI().DisablePathNormalizing = true
	return r
}

func (r *Request) BodyRaw(s string) *Request {
	r.Request.SetBodyRaw([]byte(s))
	return r
}

func (r *Request) FromRaw(s string) error {
	return r.Request.Read(bufio.NewReader(strings.NewReader(s)))
}

func (r *Request) Host(host string) *Request {
	r.Request.UseHostHeader = true
	r.Request.Header.SetHostBytes([]byte(host))
	return r
}

func (r *Request) MultipartFiles(fs Files) *Request {
	var b bytes.Buffer
	w := multipart.NewWriter(&b)
	defer w.Close()

	for n, f := range fs {
		h := make(textproto.MIMEHeader)
		if f.FileName != "" {
			h.Set("filename", f.FileName)
		}
		if f.ContentType != "" {
			h.Set("Content-Type", f.ContentType)
		}
		part, err := w.CreatePart(h)
		//part, err := w.CreateFormFile(fieldName, f.FileName)
		if err != nil {
			fmt.Printf("Upload %s failed!", n)
			panic(err)
		}
		if len(f.Content) > 0 {
			reader := bytes.NewReader(f.Content)
			_, _ = io.Copy(part, reader)
		}
	}

	r.Request.Header.SetMultipartFormBoundary(w.Boundary())
	r.Request.SetBodyRaw(b.Bytes())
	return r
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func (r *Request) Do(resp *Response) error {
	resp.body = ""
	resp.title = ""
	if r.maxRedirects > 1 {
		return defaultClient.DoRedirects(r.Request, resp.Response, r.maxRedirects)
	} else {
		return defaultClient.Do(r.Request, resp.Response)
	}
}

func (r *Request) DoWithTrace(resp *Response) error {
	if r.Trace == nil {
		return r.Do(resp)
	}
	resp.body = ""
	resp.title = ""
	start := time.Now()
	err := defaultClient.Do(r.Request, resp.Response)
	if err != nil {
		return err
	}
	*r.Trace = append(*r.Trace, TraceInfo{
		Request:  r.String(),
		Response: resp.String(),
		Duration: time.Since(start),
	})
	return nil
}
