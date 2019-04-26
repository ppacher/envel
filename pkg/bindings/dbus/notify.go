package dbus

import (
	"github.com/esiqveland/notify"
	"github.com/godbus/dbus"
	"github.com/ppacher/envel/pkg/signal"
	lua "github.com/yuin/gopher-lua"
)

// AddNotify adds the notify library
func AddNotify(L *lua.LState, t *lua.LTable) {
	notifyTable := L.NewTable()

	conn, err := GetConnection(L)
	if err != nil {
		L.RaiseError(err.Error())
		return
	}

	notifier, err := notify.New(conn)
	if err != nil {
		L.RaiseError(err.Error())
		return
	}

	mt := L.NewTable()
	L.SetFuncs(mt, map[string]lua.LGFunction{
		"__call": func(L *lua.LState) int {
			return sendNotification(L, notifier)
		},
	})

	notifyIndex := L.NewTable()
	L.SetField(mt, "__index", notifyIndex)

	L.SetMetatable(notifyTable, mt)

	_, sig := signal.Extend(L, notifyTable)

	go func() {
		for msg := range notifier.NotificationClosed() {
			sig.EmitFrom("notification::closed", func(L *lua.LState) []lua.LValue {
				return []lua.LValue{
					lua.LNumber(int(msg.ID)),
					lua.LString(msg.Reason.String()),
				}
			})
		}
	}()

	go func() {
		for msg := range notifier.ActionInvoked() {
			sig.EmitFrom("notification::action", func(L *lua.LState) []lua.LValue {
				return []lua.LValue{
					lua.LNumber(int(msg.ID)),
					lua.LString(msg.ActionKey),
				}
			})
		}
	}()

	L.SetField(t, "notify", notifyTable)
}

func sendNotification(L *lua.LState, notifier notify.Notifier) int {
	msg := notificationFromTable(L, L.CheckTable(2))

	id, err := notifier.SendNotification(*msg)
	if err != nil {
		L.Push(lua.LNil)
		L.Push(lua.LString(err.Error()))
		return 2
	}

	L.Push(lua.LNumber(id))

	return 1
}

// https://developer.gnome.org/notification-spec/#urgency-levels
var urgencyLevels = map[string]uint8{
	"low":      0,
	"normal":   1,
	"critical": 2,
}

func notificationFromTable(L *lua.LState, t *lua.LTable) *notify.Notification {
	n := notify.Notification{
		Hints:         map[string]dbus.Variant{},
		ExpireTimeout: -1, // default to use the notifcation servers settings
	}

	app := t.RawGetString("app")
	if t, ok := app.(lua.LString); ok {
		n.AppName = string(t)
	} else if app != lua.LNil {
		L.RaiseError("app must be a string or nil, got: %s", app.Type().String())
	}

	replaces := t.RawGetString("replace")
	if r, ok := replaces.(lua.LNumber); ok {
		n.ReplacesID = uint32(r)
	} else if replaces != lua.LNil {
		L.RaiseError("replaces must be a number or nil, got: %s", replaces.Type().String())
	}

	icon := t.RawGetString("icon")
	if t, ok := icon.(lua.LString); ok {
		n.AppIcon = string(t)
	} else if icon != lua.LNil {
		L.RaiseError("icon must be a string or nil, got: %s", icon.Type().String())
	}

	title := t.RawGetString("title")
	if t, ok := title.(lua.LString); ok {
		n.Summary = string(t)
	} else if title != lua.LNil {
		L.RaiseError("title must be a string or nil, got: %s", title.Type().String())
	}

	text := t.RawGetString("text")
	if t, ok := text.(lua.LString); ok {
		n.Body = string(t)
	} else if text != lua.LNil {
		L.RaiseError("text must be a string or nil, got: %s", text.Type().String())
	}

	timeout := t.RawGetString("timeout")
	if r, ok := timeout.(lua.LNumber); ok {
		n.ExpireTimeout = int32(float64(r) * 1000)
	} else if timeout != lua.LNil {
		L.RaiseError("timeout must be a number or nil, got: %s", timeout.Type().String())
	}

	urgency := t.RawGetString("urgency")
	if urgency != lua.LNil {
		switch v := urgency.(type) {
		case lua.LString:
			level := urgencyLevels[v.String()]
			if level == 0 && v.String() != "low" {
				L.RaiseError("invalid urgency level. Use 0-2 or low, normal, critical")
			}
			n.Hints["urgency"] = dbus.MakeVariant(level)
		case lua.LNumber:
			n.Hints["urgency"] = dbus.MakeVariant(int(v))
		default:
			L.RaiseError("invalid urgency value. Expected string or number")
		}
	}

	if n.Summary == "" || n.Body == "" {
		L.RaiseError("invalid notification. At least title or text must be set")
	}

	return &n
}
