# mqtt-bridgeify

This small app bridges mqtt-topics from local mosquitto to the cloud mqtt broker.

# Usage

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
