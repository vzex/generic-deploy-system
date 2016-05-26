local m = require "logic/internal/pack"
json = require "logic/internal/json"
-- SendToRemote(str, timeoutsec, callback(data))
function remote_cmd(cmd, callback, timeout)
	return SendToRemote(MsgPack.pack({Action="cmd", Cmd=cmd}), timeout or 10, callback)
end
function remote_bash(cmd, callback, timeout)
	return SendToRemote(MsgPack.pack({Action="bash", Cmd=cmd}), timeout or 10, callback)
end
function local_msg(msg)
	return SendToLocal(json.encode({Action="msg", Msg=msg}))
end
function local_dialog(msg)
	return SendToLocal(json.encode({Action="dialog", Msg=msg}))
end
function remote_print(msg)
	return SendToRemote(MsgPack.pack({Action="print", Cmd=msg}), timeout or 10, callback)
end
return m
