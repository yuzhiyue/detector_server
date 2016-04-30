package db

import (
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "protocol"
    "container/list"
    "fmt"
    "time"
    "log"
)

var session *mgo.Session;

func InitDB()  {
    var err error
    session, err = mgo.Dial("127.0.0.1:22522")
    if err != nil {
        panic(err)
    }
    session.SetMode(mgo.Monotonic, true)
    log.Println("connect to db succ")
}

func CreateDetector(mac string, imei string) {
    c := session.DB("detector").C("detector_info")
    c.UpsertId(mac, bson.M{"_id":mac, "imei":imei, "last_update_time":uint32(time.Now().Unix())})
}

func UpdateDetectorLastTime(mac string, time uint32)  {
    c := session.DB("detector").C("detector_info")
    c.Update(bson.M{"_id":mac}, bson.M{"last_update_time":time})
}

func UpdateDetectorLocate(mac string, info * protocol.DetectorSelfInfoReportRequest)  {
    c := session.DB("detector").C("detector_info")
    c.Update(bson.M{"_id":mac}, bson.M{"longitude":info.Longitude, "atitude":info.Atitude, "mcc":info.Mcc, "mnc":info.Mnc,
        "lac":info.Lac, "cell_id":info.CellId, "last_update_time":uint32(time.Now().Unix())})
}

func SaveDetectorReport(apMac string, reportInfos * list.List)  {
    c := session.DB("detector").C("detector_report")
    bulk := c.Bulk()
    for e := reportInfos.Front(); e != nil; e = e.Next(){
        info := e.Value.(*protocol.ReportInfo)
        fmt.Println(*info)
        bulk.Insert(bson.M{"ap_mac":apMac, "device_mac":info.MAC, "longitude":info.Longitude, "atitude":info.Atitude, "mcc":info.Mcc, "mnc":info.Mnc,
            "lac":info.Lac, "cell_id":info.CellId, "time":info.Time})
    }
    bulk.Run()
}