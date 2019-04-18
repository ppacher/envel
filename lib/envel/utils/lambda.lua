return function(str)
    -- do nothing if we already got a function
    if type(str) == 'function' then return str end

    local parts = {}
    for m in string.gmatch(str, '([^=>]+)') do
        table.insert(parts, m)
    end

    if #parts ~= 2 then
        error("invalid lambda function specified")
    end

    local params = {}
    for p in string.gmatch(parts[1], "%a+%d*") do
        table.insert(params, p)
    end

    local fn_str = "return function(" .. table.concat(params, ", ") .. ")\n return "
    fn_str = fn_str .. parts[2]
    fn_str = fn_str .. "\nend"

    local fn = loadstring(fn_str)
    return fn()
end