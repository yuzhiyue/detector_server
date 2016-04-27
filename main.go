package detector_server
import (
    "net"
    "fmt"
)



type Detector struct {
    id int
    mac string
    longitude float32
    atitude float32
    status int
    onn net.Conn
}

func handleMsg(detector Detector, cmd uint8, msg []byte)  {
    fmt.Println(msg)
}

func handleConn(conn net.Conn) {
    defer conn.Close()
    detector := Detector {0, "", 0, 0, 0, conn}
    buff := make([]byte, 1024 * 32)
    var buffUsed int32 = 0;
    var msgSize int32 = 0
    header := MsgHeader{}
    for {
        len, err := conn.Read(buff[buffUsed:])
        if err != nil {
        }
        buffUsed += int32(len)
        if header.msgLen == 0 {
            if buffUsed >= headerLen {
                header.decode(buff)
                if header.magic != 0xf9f9 {
                    return
                }
            }
        } else if buffUsed >= msgSize {
            if !checkCrc16(buff) {
                return
            }
            handleMsg(detector, header.cmd, buff[:buffUsed])
            copy(buff, buff[msgSize:])
            buffUsed -= msgSize
            header.magic = 0
            header.msgLen = 0
            header.cmd = 0
        }
    }
}

func main()  {
    listen, err := net.Listen("tcp", ":10000")
    if err != nil {
        return
    }
    defer listen.Close();
    for {
        conn, err := listen.Accept();
        if err != nil {

        }
        go handleConn(conn)
    }
    return
}
