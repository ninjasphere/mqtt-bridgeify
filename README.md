# mqtt-bridgeify

This small app bridges mqtt-topics from local mosquitto to the cloud mqtt broker.

# Usage

```
Usage: mqtt-bridgeify agent [options]

  Starts the MQTT bridgeify agent and runs until an interrupt is received.

Options:

  -localurl=tcp://localhost:1883           URL for the local broker.
  -localurl=ssl://dev.ninjasphere.co:8883  URL for the remote broker.
  -debug                                   Enables debug output.
  -token=                                  The ninja sphere token.
```

Typically you will just pass in your token.

```
$ mqtt-bridgeify agent -token XXX
```

# Bridge

Currently uses the following mappings.

```go
var localTopics = []replaceTopic{
  {on: "$location/calibration", replace: "$location", with: "$cloud/location"},
  {on: "$device/+/+/rssi", replace: "$device", with: "$cloud/device"},
}

var cloudTopics = []replaceTopic{
  {on: "$cloud/location/calibration/complete", replace: "$cloud/location", with: "$location"},
  {on: "$cloud/device/+/+/location", replace: "$cloud/device", with: "$device"},
}
```
