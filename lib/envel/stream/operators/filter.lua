local Observable = require("envel.stream.observable")
local Subscriber = require("envel.stream.subscription").Subscriber

function Observable:filter(handler)
    return self:lift(function(sink, source)
        local child = Subscriber.create(sink)
        local next = child.next
        child.next = function(sub, ...)
            local allow = handler(...)

            if allow then next(sub, ...) end
        end

        return source:subscribe(child)
    end)
end