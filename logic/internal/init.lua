local m = require "logic/internal/pack"
json = require "logic/internal/json"
-- SendToRemote(str, timeoutsec, callback(data))
function remote_cmd(cmd, callback, timeout)
	return SendToRemote(MsgPack.pack({Action="cmd", Cmd=cmd}), timeout or 10, callback)
end
function local_msg(msg)
	return SendToLocal(json.encode({Action="msg", Msg=msg}))
end
return m
