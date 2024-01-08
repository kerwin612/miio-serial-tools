package mst

import (
    "os"
    "fmt"
    "time"
    "strings"
    "net/http"
    "io/ioutil"
    "encoding/json"
    "github.com/gorilla/websocket"
)

var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
}
var ws *websocket.Conn
var wssender chan string
var scoff chan bool
var off chan bool
var _status bool
var autoRespMap map[string]interface{}
var shortCmdFile string
var autoRespFile string
var serialConfigFile string

func api(homedir string) {

    _status = false
    autoRespMap = make(map[string]interface{})
    shortCmdFile = homedir + "short_cmd.json"
    autoRespFile = homedir + "auto_resp.json"
    serialConfigFile = homedir + "serial_config.json"

    autoRespArray := []struct {
        ReqCmd string
        RespCmd []string
    }{}
    file, err := os.OpenFile(autoRespFile, os.O_RDWR, 0644)
    defer file.Close()
    if err == nil {
        b, err := ioutil.ReadAll(file)
        if err == nil {
            json.Unmarshal(b, &autoRespArray)

            for _, obj := range autoRespArray {
                autoRespMap[obj.ReqCmd] = obj.RespCmd
            }
            logger.Println("auto_resp: ", autoRespMap)
        }
    }

    http.HandleFunc("/v1/status", func (w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Content-Type", "application/json")
        fmt.Fprintf(w, "{\"status\": %t}", _status)
    })

    http.HandleFunc("/v1/start", _start)

    http.HandleFunc("/v1/stop", _stop)

    http.HandleFunc("/v1/send", _send)

    http.HandleFunc("/v1/serial_config", _serial_config)

    http.HandleFunc("/v1/auto_resp", _auto_resp)

    http.HandleFunc("/v1/short_cmd", _short_cmd)

    http.HandleFunc("/ws", _ws)

}

func wssend(msg string) {
    if wssender != nil {
        go func() {
            wssender <- msg
        }()
    }
}

func _start(w http.ResponseWriter, r *http.Request) {
    decoder := json.NewDecoder(r.Body)
    var sc *SerialConfig
    err := decoder.Decode(&sc)
    if err != nil {
        logger.Println("start params error: ", err)
        w.WriteHeader(500)
        return
    }
    outer := make(chan string, 1024)
    err = Start(sc, outer)
    if err != nil {
        logger.Println("start port error: ", err)
        w.WriteHeader(500)
        return
    }
    off = make(chan bool)
    go func() {
        for {
            select {
                case <- off:
                    logger.Println("off true")
                    return
                default:
                    outstr := <-outer
                    if "recv: down none" != outstr && "send: get_down" != outstr {
                        logger.Println(outstr)
                    }
                    if outstr == "stop!" {
                        return
                    }
                    wssend(outstr)
                    if strings.HasPrefix(outstr, "recv: ") {
                        req_cmd := outstr[6:]
                        if resp_cmd, ok := autoRespMap[req_cmd]; ok {
                            for _, _cmd := range resp_cmd.([]string) {
                                wssend("auto_resp: " + _cmd)
                                Send(_cmd)
                                time.Sleep(time.Millisecond * 100)
                            }
                        }
                    }
            }
        }
    }()
    scoff = make(chan bool)
    go func() {
        for {
            select {
                case <- scoff:
                    logger.Println("scoff true")
                    return
                default:
                    _status = Status()
                    if !_status {
                        logger.Println("port status is flase")
                        wssend("_cmd:stop")
                        outer <- "stop!"
                        return
                    }
            }
        }
    }()
    w.WriteHeader(200)
}

func _stop(w http.ResponseWriter, r *http.Request) {
    w.WriteHeader(200)
    if _status {
        off <- true
        scoff <- true
        _status = false
    }
    Stop()
}

func _send(w http.ResponseWriter, r *http.Request) {
    decoder := json.NewDecoder(r.Body)
    obj := struct {
        Cmd string
    }{}
    err := decoder.Decode(&obj)
    if err != nil {
        wssend("send params error: " + err.Error())
        w.WriteHeader(500)
        return
    }
    if !_status {
        wssend("send [" + obj.Cmd + "] error: server not startup.")
        w.WriteHeader(500)
        return
    }
    err = Send(obj.Cmd)
    if err != nil {
        wssend("send [" + obj.Cmd + "] error: " + err.Error())
        w.WriteHeader(500)
        return
    }
    w.WriteHeader(200)
}

func _serial_config(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
        case http.MethodGet:
            data, err := ioutil.ReadFile(serialConfigFile)
            if err != nil {
                w.WriteHeader(200)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            fmt.Fprintf(w, string(data))
        case http.MethodPost:
            decoder := json.NewDecoder(r.Body)
            var sc *SerialConfig
            err := decoder.Decode(&sc)
            if err != nil {
                panic(err)
            }
            scBytes, err := json.MarshalIndent(&sc, "", " ")
            ioutil.WriteFile(serialConfigFile, scBytes, 0666)
            w.WriteHeader(200)
        default:
            http.NotFound(w, r)
            return
    }
}

func _auto_resp(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
        case http.MethodGet:
            data, err := ioutil.ReadFile(autoRespFile)
            if err != nil {
                w.WriteHeader(200)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            fmt.Fprintf(w, string(data))
        case http.MethodPost:
            decoder := json.NewDecoder(r.Body)
            objArray := []struct {
                ReqCmd string
                RespCmd []string
            }{}
            err := decoder.Decode(&objArray)
            if err != nil {
                panic(err)
            }
            autoRespMap = make(map[string]interface{})
            for _, obj := range objArray {
                autoRespMap[obj.ReqCmd] = obj.RespCmd
            }
            logger.Println("auto_resp: ", autoRespMap)
            objArrayBytes, err := json.MarshalIndent(&objArray, "", " ")
            ioutil.WriteFile(autoRespFile, objArrayBytes, 0666)
            w.WriteHeader(200)
        default:
            http.NotFound(w, r)
            return
    }
}

func _short_cmd(w http.ResponseWriter, r *http.Request) {
    switch r.Method {
        case http.MethodGet:
            data, err := ioutil.ReadFile(shortCmdFile)
            if err != nil {
                w.WriteHeader(200)
                return
            }
            w.Header().Set("Content-Type", "application/json")
            fmt.Fprintf(w, string(data))
        case http.MethodPost:
            decoder := json.NewDecoder(r.Body)
            cmdArray := []string{}
            err := decoder.Decode(&cmdArray)
            if err != nil {
                panic(err)
            }
            logger.Println("short_cmd: ", cmdArray)
            cmdArrayBytes, err := json.MarshalIndent(&cmdArray, "", " ")
            ioutil.WriteFile(shortCmdFile, cmdArrayBytes, 0666)
            w.WriteHeader(200)
        default:
            http.NotFound(w, r)
            return
    }
}

func _ws(w http.ResponseWriter, r *http.Request) {
    upgrader.CheckOrigin = func(r *http.Request) bool { return true }

    // upgrade this connection to a WebSocket
    // connection
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
        logger.Println(err)
    }
    if ws != nil {
        ws.Close()
        ws = nil
    }
    ws = conn

    wssender = make(chan string)
    go func() {
        for {
            select {
                default:
                    if ws == nil {
                        break
                    }
                    sendstr := <-wssender
                    if err := ws.WriteMessage(websocket.TextMessage, []byte(time.Now().Format("[15:04:05.000] ") + sendstr)); err != nil {
                        logger.Println(err)
                        wssender = nil
                        ws.Close()
                        return
                    }
            }
        }
    }()

    logger.Println("Client Connected")
    // listen indefinitely for new messages coming
    // through on our WebSocket connection
    _reader()
}

// define a reader which will listen for
// new messages being sent to our WebSocket
// endpoint
func _reader() {
    for {
        // read in a message
        _, p, err := ws.ReadMessage()
        if err != nil {
            logger.Println(err)
            ws.Close()
            return
        }
        // print out that message for clarity
        fmt.Println(string(p))

        wssend(string(p))
    }
}
