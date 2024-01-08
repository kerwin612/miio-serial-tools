package mst

import (
    "time"
    lib "github.com/tarm/serial"
)

var port *lib.Port


type SerialImpl struct {
}

func (s *SerialImpl)Init(c *SerialConfig) error {
    config := &lib.Config{
        Name: c.Name,
        Baud: c.Baud,
        ReadTimeout: time.Second * 1}
    _port, err := lib.OpenPort(config)
    if err != nil {
        return err
    }
    port = _port
    return nil
}

func (s *SerialImpl)Write(data []byte) error {
    _, err := port.Write(data)
    if err != nil {
        return err
    }
    return nil
}

func (s *SerialImpl)Read() ([]byte, error) {
    buf := make([]byte, 128)
    n, err := port.Read(buf)
    if err != nil {
        return nil, err
    }
    if n != 0 {
        return buf[:n], nil
    }
    return nil, nil
}

func (s *SerialImpl)Close() error {
    port.Close()
    return nil
}
