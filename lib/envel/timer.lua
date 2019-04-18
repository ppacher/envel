local signal = require("envel.signal")
local base_timer = _G.__core.timer

local timer = {}

-- Start the timer if it is not already running
-- This will also emit a "timer::started" signal
function timer:start()
    if not self:is_started() then
        --self:emit_signal("timer::started")
        getmetatable(self).__timer:start()
    end
end

-- Stops the timer if it is running. It will also emit a
-- "timmer::stopped" signal
function timer:stop()
    if self:is_started() then
        self:emit_signal("timer::stopped")
        getmetatable(self).__timer:stop()
    end
end

-- again restarts the timer. This is equal to calling
-- timer:stop() and timer::start()
function timer:again()
    getmetatable(self).__timer:again()
end

-- Returns true if the timer is already started, false otherwise
function timer:is_started()
    return getmetatable(self).__timer:is_started()
end

function timer:emit_signal(...)
    self.signal:emit_signal(unpack(arg))
end

function timer:connect_signal(...)
    self.signal:connect_signal(unpack(arg))
end


-- new creates a new timer
local function new(_, args)
    local t = {
        signal = signal(),
    }
    local cb = args.callback

    -- we need to disable call_now as the metatable of our
    -- wrapped timer is not yet setup
    local call_now = args.call_now or false
    args.call_now = false

    args.callback = function()
        t:emit_signal("timer::tick")
        if type(cb) == 'function' then cb(t) end
    end

    local base = base_timer(args);

    local mt = {
        __timer = base,
        __index = timer,
    }

    setmetatable(t, mt)

    -- if call_now was set we should immediately execute the callback
    -- we do not emit a timer::tick signal here
    if call_now and type(cb) == 'function' then
        cb(t)
    end

    return t
end

return setmetatable({}, {
    __call = new
})