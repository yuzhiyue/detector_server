package main
import (
    "net"
    "fmt"
    "protocol"
    "time"
    "bytes"
    "encoding/binary"
)



type Detector struct {
    Id int
    ProtoVer uint8
    MAC string
    Longitude float32
    Atitude float32
    Status int
    conn net.Conn
}

func (detector * Detector)SendMsg(cmd uint8, msg []byte)  {
    buff := new(bytes.Buffer)
    binary.Write(buff, binary.BigEndian, uint16(0xf9f9))
    binary.Write(buff, binary.BigEndian, uint16(len(msg)) + protocol.CRC16Len + protocol.HeaderLen - uint16(4))
    binary.Write(buff, binary.BigEndian, cmd)
    binary.Write(buff, binary.BigEndian, msg)
    crc16 := protocol.GenCRC16(buff.Bytes())
    binary.Write(buff, binary.BigEndian, crc16)
    detector.conn.Write(buff.Bytes());
}

func onDetectorLogin(detector * Detector, request protocol.LoginRequest) {
    fmt.Println("onDetectorLogin, request:", request)
    detector.Status = 1
    detector.ProtoVer = request.ProtoVer
    response := protocol.LoginResponse{}
    response.ProtoVer = request.ProtoVer
    response.Seq = request.Seq
    response.Time = uint32(time.Now().Unix())
    buff := response.Encode()
    fmt.Println("response:", buff)
    detector.SendMsg(1, buff)
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
        detector.SendMsg(2, nil)
        break;
    }
    case 3: {
        break;
    }
    }
}

func handleConn(conn net.Conn) {
    defer conn.Close()
    detector := Detector {}
    detector.conn = conn
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
                if !protocol.CheckCRC16(buff[:header.MsgLen]) {
                    fmt.Println("check crc failed")
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
