local mqtt_connect = require("envel.mqtt")
local dbus = require("envel.dbus")
local json = require("json")
local device = require("envel.device")
local hs1xx = require("envel.platform.tplink.hs1xx")
local owm = require("openweathermap")
local notify = dbus.notify
local config = require("config")
local new_pushover = require("pushover")
local rule = require("envel.rules")
local on_exit = _G.on_exit
local stream = require("envel.stream")
local from_signal = stream.from_signal
local smarthome = require("envel.smarthome")

-- add the stream operators
require 'envel.stream.operators'

local mqtt = mqtt_connect {
    broker = "tcp://localhost:1883",
    client_id = "envel",
}
mqtt:publish({topic="envel", payload="started"})

local plug = hs1xx(config.plug.ip)

local home = smarthome.home {
    mqtt = mqtt,
}

local plug_subject = stream.Subject.create()

local test, err = smarthome.interface {
    name = 'hs100_1',
    location = 'bathroom',
    description = 'A test interface',
    metrics = 'gauge',
    items = {
        voltage = {
            unit = 'V',
            source = plug_subject,
        },
        total = {
            unit = 'Wh',
            source = from_signal(plug, 'realtime')
                        :path('total')
        },
    }
}
assert(test, err)

test:bind(home)
test:set_connected()

--os.exit()

-- on_exit allows to configure shutdown handlers when envel's event loop is stopped
-- make sure to not use any features that schedule new tasks on the event loop (i.e. timers, spawn, http, ...)
on_exit(function()
    print("good bye")
end)

local notifier = new_pushover({
    user  = config.pushover.user,
    key = config.pushover.key,
    sound = "none",
})

--[[
notifier:notify({
    text = "<b>Hello from here<b>",
    title = "Test",
    html = true,
    url = 'https://grafana.ppacher.at',
    url_title = 'test',
    priority = 'critical',
})
--]]


local weather = owm(config.owm.key, config.owm.location, config.owm.units)

-- Assuming we have connected our laundry washing machine to
-- a TP-Link HS1xx plug (featuring an energy meter like the HS110)

-- create a client for the plug itself
local plug = hs1xx(config.plug.ip)

-- create a sensor for each property (signal) the plug exposes
-- this will automatically export the values for prometheus
local plug_sensor = device.sensor {
    name = "laundry",
    default_metric = "gauge",
    distinct = false, -- only used when emitting signals, prometheus metrics will always be updated
    {
        name = "relay_state",
        from_signal = {plug, "state"}
    },
    {
        name = "voltage",
        unit = "V",
        deadband = "0.5%", -- do not emit the value if the delta to the pervious one is smaller then 0.5%
        from_signal = {plug, "realtime", function(t) return t.voltage end }
    },
    {
        name = "current",
        unit = "A",
        from_signal = {plug, "realtime", function(t) return t.current end }
    },
    {
        name = "power",
        unit = "W",
        from_signal = {plug, "realtime", function(t) return t.power end }
    },
    {
        -- This property tries do detect wether there's something connected to the
        -- power plug or not by comparing the current power consumption against the threshold of 0.1W
        name = "in_use",
        from_signal = {plug, "realtime", function(t) return t.power end},
        before_set = function(v)
            return v >= 0.1
        end
    },
    {
        name = "total",
        unit = "Wh",
        from_signal = {plug, "realtime", function(t) return t.total end }
    },
    -- Detects wether the laundry washer finished
    {
        name = "is_running",
        from_signal = {plug, "realtime", function(t)
            -- Waschmaschine benötigt ~1.7W im Standby
            return t.power > 2
        end}
    }
}

-- create a sensor device for the openweathermap client
-- it already provides a set of common sensor properties
-- using client:common_sensor_properties()
local weather_sensor = device.sensor {
    name  = "weather",
    default_metric = "gauge",
    distinct = false,
    unpack(weather:common_sensor_properties())
}

-- {{ Publish all sensor values to MQTT using the topic sensors/{sensorName}/{propertyName}
--    if the sensor has a unit, the value is sent as a JSON encoded string of format value+unit
--    otherwise the value is sent JSON encoded as it is

local publish_changes = function(sensor, prop_name, value, prop_def)
    print(sensor.name.."."..prop_name.." => "..tostring(value)..(prop_def.unit or ""))
    local payload
    if prop_def.unit then
        payload = tostring(value)..prop_def.unit
    else
        payload = value
    end

    mqtt:publish {
        topic = 'sensors/'..sensor.name.."/"..prop_name,
        payload = json.encode(payload),
        -- send the message with retained flag so consumers will always receive the latest values
        -- upon subscription
        retained = true,
    }
end

plug_sensor:connect_signal("sensor::property", publish_changes)
plug_sensor:connect_signal("sensor::property", function(sensor, name, value)
    if name == "voltage" then
        print("next(", value, ")")
        plug_subject:next(value)
    end
end)
weather_sensor:connect_signal("sensor::property", publish_changes)

-- }}

-- {{ Rules (experimental)
rule {
    name = "test rule",
    trigger = {
        rule.onSignal(weather, "weather::temp_min"),
        rule.onPropertyChange(plug_sensor, "is_running"),
        rule.onInterval(10),
    },
    when = function()
        print("when")
        return false
    end,
    action = function()
        notify{title = "foo", text = "it works"}
    end
}
-- }}

-- start watching (polling) current weather conditions
--weather:watch()

-- poll the relay status every 3 seconds
--plug:watch_relay(3)

-- poll the current power consumption (realtime) every 10 seconds
-- a lower value seems the cause troubles with the HS110 plug not
-- responding to calls
plug:watch_realtime(10)

-- allow the plug (laundry washer) to be controlled via MQTT
mqtt:subscribe {
    topic = "laundry_washer/on",
    callback = function() plug:turn_on() end
}
mqtt:subscribe {
    topic = "laundry_washer/off",
    callback = function() plug:turn_off() end
}
