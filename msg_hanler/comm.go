package msg_hanler

import "net"

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
    conn      net.Conn
}
