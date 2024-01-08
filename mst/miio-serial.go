package mst

import (
    "time"
    "errors"
    "strings"
)

type SerialConfig struct {
    Name string
    Baud int
    Bits int
    Pary int
    Stop int
}

type ISerial interface {
    Init(c *SerialConfig) error
    Write(data []byte) error
    Read() ([]byte, error)
    Close() error
}

var NOSESSIONCMD = []string{
    "none",
    "MIIO_net_change",
//    "miIO.stop_diag_mode",
}

var serial ISerial
var recvBuf []byte
var outer chan string
var getoff, readoff chan bool
var readRetry = 0
var status = false
var session = false

func Status() bool {
    return status
}

func Start(sc *SerialConfig, o chan string) error {
    serial = new(SerialImpl)
    err := serial.Init(sc)
    if err != nil {
        return err
    }
    outer = o

    readoff = make(chan bool)
    go func() {
        for {
            select {
                case <- readoff:
                    logger.Println("off recv")
                    return
                default:
                    recv()
            }
        }
    }()

    getoff = make(chan bool)
    go func() {
        for {
            select {
                case <- getoff:
                    logger.Println("off get_down")
                    return
                default:
                    if session {
                        continue
                    }
                    Send("get_down")
                    time.Sleep(time.Millisecond * 200)
            }
        }
    }()

    logger.Println("started")
    status = true
    return nil
}

func Stop() {
    logger.Println("stop.")
    if status {
        go func() {
            readoff <- true
            getoff <- true
            status = false
        }()
    }
    logger.Println("stop..")
    serial.Close()
    logger.Println("stoped")
}

func Send(cmd string) error {
    if session && !strings.HasPrefix(cmd, "result ") && !strings.HasPrefix(cmd, "error ") {
        return errors.New("Last session has not ended")
    }
    outer <- "send: " + cmd
    err := serial.Write([]byte(cmd + "\r"))
    if err != nil {
        return err
    }
    session = false
    return nil
}

func recv() {
    chs, err := serial.Read()
    if err != nil {
        logger.Println("port read error: ", err)
        readRetry = readRetry + 1
        if readRetry > 10 {
            logger.Println("Retry limit reached")
            Stop()
        }
        return
    } else if chs == nil || len(chs) == 0 {
        //logger.Println("recv empty or timeout")
        return
    }else {
        readRetry = 0
    }
    for _, ch := range chs {
        if session {
            continue
        }
        str := string(ch)
        if str == "\r" {
            cmd := string(recvBuf)
            recvBuf = recvBuf[:0]
            outer <- "recv: " + cmd
            if isNewSession(cmd) {
                session = true
                logger.Println("session start")
                go func() {
                    timeout := 0
                    for {
                        if !session {
                            logger.Println("session stop")
                            return
                        }
                        if timeout < 1000 {
                            timeout += 1
                        } else {
                            logger.Println("session timeout")
                            Send("error \"timeout\" -5000")
                            return
                        }
                        time.Sleep(time.Millisecond * 1)
                    }
                }()
            }
        } else {
            recvBuf = append(recvBuf, ch)
        }
    }
}

func isNewSession(cmd string) bool {
    array := strings.Fields(cmd)
    return len(array) > 1 && array[0] == "down" && !contains(NOSESSIONCMD, array[1])
}

func contains(arr []string, str string) bool {
   for _, a := range arr {
      if a == str {
         return true
      }
   }
   return false
}
