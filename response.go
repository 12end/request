package request

import (
	"bytes"
	"github.com/12end/fasthttp"
	"html"
	"regexp"
	"strings"
	"sync"
)

var responsePool sync.Pool
var titleReg = regexp.MustCompile("(?ims)<title.*?>(.*?)</title>")
var emptyReg = regexp.MustCompile(`[\n\r\t]+`)

func AcquireResponse() *Response {
	v := responsePool.Get()
	if v == nil {
		return &Response{
			Resp: fasthttp.AcquireResponse(),
		}
	}
	return v.(*Response)
}

func ReleaseResponse(resp *Response) {
	resp.Reset()
	responsePool.Put(resp)
}

type Response struct {
	Resp   *fasthttp.Response
	body   string
	title  string
	header string
}

func (r *Response) Reset() {
	fasthttp.ReleaseResponse(r.Resp)
	r.title = ""
	r.body = ""
	r.header = ""
}

func (r *Response) GetHeader(k string) (string, bool) {
	vb := r.Resp.Header.Peek(k)
	if vb == nil {
		return "", false
	} else {
		return b2s(vb), true
	}
}

func (r *Response) GetBody() (string, error) {
	if r.body != "" {
		return r.body, nil
	}
	body, err := r.Resp.BodyUncompressed()
	return b2s(body), err
}

func (r *Response) GetTitle() (string, error) {
	if r.title != "" {
		return r.title, nil
	}
	body, err := r.GetBody()
	if err != nil {
		return r.title, err
	}
	find := titleReg.FindStringSubmatch(body)
	if len(find) > 1 {
		r.title = find[1]
		r.title = emptyReg.ReplaceAllString(html.UnescapeString(r.title), "")
		r.title = strings.TrimSpace(r.title)
	}
	return r.title, err
}

func (r *Response) BodyContains(s string) bool {
	b, err := r.GetBody()
	if err != nil {
		return false
	}
	return strings.Contains(b, s)
}

func (r *Response) HeaderContains(s string) bool {
	return bytes.Contains(r.Resp.Header.Header(), s2b(s))
}

func (r *Response) Cookie(k string) (string, bool) {
	v := r.Resp.Header.PeekCookie(k)
	if v == nil {
		return "", false
	}
	return b2s(v), true
}

func (r *Response) String() string {
	return r.Resp.String()
}

func (r *Response) Search(reg *regexp.Regexp) map[string]string {
	result := make(map[string]string)
	body, err := r.GetBody()
	if err != nil {
		return result
	}
	match := reg.FindStringSubmatch(body)
	for i, name := range reg.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	return result
}
