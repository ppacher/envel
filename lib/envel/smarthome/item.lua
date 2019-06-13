local signal = require 'envel.signal'

local Item = {}
Item.__index = Item

--- The default metrics provider uses Prometheus to expose
-- item values
-- To configure a different metrics provider overwrite the following
-- property and call item:expose_metrics(cfg)
--
-- @tparam table cfg A configuration table for the metrics provider
-- Allowed properties depend on the selected provider
Item.default_metrics_provider = function(item, cfg)
    local prometheus = require('envel.metrics.prometheus')

    local metrics_cfg = {}

    if type(cfg) == 'string' then
        metrics_cfg.type = cfg
        metrics_cfg.name = item.name
        metrics_cfg.namespace = 'sensors'
        metrics_cfg.subsystem = item.interface and item.interface.name or ''
    else
        metrics_cfg = {
            type = cfg.type,
            name = cfg.name or item.name,
            namespace = cfg.namespace or 'sensors',
            subsystem = cfg.subsystem or (item.interface and item.interface.name or '')
        }
    end

    local metric = prometheus(metrics_cfg)

    item:connect_signal('changed', function(value)
        metric:set(value)
    end)
end

function Item:__tostring()
    if self.value then
        return tostring(self.value) .. (self.unit or '')
    end

    -- TODO(ppacher): we currently use null as it's JSON ecoded nil, what to do here?
    return 'null'
end

--- Tries to connect the item to a given source by updating it's value whenever the
-- source emits a new one. The source may either be an envel.stream.Observable or
-- a function. If source is a function it will be invoked with a callback function as it's
-- first parameter. Whenever there is a new value this callback method should be invoked with
-- the new value as it's first parameter.
--
-- Consider the following example:
-- ```lua
-- local new_value = nil
-- item:try_connect(function(cb) new_value = cb end)
--
-- -- set a new value
-- new_value(10.9)
-- assert(item.value == 10.9)
-- ```
function Item:try_connect(source)
    if not source then
        return 'no source provided'
    end

    -- check if it is an observable
    if (type(source) == 'table' or type(source) == 'userdata') and
       (type(source.subscribe) == 'function' or type(source.subscribe) == 'table') then
        source:subscribe(function(_, new_value)
            self:set(new_value)
        end)
        return
    end

    -- it's a callback function
    if type(source) == 'function' then
        source(function(value)
            self:set(value)
        end)
        return
    end

    return 'unknown source type'
end

--- Connects the item to a given source by updating it's value whenever the source
-- emits a new one. It will throw an error if the provided source is not supported.
-- For more information see Item:try_connect()
function Item:connect(source)
    local err = self:try_connect(source)
    if err then
        error(err)
    end
end

--- Expose the item's value as metrics
function Item:expose_metrics(metrics_cfg)
    -- if it's a function we use that directly
    -- this allows to pass a factory function for custom metric
    -- exporters
    if type(metrics_cfg) == 'function' then
        metrics_cfg(self)
        return
    end

    -- if it's not a function, fall back to the default_metrics_provider
    Item.default_metrics_provider(self, metrics_cfg)
end

--- Create a new item from the given configuration
-- Valid arguments for `cfg`:
--
-- * **unit**: An optional unit of the item
-- * **hot**: Whether or not the item is hot (see class Item)
-- * **group**: An optional group the item belongs to
-- * **description**: An optional description for the item
-- * **metrics**: An optional metrics setup function or a table/string configuration for
--   the default metrics provider (@see default_metrics_provider). Providing
--   this property will just call item:expose_metrics(...)
-- * **extra**: An optional (JSON serializable) table with metadata for the item
-- * **keywords**: An optional list of keywords that may be used to identify the item
--
-- @tparam string name The name for the item
-- @tparam table cfg The configuration table for the item. See above for valid properties
-- @tparam Interface cfg[opt] An optional interface the item belongs to. This is only use
--  when metrics is set in cfg and prometheus is used. TODO(ppacher): deprecate and remove it
-- @treturn table The new item
-- @staticfct envel.smarthome.Item
function Item.create(name, cfg, intf)
    local item = {
        name = name,
        value = nil,
        unit = cfg.unit or '',
        hot = cfg.hot == true,
        group = cfg.group or '',
        keywords = cfg.keywords or {},
        description = cfg.description or '',
        extra = cfg.extra or {},
        interface = intf,
        __private = {
            signal = signal(),
        }
    }
    setmetatable(item, Item)

    if cfg.metrics then
        item:expose_metrics(cfg.metrics)
    end

    if cfg.source then item:connect(cfg.source) end

    return item
end

--- Connects a callback function from a signal
-- @see envel.signal
function Item:connect_signal(...)
    return self.__private.signal:connect_signal(unpack(arg))
end

--- Disconnects a callback function from a signal
-- @see envel.signal
function Item:disconnect_signal(...)
    return self.__private.signal:disconnect_signal(unpack(arg))
end

-- Publish publishes the value of the item
function Item:publish()
    self.__private.signal:emit_signal('changed', self.value, self)
end

function Item:set(value)
    local changed = self.value ~= value
    self.value = value

    if changed then
        self:publish()
    end
end

setmetatable(Item, {
    __call = function(t, cfg) return t.create(cfg) end
})

--- Validates a item configuration table
-- @see envel.smarthome.Item
-- @treturn ?string A string if an error is found, nil if everything is fine
function Item.validate_options(cfg)
    if type(cfg) ~= 'table' then
        return 'expected a table but got '..type(cfg)
    end

    if cfg.unit ~= nil and type(cfg.unit) ~= 'string' then
        return 'item unit must be a string or nil'
    end

    if cfg.group ~= nil and type(cfg.group) ~= 'string' then
        return 'item group must be a string or nil'
    end

    if string.match(cfg.group or '', '/') then
        return 'item group may not container slashes ("/")'
    end

    return nil
end

return Item