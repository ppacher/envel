--[[ Example Rule

local was_running = false
rule{
    name = "Alert when whashing machine finished",
    trigger = {
        onSignal(plug, "property::power"),
        -- onPropertyChange(plug, "power"),
    },
    when = function()
        local prev = was_running
        was_running = plug.is_running
        if prev and not plug.is_running then
            return true
        end
    end,
    action = function()
        notify{text = "Washing machine finished"}
    end
}


local signal = require("envel.signal")
local f = signal()
rule {
    name = "emit when called",
    trigger = onInterval(100),
    action = signal.emit(f, "changed", {"foobar"})
}

-- will print "foobar" evenry 100 seconds
f:connect_signal("changed", function(e) print(e) end )

--]]

local rule_class = {}

local function subscribe_triggers(trigger, cb)
    if type(trigger) == 'function' then
        return trigger(cb)
    end

    local cleanups = {}
    for _, t in ipairs(trigger) do
        local unsub = t(cb)
        table.insert(cleanups, unsub)
    end

    return function()
        for _, t in ipairs(cleanups) do
            -- make sure we don't fail during cleanup
            pcall(t)
        end
    end
end

--- Executes the rule's action and does some logging
function rule_class:run_action()
    print("running rule: "..tostring(self.name))
    self.action()
end

--- Run checks if the rule's when condition is fullfilled (if any)
-- and executes the rule's action
function rule_class:run()
    if self.when ~= nil then
        if self.when() then
            self:run_action()
        end
    else
        self:run_action()
    end
end


--- Creates a new rule
-- @param rule  The rule description to create
-- @returns A rule
local function create_rule(rule)
    local newinst = {
        name = rule.name,
        when = rule.when,
        action = rule.action
    }

    local rule_callback = function()
        newinst:run()
    end

    newinst.close = subscribe_triggers(rule.trigger, rule_callback)

    setmetatable(newinst, {__index = rule_class})

    return newinst
end

local module = {
    rules = {}
}

-- make module.rules a value-weak table
-- to ensure we don't keep rules that could be garbage collected
-- keys are numeric so it doesn't matter if they are weak or not
setmetatable(module.rules, {
    __mode = "v"
})

--- Trigger defines the type of function(s) expected by rule.trigger
-- @class function
-- @name Trigger
-- @param cb The callback function to invoke when triggered
-- @return A function to call to remove the registration
--
-- @example: function(cb) return singal:connect_signal("foo", cb) end
--

--- Returns a trigger function that connects to a signal
-- @param obj       The object that emits the signal
-- @param signal    The signal to subscribe to
-- @returns         trigger function
function module.onSignal(obj, signal)
    return function(cb)
        obj:connect_signal(signal, cb)

        return function() obj:disconnect_signal(signal, cb) end
    end
end

--- Returns a trigger function that connects to a property changesignal
-- @param obj       The object that emits the signal
-- @param property  The property to subscribe to
-- @returns         trigger function
function module.onPropertyChange(obj, property)
    return module.onSignal(obj, "property::"..property)
end


--- Returns a trigger function that triggers whenever the provided
-- timer ticks
-- @param timer     The time object
-- @returns         trigger function
function module.onTimer(timer)
    return module.onSignal(timer, "timer::tick")
end

--- Returns a trigger function that runs the rule every given interval
-- @param interval      Number of seconds between rule executions
-- @returns             trigger function
function module.onInterval(interval)
    return function(cb)
        local t = require("envel.timer"){
            timeout = interval,
            callback = cb,
            autostart = true,
            call_now = false,
        }

        return function()
            t:stop()
        end
    end
end


--- Verifies if the provided rule has every thing setup correctly
-- @param rule  The rule to verify
-- @returns A string describing an error or nil
function module:verify_rule(rule)
    local _ = self

    -- each rule must have a unique name
    if type(rule.name) ~= 'string' then
        return 'name must be a string'
    end

    for _, r in ipairs(self.rules) do
        if r.name == rule.name then
            return "Rule with name "..tostring(r.name).." already registered"
        end
    end

    -- the rules trigger must either be a function or a list of functions
    if type(rule.trigger) ~= 'table' and type(rule.trigger) ~= 'function' then
        return 'trigger must be set to a function or list of functions'
    end

    if type(rule.trigger) == 'table' then
        for _, t in ipairs(rule.trigger) do
            if type(t) ~= 'function' then
                return 'triggers must be functions'
            end
        end
    end

    -- the rules condition must either be nil or a function
    if type(rule.when) ~= 'function' and rule.when ~= nil then
        return 'when must be a function'
    end

    -- the rules action must always be set to a function
    if type(rule.action) ~= 'function' then
        return 'action must be a function'
    end

    return nil
end

--- Configures a new rule
-- @param rule  The rule to add
function module:add_rule(rule)
    local err = self:verify_rule(rule)
    if err ~= nil then error(err) end

    local r = create_rule(rule)

    table.insert(self.rules, r)
    return r
end

return setmetatable(module, {
    __call = function(self, rule)
        return self:add_rule(rule)
    end
})