local Observable = require("envel.stream.observable")
local subscription = require("envel.stream.subscription")
local Subject = require("envel.stream.subject")
local AsyncSubject = require("envel.stream.async_subject")
local ConnectableObservable = require("envel.stream.connectable_observable")

--- Creates a new observable from a signal
-- @tparam table|userdata obj The object implementing the envel.signal interface
-- @tparam string signal The signal identifier to subscribe to
local function from_signal(obj, signal)
    return Observable.create(function(observer)
        local function handle(...)
            observer:next(...)
        end

        obj:connect_signal(signal, handle)

        return function()
            obj:disconnect_signal(signal, handle)
        end
    end)
end

local Stream = {
    Observable = Observable,
    Subscription = subscription.Subscription,
    Subscriber = subscription.Subscriber,
    Subject = Subject,
    AsyncSubject = AsyncSubject,
    ConnectableObservable = ConnectableObservable,
    from_signal = from_signal,
}

return Stream