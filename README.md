### Usage
```
Usage of ./backuppc_exporter:
-config-dir string
Path to directory BackupPC configuration config.pl (default "/etc/backuppc")
-data-dir string
Path to directory with pc, cpool and pool directories (default "/var/lib/backuppc")
-listen-address string
The address to listen on for HTTP requests. (default ":8080")
-refresh-interval int
Metric refresh interval in seconds (default 60)
```

### Installation
* Clone and build
```
git clone git@github.com:otwieracz/backuppc_exporter.git
cd backuppc_exporter
go build # or GOOS=freebsd gobuild when cross-compiling for FreeBSD
```
* Copy `backuppc_exporter` to `/usr/local/bin` on target server.
* When target server is Linux controlled with `systemd`:
  * Copy `dist/backuppc_exporter.service` to target server as `/etc/systemd/system/backuppc_exporter.service`
  * `systemctl daemon-reload`
  * `systemctl enable backuppc_exporter`
  * `systemctl start backuppc_exporter`
* When target server is legacy Ubuntu controlled with `upstart`:
  * Copy `dist/backuppc_exporter.upstart` to target server as `/etc/init.d/backuppc_exporter`
  * `update-rc.d backuppc_exporter defaults`
  * `update-rc.d backuppc_exporter enable`
  * `service backuppc_exporter start`
* When target server is FreeBSD:
  * Copy `dist/backuppc_exporter.freebsd` to target server as `/usr/local/etc/rc.d/backuppc_exporter`
  * In `/etc/rc.conf` set `backuppc_exporter_enabled="YES"`
  * `service backuppc_exporter start`

### Requirements
Has to be running as `backuppc` user in order to access BackupPC CGI interface.

### Exposed metrics
* standard `golang` metrics
* `backuppc_disabled_hosts_count`
```
# HELP backuppc_disabled_hosts_count BackupPC disabled hosts
# TYPE backuppc_disabled_hosts_count gauge
backuppc_disabled_hosts_count 15
```
* `backuppc_last_age`
```
# HELP backuppc_last_age Age of most recent backup for every host, in seconds.
# TYPE backuppc_last_age gauge
backuppc_last_age{disabled="0",hostname="alpha"} 347076
backuppc_last_age{disabled="0",hostname="beta"} 325476
backuppc_last_age{disabled="1",hostname="gamma"} 325472
```
* `backuppc_pool_usage`
```
# HELP backuppc_pool_usage BackupPC pool usage (0 to 1)
# TYPE backuppc_pool_usage gauge
backuppc_pool_usage 0.44
```

### Example Prometheus rules

``` 
  - alert: BackuppcPoolStatus
    expr: backuppc_pool_usage > 0.9
    for: 1m
    labels:
      severity: warning
    annotations:
      identifier: '{{ $labels.instance }}'
      description: The BackupPC pool of {{ $labels.instance }} is almost full.
      summary: The BackupPC pool of {{ $labels.instance }} is almost full.
```

```
  - alert: BackuppcAllBackupsOnTime
    expr: ceil(backuppc_last_age/86400) > 2
    for: 1m
    labels:
      severity: warning
    annotations:
      identifier: '{{ $labels.instance }}'
      description: 'Last backup of {{ $labels.hostname }} on {{ $labels.instance }} is more than {{ $value }} days old.'
      summary: Backup too old for {{ $labels.hostname }}.
```

### Compatibility
Tested on:
* `FreeBSD backuppc 11.2-STABLE FreeBSD 11.2-STABLE #0 r325575+dac72894653(freenas/11-stable): Sun Sep  9 19:34:18 EDT 2018     root@nemesis.tn.ixsystems.com:/freenas-11.2-releng/freenas/_BE/objs/freenas-11.2-releng/freenas/_BE/os/sys/FreeNAS.amd64  amd64`
* `Linux backupm2 3.16.0-31-generic #41~14.04.1-Ubuntu SMP Wed Feb 11 19:30:13 UTC 2015 x86_64 x86_64 x86_64 GNU/Linux` (upstart)
