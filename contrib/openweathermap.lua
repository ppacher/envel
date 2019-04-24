local http   = require("envel.http")
local timer  = require("envel.timer")
local signal = require("envel.signal")
local json   = require("json")

-- class for openweathermap clients
local weather_cls = {}

-- Returns the API URL to fetch the current weather information for
-- the configured location and units
function weather_cls:get_current_weather_url()
    return "http://api.openweathermap.org/data/2.5/weather?units="..self.units.."&q="..self.location.."&APPID="..self.key
end

-- updates the current weather information. If `cb` is set to a function
-- it will be called with the result of the API call as it's first argument
-- TODO(ppacher): add a non-blocking HTTP client that uses callbacks and pre-reads the body
-- currently this method blocks the whole loop
function weather_cls:update_current(cb)
    local res, err = http{
        method = "GET",
        url = self:get_current_weather_url(),
        headers = {
            Accept = "application/json",
        }
    }

    if err ~= nil or res.status_code ~= 200 then
        print("failed to poll openweathermap.org (status_code="..tostring(res.status_code).."): "..(err or res.status))
    else
        local payload = json.decode(res.body:read("*a"))
        self.current = payload

        self:emit_signal("weather::temp", payload.main.temp)
        self:emit_signal("weather::pressure", payload.main.pressure)
        self:emit_signal("weather::humidity", payload.main.humidity)
        self:emit_signal("weather::temp_min", payload.main.temp_min)
        self:emit_signal("weather::temp_max", payload.main.temp_max)
        self:emit_signal("weather::sunrise", payload.sys.sunrise)
        self:emit_signal("weather::sunset", payload.sys.sunset)
        if payload.wind then
            self:emit_signal("weather::windspeed", payload.wind.speed)
            self:emit_signal("weather::winddeg", payload.wind.deg)
        end

        if type(cb) == 'function' then
            cb(payload)
        end

    end
end

-- watch starts polling the openweathermap API
-- and emits signals for all monitored values
-- cb is passed to update_current()
function weather_cls:watch(timeout, cb)
    local t = timer{
        autostart = true,
        call_now = true,
        timeout = timeout or 10 * 60,
        callback = function()
            self:update_current(cb)
        end,
        current = nil,
    }

    return t
end

-- Returns a table with common properties used in sensor defintions
-- use like device.sensor{name="test", unpack(result)}
function weather_cls:common_sensor_properties()
    return {
        {
            name = "temperature",
            unit = "°C",
            from_signal = {self, "weather::temp"},
        },
        {
            name = "temperature_min",
            unit = "°C",
            from_signal = {self, "weather::temp_min"},
        },
        {
            name = "temperature_max",
            unit = "°C",
            from_signal = {self, "weather::temp_max"},
        },
        {
            name = "pressure",
            unit = "mB",
            from_signal = {self, "weather::pressure"},
        },
        {
            name = "humidity",
            unit = "%",
            from_signal = {self, "weather::humidity"},
        },
        {
            name = "sunrise",
            unit = "s",
            from_signal = {self, "weather::sunrise"},
        },
        {
            name = "sunset",
            unit = "s",
            from_signal = {self, "weather::sunset"},
        },
        {
            name = "wind_speed",
            unit = "ms",
            from_signal = {self, "weather::windspeed"}
        },
        {
            name = "wind_deg",
            unit = "°",
            from_signal = {self, "weather::winddeg"}
        }
    }
end

function weather_cls:emit_signal(...)
    self.signal:emit_signal(unpack(arg))
end

function weather_cls:connect_signal(...)
    self.signal:connect_signal(unpack(arg))
end

return setmetatable({}, {
    __call = function(_, key, location, units)
        local weather = {
            signal = signal(),
            key = key,
            location = location,
            units = units or "metric"
        }
        setmetatable(weather, {__index = weather_cls})
        return weather
    end
})