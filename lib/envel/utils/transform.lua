local L = require("envel.utils.lambda")
local module = {}

-- transform returns a function that execute a transformation pipeline
local function transform(...)
    return function(v)
        for _, fn in ipairs(arg) do
            -- make sure we also support lambda function form
            -- envel.utils.lambda
            fn = L(fn)
            if type(fn) == 'function' then
                local err
                v, err = fn(v)
                if err ~= nil then
                    return nil, err
                end
            end
        end

        return v
    end
end

-- tostring is a transform function that returns a string
-- representation of the parameter
function module.tostring(v)
    return tostring(v)
end

function module.map(tbl, fn)
    local t = {}
    for k,v in pairs(tbl) do
        -- we use L(fn) to directly support lambda functions
        -- from envel.utils.lambda
        t[k] = L(fn)(v, k)
    end
    return t
end

return setmetatable(module, {
    __call = function(table, ...)
        return transform(unpack(arg))
    end
})