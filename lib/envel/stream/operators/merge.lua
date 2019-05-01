local Observable = require("envel.stream.observable")
local Subscriber = require("envel.stream.subscription").Subscriber

function Observable:merge(...)
    local sources = {...}

    return self:lift(function(sink, source)
        local child = Subscriber.create(sink)

        local subscription = source:subscribe(child)
        for _, src in ipairs(sources) do
            local inner_subscription = src:subscribe(child)
            subscription:add(inner_subscription)
        end

        return subscription
    end)
end