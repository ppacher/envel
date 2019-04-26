--- Module signal

local methods = {}

--- emit returns a callback function that emits the given `signal`
-- on `prop` using unpacked `args` as parameters
-- @param prop      The object implementing the emit_signal interface
-- @param signal    The name of the signal to emit
-- @param args      A table containing arguments for the signal
--                  If a function is provided it will be called and the result
--                  will be used for signal parameters
function methods.emit(prop, signal, args)
    -- check that `prop` actually implements the emit_signal interface
    if type(prop.emit_signal) ~= 'function' then
        error("cannot emit signal on object not implemting emit_signal")
    end

    -- if it's function call it
    if type(args) == 'function' then args = args() end

    -- make sure we have a empty table to unpack
    if not args then args = {} end

    -- return the actual callback
    return function()
        prop:emit_signal(signal, unpack(args))
    end
end

-- update the matatable of the built-in __signal
-- table to provide our own methods as well
setmetatable(_G.__signal, {
    __index = methods,
    __call = getmetatable(_G.__signal).__call
})

return _G.__signal