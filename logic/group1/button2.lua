
if bInit then
        return 
end
print(MachineName, MachineAddr, 1+2)
local endStr = remote_bash("ls -r|wc -l", function(aa) 
	local tbl = MsgPack.unpack(aa)
	for a,b in pairs(tbl) do 
                if type(b) == "string" then
                        local_msg(b)
                end
        end
end)
