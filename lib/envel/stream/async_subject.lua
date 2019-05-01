local Subject = require("envel.stream.subject")
local to_subscriber = require("envel.stream.subject").to_subscriber

local AsyncSubject = {}
AsyncSubject.__index = AsyncSubject
AsyncSubject.__tostring = function() return "AsyncSubject" end
setmetatable(AsyncSubject.__index, Subject)

function AsyncSubject.create()
    local subject = Subject.create()
    setmetatable(subject, AsyncSubject)

    subject.has_next = false
    subject.next_value = nil
    subject.has_completed = false

    return subject
end

function AsyncSubject:__subscribe(sink)
    if self.has_thrown ~= nil then
        sink:error(self.has_thrown)
        return sink
    end

    if self.has_completed then
        if self.has_next then
            sink:next(self.next_value)
        end

        sink:complete()
        return sink
    end

    return Subject.__subscribe(self, sink)
end

function AsyncSubject:next(value)
    if not self.has_completed then
        self.next_value = value
        self.has_next = true
    end
end

function AsyncSubject:error(err)
    if not self.has_completed then
        Subject.error(self, err)
    end
end

function AsyncSubject:complete()
    if self.has_completed then return end

    self.has_completed = true

    if self.has_next then
        Subject.next(self, self.next_value)
    end

    Subject.complete(self)
end

return AsyncSubject