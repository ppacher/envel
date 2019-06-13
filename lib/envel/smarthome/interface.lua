local append = table.insert
local signal = require 'envel.signal'
local Item = require 'envel.smarthome.item'

local Interface = {}
Interface.__index = Interface
function Interface:__tostring()
    return 'Interface<' .. (self and self.name or 'nil') .. '>'
end

--- Interface states
-- An interface exposed via MQTT can have one of three different states:
--
-- * **0**: disconnected from MQTT broker
-- * **1**: connected to MQTT, but disconnected from hardware
-- * **2**: connected to MQTT and hardware, device fully operational
--
-- interfaces created by this module may only be DISCONNECTED or CONNECTED
-- to the hardware. The MQTT connection part is handled by a envel.smarthome.home
-- instance
-- @see envel.smarthome.home
Interface.state = {
    DISCONNECTED = 1,
    CONNECTED = 2
}

--- Connects a callback function to a given signal
-- @see envel.signal
function Interface:connect_signal(...)
    return self.__private.signal:connect_signal(unpack(arg))
end

--- Disconnects a callback function from a signal
-- @see envel.signal
function Interface:disconnect_signal(...)
    return self.__private.signal:disconnect_signal(unpack(arg))
end

--- Binds the interface to one or more MQTT-Home brokers. This function may also be called
-- multiple times.
-- @param args[...] A list of envel.smarthome.homes
function Interface:bind(...)
    for _, m in ipairs(arg) do
        m:bind_interface(self)
    end
    return self
end

--- Set the interface status to CONNECTED
-- @see envel.smarthome.interface.state
function Interface:set_connected()
    self:set_status(Interface.state.CONNECTED)
end

-- Set the interface status to DISCONNECTED
-- @see envel.smarthome.interface.state
function Interface:set_disconnected()
    self:set_status(Interface.state.DISCONNECTED)
end

--- Updates the interface status.
-- @see Interface.state
-- @tparam number status The new status of the interface (1 or 2)
function Interface:set_status(status)
    if not (status == Interface.state.CONNECTED or status == Interface.state.DISCONNECTED) then
        error('Invalid state for interface')
    end

    local changed = self.__private.status ~= status
    self.__private.status = status

    -- notify listeners about the state change as well
    if changed then
        self.__private.signal:emit_signal('status::update', status, self)
        if status == Interface.state.CONNECTED then
            self.__private.signal:emit_signal('status::connected', status, self)
        else
            self.__private.signal:emit_signal('status::disconnected', status, self)
        end
    end
end

--- Create a new mqtt-smarthome interface
-- Valid arguments for `cfg`:
--
-- * **name**: The name of the new interface (required)
-- * **location**: An optional location for the new interface
-- * **description**: An optional description for the new interface
-- * **items**: An optional table of items exposed by the interface
-- * **metrics**: An optional default metric type to use for all items.
-- * **keywords**: An optional list of keywords that may be used to identify the interface
--
-- @tparam table cfg A configuration table, see above for valid properties
-- @treturn table a new interface instance
-- @treturn ?string An error message if appropriate
function Interface.create(cfg)
    if not cfg then return nil, "No configuration provided" end
    if not cfg.name then return nil, "Interface name must be configured" end
    if not cfg.items then cfg.items = {} end

    -- make sure we only have valid items
    for _, item in pairs(cfg.items) do
        local err = Item.validate_options(item)
        if err ~= nil then return nil, err end
    end

    local intf = {
        name = cfg.name,
        location = cfg.location,
        -- TODO(ppacher): we could move private data to a dedicated items table that uses
        -- weak-keys based on the item table instance
        __private = {
            -- signal instance used to listen for and emit signals
            signal = signal(),
            status = Interface.state.DISCONNECTED,
        },
    }
    setmetatable(intf, Interface)

    -- create a item instance for each item configuration and store it in a table
    -- make the talbe readonly afterwards
    local items = {}
    for name, icfg in pairs(cfg.items) do
        if icfg.metrics == nil and type(cfg.metrics) == 'string' then
            icfg.metrics = cfg.metrics
        end

        items[name] = Item.create(name, icfg, intf)

        -- forward item signals
        items[name]:connect_signal('changed', function(value)
            intf.__private.signal:emit_signal('items::changed', value, items[name], intf)
            intf.__private.signal:emit_signal('item::'..name, value, items[name], intf)
        end)
    end

    -- items:names() returns a list of item names
    function items:names()
        local names = {}
        for n, _ in pairs(self) do append(names, n) end
        return names
    end

    -- make the items table read only
    intf.items = setmetatable({}, {
        __index = items,
        __newindex = function(a1, name, value)
            assert(getmetatable(a1).__index == items)
            assert(items[name], "item with name "..name.." does not exist on interface "..intf.name)

            -- delegate the set operation to the item instance
            items[name]:set(value)
        end,
    })

    return intf
end

-- Allow interface to be called directly so we can use a more declarative style
-- like:
--
-- envel.home.interface {
--   name = "..."
-- }
setmetatable(Interface, {
    __call = function(t, cfg) return t.create(cfg) end
})

return Interface