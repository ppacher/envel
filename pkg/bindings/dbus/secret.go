package dbus

import (
	keyring "github.com/ppacher/go-dbus-keyring"
	lua "github.com/yuin/gopher-lua"
)

// AddSecret adds the SecretService keyring API
func AddSecret(L *lua.LState, t *lua.LTable) {
	secret := L.NewTable()

	L.SetFuncs(secret, map[string]lua.LGFunction{
		"get": secretGet,
	})

	t.RawSetString("secret", secret)
}

func secretGet(L *lua.LState) int {
	conn, err := GetConnection(L)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	var collection string
	var secretLabel string

	if L.GetTop() == 2 {
		collection = L.CheckString(1)
		secretLabel = L.CheckString(2)
	} else {
		secretLabel = L.CheckString(1)
	}

	svc, err := keyring.GetSecretService(conn)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	var col keyring.Collection

	if collection != "" {
		col, err = svc.GetCollection(collection)
	} else {
		col, err = svc.GetDefaultCollection()
	}
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	item, err := col.GetItem(secretLabel)
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	session, err := svc.OpenSession()
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	defer session.Close()

	locked, err := item.Locked()
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	// try to unlock the item if it's locked
	if locked {
		_, err := item.Unlock()
		if err != nil {
			L.RaiseError(err.Error())
			return 0
		}
	}

	secret, err := item.GetSecret(session.Path())
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	result := string(secret.Value)
	L.Push(lua.LString(result))

	attr, err := item.GetAttributes()
	if err != nil {
		L.RaiseError(err.Error())
		return 0
	}

	if attr["username_value"] != "" {
		L.Push(lua.LString(attr["username_value"]))
		return 2
	}

	return 1
}
