if bInit then
        return {name="input"}
end
if not Single() then return end
local v = LocalGetInput()
if v and v ~= "" then
        local_dialog(v)
end
