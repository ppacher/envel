package core

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	"io/ioutil"

	"github.com/ppacher/envel/pkg/callback"
	lua "github.com/yuin/gopher-lua"
)

const readerTypeName = "reader"

// AddReader is the module loader function for lualib.reader
func AddReader(L *lua.LState, m *lua.LTable) {
	t := L.NewTable()
	L.SetFuncs(t, readerAPI)

	typeMt := L.NewTypeMetatable(readerTypeName)

	L.SetField(typeMt, "__index", L.SetFuncs(L.NewTable(), readerTypeAPI))
	t.RawSetString("__reader_mt", typeMt)

	m.RawSetString("reader", t)
}

// Reader wraps the io.Reader interface into a dedicated type exposed to
// Lua
type Reader struct {
	io.Reader
}

// Close implements io.ReadCloser
func (r *Reader) Close() error {
	if c, ok := r.Reader.(io.Closer); ok {
		return c.Close()
	}

	return fmt.Errorf("Not an io.ReadCloser")
}

// readerTypeAPI defines all methods available on reader objects
// (using metatables)
var readerTypeAPI = map[string]lua.LGFunction{
	"read":               readerRead,
	"close":              readerClose,
	"with_line_callback": withLineCallback,
}

// readerAPI defines all methods available on the returned module
// table
var readerAPI = map[string]lua.LGFunction{
	"from_string": fromString,
}

// NewReader is a utility method that creates a new reader object. The returned LUserData still
// needs to be pushed to the Lua stack using L.Push
func NewReader(L *lua.LState, source io.Reader) (*lua.LUserData, *Reader) {
	reader := &Reader{
		Reader: source,
	}

	ud := L.NewUserData()
	ud.Value = reader
	L.SetMetatable(ud, L.GetTypeMetatable(readerTypeName))

	return ud, reader
}

// checkReader checks if the first paramter is a Reader UserData
// or errors
func checkReader(L *lua.LState) *Reader {
	ud := L.CheckUserData(1)
	if v, ok := ud.Value.(*Reader); ok {
		return v
	}
	L.ArgError(1, "person expected")
	return nil
}

// readerRead reads all data from the reader and should be used in lue like `r:read()`
func readerRead(L *lua.LState) int {
	r := checkReader(L)

	mode := L.Get(2)

	if mode.Type() == lua.LTNumber {
		size := int(mode.(lua.LNumber))
		buffer := make([]byte, size)

		n, err := r.Read(buffer)
		if n == 0 && err == io.EOF {
			L.Push(lua.LNil)
			return 1
		}

		L.Push(lua.LString(buffer[:n]))
		return 1
	}

	if mode == lua.LNil {
		// lua defaults to *line if no read mode is given
		mode = lua.LString("*line")
	}

	if mode.Type() != lua.LTString {
		// error
		L.ArgError(2, fmt.Sprintf("Exected string or number but got %#v", mode))
	}

	mStr := mode.(lua.LString).String()

	if mStr == "*all" || mStr == "*a" {
		data, err := ioutil.ReadAll(r)
		// ioutil.ReadAll() never returns io.EOF as an error so we
		// are safe to raise as soon as an error is returned
		if err != nil {
			L.RaiseError(err.Error())
		}

		if len(data) == 0 {
			// check if we reached EOF
			// find a better solution here because if the read now works we lost one byte
			// to garbage collection
			if _, err := r.Read(make([]byte, 1)); err == io.EOF {
				L.Push(lua.LNil)
				return 1
			}
		}

		L.Push(lua.LString(data))
		return 1
	}

	if mStr == "*line" || mStr == "*l" {
		line := ""
		for {
			b := make([]byte, 1)
			n, err := r.Read(b)
			if n == 1 {
				if b[0] == '\n' {
					L.Push(lua.LString(line))
					return 1
				}
				line = line + string(b)
			}

			if err != nil {
				if err == io.EOF {
					if len(line) > 0 {
						L.Push(lua.LString(line))
						return 1
					}

					L.Push(lua.LNil)
					return 1
				}

				L.RaiseError(err.Error())
				return 0
			}
		}
	}

	L.ArgError(2, "Invalid read mode")
	return 0
}

func isWhitespace(n byte) bool {
	return n == '\n' || n == '\t' || n == ' '
}

func isNumber(n byte) bool {
	return n >= byte('0') && n <= byte('9')
}

func isPunct(n byte) bool {
	return n == '.'
}

// readerClose provides `r:close()` and closes the underlying reader if it also implements io.Closer
// an error is returned otherwise
func readerClose(L *lua.LState) int {
	r := checkReader(L)

	if closer, ok := r.Reader.(io.Closer); ok {
		closer.Close()
		return 0
	}

	L.RaiseError("Reader cannot be closed")
	return 1
}

// withLineCallback provides a `l:with_line_callback()` method that reads data from
// a *Reader asynchronously and calls the provided method for each line
func withLineCallback(L *lua.LState) int {
	r := checkReader(L)
	lineCb := callback.LGet(2, L)

	r.WithLineCallback(lineCb, nil)

	return 0
}

// LineCallbackOptions holds additional options for Reader.WithLineCallback
type LineCallbackOptions struct {
	// PrefixArgs holds a list of arguments that should be passed to the callback
	// before the actual line
	PrefixArgs []lua.LValue

	// SuffixArgs holds a list of arguments that should be passed to the callback
	// after the actual line
	SuffixArgs []lua.LValue
}

// WithLineCallback starts reading from the reader and calls the provided
// callback for each line
func (r *Reader) WithLineCallback(lineCb callback.Callback, opts *LineCallbackOptions) {
	lineReader := bufio.NewReader(r)

	var prefix []lua.LValue
	var suffix []lua.LValue

	if opts != nil {
		prefix = opts.PrefixArgs
		suffix = opts.SuffixArgs
	}

	go func() {
		for {
			d, err := lineReader.ReadString('\n')

			// remove the delimiter if it's set
			if len(d) > 0 && d[len(d)-1] == '\n' {
				d = d[:len(d)-1]
			}

			args := append(prefix, lua.LString(d))
			args = append(args, suffix...)
			if d != "" {
				<-lineCb.Do(args...)
			}

			if err != nil {
				if err == io.EOF {
					<-lineCb.Do(lua.LNil)
				} else {
					<-lineCb.Do(lua.LNil, lua.LString(err.Error()))
				}
				return
			}
		}
	}()
}

// fromString provides the `reader.from_string()` method that is used
// to create a *Reader from a string inside Lua
func fromString(L *lua.LState) int {
	s := L.CheckString(1)
	r := bytes.NewBufferString(s)

	ud, _ := NewReader(L, r)
	L.Push(ud)
	return 1
}
