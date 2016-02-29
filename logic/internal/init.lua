local m = require "logic/internal/pack"
function remote_cmd(cmd, callback)
	return SendToRemote(MsgPack.pack({Head=cmd}), 3, callback)
end
return m
