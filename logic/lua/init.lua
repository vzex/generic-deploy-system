
function class(parent)
        local t = {__declared = {}, __subclass = {}, __functions = {}, __parent = parent}
        if parent then
                local f = function(key)
                        for k, v in pairs(parent[key]) do
                                t[key][k] = v
                        end
                end
                f("__declared")
                f("__subclass")
                f("__functions")
		t.__functions["ctor"] = nil
        end
        local mt = {
                __index = function(tbl, k)
                        if k == "pack" then
                                return function() return mp.pack(tbl) end
                        elseif k == "unpack" then
                                return function(str) 
                                        local _tbl = mp.unpack(str) 
                                        local function f(c, t, tmp)
                                                for m, n in pairs(tmp) do
                                                        local sub = c.__subclass[m]
                                                        if not sub then
                                                                t[m] = n
                                                        else
                                                                local subc = sub.new()
                                                                f(sub, subc, n)
                                                                t[m] = subc
                                                        end
                                                end
                                        end
                                        f(t, tbl, _tbl)
                                end	
                        end
                        if not t.__declared[k] then
                                print("not declared key", k)
                                return nil
                        end
                        return t.__functions[k]
                end,
                __newindex = function(tbl, k, v)
                        if not t.__declared[k] then
                                print("set not declared key", k)
                                return
                        end
                        rawset(tbl, k, v)
                end
        }
        t.new = function(...)
                local n = {}
                setmetatable(n, mt)
                for member, sub in pairs(t.__subclass) do
                        n[member] = sub.new()
                end
		local function f(_t, ...)
			if _t then
				f(_t.__parent, ...)
				if _t.__functions["ctor"] then
					_t.__functions["ctor"](n, ...)
				end
			end
		end
		f(t.__parent, ...)
                if t.__functions["ctor"] then
                        t.__functions["ctor"](n, ...)
                end
                return n
        end
	t.reg = function(membername, subclass)
		if t.__declared[membername] then
			print("already defined", membername)
			return
		end
		--local m = class.__indextokey
		--table.insert(m, membername)
		--class.__declared[membername] = #m
		t.__declared[membername] = true
		t.__subclass[membername] = subclass
	end
        local mf = {
                __newindex = function(tbl, k, v)
                        if type(v) == "function" then
				t.__functions[k] = v
                                t.__declared[k] = true
                                return
                        end
                end,
		__index = function(tbl, k)
			return t.__functions[k]
		end
        }
        setmetatable(t, mf)
        return t
end

