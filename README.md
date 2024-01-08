# MIIO Serial Tools  
**软件所有功能基于[iot.mi.com](https://iot.mi.com/new/doc/embedded-development/wifi/standard-protocol.html)开发**  
`后端`开发者（固件开发）可以依据**产品功能定义**利用[mst](https://github.com/kerwin612/miio-serial-tools)来**模拟固件功能**，便于`前端`开发者（米家APP插件开发）真实且高效的**联调接口**  

> 串口配置信息：  
串口 -> miot模组  
VCC  -> (ESP-WROOM-02: 3V3) / (ESP-WROOM-32: 3V3) / (MHCWB4P: VDD)  
GND  -> GND  
RXD	 -> (ESP-WROOM-02: IO15-CMD|IO2-LOG) / (ESP-WROOM-32: GPIO17-CMD|TXD0-LOG) / (MHCWB4P: IO14-CMD)  
TXD  -> (ESP-WROOM-02: IO13-CMDIN) / (ESP-WROOM-32: GPIO16-CMDIN) / (MHCWB4P: IO13-CMDIN)  

