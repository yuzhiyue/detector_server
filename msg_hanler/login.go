package msg_hanler

import (
    "log"
    "gopkg.in/mgo.v2/bson"
    "time"
    "detector_server/db"
    "detector_server/protocol"
)

func OnDetectorLogin(cmd uint8, seq uint16, detector * Detector, request * protocol.LoginRequest) {
    log.Println("onDetectorLogin, request:", request)
    result := bson.M{}
    err := db.GetDetectorInfo(request.IMEI, &result)
    if err != nil {
        detector.No = db.CreateDetector(request.IMEI, request.MAC)
        //db.CreateDetector(request.MAC, request.IMEI)
    } else {
        db.UpdateLoginTime(request.IMEI)
        detector.No = int(db.GetNumber(result, "no"))
        detector.Longitude = int32(db.GetNumber(result, "longitude") * protocol.GeoMmultiple)
        detector.Latitude = int32(db.GetNumber(result, "latitude") * protocol.GeoMmultiple)
        detector.GeoUpdateType = int(db.GetNumber(result, "geo_update_type"))
    }
    if detector.No == 0 {
        detector.No = db.CreateDetectorNo(request.IMEI)
    }
    detector.MAC = request.MAC
    detector.IMEI = request.IMEI
    detector.Status = 1
    detector.ProtoVer = request.ProtoVer
    detector.ReportData = make(map[string]*protocol.ReportInfo)
    detector.LastRecvReportTime = uint32(time.Now().Unix())
    response := protocol.LoginResponse{}
    response.ProtoVer = protocol.MaxProtoVer
    response.Time = uint32(time.Now().Unix())
    buff := response.Encode()
    log.Println("response:", buff)
    detector.SendMsg(cmd, seq, buff)
}

func OnDetectorLoginV2(cmd uint8, seq uint16, detector * Detector, request * protocol.LoginRequest) {
    log.Println("onDetectorLogin, request:", request)
    result := bson.M{}
    err := db.GetDetectorInfo(request.MAC, &result)
    if err != nil {
        detector.No = db.CreateDetector(request.MAC, request.IMEI)
    } else {
        db.UpdateLoginTime(request.MAC)
        detector.No = int(db.GetNumber(result, "no"))
        detector.Longitude = int32(db.GetNumber(result, "longitude") * protocol.GeoMmultiple)
        detector.Latitude = int32(db.GetNumber(result, "latitude") * protocol.GeoMmultiple)
        detector.GeoUpdateType = int(db.GetNumber(result, "geo_update_type"))
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
    }
    if detector.No == 0 {
        detector.No = db.CreateDetectorNo(request.MAC)
    }
    detector.MAC = request.MAC
    detector.IMEI = request.IMEI
    detector.Status = 1
    detector.ProtoVer = request.ProtoVer
    detector.ReportData = make(map[string]*protocol.ReportInfo)
    detector.LastRecvReportTime = uint32(time.Now().Unix())
    response := protocol.LoginResponse{}
    response.ProtoVer = protocol.MaxProtoVer
    response.Time = uint32(time.Now().Unix())
    buff := response.Encode()
    log.Println("response:", buff)
    detector.SendMsg(cmd, seq, buff)
    detector.SendScanConf()
}

func HandleLoginMsgV1(detector * Detector, cmd uint8, seq uint16, msg []byte) bool {
    request := protocol.LoginRequest{}
    if !request.Decode(msg){
        return false
    }
    OnDetectorLogin(cmd, seq, detector, &request)
    return true
}

func HandleLoginMsgV2(detector * Detector, cmd uint8, seq uint16, msg []byte) bool {
    request := protocol.LoginRequest{};
    if !request.DecodeV2(msg){
        return false;
    }
    OnDetectorLoginV2(cmd, seq, detector, &request)
    return true
}
