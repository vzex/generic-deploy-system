if bInit then
        return {hide=true}
end
if not Single() then return end

remote_print("haha, this is called from another machine", ExtraArg)
return "back over"
