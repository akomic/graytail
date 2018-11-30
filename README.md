GrayTail
========

WebSocket client for live log tailing, connects to [Grayproxy](https://github.com/andviro/grayproxy).
Supports server side filtering of log messages.

# Configuration

It can be specified either through cli params or via config file.
Default config file location is ~/.graytailrc.yml and example of the file is
graytailrc.yml.example.

# Running
```shell
graytail --uri ws://MyToken@127.0.0.1:20221 -f container_name=nginx
```

# Making it work with Graylog

Run [Grayproxy](https://github.com/andviro/grayproxy) somewhere
```shell
./grayproxy -in udp://:8181 -out ws://MyToken@0.0.0.0:20221
```

Configure Graylog Output:
System > Outputs > GELF Output

```
protocol: UDP
port: 8181
```

Configure Stream:
Streams > All messages > Manage Outputs > Assign existing Output
