-- File providing mocks for envel bindings
-- required to successfully run tests


----------------------------------------------------------
-- Mock object for signals

_G.__signal = {}
_G.__signal.__index = _G.__signal
function _G.__signal.emit_signal() end
function _G.__signal.connect_signal() end
setmetatable(_G.__signal, {
    __index = _G.__signal,
    __call = function()
        return setmetatable({}, _G.__signal)
    end
})


----------------------------------------------------------
-- Mock object for timer bindings

local timer_mock = {}
timer_mock.__index = timer_mock
function timer_mock:start() self.started = true end
function timer_mock:stop() self.started = false end
function timer_mock:again() self.started = true end
function timer_mock:is_started() return self.started end

setmetatable(timer_mock, {
    __index = timer_mock,
    __call = function()
        return setmetatable({}, timer_mock)
    end
})

_G.__core = {
    timer = timer_mock
}