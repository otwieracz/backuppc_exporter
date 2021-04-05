### Usage
```
Usage of ./backuppc_exporter:
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
go get -u github.com/prometheus/client_golang/prometheus
go build -o build/backuppc_exporter # or GOOS=freebsd gobuild when cross-compiling for FreeBSD
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
* `backuppc_last_age`
```
# HELP backuppc_last_age Age of most recent backup for every host, in seconds.
# TYPE backuppc_last_age gauge
backuppc_last_age{hostname="alpha"} 347076
backuppc_last_age{hostname="beta"} 325476
backuppc_last_age{hostname="gamma"} 325472
```
* `backuppc_pool_usage`
```
# HELP backuppc_pool_usage BackupPC pool usage (0 to 1)
# TYPE backuppc_pool_usage gauge
backuppc_pool_usage 0.44
```

* `backuppc_number_of_backups`
```
# HELP backuppc_number_of_backups Number of backups for every host.
# TYPE backuppc_number_of_backups gauge
backuppc_number_of_backups{hostname="alpha"} 10
backuppc_number_of_backups{hostname="beta"} 10
backuppc_number_of_backups{hostname="gamma"} 0
```

* `backuppc_number_incremental_backups`
```
# HELP backuppc_number_incremental_backups Number of incremental backups for every host.
# TYPE backuppc_number_incremental_backups gauge
backuppc_number_incremental_backups{hostname="alpha"} 7
backuppc_number_incremental_backups{hostname="beta"} 7
backuppc_number_incremental_backups{hostname="gamma"} 0
```

* `backuppc_number_full_backups`
```
# HELP backuppc_number_full_backups Number of incremental backups for every host.
# TYPE backuppc_number_full_backups gauge
backuppc_number_full_backups{hostname="alpha"} 3
backuppc_number_full_backups{hostname="beta"} 3
backuppc_number_full_backups{hostname="gamma"} 0
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
* `Linux ubuntu 5.4.0-1032-raspi #35-Ubuntu SMP PREEMPT Fri Mar 19 20:52:40 UTC 2021 aarch64 aarch64 aarch64 GNU/Linux`
