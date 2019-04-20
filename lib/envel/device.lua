local signal = require("envel.signal")

local module = {}

-- sensor_class represents the class of a sensor
local sensor_class = {}

function sensor_class:emit_signal(...)
    self.signal:emit_signal(unpack(arg))
end

function sensor_class:connect_signal(...)
    self.signal:connect_signal(unpack(arg))
end

-- registers the sensor at the hosting application
local function register_sensor(sensor)

end

-- sensor creates and registers a new sensor
function module.sensor(cfg)
    if not cfg then error('Missing sensor configuration') end
    if not cfg.name then error('Sensor name must be set') end

    local sensor = {
        name = cfg.name,
        description = cfg.description,
        labels = {},
        signal = signal(),
        __properties = {},
        __allowed_properties = {}
    }

    -- add properties
    local prop_count = 0
    for _, prop in ipairs(cfg) do
        if type(prop) ~= 'table' then error("Invalid property. Expected a table but got nil") end
        if type(prop.name) ~= 'string' then error('Property names must be string') end
        if type(prop.unit) ~= 'string' and prop.unit ~= nil then error("Unit type must either be a string or nil") end

        if sensor.__allowed_properties[prop.name] ~= nil then
            error("property "..prop.name.." already defined for sensor "..cfg.name)
        end

        if prop.metric or (cfg.default_metric and prop.metric ~= false) then
            local user_cfg = prop.metric or cfg.default_metric

            local prometheus = require("envel.metrics.prometheus")
            local metric_config = {};

            if type(user_cfg) == 'string' then
                metric_config.type = user_cfg
            elseif type(user_cfg) == 'table' then
                metric_config = prop.metric
            else
                error("Invalid type form sensor.metric")
            end

            if not metric_config.type and type(cfg.default_metric) == 'string' then
                metric_config.type = cfg.default_metric
            end

            if not metric_config.name then
                metric_config.name = prop.name
            end

            if not metric_config.namespace then
                metric_config.namespace = "sensors"
            end

            if not metric_config.subsystem then
                metric_config.subsystem = cfg.name
            end

            prop.__metric = prometheus(metric_config)
        end

        if cfg.distinct ~= nil and prop.distinct == nil then
            prop.distinct = cfg.distinct
        end

        sensor.__allowed_properties[prop.name] = prop
        prop_count = prop_count + 1
    end
    if prop_count == 0 then error("No properties defined for sensor "..sensor.name) end

    -- add labels
    for name, value in pairs(cfg.labels or {}) do
        if type(name) ~= nil then
            error('Sensor label names must be strings')
        end

        if type(value) ~= nil then
            error('Sensor label values must be strings')
        end

        sensor.labels[name] = value
    end

    -- mark property and labels as readonly
    setmetatable(sensor.labels, {__newindex = function() error('Sensor labels are readonly') end})
    setmetatable(sensor.__allowed_properties, {__newindex = function() error('Sensor properties are readonly') end})

    local sensor_mt = {
        __index = function(t, k)
            if getmetatable(sensor).__class[k] then return getmetatable(sensor).__class[k] end
            if not sensor.__allowed_properties[k] then
                local mt = getmetatable(getmetatable(t).__index)

                -- since we extend signal we must also lookup properties
                -- in the metatable of __index
                if mt ~= nil then
                    if getmetatable(t).__index[k] ~= nil then
                        return getmetatable(t).__index[k]
                    end
                end

                error('unknown sensor property'..tostring(k)..' on table '..tostring(t)..' sensor='..tostring(sensor))
            end

            if sensor.__properties[k] ~= nil then
                return sensor.__properties[k]
            end

            return nil
        end,

        __newindex = function(t, k, v)
            if not sensor.__allowed_properties[k] then
                error('Only configured sensor values may be updated')
            end

            -- if we should even emit the value if it's the same as before we always
            -- set has_changed to true
            local has_changed = sensor.__properties[k] ~= v or sensor.__allowed_properties[k].distinct == false

            local deadband = sensor.__allowed_properties[k].deadband
            local current = sensor.__properties[k]

            if deadband and current ~= nil then
                local deadband_value
                local is_percent = false

                if type(deadband) == 'string' then
                    if string.match(deadband, "%%") then
                        is_percent = true
                        local value = string.gsub(deadband, "%%", "")
                        deadband_value = tonumber(value)
                    else
                        deadband_value = tonumber(deadband)
                    end
                else
                    deadband_value = deadband
                end

                if type(deadband_value) ~= 'number' then
                    error("invalid configuration for deadband filter")
                end

                if type(v) ~= 'number' then
                    error("deadband can only be applied to numbers")
                end

                local lower_boundary = current - deadband_value
                local upper_boundary = current + deadband_value

                if is_percent  then
                    lower_boundary = current - (current / 100 * deadband_value)
                    upper_boundary = current + (current / 100 * deadband_value)
                end
                has_changed = v <= lower_boundary or v >= upper_boundary
            end

            if type(sensor.__allowed_properties[k].before_set) == 'function' then
                v = sensor.__allowed_properties[k].before_set(v)
            end

            if has_changed then
                if sensor.__allowed_properties[k].__metric and type(v) == 'number' then
                    sensor.__allowed_properties[k].__metric:set(v)
                end

                sensor.__properties[k] = v
                t:emit_signal("sensor::property", sensor, k, v, sensor.__allowed_properties[k])
                t:emit_signal("property::"..k, v)
            end
        end,

        __class = sensor_class,
    }

    -- setup setters and getters for registered properties
    setmetatable(sensor, sensor_mt)

    -- register the sensor at the host application
    register_sensor(sensor)

    -- setup property subscriptions
    for name, value in pairs(sensor.__allowed_properties) do
        -- TODO(ppacher) add better error handling
        if value.from_signal ~= nil then
            local obj = value.from_signal[1]
            local signal_name = value.from_signal[2]
            local arg_index = value.from_signal[3] or 1

            if type(obj.connect_signal) == 'function' then
                obj:connect_signal(signal_name, function(...)
                    if type(arg_index) == 'function' then
                        sensor[name] = arg_index(unpack(arg))
                    elseif type(arg_index) == 'string' then
                        -- if arg_index is a string we'll use it
                        -- as a map key for the first parameter
                        sensor[name] = arg[1][arg_index]
                    else
                        sensor[name] = arg[arg_index]
                    end
                end)
            end
        end
    end

    return sensor
end

return setmetatable(module, {
    __call = function(_, ...)
        error('devices not yet implemented')
    end
})