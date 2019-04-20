package mqtt

import (
	"log"

	mqtt "github.com/eclipse/paho.mqtt.golang"
	"github.com/ppacher/envel/pkg/bindings/callback"
	"github.com/ppacher/envel/pkg/loop"
	lua "github.com/yuin/gopher-lua"
)

const mqttTypeName = "mqtt"

func createMQTTTypeTable(L *lua.LState, t *lua.LTable) {
	typeMT := L.NewTypeMetatable(mqttTypeName)

	L.SetField(typeMT, "__index", L.SetFuncs(L.NewTable(), mqttTypeAPI))
	L.SetField(t, "__mqtt_mt", typeMT)
}

var mqttTypeAPI = map[string]lua.LGFunction{
	"unsubscribe": mqttUnsubscribe,
	"subscribe":   mqttSubscribe,
	"publish":     mqttPublish,
	"close":       mqttClose,
}

// MQTT is stored inside a UserData Lua value and provides access to
// the mqtt client
type MQTT struct {
	mqtt.Client
}

func checkMQTT(L *lua.LState) *MQTT {
	ud := L.CheckUserData(1)
	if m, ok := ud.Value.(*MQTT); ok {
		return m
	}

	L.ArgError(1, "Expected an mqttClient")

	return nil
}

// NewMQTTClient creates a new MQTT client
func NewMQTTClient(opts *mqtt.ClientOptions) (*MQTT, error) {
	c := mqtt.NewClient(opts)

	// connect to the brokers
	if token := c.Connect(); token.Wait() && token.Error() != nil {
		return nil, token.Error()
	}

	return &MQTT{
		Client: c,
	}, nil
}

func newMQTT(L *lua.LState) int {
	opts := L.CheckTable(2)

	cfg := mqtt.NewClientOptions()

	// TODO(ppacher): add support for multiple brokers
	broker := opts.RawGetString("broker")
	if b, ok := broker.(lua.LString); ok {
		cfg.AddBroker(string(b))
	} else {
		L.ArgError(1, "broker must be set to a string")
	}

	clientID := opts.RawGetString("client_id")
	if c, ok := clientID.(lua.LString); ok {
		cfg.SetClientID(string(c))
	} else {
		L.ArgError(1, "client_id must be set to a string")
	}

	cleanSession := opts.RawGetString("clean_session")
	if c, ok := cleanSession.(lua.LBool); ok {
		cfg.SetCleanSession(bool(c))
	} else if cleanSession != lua.LNil {
		L.ArgError(1, "clean_session must be nil or boolean")
	}

	usernameValue := opts.RawGetString("username")
	if c, ok := usernameValue.(lua.LString); ok {
		cfg.SetUsername(string(c))
	} else if usernameValue != lua.LNil {
		L.ArgError(1, "username must be nil or a string")
	}

	passwordValue := opts.RawGetString("password")
	if c, ok := passwordValue.(lua.LString); ok {
		cfg.SetPassword(string(c))
	} else if passwordValue != lua.LNil {
		L.ArgError(1, "password must be nil or a string")
	}

	// we always enable auto-reconnect
	cfg.SetAutoReconnect(true)

	mq, err := NewMQTTClient(cfg)
	if err != nil {
		L.RaiseError("mqtt: %s", err.Error())
		return 0
	}

	ud := L.NewUserData()
	ud.Value = mq
	L.SetMetatable(ud, L.GetTypeMetatable(mqttTypeName))

	L.Push(ud)

	return 1
}

func mqttSubscribe(L *lua.LState) int {
	mq := checkMQTT(L)

	opts := L.CheckTable(2)

	topic := opts.RawGetString("topic")
	if _, ok := topic.(lua.LString); !ok {
		L.ArgError(1, "topic must be set to a string")
	}

	qos := opts.RawGetString("qos")
	if _, ok := qos.(lua.LNumber); !ok {
		if qos == lua.LNil {
			qos = lua.LNumber(0)
		} else {
			L.ArgError(1, "qos must be set to nil or a number")
		}
	}

	fn := opts.RawGetString("callback")
	if _, ok := fn.(*lua.LFunction); !ok {
		L.ArgError(1, "callback must be set to a function")
	}

	cb := callback.New(fn.(*lua.LFunction), loop.LGet(L))

	token := mq.Subscribe(
		topic.(lua.LString).String(),
		byte(qos.(lua.LNumber)),
		func(cli mqtt.Client, msg mqtt.Message) {
			<-cb.From(func(L *lua.LState) []lua.LValue {
				t := L.NewTable()
				L.SetField(t, "body", lua.LString(msg.Payload()))
				L.SetField(t, "topic", lua.LString(msg.Topic()))
				L.SetField(t, "duplicate", lua.LBool(msg.Duplicate()))

				return []lua.LValue{t}
			})
		},
	)
	token.Wait()

	if token.Error() != nil {
		L.RaiseError(token.Error().Error())
	}

	return 0
}

func mqttPublish(L *lua.LState) int {
	mq := checkMQTT(L)
	opts := L.CheckTable(2)

	msgTopic := ""
	msgQoS := byte(0)
	msgRetained := false
	msgPayload := ""

	topic := opts.RawGetString("topic")
	if v, ok := topic.(lua.LString); ok {
		msgTopic = string(v)
	} else {
		L.ArgError(1, "topic must be set to a string")
	}

	qos := opts.RawGetString("qos")
	if v, ok := qos.(lua.LNumber); ok {
		msgQoS = byte(v)
	} else if qos != lua.LNil {
		L.ArgError(1, "qos must be set to nil or a number")
	}

	payload := opts.RawGetString("payload")
	if v, ok := payload.(lua.LString); ok {
		msgPayload = v.String()
	} else {
		L.ArgError(1, "payload must be set to a string")
	}

	retained := opts.RawGetString("retained")
	if v, ok := retained.(lua.LBool); ok {
		msgRetained = bool(v)
	} else if retained != lua.LNil {
		L.ArgError(1, "retained must be set to a bool")
	}

	token := mq.Publish(msgTopic, msgQoS, msgRetained, msgPayload)

	token.Wait()

	if token.Error() != nil {
		log.Printf("Error: %s\n", token.Error())
		L.RaiseError(token.Error().Error())
	}

	return 0
}

func mqttClose(L *lua.LState) int {
	mq := checkMQTT(L)
	// do not block the event loop
	go func() {
		mq.Disconnect(100)
	}()

	return 0
}

func mqttUnsubscribe(L *lua.LState) int {
	mq := checkMQTT(L)

	topics := []string{}
	for i := 2; i < L.GetTop(); i++ {
		t := L.CheckString(i)
		topics = append(topics, t)
	}

	if token := mq.Unsubscribe(topics...); token.Wait() && token.Error() != nil {
		L.RaiseError(token.Error().Error())
	}

	return 0
}
