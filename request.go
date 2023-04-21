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
			Req: fasthttp.AcquireRequest(),
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
	Req          *fasthttp.Request
	Trace        *[]TraceInfo
	maxRedirects int
}

func (r *Request) Reset() {
	r.Trace = nil
	r.maxRedirects = 0
	fasthttp.ReleaseRequest(r.Req)
}

func (r *Request) SetMaxRedirects(t int) *Request {
	r.maxRedirects = t
	return r
}

func (r *Request) String() string {
	return r.Req.String()
}

func (r *Request) Method(method string) *Request {
	r.Req.Header.SetMethod(method)
	return r
}

func (r *Request) URL(url string) *Request {
	r.Req.SetRequestURIBytes(s2b(url))
	return r
}

func (r *Request) WithTrace(t *[]TraceInfo) *Request {
	r.Trace = t
	return r
}

func (r *Request) SetParams(p Params) *Request {
	for k, v := range p {
		r.Req.URI().QueryArgs().Set(k, v)
	}
	return r
}

func (r *Request) SetTimeout(t time.Duration) *Request {
	r.Req.SetTimeout(t)
	return r
}

func (r *Request) SetData(p Data) *Request {
	for k, v := range p {
		r.Req.PostArgs().Set(k, v)
	}
	return r
}

func (r *Request) DisableNormalizing() *Request {
	r.Req.Header.DisableNormalizing()
	r.Req.URI().DisablePathNormalizing = true
	return r
}

func (r *Request) BodyRaw(s string) *Request {
	r.Req.SetBodyRaw(s2b(s))
	return r
}

func (r *Request) FromRaw(s string) error {
	return r.Req.Read(bufio.NewReader(strings.NewReader(s)))
}

func (r *Request) Host(host string) *Request {
	r.Req.UseHostHeader = true
	r.Req.Header.SetHostBytes(s2b(host))
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

	r.Req.Header.SetMultipartFormBoundary(w.Boundary())
	r.Req.SetBodyRaw(b.Bytes())
	return r
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func (r *Request) Do(resp *Response) error {
	if r.maxRedirects > 1 {
		return defaultClient.DoRedirects(r.Req, resp.Resp, r.maxRedirects)
	} else {
		return defaultClient.Do(r.Req, resp.Resp)
	}
}

func (r *Request) DoWithTrace(resp *Response) error {
	if r.Trace == nil {
		return r.Do(resp)
	}
	start := time.Now()
	err := defaultClient.Do(r.Req, resp.Resp)
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
