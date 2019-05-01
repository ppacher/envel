local Observable = require("envel.stream.observable")
local Subscriber = require("envel.stream.subscription").Subscriber
local new_timer = require("envel.timer")

function Observable:debounceTime(timeout)
    return self:lift(function(sink, source)
        local child = Subscriber.create(sink)
        local next = child.next
        local timer = nil

        child.next = function(sub, value)
            if timer ~= nil then
                timer:stop()
            end

            timer = new_timer {
                timeout = timeout,
                single_shot = true,
                autostart = true,
                callback = function()
                    next(sub, value)
                    timer = nil
                end
            }
        end

        return source:subscribe(child)
    end)
end