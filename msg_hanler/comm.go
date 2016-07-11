package msg_hanler

import (
    "net"
    "bytes"
    "encoding/binary"
    "detector_server/protocol"
    "log"
    "encoding/hex"
)

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
    for i := 0; i < len(scanConf.Channel); i++ {
        channel := &scanConf.Channel[i]
        channel.Channel = uint8(i + 1)
        channel.Seq = uint8(i + 1)
        channel.Open = 0xFF
        channel.Interval = 30
    }

    buff := scanConf.Encode()
    detector.SendMsg(6, 0, buff)
}