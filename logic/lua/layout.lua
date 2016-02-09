add_button("qa1", "test1", testa)
add_button("qa1", "test2", testb)
add_button("qa2", "test1", testc)


function testa()
end

function testb()
end

function testc(arg)
	--arg:{name="test1", sessionId=123, group="qa1", target={nick1,nick2}}
	--remote_cmd("ls -l", function(status, recv)
	--	print(status, recv)
	--end)
	--server_upload("/tmp/a.txt", "/home/pangu/a.txt", callback)
	--server_download("/home/pangu/a.txt", "/tmp/a.txt", callback)
	--
	--local_server_download("/tmp/a.txt", callback)
	--local_remote_download("/tmp/a.txt", targetNick, callback)
	--local_remote_upload("/tmp/a.txt", targetNick, callback) arg1 is target file.this will choose a file from broswer
	--local_server_upload("/tmp/a.txt", targetNick, callback)
	--local_getinput("ok?", callback)
	--local_confirm("ok?", callback)
	--local_output("ssss")
	--
	--global_setmap(key, value)
	--global_setmap_ifeq(key, value, old)
	--global_getmap(key)
	--
	--
	--session_defer_func(callback)
end

