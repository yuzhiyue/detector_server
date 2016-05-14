package db

import (
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "protocol"
    "container/list"
    "time"
    "log"
)

var session *mgo.Session;

func GetNumber(m bson.M, key string) float64 {
    v := m[key]
    if v == nil {
        return 0
    }
    switch v.(type) {
    case float64:
        return v.(float64)
    case float32:
        return float64(v.(float32))
    case int:
        return float64(v.(int))
    }
    return 0
}

func InitDB()  {
    var err error
    session, err = mgo.Dial("127.0.0.1:22522")
    if err != nil {
        panic(err)
    }
    session.SetMode(mgo.Monotonic, true)
    log.Println("connect to db succ")
}

func GetDetectorInfo(mac string, result interface{}) error {
    c := session.DB("detector").C("detector_info")
    return c.FindId(mac).One(result)
}

func CreateDetector(mac string, imei string) {
    c := session.DB("detector").C("detector_info")
    c.Insert(mac, bson.M{"_id":mac, "imei":imei, "company":"01", "last_active_time":uint32(time.Now().Unix())})
}

func UpdateLoginTime(mac string)  {
    c := session.DB("detector").C("detector_info")
    c.UpsertId(mac, bson.M{"$set": bson.M{"last_login_time":uint32(time.Now().Unix())}})
}

func UpdateDetectorLastActiveTime(mac string, time uint32)  {
    c := session.DB("detector").C("detector_info")
    c.Update(bson.M{"_id":mac}, bson.M{"$set": bson.M{"last_active_time":time}})
}

func UpdateDetectorLocate(mac string, info * protocol.DetectorSelfInfoReportRequest)  {
    c := session.DB("detector").C("detector_info")
    c.Update(bson.M{"_id":mac},  bson.M{"$set": bson.M{"longitude":float64(info.Longitude) / protocol.GeoMmultiple, "latitude":float64(info.Latitude) / protocol.GeoMmultiple, "mcc":info.Mcc, "mnc":info.Mnc,
        "lac":info.Lac, "cell_id":info.CellId, "last_active_time":uint32(time.Now().Unix())}})
}

func SaveDetectorReport(apMac string, reportInfos * list.List)  {
    c := session.DB("detector").C("detector_report")
    bulk := c.Bulk()
    for e := reportInfos.Front(); e != nil; e = e.Next(){
        info := e.Value.(*protocol.ReportInfo)
        log.Println(*info)
        if(info.Longitude == 0 || info.Latitude == 0) {
            continue
        }
        bulk.Insert(bson.M{"ap_mac":apMac, "device_mac":info.MAC, "rssi":info.RSSI, "longitude":float64(info.Longitude) / protocol.GeoMmultiple, "latitude":float64(info.Latitude) / protocol.GeoMmultiple, "mcc":info.Mcc, "mnc":info.Mnc,
            "lac":info.Lac, "cell_id":info.CellId, "time":info.Time})
    }
    bulk.Run()
}

func GetGeoByBaseStation(lac int, cell int, mcc int) (float64, float64)  {
    if lac != 0 && cell != 0 {
        result := bson.M{}
        c := session.DB("detector").C("base_station_info")
        err := c.Find(bson.M{"lac":lac, "cell_id":cell, "mcc":mcc}).One(&result)
        if err == nil {
            return GetNumber(result, "longitude"), GetNumber(result, "latitude")
        }
    }
    return 0, 0
}