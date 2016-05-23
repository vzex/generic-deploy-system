if bInit then
        return {multicontrol=true, name="upload"}
end
if not Single() then return end
--[[ServerUploadToRemote("./server.go", "/tmp/a.go", function(er) 
        if er ~= "" then
                local_msg(er)
        else
                local_msg("upload ok")
        end
end)
ServerDownFromRemote("/tmp/a.go", "/tmp/b.go", function(er) 
        if er ~= "" then
                local_msg(er)
        else
                local_msg("download ok")
        end
end)]]
LocalUploadToServer("test")
--[[
ServerUploadToRemote()
LocalUploadToServer()
RemoteDownToServer()
ServerDownToLocal()

SendToSpecRemote()
Connect(ip, port, function(recv, status)
	print(recv)
end)
]]
--print("test over", endStr)
