function _handle(msg)
	local info = MsgPack.unpack(msg)
	_G[info.Head](info)
end

function test(info)
	for a,b in pairs(info) do
		print(a,b)
		SendToRemote(MsgPack.pack({"response", a, b}))
	end
	SendToRemoteEnd("final")
end

