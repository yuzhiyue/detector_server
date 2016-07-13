package db

import (
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "time"
    "log"
    "gopkg.in/olivere/elastic.v3"
    "detector_server/protocol"
)
var es_client *elastic.Client
func InitES() {
    var err error
    es_client, err = elastic.NewClient(elastic.SetURL("http://120.24.7.62:9200"))
    if err != nil {
        // Handle error
        panic(err)
    }
}

func InitIndex()  {
    exists, err := es_client.IndexExists(dbName).Do()
    if err != nil {
        // Handle error
        panic(err)
    }
    if !exists {
        createIndex, err := es_client.CreateIndex(dbName).Do()
        if err != nil {
            // Handle error
            panic(err)
        }
        if !createIndex.Acknowledged {
            // Not acknowledged
        }
    }
}

var g_session *mgo.Session;
var dbName string;
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

func GetSession() *mgo.Session {
    return g_session.Clone()
}

func InitDB(db string)  {
    var err error
    g_session, err = mgo.Dial("127.0.0.1:22522")
    if err != nil {
        panic(err)
    }
    g_session.SetMode(mgo.Monotonic, true)
    dbName = db
    log.Println("connect to db succ")
}

func GetDetectorInfo(mac string, result interface{}) error {
    session := GetSession()
    defer session.Close()
    c := session.DB(dbName).C("detector_info")
    return c.FindId(mac).One(result)
}

func CreateDetector(mac string, imei string) {
    session := GetSession()
    defer session.Close()
    c := session.DB(dbName).C("detector_info")
    c.Insert(bson.M{"_id":mac, "imei":imei, "company":"01", "last_active_time":uint32(time.Now().Unix()), "last_login_time":uint32(time.Now().Unix())})
}

func UpdateLoginTime(mac string)  {
    session := GetSession()
    defer session.Close()
    c := session.DB(dbName).C("detector_info")
    c.UpsertId(mac, bson.M{"$set": bson.M{"last_login_time":uint32(time.Now().Unix())}})
}

func UpdateDetectorLastActiveTime(mac string, time uint32)  {
    session := GetSession()
    defer session.Close()
    c := session.DB(dbName).C("detector_info")
    c.Update(bson.M{"_id":mac}, bson.M{"$set": bson.M{"last_active_time":time}})
}

func UpdateDetectorLocate(mac string, info * protocol.DetectorSelfInfoReportRequest)  {
    session := GetSession()
    defer session.Close()
    c := session.DB(dbName).C("detector_info")
    c.Update(bson.M{"_id":mac},  bson.M{"$set": bson.M{"report_longitude":float64(info.Longitude) / protocol.GeoMmultiple, "report_latitude":float64(info.Latitude) / protocol.GeoMmultiple, "mcc":info.Mcc, "mnc":info.Mnc,
        "lac":info.Lac, "cell_id":info.CellId, "last_active_time":uint32(time.Now().Unix())}})
}

func SaveDetectorReport(apMac string, reportInfos * map[string]*protocol.ReportInfo)  {
    session := GetSession()
    defer session.Close()
    c := session.DB(dbName).C("detector_report")
    bulk := c.Bulk()
    for _, info := range *reportInfos{
        log.Println(*info)
        if(info.Longitude == 0 || info.Latitude == 0) {
            continue
        }
        bulk.Insert(bson.M{"ap_mac":apMac, "device_mac":info.MAC, "rssi":info.RSSI, "longitude":float64(info.Longitude) / protocol.GeoMmultiple, "latitude":float64(info.Latitude) / protocol.GeoMmultiple, "report_longitude":float64(info.ReportLongitude) / protocol.GeoMmultiple, "report_latitude":float64(info.ReportLatitude) / protocol.GeoMmultiple, "mcc":info.Mcc, "mnc":info.Mnc,
            "lac":info.Lac, "cell_id":info.CellId, "time":info.Time})
    }
    bulk.Run()
}

func GetGeoByBaseStation(lac int, cell int, mcc int) (float64, float64)  {
    session := GetSession()
    defer session.Close()
    if lac != 0 && cell != 0 {
        result := bson.M{}
        c := session.DB(dbName).C("base_station_info")
        err := c.Find(bson.M{"lac":lac, "cell_id":cell, "mcc":mcc}).One(&result)
        if err == nil {
            return GetNumber(result, "longitude"), GetNumber(result, "latitude")
        }
    }
    return 0, 0
}