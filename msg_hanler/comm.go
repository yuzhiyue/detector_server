package msg_hanler

import (
    "net"
    "bytes"
    "encoding/binary"
    "detector_server/protocol"
    "log"
    "encoding/hex"
    "time"
    "detector_server/db"
    "gopkg.in/mgo.v2/bson"
)

type ScanConf struct {
    Channel uint32
    Interval uint32
}

type ReportInfo struct {
    Longitude int32
    Latitude  int32
    Time uint32
}

type Detector struct {
    Id        int
    ProtoVer  uint8
    MAC       string
    IMEI      string
    Longitude int32
    Latitude  int32
    GeoUpdateType int
    Status    int
    LastRecvTime uint32
    Conn      net.Conn
    ReportData map[string]*protocol.ReportInfo
    LastSaveReportTime uint32
    ScanConf []ScanConf
    ScanConfUpdateTime uint32
    ScanConfSendTime uint32
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
    detector.Conn.Write(buff.Bytes());
    log.Println("response:", cmd, "\n", hex.Dump(buff.Bytes()))
}

func (detector * Detector)SendScanConf() {
    scanConf := protocol.ScanConf{}
    scanConf.ConfVer = 1
    if len(detector.ScanConf) == 0 {
        for i := 0; i < len(scanConf.Channel); i++ {
            channel := &scanConf.Channel[i]
            channel.Channel = uint8(i + 1)
            channel.Seq = uint8(i + 1)
            channel.Open = 0xFF
            channel.Interval = 2
        }
    } else {
        for i := 0; i < len(scanConf.Channel); i++ {
            channel := &scanConf.Channel[i]
            channel.Channel = uint8(i + 1)
            channel.Seq = uint8(i + 1)
            channel.Open = 0x0
            channel.Interval = 0
        }
        for _, e := range detector.ScanConf {
            if e.Channel > uint32(len(scanConf.Channel)) {
                continue
            }
            channel := &scanConf.Channel[e.Channel - 1]
            channel.Channel = uint8(e.Channel)
            channel.Open = 0xFF
            channel.Interval = uint16(e.Interval)
        }
    }
    log.Println("send scan conf", detector.MAC, scanConf)
    buff := scanConf.Encode()
    detector.ScanConfSendTime = uint32(time.Now().Unix())
    detector.SendMsg(6, 0, buff)
}

func (detector * Detector)SaveReport() {
    now := uint32(time.Now().Unix())
    if now - detector.LastSaveReportTime < 30 {
        return
    }
    detector.LastSaveReportTime = now
    if detector.ProtoVer == 1 {
        db.SaveDetectorReport(detector.IMEI, &detector.ReportData)
    } else {
        db.SaveDetectorReport(detector.MAC, &detector.ReportData)
    }
    detector.ReportData = make(map[string]*protocol.ReportInfo)
}

func (detector *Detector) ReloadDetectorInfo() {
    result := bson.M{}
    var err error;
    if detector.ProtoVer == 1 {
        err = db.GetDetectorInfo(detector.IMEI, &result)
    } else {
        err = db.GetDetectorInfo(detector.MAC, &result)
    }
    if err == nil {
        detector.Longitude = int32(db.GetNumber(result, "longitude") * protocol.GeoMmultiple)
        detector.Latitude = int32(db.GetNumber(result, "latitude") * protocol.GeoMmultiple)
        detector.GeoUpdateType = int(db.GetNumber(result, "geo_update_type"))
    }
    detector.ScanConfUpdateTime = uint32(db.GetNumber(result, "scan_conf_update_time"))
    scanConf, ok := result["scan_conf"]
    if ok {
        for _, e := range scanConf.([]interface {}) {
            conf := ScanConf{}
            conf.Channel = uint32(db.GetNumber(e.(bson.M), "channel"))
            conf.Interval = uint32(db.GetNumber(e.(bson.M), "interval"))
            detector.ScanConf = append(detector.ScanConf, conf)
        }
    }
    if detector.ProtoVer != 1 {
        if detector.ScanConfSendTime < detector.ScanConfUpdateTime {
            detector.SendScanConf()
        }
    }
}