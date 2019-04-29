local Observable = require("envel.stream.observable")
local Subscriber = require("envel.stream.subscription").Subscriber

function Observable:map(handler)
    return self:lift(function(sink, source)
        print("lifted")
        local child = Subscriber.create(sink)
        local next = child.next

        child.next = function(sub, ...)
            local value = handler(...)
            next(sub, value)
        end

        return source:subscribe(child)
    end)
end