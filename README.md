# MQTT Temperature (DS18S20)

Raspberry Pi One-Wire temperature publisher

## Build for raspberry pi

```
make
```

## Configuration

Default: `/etc/mqtemperature/config.yaml`

```
devices:
    "28-03146c2288ff": "hotwater"
    "28-03146cd21fff": "hotwater-return"
diffs:
    - topic: "hotwater-diff"
      device1: "28-03146c2288ff"
      device2: "28-03146cd21fff"
```

## Run

```
mqtemperature  --host=my-mqtt.server --interval=20
```
