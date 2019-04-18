local function get_path(tbl, path)
    for _, p in ipairs(path) do
        tbl = tbl[p]

        if tbl == nil then
            return nil
        end
    end

    return tbl
end

return {
    transform = require("envel.utils.transform"),
    lambda = require("envel.utils.lambda"),
    get_path = get_path,
}