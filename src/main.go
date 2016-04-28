package main
import (
    "net"
    "fmt"
    "protocol"
)



type Detector struct {
    id int
    mac string
    longitude float32
    atitude float32
    status int
    onn net.Conn
}

func onDetectorLogin(detetctor * Detector, request protocol.LoginRequest) {
    fmt.Println("onDetectorLogin, request:", request)
    detetctor.status = 1
}

func handleMsg(detector * Detector, cmd uint8, msg []byte)  {
    fmt.Println("recv request, cmd:", cmd, msg)
    switch cmd {
    case 1: {
        reqest := protocol.LoginRequest{};
        reqest.Decode(msg)
        onDetectorLogin(detector, reqest)
        break;
    }
    case 2: {
        break;
    }
    case 3: {
        break;
    }
    }
}

func handleConn(conn net.Conn) {
    defer conn.Close()
    detector := Detector {0, "", 0, 0, 0, conn}
    buff := make([]byte, 1024 * 32)
    var buffUsed uint16 = 0;
    header := protocol.MsgHeader{}
    for {
        len, err := conn.Read(buff[buffUsed:])
        if err != nil {
            fmt.Println("recv data err", err)
            return;
        }
        fmt.Println("recv data, len:", len)
        buffUsed += uint16(len)
        for {
            if header.MsgLen == 0 {
                if buffUsed >= protocol.HeaderLen {
                    header.Decode(buff)
                    if header.Magic != 0xf9f9 {
                        fmt.Println("decode header, magic err", header.Magic)
                        return
                    }
                    if header.MsgLen > uint16(cap(buff)) {
                        fmt.Println("msg too big, size", header.MsgLen)
                        return;
                    }
                    fmt.Println("decode header, msg len", header.MsgLen)
                }
            }
            if header.MsgLen != 0 && buffUsed >= header.MsgLen {
                if !protocol.CheckCrc16(buff) {
                    return
                }
                handleMsg(&detector, header.Cmd, buff[protocol.HeaderLen : header.MsgLen - protocol.CRC16Len])
                copy(buff, buff[header.MsgLen:buffUsed])
                buffUsed -= header.MsgLen
                header.Magic = 0
                header.MsgLen = 0
                header.Cmd = 0
            } else {
                break;
            }
        }

    }
}

func main()  {
    listen_address := ":10001"
    listen, err := net.Listen("tcp", listen_address)
    if err != nil {
        return
    }
    fmt.Println("server start, listen on", listen_address)
    defer listen.Close();
    for {
        conn, err := listen.Accept();
        if err != nil {
            return
        }
        fmt.Println("accept new connection")
        go handleConn(conn)
    }
    return
}
