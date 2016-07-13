package msg_hanler

import (
    "log"
    "time"
    "detector_server/protocol"
)

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
        //if (info.Latitude == 0 || info.Longitude == 0) {
        //    info.Longitude, info.Latitude = detector.Longitude, detector.Latitude
        //}
        if (detector.Latitude != 0 && detector.Longitude != 0) {
            info.Longitude, info.Latitude = detector.Longitude, detector.Latitude
        }
        detector.ReportData[info.MAC] = info
    }
    detector.SendMsg(cmd, seq, nil)
}

func HandleReportTraceMsgV1(detector * Detector, cmd uint8, seq uint16, msg []byte) bool {
    request := protocol.ReportRequest{};
    if !request.Decode(msg){
        return false;
    }
    OnReport(cmd, seq, detector, &request)
    return true
}

func HandleReportTraceMsgV2(detector * Detector, cmd uint8, seq uint16, msg []byte) bool {
    request := protocol.ReportRequest{};
    if !request.DecodeV2(msg){
        return false;
    }
    OnReport(cmd, seq, detector, &request)
    return true
}

func HandleReportTraceMsg(detector * Detector, cmd uint8, seq uint16, msg []byte) bool {
    if detector.ProtoVer == 1 {
        HandleReportTraceMsgV1(detector, cmd, seq, msg)
    } else {
        HandleReportTraceMsgV2(detector, cmd, seq, msg)
    }
    return true
}
