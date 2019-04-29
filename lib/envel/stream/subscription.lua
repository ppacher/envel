-------------------------------------------------------------------------------

--- @class  Subscription
--
local Subscription = {}
Subscription.__index = Subscription
Subscription.__tostring = function() return 'Subscription' end
Subscription.__is_subscription = true

function Subscription.create(unsubscribe, o)
    local instance = o or {}
    instance.closed = false
    instance._parents = nil
    instance._subscriptions = nil
    instance._unsubscribe = unsubscribe

    setmetatable(instance, Subscription)

    return instance
end

function Subscription:unsubscribe()
    if self.closed then return end
    self.closed = true

    local _parents, _subscriptions, _unsubscribe = self._parents, self._subscriptions, self._unsubscribe

    self._parents = nil
    self._subscriptions = nil
    self._unsubscribe = nil


    -- make sure to remove ourself from any parent we might be linked to
    if _parents ~= nil and _parents.__remove_subscription then
        _parents:__remove_subscription(self)
    elseif _parents ~= nil then
        for _, p in ipairs(_parents) do
            p:__remove_subscription(self)
        end
    end

    -- call the cleanup function
    if type(_unsubscribe) == 'function' then
        _unsubscribe(self)
    end

    -- unsubscribe all of our child subscriptions
    if type(_subscriptions) == 'table' then
        for _, sub in ipairs(_subscriptions) do
            if type(sub.unsubscribe) == 'function' then
                sub:unsubscribe()
            end
        end
    end
end

function Subscription:add(teardown)
    local subscription

    if type(teardown) == 'function' then
        subscription = Subscription.create(teardown)
    elseif type(teardown) == 'table' and (teardown == self or (teardown.__is_subscription and teardown.closed) or
           type(teardown.unsubscribe) ~= 'function') then
        return teardown
    else
        subscription = teardown
    end

    -- there's a unsubscribe method available but it's not a subscription per-se, wrap it into one
    if subscription == nil or not subscription.__is_subscription then
        subscription = Subscription.create()
        subscription._subscriptions = {teardown}
    end

    if self.closed then
        subscription:unsubscribe()
        return subscription
    end

    -- Add `self` as a parent of `subscription` if that's not already the case
    local _parents = subscription._parents

    if _parents == nil then
        -- there not parent yet
        subscription._parents = self
    elseif _parents.__is_subscription then
        -- check if we are already the parent of those subscription
        if _parents == self then return subscription end

        -- if there's already another parent, but not multiple, allocate an array
        -- to store the rest of the parent subscriptions
        subscription._parents = {_parents, self}
    else
        local found = false
        for _, p in ipairs(_parents) do
            if p == self then
                found = true
                break
            end
        end

        if not found then
            table.insert(subscription._parents, self)
        else
            -- we are already a parent of the subscription, nothing to do
            return subscription
        end
    end

    if self._subscriptions == nil  then
        self._subscriptions = {subscription}
    else
        table.insert(self._subscriptions, subscription)
    end

    return subscription
end

function Subscription:__remove_subscription(sub)
    if self._subscriptions == nil then return end

    local new_subscriptions = {}
    for _, s in ipairs(self._subscriptions) do
        if s ~= sub then
            table.insert(new_subscriptions, s)
        end
    end

    self._subscriptions = new_subscriptions
end
-------------------------------------------------------------------------------

--- @class  Subscriber
--
local SafeSubscriber = {}

local Subscriber = {}
Subscriber.__index = Subscriber
Subscriber.__tostring = function() return 'Subscriber' end
Subscriber.__is_subscriber = true

-- make Subscriber inherit from subscription
setmetatable(Subscriber.__index, Subscription)

function Subscriber.create(onNext, onError, onCompleted)
    local instance = {
        _is_stopped = false,
        _destination = nil,
    }

    -- TODO(ppacher): add support for empty observers
    if onNext ~= nil then
        if type(onNext) == 'table' and onNext.__is_subscriber then
            instance._destination = onNext
            onNext:add(instance)
        elseif type(onNext) == 'table' then
            instance._destination = SafeSubscriber.create(instance, onNext)
        else
            instance._destination = SafeSubscriber.create(instance, onNext, onError, onCompleted)
        end
    end

    return setmetatable(instance, Subscriber)
end

function Subscriber:next(value)
    if self._is_stopped then return end

    self._destination:next(value)
end

function Subscriber:error(value)
    if self._is_stopped then return end

    self._is_stopped = true

    self._destination:error(value)
    self:unsubscribe()
end

function Subscriber:complete()
    if self._is_stopped then return end

    self._is_stopped = true

    self._destination:complete()
    self:unsubscribe()
end

function Subscriber:unsubscribe()
    if self.closed then
        return
    end

    self._is_stopped = true
    Subscription.unsubscribe(self)
end

-------------------------------------------------------------------------------

function SafeSubscriber.create(parent, observerOrNext, onError, onComplete)
    -- create a new subscriber without a destination
    local instance = Subscriber.create()
    local next
    local context = instance

    instance._parentSubscriber = parent
    if type(observerOrNext) == 'function' then
        next = observerOrNext
    else
        next = observerOrNext.next
        onError = observerOrNext.error
        onComplete = observerOrNext.complete
        if not observerOrNext.__empty_observer then
            context = setmetatable({}, {__index = observerOrNext})
            if type(context.unsubscribe) == 'function' then
                local unsub = context.unsubscribe
                instance:add(function() unsub(context) end)
            end
            context.unsubscribe = function()
                instance:unsubscribe()
            end
        end
    end

    instance._next = next
    instance._error = onError
    instance._complete = onComplete
    instance._context = context

    function instance:__tryOrUnsubscribe(fn, value)
        if fn == nil then return end
        return fn(instance._context, value)
--[[
        if not pcall(function()
            return fn(value)
        end) then
            print("failed to call")
            self:unsubscribe()
        end
--]]
    end

    function instance:next(value)
        if instance._is_stopped or not instance._next then return end

        return instance:__tryOrUnsubscribe(instance._next, value)
    end

    function instance:error(err)
        if instance._is_stopped or not instance._error then return end

        return instance:__tryOrUnsubscribe(instance._error, err)
    end

    function instance:complete()
        if instance._is_stopped or not instance._complete then return end

        return instance:__tryOrUnsubscribe(instance._complete)
    end

    return instance
end

local function to_subscriber(nextOrObserver, onError, onCompleted)
    if nextOrObserver == nil and onError == nil and onCompleted == nil then
        return Subscriber.create()
    end

    if type(nextOrObserver) == 'table' and nextOrObserver.__is_subscriber then
        return nextOrObserver
    end

    return Subscriber.create(nextOrObserver, onError, onCompleted)
end

return {
    Subscriber = Subscriber,
    Subscription = Subscription,
    to_subscriber = to_subscriber,
}