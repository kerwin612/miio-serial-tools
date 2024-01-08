go install github.com/rakyll/statik
go get github.com/kerwin612/miio-serial-tools/mst
statik -f -src=static && go build -ldflags -H=windowsgui
