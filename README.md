# SML to HTTP proxy

This application is designed to read a binary SML stream via TCP/IP and export all values via a simple HTTP REST API using JSON.
It is intended to be used with home automation systems to read the values of smart meters.
In the most basic configuration, a serial to TCP/IP converter in conjunction with a D0 read head can be used.

## Building

To build this application first install Go 1.20 or later.
Then run the provided Makefile:

```shell
go build -o smlToHttp
```

## Usage

First create a configuration file named `config.yml`.
Replace the IP address serial to TCP/IP converter in the meters section.
Note: You can add multiple meters, not just one.

To disable the HTTP request log, set `web/disable_request_log` to `true`.
Similarly, you can disable the SML reception log by setting `meters/disable_reception_log` to `true` on a per meter basis.

```yaml
web:
  address: 127.0.0.1:11123
  disable_request_log: false

meters:
  - id: my_smartmeter
    address: 192.168.0.1:8234
    reconnect_delay: 10
    read_timeout: 5
    connect_timeout: 10
    debug: false
    disable_reception_log: false
```

Then start the application:

```shell
./smlToHttp -config config.yml
```

The log should now yield that the connection is successful and that SML frames are being decoded.
You can then try to access the API:

```shell
curl http://127.0.0.1:11123/processImage
```

The response should look something along the lines of:

```json
{
  "meters": {
    "my_smartmeter": {
      "connected": true,
      "lastUpdate": "2023-06-06T13:12:10.064515753Z",
      "values": {
        "1-0:1.8.0*255": {
          "value": 123456.7,
          "unit": 30
        },
        "1-0:2.8.0*255": {
          "value": 234567.8,
          "unit": 30
        },
        "1-0:16.7.0*255": {
          "value": -3210,
          "unit": 27
        },
        "... continued ...": {}
      }
    }
  }
}
```

The response above has been truncated a bit, but you should get the gist out of it.
In the example above, the OBIS key `1-0:1.8.0*255` yields a value of `123456.7`, which represents 123456.7 kWh of retrieved energy from the energy provider.
Also, we have sold 234567.8 kWh of energy to the service provider.
And our current power draw is -3210 W, so we are currently selling 3210 Watts to the service provider.
A positive value here would mean that we currently buy energy from the provider.
Please refer to your smart meter user manual for exported OBIS items.

Note:
Most smart meters will only export basic information via the optical interface when the PIN protection is not deactivated.
This is by design, as the currently used power is considered privacy sensitive.
To get the full data set, please refer to the manual of your smart meter on how to enable the full data set.

## Debugging SML output

When you have a dump of the meter's output in a file, you can decode the file's content using the following command.
It will dump every *valid* SML message in the file.
Invalid messages (e.g. CRC does not match or invalid structure) are ignored.

```shell
./smlToHttp -dump <file>
```

## Integration with OpenHAB

The proxy is currently in production use in combination with OpenHAB, but may of course serve other systems.
For this example, the HTTP binding and the JSONPath transformation addons are required.
An example configuration might be:

```text
Thing http:url:smlToHttp "SML to HTTP" [
    baseURL="http://127.0.0.1:11123/processImage",
    refresh=2
] {
    Channels:
        Type number : my_smartmeter_1_8_0 "My SmartMeter 1.8.0" [ stateTransformation="JSONPATH:$.meters.my_smartmeter.values.['1-0:1.8.0*255'].value", unit="Wh", mode="READONLY" ]
        Type number : my_smartmeter_2_8_0 "My SmartMeter 2.8.0" [ stateTransformation="JSONPATH:$.meters.my_smartmeter.values.['1-0:2.8.0*255'].value", unit="Wh", mode="READONLY" ]
        Type number : my_smartmeter_16_7_0 "My SmartMeter 16.7.0" [ stateTransformation="JSONPATH:$.meters.my_smartmeter.values.['1-0:16.7.0*255'].value", unit="W", mode="READONLY" ]
}
```

## License

    SML to HTTP proxy
    Copyright (C) 2023  Stephan Brunner

    This program is free software: you can redistribute it and/or modify
    it under the terms of the GNU General Public License as published by
    the Free Software Foundation, either version 3 of the License, or
    (at your option) any later version.

    This program is distributed in the hope that it will be useful,
    but WITHOUT ANY WARRANTY; without even the implied warranty of
    MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.  See the
    GNU General Public License for more details.

    You should have received a copy of the GNU General Public License
    along with this program.  If not, see <https://www.gnu.org/licenses/>.
