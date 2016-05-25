local m = require "logic/internal/pack"
function handle_cmd(info)
        local commond = info.Cmd
        local s, ok = cmd(commond)
        SendBack(MsgPack.pack({str=s, ok = ok}))
end
function handle_bash(info)
        local commond = info.Cmd
        local s, ok = bash(commond)
        SendBack(MsgPack.pack({str=s, ok = ok}))
end
function handle_print(info)
        print(info.Cmd)
end
return m
