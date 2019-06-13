local append = table.insert
local MQTT = require('envel.mqtt')
local JSON = require('json')

local Home = {}
Home.__index = Home
Home.__tostring = function() return 'Home' end

--- Returns MQTT connection base on the given
-- "Home" configuration table
local function get_mqtt(cfg)
    if type(cfg.mqtt) == 'table' then -- a MQTT connection will be uservalue
        return MQTT(cfg.mqtt)
    end

    return cfg.mqtt
end

local function default_status_topic_generator(home, item, interface)
    local topic = interface.name .. '/status'

    if type(interface.location) == 'string' and interface.location ~= '' then
        topic = topic .. '/' .. interface.location
    end

    if type(item.group) == 'string' and item.group ~= '' then
        topic = topic .. '/' .. item.group
    end

    topic = topic .. '/' .. item.name

    if home.cfg.topic_prefix ~= '' then
        topic = home.cfg.topic_prefix .. '/' .. topic
    end

    return topic
end

local function default_subscribe_topic_generator()

end

--- Creates a new home instance
-- The following properties are valid:
--
-- * **mqtt**: The MQTT client or configuration
-- * **topic_prefix**: An optional MQTT topic prefix to use. Only used by default_status_topic_generator
--   and default_subscribe_topic_generator
-- * **status_topic_generator**: An optional function returning the topic required for publishing
--   see default_status_topic_generator
-- * **status_message_qos**: The Quality-Of-Service to use for status messages. Defaults to 2
-- * **subscribe_topic_generator**: An optional function returning a topic subscription string
--   see default_subscribe_topic_generator
--   updates. Note that the provided topic MUST ONLY MATCH the given item.
--
-- @tparam table cfg The configuration table to use
-- @treturns Home A home instance if everything was successful
function Home.create(cfg)
    local mqtt = get_mqtt(cfg)
    local inst = {
        name = cfg.name,
        mqtt = mqtt,
        interfaces = {},
        cfg = {
            topic_prefix = cfg.topic_prefix or '',
            status_topic_generator = cfg.status_topic_generator or default_status_topic_generator,
            status_message_qos = cfg.status_message_qos or 2,
            subscribe_topic_generator = cfg.subscribe_topic_generator or default_subscribe_topic_generator,
        }
    }
    setmetatable(inst, Home)

    -- TODO(ppacher) setup topic subscriptions

    -- we need to use a wrapper functions for connect_signal
    -- because they will be called unbound (i.e no self context)
    inst.__item_handler = function(...)
        inst:publish_item(unpack(arg))
    end
    inst.__status_handler = function(...)
        inst:publish_status(unpack(arg))
    end

    return inst
end

--- internal function to handle item value updates
function Home:publish_item(value, item, interface)
    local payload = {}
    local topic = self.cfg.status_topic_generator(self, item, interface)

    for ek, ev in pairs(item.extra or {}) do
        payload[ek] = ev
    end

    -- we set the value at last so item.extra cannot overwrite it
    payload.val = value

    self.mqtt:publish {
        topic = topic,
        payload = JSON.encode(payload),
        qos = self.cfg.status_message_qos,
        retained = not item.hot,
    }
end

-- internal function to handle interface status changes
function Home:publish_status(status, interface)
    local topic = string.format('%s/connected', interface.name)
    if self.cfg.topic_prefix ~= '' then
        topic = self.cfg.topic_prefix + '/' + topic
    end

    self.mqtt:publish {
        topic = topic,
        payload = tostring(status),
        qos = 2,
        retained = true,
    }
end

--- Binds an interface to the given home. It is an error to bind the same interface
-- multiple times
-- @tparam Interface intf The interface to bind to the home
function Home:bind_interface(intf)
    -- make sure the interface is not already bound to the home
    for _, i in ipairs(self.interfaces) do
        if i == intf then
            error(tostring(intf) .. ' already added to home')
        end
    end

    -- connect the signal handler for item changes
    intf:connect_signal('items::changed', self.__item_handler)
    intf:connect_signal('status::update', self.__status_handler)

    append(self.interfaces, intf)
end

--- Unbinds an interface from the home
-- @tparam Interface intf The interface to unbind
function Home:unbind_interface(intf)
    local new = {}
    local found = false

    for _, i in ipairs(self.interfaces) do
        if i ~= intf then
            append(new, i)
        else
            intf:disconnect_signal('items::changed', self.__item_handler)
            intf:disconnect_signal('status::update', self.__status_handler)
            found = true
        end
    end

    self.interfaces = new
    assert(found, "unknown interface")
end

setmetatable(Home, {
    __call = function(_, cfg)
        return Home.create(cfg)
    end
})

return Home