if bInit then
        return {multicontrol=true}
end
if not Single() then return end
ServerUploadToRemote()
LocalUploadToServer()
RemoteDownToServer()
ServerDownToLocal()

SendToSpecRemote()
--print("test over", endStr)
