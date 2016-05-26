local m = require "logic_remote/internal/pack"
function handle_cmd(info)
        local commond = from64(info.Cmd)
        local s, ok = cmd(commond)
        SendBack(MsgPack.pack({str=base64(s), ok = ok}))
end
function handle_bash(info)
        local commond = from64(info.Cmd)
        local s, ok = bash(commond)
        SendBack(MsgPack.pack({str=base64(s), ok = ok}))
end
function handle_print(info)
        print(from64(info.Cmd))
end
return m
