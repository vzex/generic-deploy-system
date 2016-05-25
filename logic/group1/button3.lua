if bInit then
        return {multicontrol=true, name="updown"}
end
if not Single() then return end
local er = LocalUploadToServer("test")
if er == nil then
        local_msg("upload ok")
else
        local_msg("upload fail:"..er)
end

local er = LocalDownFromServer("test")
if er == nil then
        local_msg("down ok")
else
        local_msg("down fail:"..er)
end

--[[
SendToSpecRemote()
]]
