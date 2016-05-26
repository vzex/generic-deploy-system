if bInit then
        return {name="测试1"}
end
if not Single() then return end

remote_cmd("sleep 2", function() end)
local endStr = remote_cmd("ls -r", function(aa) 
	local tbl = MsgPack.unpack(aa)
	for a,b in pairs(tbl) do 
                if type(b) == "string" then
                        local_msg(from64(b))
                end
        end
end)

--print("test over", endStr)
