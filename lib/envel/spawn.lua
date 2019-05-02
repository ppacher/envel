--- Module spawn also to call external processes in an asynchronous manner which does not
-- block the event loop of envel

local exec = _G.__core.exec
local timer = require("envel.timer")

local spawn = {}

local function call_maybe_shell(cmd, shell, on_done)
    local line_cb = nil
    local done_cb = nil

    if on_done ~= nil then
        local out = ""
        local err = ""
        line_cb = function(stdout, stderr)
            -- we need to add back the newline
            -- for stdout and stderr
            if stdout ~= nil then
                out = out .. '\n' .. stdout
            end

            if stderr ~= nil then
                err = err .. '\n' .. stderr
            end
        end

        done_cb = function(reason, code_or_signal)
            on_done(out, err, reason, code_or_signal)
        end
    end

    exec(cmd, shell or false, line_cb, done_cb)
end

-- executes a command and calls the on_done callback providing
-- stdout, stderr, exit reason and exit code
function spawn.easy_async(cmd, on_done)
    return call_maybe_shell(cmd, false, on_done)
end

-- Like spawn.easy_async but executes the command inside a shell
function spawn.easy_async_with_shell(cmd, on_done)
    return call_maybe_shell(cmd, true, on_done)
end

-- Executes a command and calls on_line for each newline printed to
-- stdout or stderr. on_done is execute once the command exits and
-- provides the exit reason (exit or signal) and the code/signal
-- depending on the reason
function spawn.with_line_callback(cmd, on_line, on_done)
    return exec(cmd, false, on_line, on_done)
end

-- Periodically executes a command
function spawn.watch(args)
    local cmd = args.cmd
    local timeout = args.timeout
    local on_start = args.start_callback
    local on_line = args.line_callback
    local on_done = args.exit_callback

    if type(cmd) ~= 'string' then
        error('cmd must be set to a string, got "'..type(cmd)..'"')
    end

    local t = timer {
        timeout = timeout,
        autostart = true,
        call_now = true,
        callback = function(t1)
            -- we stop the timer and restart it once the command finishes
            -- so we don't run the command multiple times if it takes longer to
            -- execute than the configured timeout
            t1:stop()

            if type(on_start) == 'function' then on_start() end
            spawn.with_line_callback(cmd, on_line, function(...)
                if type(on_done) == 'function' then on_done(unpack(arg)) end
                t1:start()
            end)
        end
    }
    return t
end

setmetatable(spawn, {
    __call = function(_, cmd)
        return call_maybe_shell(cmd, false, nil)
    end
})

return spawn