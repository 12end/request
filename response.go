package request

import (
	"bytes"
	"github.com/valyala/fasthttp"
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
	Resp  *fasthttp.Response
	body  string
	title string
}

func (r *Response) Reset() {
	fasthttp.ReleaseResponse(r.Resp)
	r.title = ""
	r.body = ""
}

func (r *Response) GetHeader(k string) (string, bool) {
	vb := r.Resp.Header.Peek(k)
	if vb == nil {
		return "", false
	} else {
		return b2s(vb), true
	}
}

func (r *Response) Body() string {
	if r.body != "" {
		return r.body
	}
	body, err := r.Resp.BodyUncompressed()
	if err != nil {
		body = r.Resp.Body()
	}
	r.body = b2s(body)
	return r.body
}

func (r *Response) Title() string {
	if r.title != "" {
		return r.title
	}
	find := titleReg.FindStringSubmatch(r.Body())
	if len(find) > 1 {
		r.title = find[1]
		r.title = emptyReg.ReplaceAllString(html.UnescapeString(r.title), "")
		r.title = strings.TrimSpace(r.title)
	}
	return r.title
}

func (r *Response) BodyContains(s string) bool {
	return strings.Contains(r.Body(), s)
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
	match := reg.FindStringSubmatch(r.Body())
	for i, name := range reg.SubexpNames() {
		if i != 0 && name != "" {
			result[name] = match[i]
		}
	}
	return result
}
