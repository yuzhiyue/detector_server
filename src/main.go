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
    "fmt"
    "gopkg.in/mgo.v2/bson"
)

type Detector struct {
    Id        int
    ProtoVer  uint8
    MAC       string
    IMEI      string
    Longitude int32
    Latitude  int32
    Status    int
    GeoUpdateType int
    LastRecvTime uint32
    conn      net.Conn
}

func (detector * Detector)SendMsg(cmd uint8, seq uint16, msg []byte)  {
    buff := new(bytes.Buffer)
    binary.Write(buff, binary.BigEndian, uint16(0xf9f9))
    msgLen := uint16(len(msg)) + protocol.CRC16Len + protocol.HeaderLen - uint16(4);
    if cmd != 2 {
        msgLen += protocol.SeqLen
    }
    binary.Write(buff, binary.BigEndian, msgLen)
    binary.Write(buff, binary.BigEndian, cmd)
    binary.Write(buff, binary.BigEndian, msg)
    if cmd != 2 {
        binary.Write(buff, binary.BigEndian, seq)
    }
    crc16 := protocol.GenCRC16(buff.Bytes())
    binary.Write(buff, binary.BigEndian, crc16)
    detector.conn.Write(buff.Bytes());
}

func OnDetectorLogin(cmd uint8, seq uint16, detector * Detector, request * protocol.LoginRequest) {
    log.Println("onDetectorLogin, request:", request)
    result := bson.M{}
    err := db.GetDetectorInfo(request.IMEI, &result)
    if err != nil {
        db.CreateDetector(request.IMEI, request.MAC)
        //db.CreateDetector(request.MAC, request.IMEI)
    } else {
        db.UpdateLoginTime(request.IMEI)
        detector.Longitude = int32(db.GetNumber(result, "longitude") * protocol.GeoMmultiple)
        detector.Latitude = int32(db.GetNumber(result, "latitude") * protocol.GeoMmultiple)
        detector.GeoUpdateType = int(db.GetNumber(result, "geo_update_type"))
    }
    detector.MAC = request.MAC
    detector.IMEI = request.IMEI
    detector.Status = 1
    detector.ProtoVer = request.ProtoVer

    response := protocol.LoginResponse{}
    response.ProtoVer = protocol.MaxProtoVer
    response.Time = uint32(time.Now().Unix())
    buff := response.Encode()
    log.Println("response:", buff)
    detector.SendMsg(cmd, seq, buff)
}

func OnReport(cmd uint8, seq uint16,detector *Detector, request * protocol.ReportRequest)  {
    if detector.Status != 1 {
        log.Println("invalid detector report")
        return
    }
    log.Println("onReport, request:", request)
    for e := request.ReportList.Front(); e != nil; e = e.Next() {
        info := e.Value.(*protocol.ReportInfo)
        if info.Time == 0 {
            info.Time = uint32(time.Now().Unix())
        }
        if (info.Latitude == 0 || info.Longitude == 0) {
            info.Longitude, info.Latitude = detector.Longitude, detector.Latitude
        }
    }
    //db.SaveDetectorReport(detector.MAC, &request.ReportList)
    db.SaveDetectorReport(detector.IMEI, &request.ReportList)
    detector.SendMsg(cmd, seq, nil)
}

func OnDetectSelfReport(cmd uint8, seq uint16, detector *Detector, request * protocol.DetectorSelfInfoReportRequest)  {
    log.Println("OnDetectSelfReport, request:", request)

    if request.Latitude == 0 || request.Longitude == 0 {
        lx, ly := db.GetGeoByBaseStation(int(request.Lac), int(request.CellId), int(request.Mcc))
        log.Println("GetGeoByBaseStation :", request.Lac, request.CellId, request.Mcc, lx, ly)
        if lx == 0 || ly == 0 {
            request.Longitude, request.Latitude = detector.Longitude, detector.Latitude
        } else {
            request.Longitude, request.Latitude = int32(lx * protocol.GeoMmultiple), int32(ly * protocol.GeoMmultiple)
            detector.Longitude, detector.Latitude = request.Longitude, request.Latitude
        }
        if (request.Longitude == 0 || request.Latitude == 0){
            db.UpdateDetectorLastActiveTime(detector.IMEI, uint32(time.Now().Unix()))
        } else {
            db.UpdateDetectorLocate(detector.IMEI, request)
        }
    }

    detector.SendMsg(cmd, seq, nil)
}

func handleMsg(detector * Detector, cmd uint8, seq uint16, msg []byte) bool {
    log.Println("recv request", detector.IMEI ,"cmd:", cmd, msg)
    switch cmd {
    case 1: {
        request := protocol.LoginRequest{};
        if !request.Decode(msg){
            return false;
        }
        OnDetectorLogin(cmd, seq, detector, &request)
        break;
    }
    case 2: {
        if detector.Status != 1 {
            log.Println("recv cmd 2 without login")
        }
        detector.SendMsg(cmd, 0, nil)
        //db.UpdateDetectorLastActiveTime(detector.MAC, uint32(time.Now().Unix()))
        db.UpdateDetectorLastActiveTime(detector.IMEI, uint32(time.Now().Unix()))
        break;
    }
    case 3: {
        if detector.Status != 1 {
            log.Println("recv cmd 3 without login")
        }
        request := protocol.ReportRequest{};
        if !request.Decode(msg){
            return false;
        }
        OnReport(cmd, seq, detector, &request)
        break;
    }
    case 4:{
        if detector.Status != 1 {
            log.Println("recv cmd 4 without login")
        }
        request := protocol.DetectorSelfInfoReportRequest{}
        if !request.Decode(msg){
            return false;
        }
        OnDetectSelfReport(cmd, seq, detector, &request)
        break;
    }
    }
    return true
}

func handleConn(conn net.Conn) {
    defer conn.Close()
    detector := Detector {}
    detector.conn = conn
    buff := make([]byte, 1024 * 32)
    var buffUsed uint16 = 0;
    header := protocol.MsgHeader{}
    for {
        conn.SetReadDeadline(time.Now().Add(120 * time.Second))
        len, err := conn.Read(buff[buffUsed:])
        if err != nil {
            log.Println("recv data err", err)
            return;
        }
        detector.LastRecvTime = uint32(time.Now().Unix())
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
                handleRet := true;
                if header.Cmd != 2 {
                    var seq uint16 = 0
                    reader := bytes.NewReader(buff[header.MsgLen - protocol.CRC16Len - protocol.SeqLen : header.MsgLen - protocol.CRC16Len])
                    binary.Read(reader, binary.BigEndian, &seq)
                    handleRet = handleMsg(&detector, header.Cmd, seq, buff[protocol.HeaderLen : header.MsgLen - protocol.CRC16Len - protocol.SeqLen])
                } else {
                    handleRet = handleMsg(&detector, header.Cmd, 0, buff[protocol.HeaderLen : header.MsgLen - protocol.CRC16Len])
                }
                if !handleRet {
                    log.Println("handleMsg failed, disconnect")
                    return;
                }
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
    fmt.Println("server is starting...")
    logFile, logErr := os.OpenFile("./detector_server.log", os.O_CREATE|os.O_RDWR|os.O_APPEND, 0666)
    if logErr != nil {
        log.Println("Fail to find", "./log/detector_server.log", "cServer start Failed")
        os.Exit(1)
    }
    //logFile.Close()
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
    fmt.Println("server start done...")
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
