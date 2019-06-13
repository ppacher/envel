local Observable = require("envel.stream.observable")
local Subscriber = require("envel.stream.subscription").Subscriber

function Observable:map(handler)
    return self:lift(function(sink, source)
        local child = Subscriber.create(sink)
        local next = child.next

        child.next = function(sub, ...)
            local value = handler(...)
            next(sub, value)
        end

        return source:subscribe(child)
    end)
end

function Observable:select_arg(index)
    return self:map(function(...)
        return ({...})[index]
    end)
end

function Observable:path(p)
    return self:map(function(value)
        return value[p]
    end)
end