# mqtt-bridgeify

This small app bridges mqtt-topics from local mosquitto to the cloud mqtt broker.

# Usage

```
Usage: mqtt-bridgeify agent [options]

  Starts the MQTT bridgeify agent and runs until an interrupt is received.

Options:

  -localurl=tcp://localhost:1883           URL for the local broker.
  -debug                                   Enables debug output.
```

Typically you will just pass in your token.

```
$ mqtt-bridgeify agent
```

To instruct it to connect to the cloud you can just.

```
mosquitto_pub -m '{"id": "123", "url":"ssl://dev.ninjasphere.co:8883","token":"XXXX"}' -t '$sphere/bridge/connect'
```

And likewise to disconnect.

```
mosquitto_pub -m '{"id": "123"}' -t '$sphere/bridge/disconnect'
```

To listen to status messages just run.

```
$ mosquitto_sub -t '$sphere/bridge/status'
{"alloc":499952,"heapAlloc":499952,"totalAlloc":631704,"lastError":"","connected":true,"configured":true,"count":0}
```

To listen for responses.

```
$ mosquitto_sub -t '$sphere/bridge/response'
{"id":"123","connected":true,"configured":true,"lastError":""}
```

# Bridge

Currently uses the following mappings.

```go
var localTopics = []replaceTopic{
	{on: "$location/calibration", replace: "$location", with: "$cloud/location"},
	{on: "$location/delete", replace: "$location", with: "$cloud/location"},
	{on: "$device/+/+/rssi", replace: "$device", with: "$cloud/device"},
	{on: "$node/+/module/status", replace: "$node", with: "$cloud/node"},
}

var cloudTopics = []replaceTopic{
	{on: "$cloud/location/calibration/progress", replace: "$cloud/location", with: "$location"},
	{on: "$cloud/device/+/+/location", replace: "$cloud/device", with: "$device"},
}
```
