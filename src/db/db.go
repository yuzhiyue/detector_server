package db

import (
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "protocol"
    "container/list"
    "fmt"
    "time"
)

var session *mgo.Session;

func InitDB()  {
    var err error
    session, err = mgo.Dial("112.74.90.113:22522")
    if err != nil {
        panic(err)
    }
    session.SetMode(mgo.Monotonic, true)
}

func CreateDetector(mac string, imei string) {
    c := session.DB("detector").C("detector_info")
    c.UpsertId(mac, bson.M{"_id":mac, "imei":imei, "last_update_time":uint32(time.Now().Unix())})
}

func UpdateDetector(mac string, longitude int, atitude int)  {
    c := session.DB("detector").C("detector_info")
    c.Update(bson.M{"_id":mac}, bson.M{"longitude":longitude, "atitude":atitude,"last_update_time":uint32(time.Now().Unix())})
}

func SaveDetectorReport(apMac string, reportInfos list.List)  {
    c := session.DB("detector").C("detector_report")
    bulk := c.Bulk()
    for e := reportInfos.Front(); e != nil; e = e.Next(){
        info := e.Value.(*protocol.ReportInfo)
        fmt.Println(*info)
        bulk.Insert(bson.M{"ap_mac":apMac, "device_mac":info.MAC, "longitude":info.Longitude, "atitude":info.Atitude, "time":info.Time})
    }
    bulk.Run()
}