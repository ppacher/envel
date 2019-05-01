local Observable = require("envel.stream.observable")
local Subject = require("envel.stream.subject")
local Subscription = require("envel.stream.subscription").Subscription
local ConnectableSubscriber = require("envel.stream.connectable_subscriber")

local ConnectableObservable = {}
ConnectableObservable.__index = ConnectableObservable
ConnectableObservable.__tostring = function() return "ConnectableObservable" end
setmetatable(ConnectableObservable.__index, Observable)

function ConnectableObservable.create(source, subjectOrFactory)
    if subjectOrFactory == nil then
        subjectOrFactory = Subject.create()
    end

    local instance = Observable.create()
    setmetatable(instance, ConnectableObservable)

    -- Source holds the source observable we are going to 
    -- subscribe upon connect()
    instance._source = source

    -- Subject holds the subject used for all subscribers
    -- it will be connected to source when connect() is called
    instance._subject = nil

    -- wether or not the underlying source has completed
    instance._hasCompleted = false

    -- The actual subscription to the source observable
    instance._connection = nil

    -- The subject or a factory function
    instance._subjectOrFactory = subjectOrFactory

    -- The number of references to the connectable observable
    instance._refCount = 0

    return instance
end

function ConnectableObservable:connect()
    -- if we are already subscribed to the source observable
    -- return the current subscription and do not resubscribe
    if self._connection then
        return self._connection
    end

    self._connection = Subscription.create()
    self._connection:add(
        self._source:subscribe(
            ConnectableSubscriber.create(self:getSubject(), self)
        )
    )

    if self._connection.closed then
        self._connection = nil
        -- TODO(ppacher): return empty subscription
        return Subscription.create()
    end

    return self._connection
end

-- Subscribe adds an observer to the underlying subject
function ConnectableObservable:subscribe(...)
    return self:getSubject():subscribe(...)
end

-- getSubject returns the subject used for providers
function ConnectableObservable:getSubject()
    if self._subject == nil then
        if type(self._subjectOrFactory) == 'function' then
            self._subject = self._subjectOrFactory()
        else
            self._subject = self._subjectOrFactory
        end
    end

    return self._subject
end

function ConnectableObservable:refCount()
    return self:lift(function(sink, source)
        local subscription = source:subscribe(sink)
        self._refCount = self._refCount + 1
        if self._connection == nil then
            self:connect()
        end

        subscription:add(function()
            self._refCount = self._refCount - 1

            if self._refCount <= 0 then
                self._connection:unsubscribe()
            end
        end)

        return subscription
    end)
end

return ConnectableObservable