-- Example Lua plugin for Solar.
-- Build server with: go build -tags=lua -o bin/solar ./cmd/solar
-- Place this file in data/lua/example.lua and set [lua] enabled = true.

-- Welcome players on connect
on_connect(function()
    broadcast("&e[Lua] A player joined!")
end)

-- Filter chat: block "badword", allow modification
on_chat(function(name, msg)
    if string.find(string.lower(msg), "badword") then
        broadcast("&c" .. name .. ", watch your language!")
        return false  -- cancel
    end
    -- return a string to modify the message
    return msg
end)

-- Log block placements
on_block_change(function(name, x, y, z, block, placing)
    if placing then
        broadcast("&a" .. name .. " placed block " .. tostring(block) .. " at " .. x .. "," .. y .. "," .. z)
    end
end)

-- Periodic tick announcement (every 400 ticks = 20 seconds at 20 TPS)
on_tick(function(tick)
    if tick % 400 == 0 then
        broadcast("&7[Lua] Server tick: " .. tostring(tick))
    end
end)

-- Cancel /kick command (example of command interception)
on_command(function(name, cmd, args)
    if cmd == "nokick" then
        broadcast("&c" .. name .. " tried to use a blocked command")
        return false
    end
end)

-- Register a custom command
register_command("luahello", "Say hello from Lua", function(name, args)
    return "&eHello from Lua, " .. name .. "!"
end)
