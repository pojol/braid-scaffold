
-- 字节序
-- 大端：BigEndian
-- 小端：LittleEndian
ByteOrder = "LittleEndian"


-- 从报文中解析出消息ID和消息体
-- 示例，用户可以参照实际报文格式进行解析
function WSUnpackMsg(buf, errmsg)

    if errmsg ~= "nil" then
        return "", ""
    end

    local msg = message.new(buf, ByteOrder, 0)

    local headerLen = msg:readi2()
    local msgHeader = msg:readBytes(headerLen)
    local msgbody = msg:readBytes(-1)

    return msgHeader, msgbody

end

function WSPackMsg(msgHead, msgBody)

    local msg = message.new("", ByteOrder, 2+#msgHead+#msgBody)
    msg:writei2(#msgHead)
    msg:writeBytes(msgHead)
    msg:writeBytes(msgBody)
    return msg:pack()

end

------------------------------------------------------------------------
-- msglen : conn will first read the predefined message length field 
function TCPUnpackMsg(msglen, buf, errmsg)
    if errmsg ~= "nil" then
        return 0, ""
    end

    local msg = message.new(buf, ByteOrder, 0)
    local msgId = msg:readi2()
    local msgbody = msg:readBytes(2, -1)

    return msgId, msgbody

end

function TCPPackMsg(msgid, msgbody)
    local msglen = #msgbody+2

    local msg = message.new("", ByteOrder, msglen)
    msg:writei2(msgid)
    msg:writeBytes(msgbody)

    return msg:pack()

end