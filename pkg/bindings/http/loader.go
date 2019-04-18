package http

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/ppacher/envel/pkg/bindings/core"

	lua "github.com/yuin/gopher-lua"
)

func Preload(L *lua.LState) {
	L.PreloadModule("envel.http", Loader)
}

func Loader(L *lua.LState) int {
	tbl := L.NewTable()

	L.SetMetatable(tbl, L.SetFuncs(L.NewTable(), map[string]lua.LGFunction{
		"__call": httpDo,
	}))

	L.Push(tbl)
	return 1
}

func assertString(L *lua.LState, value lua.LValue, key string) string {
	val := L.GetField(value, key)
	if val == lua.LNil {
		return ""
	}

	if v, ok := val.(lua.LString); ok {
		return string(v)
	}

	L.RaiseError(fmt.Sprintf("expected a string for property %s, got %s", key, val.Type().String()))

	return ""
}

func assertTable(L *lua.LState, value lua.LValue, key string, def *lua.LTable) *lua.LTable {
	v := L.GetField(value, key)
	if v == lua.LNil {
		return def
	}

	if t, ok := v.(*lua.LTable); ok {
		return t
	}

	L.RaiseError("expected a table for " + key + " but got " + v.Type().String())
	return def
}

func httpDo(L *lua.LState) int {
	request := L.CheckTable(2)

	method := assertString(L, request, "method")
	urlS := assertString(L, request, "url")
	body := assertString(L, request, "url")
	headers := assertTable(L, request, "headers", nil)

	httpHeader := make(http.Header)

	if headers != nil {
		headers.ForEach(func(key, value lua.LValue) {
			headerName, ok := key.(lua.LString)
			if !ok {
				L.ArgError(1, "HTTP headers must have string keys")
				return
			}

			if headerValue, ok := value.(lua.LString); ok {
				httpHeader[string(headerName)] = append(httpHeader[string(headerName)], string(headerValue))
			}

			if headerValues, ok := value.(*lua.LTable); ok {
				headerValues.ForEach(func(k, v lua.LValue) {
					if _, ok := k.(lua.LNumber); !ok {
						L.ArgError(1, "HTTP header fields must either be strings or list of strings")
						return
					}

					if s, ok := v.(lua.LString); ok {
						httpHeader[string(headerName)] = append(httpHeader[string(headerName)], string(s))
						return
					}

					L.ArgError(1, "HTTP header fields must either be strings or list of strings")
				})
			}
		})
	}

	u, err := url.Parse(urlS)
	if err != nil {
		L.RaiseError("expected an url: " + err.Error())
		return 0
	}

	req := &http.Request{
		Method: strings.ToUpper(method),
		URL:    u,
		Body:   ioutil.NopCloser(bytes.NewBufferString(body)),
		Header: httpHeader,
	}

	cli := http.Client{
		Timeout: time.Second * 30, // TODO(ppacher): make it configurable
	}

	res, err := cli.Do(req)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))

		log.Printf("http request failed: %s\n", err.Error())
		return 2
	}

	L.Push(convertResponseToTable(L, res))

	return 1
}

func convertResponseToTable(L *lua.LState, res *http.Response) *lua.LTable {
	t := L.NewTable()

	t.RawSetString("status", lua.LString(res.Status))
	t.RawSetString("status_code", lua.LNumber(res.StatusCode))

	headers := L.NewTable()
	for key, values := range res.Header {
		ht := L.NewTable()
		for _, v := range values {
			ht.Append(lua.LString(v))
		}

		headers.RawSetString(key, ht)
	}

	reader, _ := core.NewReader(L, res.Body)

	t.RawSetString("body", reader)

	cookies := L.NewTable()
	for _, cookie := range res.Cookies() {
		cookies.Append(convertCookieToTable(L, cookie))
	}

	t.RawSetString("cookies", cookies)

	return t
}

func convertCookieToTable(L *lua.LState, cookie *http.Cookie) *lua.LTable {
	c := L.NewTable()

	c.RawSetString("name", lua.LString(cookie.Name))
	c.RawSetString("value", lua.LString(cookie.Value))
	c.RawSetString("path", lua.LString(cookie.Path))
	c.RawSetString("domain", lua.LString(cookie.Domain))
	c.RawSetString("expires", lua.LNumber(cookie.Expires.Unix()))
	c.RawSetString("max_age", lua.LNumber(cookie.MaxAge))
	c.RawSetString("secure", lua.LBool(cookie.Secure))
	c.RawSetString("http_only", lua.LBool(cookie.Secure))

	return c
}
