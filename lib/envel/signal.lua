local methods = {}
function methods.is_a(what)
    repeat
        if getmetatable(what) == _G.__signal.__signal_mt then
            return true
        end

        what = getmetatable(what)
        if what then what = what.__index end

    until what == nil

    return false
end

setmetatable(_G.__signal, {
    __index = methods,
    __call = getmetatable(_G.__signal).__call
})

return _G.__signal