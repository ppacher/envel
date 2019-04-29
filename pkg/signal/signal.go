package signal

import (
	"fmt"
	"sync"

	"github.com/ppacher/envel/pkg/callback"
	lua "github.com/yuin/gopher-lua"
)

const signalTypeName = "signal"

func createSignalTypeMetatable(L *lua.LState, t *lua.LTable) {
	mt := L.NewTypeMetatable(signalTypeName)
	L.SetField(mt, "__index", L.SetFuncs(L.NewTable(), signalTypeAPI))
	L.SetField(t, "__signal_mt", mt)

	L.SetField(t, "extend", L.NewFunction(signalExtend))
}

var signalTypeAPI = map[string]lua.LGFunction{
	"connect_signal":    signalConnect,
	"disconnect_signal": signalDisconnect,
	"emit_signal":       signalEmit,
	"topics":            signalTopics,
	"subscribers":       signalSubscribers,
}

// Signal provides a singal framework for lua
type Signal struct {
	lock      sync.RWMutex
	listeners map[string][]callback.Callback
}

// NewSignal creates a new signal for the lua VM
func NewSignal(L *lua.LState) (*lua.LUserData, *Signal) {
	sig := &Signal{
		listeners: make(map[string][]callback.Callback),
	}

	ud := L.NewUserData()
	ud.Value = sig
	L.SetMetatable(ud, L.GetTypeMetatable(signalTypeName))
	return ud, sig
}

// Extend extends the given table by adding methods for signal handling
// It works by extending the metatable of the provided table. Any existing
// metatable is preserved and a __index metatable moved behind a proxy table
func Extend(L *lua.LState, obj lua.LValue) (*lua.LUserData, *Signal) {

	// obj == newinst
	// getmetatable(obj) == class_mt
	mt := L.GetMetatable(obj) // == class_mt
	if mt == lua.LNil {
		mt = L.NewTable()
	}

	proxy := L.NewTable()
	ud, sig := NewSignal(L)
	L.SetField(proxy, "__index", ud)

	index := L.GetField(mt, "__index")

	if index != lua.LNil {
		lastMt := mt
		for {
			lastMt = L.GetMetatable(index)
			if lastMt == lua.LNil {
				break
			}

			index = L.GetField(lastMt, "__index")
			if index == lua.LNil {
				index = L.NewTable()
				L.SetField(lastMt, "__index", index)
				break
			}
		}

		// attach our proxy as the metatable for __index
		L.SetMetatable(index, proxy)
	} else {
		// theres no __index meta field on the object so
		// we set it to our signals userdata
		L.SetField(mt, "__index", ud)
	}

	L.SetMetatable(obj, mt)

	return ud, sig
}

// CheckSignal get the lua parameter at stack index arg and ensures
// it's a signal or extends a signal object
func CheckSignal(L *lua.LState, arg int) (lua.LValue, *Signal) {
	val := L.Get(arg)
	ud, sig, err := getUDandSignal(L, val)
	if err != nil {
		L.ArgError(arg, "expected a signal")
	}

	return ud, sig
}

// GetSignal searches for a signal userdata inside the metatable chain
// of the provided value and returns the Signal instance.
func GetSignal(L *lua.LState, val lua.LValue) (*Signal, error) {
	_, sig, err := getUDandSignal(L, val)
	return sig, err
}

// getUDandSignal returns the signal instance of the given lua value, if any
func getUDandSignal(L *lua.LState, val lua.LValue) (*lua.LUserData, *Signal, error) {
	for val != lua.LNil {
		if ud, ok := val.(*lua.LUserData); ok {
			if sig, ok := ud.Value.(*Signal); ok {
				return ud, sig, nil
			}
		}

		val = L.GetMetatable(val)
		if val == lua.LNil {
			break
		}

		val = L.GetField(val, "__index")
	}

	return nil, nil, fmt.Errorf("failed to find signal")
}

func checkSignal(L *lua.LState) *Signal {
	_, sig := CheckSignal(L, 1)
	return sig
}

func newSignal(L *lua.LState) int {
	ud, _ := NewSignal(L)
	L.Push(ud)
	return 1
}

func signalConnect(L *lua.LState) int {
	sig := checkSignal(L)
	name := L.CheckString(2)
	handler := callback.LGet(3, L)

	sig.lock.Lock()
	defer sig.lock.Unlock()

	sig.listeners[name] = append(sig.listeners[name], handler)

	return 0
}

func signalDisconnect(L *lua.LState) int {
	sig := checkSignal(L)
	name := L.CheckString(2)
	handler := L.CheckFunction(3)

	sig.lock.Lock()
	defer sig.lock.Unlock()

	handlers := sig.listeners[name]
	// find the actual callback value
	for i := 0; i < len(handlers); i++ {
		if handlers[i].Callable() != handler {
			continue
		}

		copy(handlers[i:], handlers[i+1:])
		handlers[len(handlers)-1] = nil
		handlers = handlers[:len(handlers)-1]

		if len(handlers) == 0 {
			delete(sig.listeners, name)
		}

		break
	}

	return 0
}

func signalTopics(L *lua.LState) int {
	sig := checkSignal(L)

	tb := L.NewTable()

	for _, t := range sig.Topics() {
		tb.Append(lua.LString(t))
	}
	L.Push(tb)

	return 1
}

func signalSubscribers(L *lua.LState) int {
	sig := checkSignal(L)
	topic := L.CheckString(2)

	L.Push(lua.LNumber(sig.Listeners(string(topic))))

	return 1
}

// Topics returns a slice of topics that have been subscribed
func (sig *Signal) Topics() []string {
	sig.lock.RLock()
	defer sig.lock.RUnlock()

	topic := make([]string, len(sig.listeners))
	i := 0
	for name := range sig.listeners {
		topic[i] = name
		i++
	}

	return topic
}

// Listeners returns the number of subscribers for a given topic
func (sig *Signal) Listeners(topic string) int {
	sig.lock.RLock()
	defer sig.lock.RUnlock()

	return len(sig.listeners[topic])
}

// Emit emits a new signal to any subscriber
func (sig *Signal) Emit(name string, args ...lua.LValue) {
	sig.lock.RLock()
	defer sig.lock.RUnlock()
	for _, cb := range sig.listeners[name] {
		cb.Do(args...)
	}
}

// EmitFrom emits a signal to any subscriber using the returnd value slice
// as spreaded parameters
func (sig *Signal) EmitFrom(name string, fn func(*lua.LState) []lua.LValue) {
	sig.lock.RLock()
	defer sig.lock.RUnlock()
	for _, cb := range sig.listeners[name] {
		cb.From(fn)
	}
}

func signalExtend(L *lua.LState) int {
	table := L.CheckTable(1)

	Extend(L, table)

	L.Push(table)

	return 1
}

func signalEmit(L *lua.LState) int {
	sig := checkSignal(L)
	name := L.CheckString(2)
	args := make([]lua.LValue, L.GetTop()-2)

	for i := 0; i < L.GetTop()-2; i++ {
		args[i] = L.Get(i + 3)
	}

	sig.Emit(name, args...)
	return 0
}
