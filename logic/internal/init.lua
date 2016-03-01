local m = require "logic/internal/pack"
-- SendToRemote(str, timeoutsec, callback(data))
function remote_cmd(cmd, callback, timeout)
	return SendToRemote(MsgPack.pack({Action="cmd", Cmd=cmd}), timeout or 10, callback)
end
return m
