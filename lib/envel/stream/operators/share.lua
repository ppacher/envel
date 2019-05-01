local Observable = require("envel.stream.observable")
local Subject = require("envel.stream.subject")

function Observable:share()
    local inner_subscription = nil
    local subject = nil
    local refCount = 0

    local next = function(_, value) subject:next(value) end
    local error = function(_, err) subject:error(err) end
    local complete = function()
        subject:complete()
        if inner_subscription ~= nil then
            inner_subscription:unsubscribe()
        end
    end

    return self:lift(function(sink, source)
        if subject == nil then
            subject = Subject.create()
        end

        refCount = refCount + 1
        local outer =  subject:subscribe(sink)
        local original_unsubscribe = outer.unsubscribe

        outer.unsubscribe = function()
            refCount = refCount - 1

            if refCount == 0 then
                inner_subscription:unsubscribe()
            end

            original_unsubscribe(outer)
        end

        if inner_subscription == nil then
            inner_subscription = source:subscribe(next, error, complete)

            inner_subscription:add(function()
                inner_subscription = nil
                subject = nil
                refCount = 0
            end)
        end

        return outer

    end)
end