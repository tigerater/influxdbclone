module github.com/influxdata/promqltests

go 1.12

require (
	github.com/glycerine/go-unsnap-stream v0.0.0-20190901134440-81cf024a9e0a // indirect
	github.com/glycerine/goconvey v0.0.0-20190410193231-58a59202ab31 // indirect
	github.com/gogo/protobuf v1.3.0 // indirect
	github.com/google/go-cmp v0.3.1
	github.com/influxdata/flux v0.59.4
	github.com/influxdata/influxdb v0.0.0-20190925213338-8af36d5aaedd
	github.com/influxdata/influxql v1.0.1 // indirect
	github.com/influxdata/promql/v2 v2.12.0
	github.com/mattn/go-isatty v0.0.9 // indirect
	github.com/prometheus/common v0.7.0
	github.com/prometheus/prometheus v2.9.2+incompatible
	github.com/prometheus/tsdb v0.10.0
	github.com/willf/bitset v1.1.10 // indirect
	go.uber.org/zap v1.10.0 // indirect
	golang.org/x/time v0.0.0-20190921001708-c4c64cad1fd0 // indirect
	google.golang.org/grpc v1.24.0 // indirect
)

replace github.com/influxdata/influxdb => ../../../../
