-- Example Lua plugin for Solar — demonstrates the full API.
-- Build server with: go build -tags=lua -o bin/solar ./cmd/solar
-- Place this file in data/lua/example.lua and set [lua] enabled = true.

-- ─── events ───

on_connect(function(player)
    broadcast("&eWelcome, " .. player:name() .. "!")
end)

on_disconnect(function(player, reason)
    broadcast("&7" .. player:name() .. " left: " .. reason)
end)

on_chat(function(player, msg)
    if string.find(string.lower(msg), "badword") then
        player:message("&cWatch your language!")
        return false
    end
    return msg  -- allow modification
end)

on_move(function(player, x, y, z, yaw, pitch)
    -- return false to reject movement
end)

on_block_change(function(player, x, y, z, block, placing)
    if placing then
        broadcast("&a" .. player:name() .. " placed " .. tostring(block) .. " at " .. x .. "," .. y .. "," .. z)
    end
    -- return false to cancel
end)

on_block_changed(function(player, x, y, z, block, placing)
    -- fires after the change, not cancelable
end)

on_click(function(player, button, action, entity_id, x, y, z, face)
    -- button: 0=left, 1=right, 2=middle
end)

on_command(function(player, cmd, args)
    if cmd == "blocked" then
        player:message("&cThat command is blocked!")
        return false
    end
end)

on_tick(function(tick)
    if tick % 400 == 0 then
        broadcast("&7[Lua] Tick: " .. tostring(tick))
    end
end)

on_joining_level(function(player, level_name)
    broadcast("&b" .. player:name() .. " is going to " .. level_name)
end)

on_joined_level(function(player, level_name, prev_level)
    broadcast("&b" .. player:name() .. " arrived from " .. prev_level .. " to " .. level_name)
end)

on_dying(function(player, cause)
    broadcast("&c" .. player:name() .. " is dying! Cause: " .. tostring(cause))
    -- return false to prevent death
end)

on_died(function(player, cause, cooldown)
    broadcast("&c" .. player:name() .. " died. Cooldown: " .. tostring(cooldown))
    return cooldown * 2  -- double the cooldown
end)

on_player_spawn(function(player, x, y, z, yaw, pitch)
    -- can modify spawn position by returning new values
    return x, y + 1, z, yaw, pitch
end)

on_shutdown(function(reason)
    broadcast("&cServer shutting down: " .. reason)
end)

on_level_save(function()
    -- return false to skip save
end)

on_level_loaded(function(name)
    broadcast("&aLevel loaded: " .. name)
end)

on_physics_update(function(x, y, z, block, level)
    -- physics block updated
end)

-- ─── commands ───

register_command("luahello", "Say hello from Lua", function(player_name, args)
    return "&eHello from Lua, " .. player_name .. "!"
end)

register_command("luatp", "Teleport to coordinates", function(player_name, args)
    local p = server:find_player(player_name)
    if p then
        local x = tonumber(args[1]) or 0
        local y = tonumber(args[2]) or 0
        local z = tonumber(args[3]) or 0
        p:teleport(x, y, z, 0, 0)
        return "&aTeleported to " .. x .. "," .. y .. "," .. z
    end
    return "&cPlayer not found"
end)

register_command("luafreeze", "Freeze yourself", function(player_name, args)
    local p = server:find_player(player_name)
    if p then
        p:set_frozen(true)
        return "&bYou are now frozen!"
    end
    return "&cPlayer not found"
end)

-- ─── server API ───

register_command("luaonline", "List online players", function(player_name, args)
    local count = server:online_count()
    return "&aOnline: " .. tostring(count) .. " players"
end)

register_command("lualevels", "List loaded levels", function(player_name, args)
    local levels = server:levels()
    local names = levels:list()
    local result = "&aLevels: "
    for i, name in ipairs(names) do
        result = result .. name
        if i < #names then result = result .. ", " end
    end
    return result
end)

-- ─── scheduler ───

-- Run a task every 5 seconds
server:scheduler():every(5000, function()
    broadcast("&7[Lua] Scheduled tick from Lua!")
end)
