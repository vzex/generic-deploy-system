if bInit then
        return {multicontrol=true}
end
if not Single() then return end
ServerUploadToRemote()
LocalUploadToServer()
RemoteDownToServer()
ServerDownToLocal()

SendToSpecRemote()
Connect(ip, port, function(recv, status)
	print(recv)
end)
--print("test over", endStr)
