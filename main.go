package main
import (
    "net"
    "time"
    "bytes"
    "encoding/binary"
    "os"
    "fmt"
    "detector_server/db"
    "detector_server/protocol"
    "detector_server/msg_hanler"
    "gopkg.in/mgo.v2/bson"
    "github.com/golang/glog"
    "flag"
)




func OnDetectSelfReport(cmd uint8, seq uint16, detector *msg_hanler.Detector, request * protocol.DetectorSelfInfoReportRequest)  {
    glog.Info("OnDetectSelfReport, request:", request)
    result := bson.M{}
    err := db.GetDetectorInfo(detector.IMEI, &result)
    if err == nil {
        detector.Longitude = int32(db.GetNumber(result, "longitude") * protocol.GeoMmultiple)
        detector.Latitude = int32(db.GetNumber(result, "latitude") * protocol.GeoMmultiple)
        detector.GeoUpdateType = int(db.GetNumber(result, "geo_update_type"))
    }
    //if request.Latitude == 0 || request.Longitude == 0 {
    //    lx, ly := db.GetGeoByBaseStation(int(request.Lac), int(request.CellId), int(request.Mcc))
    //    log.Println("GetGeoByBaseStation :", request.Lac, request.CellId, request.Mcc, lx, ly)
    //    if lx == 0 || ly == 0 {
    //        request.Longitude, request.Latitude = detector.Longitude, detector.Latitude
    //    } else {
    //        request.Longitude, request.Latitude = int32(lx * protocol.GeoMmultiple), int32(ly * protocol.GeoMmultiple)
    //        detector.Longitude, detector.Latitude = request.Longitude, request.Latitude
    //    }
    //
    //    if (request.Longitude == 0 || request.Latitude == 0){
    //        db.UpdateDetectorLastActiveTime(detector.IMEI, uint32(time.Now().Unix()))
    //    } else {
    //        db.UpdateDetectorLocate(detector.IMEI, request)
    //    }
    //}
    db.UpdateDetectorLocate(detector.IMEI, request)
    db.UpdateDetectorLastActiveTime(detector.IMEI, uint32(time.Now().Unix()))
    detector.SendMsg(cmd, seq, nil)
}

func handleMsg(detector * msg_hanler.Detector, cmd uint8, seq uint16, msg []byte) bool {
    glog.Info("recv request", detector.MAC, detector.IMEI ,"cmd:", cmd)
    switch cmd {
    case 0x01: {
        return msg_hanler.HandleLoginMsgV1(detector, cmd, seq, msg)
        break
    }
    case 0x11: {
        return msg_hanler.HandleLoginMsgV2(detector, cmd, seq, msg)
        break
    }
    case 0x02: {
        if detector.Status != 1 {
            glog.Error("recv cmd 2 without login")
            return false
        }
        detector.SendMsg(cmd, 0, nil)
        if detector.ProtoVer == 1 {
            db.UpdateDetectorLastActiveTime(detector.IMEI, uint32(time.Now().Unix()))
        } else {
            db.UpdateDetectorLastActiveTime(detector.MAC, uint32(time.Now().Unix()))
        }
        detector.ReloadDetectorInfo()
        detector.SaveReport()
        if detector.ProtoVer >= 3 {
            newFirmware := detector.CheckNewFirmware()
            if newFirmware != "" {
                detector.Reboot()
            }
        }
        if uint32(time.Now().Unix()) - detector.LastRecvReportTime > 1800 {
            detector.Reboot()
        }
        break
    }
    case 0x03: {
        if detector.Status != 1 {
            glog.Error("recv cmd 2 without login")
            return false
        }
        msg_hanler.HandleReportTraceMsg(detector, cmd, seq, msg)
        break
    }
    case 0x04:{
        if detector.Status != 1 {
            glog.Error("recv cmd 2 without login")
            return false
        }
        request := protocol.DetectorSelfInfoReportRequest{}
        if !request.Decode(msg){
            return false
        }
        OnDetectSelfReport(cmd, seq, detector, &request)
        detector.SaveReport()
        break
    }
    }
    return true
}


func handleConn(conn net.Conn) {
    defer conn.Close()
    detector := msg_hanler.Detector {}
    detector.Conn = conn
    detector.NeedClose = false
    buff := make([]byte, 1024 * 32)
    var buffUsed uint16 = 0;
    header := protocol.MsgHeader{}
    for {
        conn.SetReadDeadline(time.Now().Add(120 * time.Second))
        len, err := conn.Read(buff[buffUsed:])
        if err != nil {
            glog.Info("recv data err", err, detector.MAC)
            return
        }
        if detector.NeedClose {
            glog.Info("detector need close", detector.MAC)
            return
        }

        detector.LastRecvTime = uint32(time.Now().Unix())
        glog.Info("recv data, len:", len)
        //log.Println("dump data:\n", hex.Dump(buff[buffUsed: buffUsed+uint16(len)]))
        buffUsed += uint16(len)
        for {
            if header.MsgLen == 0 {
                if buffUsed >= protocol.HeaderLen {
                    header.Decode(buff)
                    if header.Magic != 0xf9f9 {
                        glog.Error("decode header, magic err", header.Magic)
                        return
                    }
                    if header.MsgLen > uint16(cap(buff)) {
                        glog.Error("msg too big, size", header.MsgLen)
                        return
                    }
                    glog.Error("decode header, msg len", header.MsgLen)
                }
            }
            if header.MsgLen != 0 && buffUsed >= header.MsgLen {
                if !protocol.CheckCRC16(buff[:header.MsgLen]) {
                    glog.Error("check crc failed")
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
                    glog.Error("handleMsg failed, disconnect", detector.MAC)
                    return
                }
                copy(buff, buff[header.MsgLen:buffUsed])
                buffUsed -= header.MsgLen
                header.Magic = 0
                header.MsgLen = 0
                header.Cmd = 0
            } else {
                break
            }
        }

    }
}

func main()  {
    flag.Parse()
    dbName := "detector"
    listen_address := ":10001"
    if len(os.Args) == 2 && os.Args[1] == "test_svr" {
        dbName = "detector"
        listen_address = ":11001"
    }
    fmt.Println("server is starting...")


    db.InitDB(dbName)
    //db.InitES()
    //db.InitESIndex()
    listen, err := net.Listen("tcp4", listen_address)
    if err != nil {
        return
    }
    glog.Info("server start, listen on", listen_address)
    defer listen.Close();
    fmt.Println("server start done...")
    for {
        conn, err := listen.Accept();
        if err != nil {
            return
        }
        glog.Info("accept new connection")
        go handleConn(conn)
    }
    return
}
