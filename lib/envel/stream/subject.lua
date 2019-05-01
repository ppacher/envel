local Observable = require("envel.stream.observable")
local Subscription = require("envel.stream.subscription").Subscription
local Subscriber = require("envel.stream.subscription").Subscriber

local AnonymousSubject = {}
local Subject = {}
Subject.__index = Subject
Subject.__tostring = function() return "Subject" end
setmetatable(Subject.__index, Observable)

function Subject.create()
    local instance = {
        is_stopped = false,
        has_thrown = nil,
        closed = false,
        observers = {},
    }

    Observable.create(nil, instance)
    setmetatable(instance, Subject)

    function instance._subscribe(sink)
        instance:__subscribe(sink)
    end

    return instance
end

function Subject:lift(operator)
    local subject = AnonymousSubject.create(self, self)
    subject._operator = operator
    return subject
end

function Subject:next(value)
    if self.closed then error("subject already closed") end

    if not self.is_stopped then
        for _, observer in ipairs(self.observers) do
            observer:next(value)
        end
    end
end

function Subject:error(err)
    if self.closed then error("subject already closed") end

    for _, observer in ipairs(self.observers) do
        observer:error(err)
    end

    self.observers = {}
    self.is_stopped = true
    self.has_thrown = err
end

function Subject:complete()
    if self.closed then error("subject already closed") end


    for _, observer in ipairs(self.observers) do
        observer:complete()
    end

    self.is_stopped = true
    self.observers = {}
end

function Subject:unsubscribe()
    self.is_stopped = true
    self.closed = true
    self.observers = {}
end

function Subject:__subscribe(sink)
    if self.closed then error("subject already closed") end

    if self.has_thrown ~= nil then
        sink:error(self.has_thrown)
        return sink
    end

    if self.is_stopped then
        sink:complete()
        return sink
    end

    local subscriber = Subscriber.create(sink)
    table.insert(self.observers, subscriber)

    subscriber._subject = self

    -- wrap the unsubscribe method of the new subscriber so we delete our self
    -- form the observers list
    function subscriber:unsubscribe()
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

    return subscriber
end

function AnonymousSubject.create(destination, source)
    local subject = Subject.create()
    subject._destination = destination
    subject._source = source

    function subject:next(value)
        if type(self._destination) == 'table' and type(self._destination.next) == 'function' then
            self._destination:next(value)
        end
    end

    function subject:error(err)
        if type(self._destination) == 'table' and type(self._destination.error) == 'function' then
            self._destination:error(err)
        end
    end

    function subject:complete()
        if type(self._destination) == 'table' and type(self._destination.complete) == 'function' then
            self._destination:complete()
        end
    end

    function subject:_subscribe(sink)
        if type(self._source) == 'table' then
            return self._source:subscribe(sink)
        end

        local empty = Subscription.create()
        --empty:unsubscribe()

        return empty
    end

    return subject
end

return Subject