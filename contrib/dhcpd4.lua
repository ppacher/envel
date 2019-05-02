local stream = require("envel.stream")
local spawn  = require("envel.spawn")

local function split(inputstr, sep)
    if sep == nil then
            sep = "%s"
    end
    local t={}
    for str in string.gmatch(inputstr, "([^"..sep.."]+)") do
        table.insert(t, str)
    end
    return t
end

return function(cfg)
    if not cfg then cfg = {} end

    cfg.log_file = cfg.log_file or '/var/log/dhcpd4.log'
    cfg.timeout = cfg.timeout or 10 -- TODO(ppacher): doesn't work for non-number values

    local function producer(observer)
        local timer = spawn.watch {
            -- TODO(ppacher): add support to specify the command
            cmd = "tail -f " .. cfg.log_file,
            timeout = cfg.timeout,
            line_callback = function(out)
                if out == nil then return end

                local parts = split(out)
                local dhcprequest = parts[6]
                local ip = parts[8]
                local mac
                local check = false

                if string.match(out, "DHCPREQUEST") then
                    check = true
                    if parts[9] == "from" then
                        mac = parts[10]
                    else
                        mac = parts[11]
                    end

                elseif string.match(out, "DHCPOFFER") then
                    check = true
                    mac = parts[10]
                end

                if check then
                    observer:next({
                        ip = ip,
                        mac = mac,
                        type = dhcprequest,
                    })
                end
            end
        }

        -- we return a cleanup function called once
        -- everyone unsubscribed
        return function()
            -- using a copy we can ensure garbage collection of the
            -- timer even if something during :stop() wents terribly
            --
            local copy = timer
            timer = nil
            copy:stop()
        end
    end

    return stream.Observable.create(producer)
        -- TODO(ppacher): add :publishLast() once it's implemented
        :multicast(function() return stream.Subject.create() end)
        :refCount()
end
