package dbus

import (
	"github.com/godbus/dbus"
	lua "github.com/yuin/gopher-lua"
)

// GetConnection returns a dbus connection to the session bus
func GetConnection(L *lua.LState) (*dbus.Conn, error) {
	cud := L.GetGlobal("__dbus_connection")
	if cud == lua.LNil {
		cud = L.NewUserData()

		c, err := dbus.SessionBus()
		if err != nil {
			return nil, err
		}

		cud.(*lua.LUserData).Value = c

		L.SetGlobal("__dbus_connection", cud)
	}

	return cud.(*lua.LUserData).Value.(*dbus.Conn), nil
}
