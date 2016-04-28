package protocol

import (
    "bytes"
    "encoding/binary"
)

const HeaderLen int32 = 16 + 16 + 8

type MsgHeader struct {
    Magic uint16
    MsgLen uint16
    Cmd uint8
}

type LoginRequest struct {
}

func CheckCrc16(buff []byte) bool {
    var crc16 uint16
    reader := bytes.NewReader(buff[len(buff)-2:])
    binary.Read(reader, binary.BigEndian, &crc16)
    return true
}

func (msgHeader * MsgHeader)Decode(buff []byte)  {
    reader := bytes.NewReader(buff)
    binary.Read(reader, binary.BigEndian, &msgHeader.Magic)
    binary.Read(reader, binary.BigEndian, &msgHeader.MsgLen)
    binary.Read(reader, binary.BigEndian, &msgHeader.Cmd)
}