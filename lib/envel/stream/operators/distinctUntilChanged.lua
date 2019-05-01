local Observable = require("envel.stream.observable")
local Subscriber = require("envel.stream.subscription").Subscriber

function Observable:distinctUntilChanged()
    return self:lift(function(sink, source)
        local child = Subscriber.create(sink)
        local next = child.next
        local old = nil
        local old_set = false

        child.next = function(sub, value)
            if not old_set then
                old_set = true
                old = value
                next(sub, value)
                return
            end

            if old ~= value then
                old = value
                next(sub, value)
            end
        end

        return source:subscribe(child)
    end)
end