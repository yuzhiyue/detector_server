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

func OnDetectorLoginV2(cmd uint8, seq uint16, detector * Detector, request * protocol.LoginRequest) {
    log.Println("onDetectorLogin, request:", request)
    result := bson.M{}
    err := db.GetDetectorInfo(request.MAC, &result)
    if err != nil {
        db.CreateDetector(request.MAC, request.IMEI)
    } else {
        db.UpdateLoginTime(request.MAC)
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
