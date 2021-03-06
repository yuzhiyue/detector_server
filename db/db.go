package db

import (
    "gopkg.in/mgo.v2"
    "gopkg.in/mgo.v2/bson"
    "time"
    "gopkg.in/olivere/elastic.v3"
    "detector_server/protocol"
    "github.com/golang/glog"
)

type LastActiveTimeRequest struct {
    MAC string;
    time uint32;
}
var lastActiveTimeRequestChannel chan *LastActiveTimeRequest
var reportChannel chan *protocol.ReportInfo
var es_client *elastic.Client


func InitSQLDB() error {
    //const addr = "postgresql://218.15.154.6:26257/detector?sslmode=disable"
    //db, err := gorm.Open("postgres", addr)
    //if err != nil {
    //    log.Fatal(err)
    //}
    return nil
}

func InitES() {
    var err error
    es_client, err = elastic.NewClient(elastic.SetURL("http://120.24.7.62:29200"))
    if err != nil {
        // Handle error
        panic(err)
    }
}

func InitESIndex()  {
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
    g_session, err = mgo.Dial("218.15.154.6:22522")
    if err != nil {
        panic(err)
    }
    g_session.SetMode(mgo.Monotonic, true)
    dbName = db
    reportChannel = make(chan *protocol.ReportInfo, 10000)
    lastActiveTimeRequestChannel = make(chan *LastActiveTimeRequest, 10000)
    go dbWiter();
    glog.Info("connect to db succ")
}

func GetDetectorInfo(mac string, result interface{}) error {
    session := GetSession()
    defer session.Close()
    c := session.DB(dbName).C("detector_info")
    return c.FindId(mac).One(result)
}

func CreateDetector(mac string, imei string) int {
    session := GetSession()
    defer session.Close()
    c := session.DB(dbName).C("detector_info")
    c.Insert(bson.M{"_id":mac, "imei":imei, "company":"01", "last_active_time":uint32(time.Now().Unix()), "last_login_time":uint32(time.Now().Unix())})
    return CreateDetectorNo(mac)
}

func CreateDetectorNo(mac string) int {
    session := GetSession()
    defer session.Close()
     change := mgo.Change{
             Update: bson.M{"$inc": bson.M{"value": 1}},
             ReturnNew: true,
             Upsert: true,
     }
    doc := bson.M{}
    _, err := session.DB(dbName).C("ids").Find(bson.M{"_id": "detector_no"}).Apply(change, &doc)
    if err == nil {
        no := int(GetNumber(doc, "value"))
        c := session.DB(dbName).C("detector_info")
        err = c.UpdateId(mac, bson.M{"$set": bson.M{"no":no}})
        if err == nil {
            return no
        }
    }
    return 0
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
    for _, info := range *reportInfos{
        info.ApMAC = apMac
        if(info.Longitude == 0 || info.Latitude == 0) {
            continue
        }
        reportChannel <- info
    }
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


type ReportInfoRecord struct {
    ID        uint64 `gorm:"primary_key;AUTO_INCREMENT"`
    MAC       string `gorm:"primary_key"`
    RSSI      uint8
    Longitude float32
    Latitude  float32
    ReportLongitude float32
    ReportLatitude  float32
    Mcc       uint16
    Mnc       uint8
    Lac       uint16
    CellId    uint16
    Channel   uint8
    Time      uint32
    ApMAC     string
}

func dbWiter()  {
    infoList := make([]*protocol.ReportInfo, 0)
    go func() {
        time.Sleep(30)
        reportChannel <- nil
    }()

    for ;;  {
        e := <- reportChannel
        if e != nil {
            infoList = append(infoList, e)
        }

        if e == nil || len(infoList) > 10 {
            if (len(infoList) != 0) {
                session := GetSession()
                c := session.DB(dbName).C("detector_report")
                bulk := c.Bulk()
                //es_bulk := es_client.Bulk()
                for _, info := range infoList {
                    glog.Info(*info)
                    //continue
                    doc := bson.M{"ap_mac":info.ApMAC, "device_mac":info.MAC, "rssi":info.RSSI, "longitude":float64(info.Longitude) / protocol.GeoMmultiple, "latitude":float64(info.Latitude) / protocol.GeoMmultiple, "report_longitude":float64(info.ReportLongitude) / protocol.GeoMmultiple, "report_latitude":float64(info.ReportLatitude) / protocol.GeoMmultiple, "mcc":info.Mcc, "mnc":info.Mnc,
                        "lac":info.Lac, "cell_id":info.CellId, "time":info.Time, "channel":info.Channel}
                    bulk.Insert(doc)
                    //indexRequest := elastic.NewBulkIndexRequest()
                    //indexRequest.Index(dbName).Type("trace").Doc(doc)
                    //es_bulk.Add(indexRequest)
                }
                bulk.Run()
                //es_bulk.Do()
                session.Close()
                infoList = make([]*protocol.ReportInfo, 0)
            }
        }

    }

}