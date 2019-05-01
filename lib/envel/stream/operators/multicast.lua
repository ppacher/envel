local Observable = require("envel.stream.observable")
local ConnectableObservable = require("envel.stream.connectable_observable")

function Observable:multicast(subjectOrFactory)
    return ConnectableObservable.create(self, subjectOrFactory)
end