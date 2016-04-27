package detector_server

import (
    "bytes"
    "encoding/binary"
)

const headerLen int32 = 16 + 16 + 8

type MsgHeader struct {
    magic uint16
    msgLen uint16
    cmd uint8
}

type LoginRequest struct {
}

func checkCrc16(buff []byte) bool {
    var crc16 uint16
    reader := bytes.NewReader(buff[len(buff)-2:])
    binary.Read(reader, binary.BigEndian, &crc16)
    return true
}

func (msgHeader * MsgHeader)decode(buff []byte)  {
    reader := bytes.NewReader(buff)
    binary.Read(reader, binary.BigEndian, &msgHeader.magic)
    binary.Read(reader, binary.BigEndian, &msgHeader.msgLen)
    binary.Read(reader, binary.BigEndian, &msgHeader.cmd)
}