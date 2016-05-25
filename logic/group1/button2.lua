
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
local res = connect("www.163.com:80", 10, function(conn, status, str)
        --conn("close")
        if status == "connected" then
                conn("send", "GET / HTTP/1.1\r\n\n")
        else
                print(str)
        end
        end)
print("dial over", res)
local tbl = GetNickList()
for _, nick in ipairs(tbl) do
        print("nicklist of", nick)
end
local nick = tbl[math.random(1, #tbl)]
print("choose nick", nick, "for remote call")
local back = SendToNick(nick, "hidefunc", "extraarg")
print("test back", back)
