hkswitch [![tests badge][tests-badge]][tests-page]
===

Control programs and services on Linux and macOS from HomeKit.

Install
---

```shell
$ go get mrz.io/hkswitch 
```

Usage example
---

This _not very useful_ example show how to create an HomeKit bridge with one switch to start the `sleep` program.

1. Create a `config.yaml` file

    ```yaml
    # publish Prometheus metrics (hkswitch_up=1/0) at localhost:9102/metrics .
    metrics:
      address: :9102
   
    bridge:
      # name of the bridge
      name: Services Example
      
      # port, leave empty to use a random one
      port: 5559
      
      # pairing pin
      pin: 12345678
      
      # directory where the bridge state is stored across runs; when not set, 
      # a directory with the same name as the bridge will be created in the
      # current working directory -- the one from which `hkswitch` is started,
      # not the service(s) work-dir
      storage-dir: /Users/username/Library/Caches/hkswitch-Services-Examples
      
      # you can also set how the bridge identifies itself in the HomeKit app
      # manufacturer: ...
      # serial-number: ...
      # model: ...
      # firmware: ...

    services:
      - 
        # set the name for the switch accessory representing this service
        name: "sleep"
        
        # service's working directory
        work-dir: /Users/username
        
        # the service will inherit the environment from hkswitch by default, use
        # the `env` field to add or redefine environment variables
        env: 
          - DURATION=30
          
        # command line to start the service
        command: [bash, -c, "sleep $DURATION"]
   
        # optionally set to the signal preferred by the service for a clean shutdown
        stop-signal: INT
        
        # optionally set to true to start the program (eg. "turn on the switch")
        # when hkswitch starts
        autostart: false
    ```
   
2. Start the bridge
   
   ```shell
   $ hkswitch config.yaml
   ```

3. Head over to the HomeKit app on your phone, tap Add Accessory and add the `Services` bridge to a room
   
At the end of the wizard you'll find a new Switch in the room: tap it to "turn it on" and start the backup, tap it
again to "turn it off", and stop `sleep` by sending it the `TERM` signal.

Run on boot
---

Use the `print-conf [launchd|systemd]` subcommand to generate a configuration files to start `hkswitch` as
a daemon with `launchd` or `systemd` using the specified configuration file, eg.:

```shell
$ hkswitch print-conf launchd -e PATH config.yaml > ~/Library/LaunchDaemons/io.mrz.hkswitch.example.plist
$ launchctl load -w ~/Library/LaunchDaemons/io.mrz.hkswitch.example.plist
```

Caveats
---

- On macOS, certain kind of services will require granting `hkswitch` Full Disk Access.
- Changing the order of services in between runs of `hkswitch` after the bridge was paired will
  confuse HomeKit, always add new services by appending to the `services` key.

[tests-badge]: https://github.com/marzocchi/hkswitch/actions/workflows/test.yaml/badge.svg
[tests-page]: https://github.com/marzocchi/hkswitch/actions/workflows/test.yaml
