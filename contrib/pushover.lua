local http = require("envel.http")
local json = require("json")

-- converts a dbus.notify priorty to a Pushover priority
local function convert_priority(t)
    if type(t) == 'number' then
        t = t - 1
    end

    if t == 'low' then return -1 end
    if t == 'normal' then return 0 end
    if t == 'high' then return 1 end
    if t == 'critical' then return 2 end

    error('unsupported priority')
end

-- send_message sends a message via the pushover service
local function send_message(key, msg)
    if msg.html and msg.monospace then 
        error('Only html or monospace can be set')
    end

    local html = 0
    if msg.html then html = 1 end
    local monospace = 0
    if msg.monospace then monospace = 1 end

    local retry = msg.retry
    local expire = msg.expire

    local priority = convert_priority(msg.priority or 'normal')

    if priority == 2 then
        if retry == nil then retry = 60 end
        if expire == nil then expire = 10*60 end
    end

    local body = json.encode({
        token = key,
        user = msg.user or error('Missing user'),
        title = msg.title or '',
        message = msg.text or '',
        html = html,
        monospace = monospace,
        device = msg.device,
        sound = msg.sound or 'none',
        timestamp = msg.timestamp or nil,
        priority = priority,
        retry = retry,
        expire = expire,
        url = msg.url,
        url_title = msg.url_title,
    })

    local res, err = http {
        method = "POST",
        url = "https://api.pushover.net/1/messages.json",
        --url = "http://postman-echo.com/post",
        headers = {
            ["Content-Type"] = "application/json",
        },
        body = body,
    }

    if err ~= nil or res.status_code ~= 200 then
        print("failed to send notification (status_code="..tostring(res.status_code).."): "..(err or res.status))
        print(res.body:read("*a"))
    end
end

local pushover_cls = {}

function pushover_cls:notify(msg)
    if not msg.user then
        msg.user = self.user
    end

    local key = self.apiKey
    if msg.key then
        key = msg.key
    end

    return send_message(key, self:set_defaults(msg))
end

function pushover_cls:set_defaults(msg)
    msg.user = msg.user or self.user or nil
    msg.retry = msg.retry or self.retry or nil
    msg.expire = msg.expire or self.expire or nil
    msg.sound = msg.sound or self.sound or nil
    msg.device = msg.device or self.device or nil
    msg.priority = msg.priority or self.priority or nil

    return msg
end

local function new(opts)
    local newinst = {
        apiKey = opts.key or error('Missing API key for pushover'),
        user = opts.user or nil,
        retry = opts.retry or nil,
        expire = opts.expire or nil,
        priority = opts.priority or nil,
        sound = opts.sound or nil,
        device = opts.device or nil,
    }

    setmetatable(newinst, {__index = pushover_cls})
    return newinst
end

return setmetatable({}, {
    __call = function(_, opts)
        return new(opts)
    end
})