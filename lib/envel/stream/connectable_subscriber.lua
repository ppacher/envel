local Subscriber = require("envel.stream.subscription").Subscriber

local ConnectableSubscriber = {}
ConnectableSubscriber.__index = ConnectableSubscriber
ConnectableSubscriber.__tostring = function() return "ConnectableSubscriber" end
setmetatable(ConnectableSubscriber.__index, Subscriber)

function ConnectableSubscriber.create(subject, connectable)
    local instance = Subscriber.create(subject)
    setmetatable(instance, ConnectableSubscriber)

    instance._connectable = connectable

    return instance
end

function ConnectableSubscriber:error(value)
    self:unsubscribe()
    Subscriber.error(self, value)
end

function ConnectableSubscriber:complete()
    self._connectable._hasCompleted = true
    self:unsubscribe()
    Subscriber.complete(self)
end

function ConnectableSubscriber:unsubscribe()
    local connectable = self._connectable
    self._connectable = nil

    if connectable then
        local connection = connectable._connection
        connectable._connection = nil
        connectable._subject = nil
        connectable._hasCompleted = true

        if connection then
            connection:unsubscribe()
        end
    end

    Subscriber.unsubscribe(self)
end

return ConnectableSubscriber