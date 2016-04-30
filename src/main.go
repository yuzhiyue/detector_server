package main
import (
    "net"
    "protocol"
    "time"
    "bytes"
    "encoding/binary"
    "db"
    "os"
    "log"
)

type Detector struct {
    Id int
    ProtoVer uint8
    MAC string
    Longitude int32
    Atitude int32
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

func OnDetectorLogin(detector * Detector, request protocol.LoginRequest) {
    log.Println("onDetectorLogin, request:", request)
    detector.MAC = request.MAC
    detector.Status = 1
    detector.ProtoVer = request.ProtoVer

    db.CreateDetector(request.MAC, request.IMEI)

    response := protocol.LoginResponse{}
    response.ProtoVer = request.ProtoVer
    response.Seq = request.Seq
    response.Time = uint32(time.Now().Unix())
    buff := response.Encode()
    log.Println("response:", buff)
    detector.SendMsg(1, buff)
}

func OnReport(detector *Detector, request protocol.ReportRequest)  {
    log.Println("onReport, request:", request)
    db.SaveDetectorReport(detector.MAC, request.ReportList)
}

func handleMsg(detector * Detector, cmd uint8, msg []byte)  {
    log.Println("recv request, cmd:", cmd, msg)
    switch cmd {
    case 1: {
        request := protocol.LoginRequest{};
        request.Decode(msg)
        OnDetectorLogin(detector, request)
        break;
    }
    case 2: {
        detector.SendMsg(2, nil)
        db.UpdateDetector(detector.MAC, 0, 0)
        break;
    }
    case 3: {
        request := protocol.ReportRequest{};
        request.Decode(msg)
        OnReport(detector, request)
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
            log.Println("recv data err", err)
            return;
        }
        log.Println("recv data, len:", len)
        buffUsed += uint16(len)
        for {
            if header.MsgLen == 0 {
                if buffUsed >= protocol.HeaderLen {
                    header.Decode(buff)
                    if header.Magic != 0xf9f9 {
                        log.Println("decode header, magic err", header.Magic)
                        return
                    }
                    if header.MsgLen > uint16(cap(buff)) {
                        log.Println("msg too big, size", header.MsgLen)
                        return;
                    }
                    log.Println("decode header, msg len", header.MsgLen)
                }
            }
            if header.MsgLen != 0 && buffUsed >= header.MsgLen {
                if !protocol.CheckCRC16(buff[:header.MsgLen]) {
                    log.Println("check crc failed")
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
    logFile, logErr := os.OpenFile("./detector_server.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
    if logErr != nil {
        log.Println("Fail to find", "./log/detector_server.log", "cServer start Failed")
        os.Exit(1)
    }
    log.SetOutput(logFile)
    log.SetFlags(log.Ldate | log.Ltime | log.Lshortfile)

    db.InitDB()
    listen_address := ":10001"
    listen, err := net.Listen("tcp", listen_address)
    if err != nil {
        return
    }
    log.Println("server start, listen on", listen_address)
    defer listen.Close();
    for {
        conn, err := listen.Accept();
        if err != nil {
            return
        }
        log.Println("accept new connection")
        go handleConn(conn)
    }
    return
}
