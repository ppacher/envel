local Subscriber = require("envel.stream.subscription").Subscriber

local SubjectSubscriber = {}
SubjectSubscriber.__index = SubjectSubscriber
SubjectSubscriber.__tostring = function() return "SubjectSubscriber" end
setmetatable(SubjectSubscriber.__index, Subscriber)

function SubjectSubscriber.create(sink, subject)
    local instance = Subscriber.create(sink)
    setmetatable(instance, SubjectSubscriber)

    -- store a reference to the subject we are subscribed to
    instance._subject = subject

    -- add ourself to the subjects list of observers
    table.insert(subject.observers, instance)

    return instance
end

function SubjectSubscriber:unsubscribe()
    -- do the actual unsubscription handling
    Subscriber.unsubscribe(self)

    local subject = self._subject
    self._subject = nil

    self.closed = true
    if not subject.observers or #subject.observers == 0 or subject.is_stopped or subject.closed then
        return
    end

    local new_observers = {}
    for _, i in ipairs(subject.observers) do
        if i ~= self then
            table.insert(new_observers, i)
        end
    end
    subject.observers = new_observers
end

return SubjectSubscriber