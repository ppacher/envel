local subscription = require("envel.stream.subscription")

--- @class Observable
local Observable = {}
Observable.__index = Observable
Observable.__tostring = function() return 'Observable' end

function Observable.create(producer)
    local self = {
        _subscribe = producer,
        _source = nil,
        _operator = nil,
    }

    return setmetatable(self, Observable)
end

--- Creates a new observable, with this observable as the source, and the passed
-- operator defined as the new observable's operator.
-- @param   operation       the operator defining the operation to take on the observable
-- @return  Observable      a new observable with the operator applied
function Observable:lift(operator)
    local obs = Observable.create()
    obs._source = self
    obs._operator = operator
    return obs
end

function Observable:subscribe(observerOrNext, onError, onCompleted)
    local operator = self._operator
    local sink = subscription.to_subscriber(observerOrNext, onError, onCompleted)
    
    if operator then
        sink:add(operator(sink, self._source))
    elseif self._source then
        sink:add(self._source:subscribe(sink))
    else
        sink:add(self._subscribe(sink))
    end

    return sink
end

return Observable