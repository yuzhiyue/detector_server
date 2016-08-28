package protocol

import (
    "bytes"
    "encoding/binary"
)

type UpgradeFirmware struct {
    FirmwareUrl [128]byte
}

func (msg * UpgradeFirmware)Encode() []byte {
    buf := new(bytes.Buffer)
    binary.Write(buf, binary.BigEndian, msg.FirmwareUrl)
    return buf.Bytes()
}