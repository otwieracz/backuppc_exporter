### Usage
```
Usage of ./backuppc_exporter:
-listen-address string
The address to listen on for HTTP requests. (default ":8080")
-refresh-interval int
Metric refresh interval in seconds (default 60)
```

### Requirements
Has to be running as `backuppc` user in order to access BackupPC CGI interface.

### Exposed metrics
* standard `golang` metrics
* `max_last_age`
```
# HELP max_last_age Maximum age of last backup for every host. There is no host without backup for more days than this.
# TYPE max_last_age gauge
max_last_age 0.47994212962963
```
* `pool_usage` 
```
# HELP pool_usage BackupPC pool usage (0 to 1)
# TYPE pool_usage gauge
pool_usage 0.44
```

### Compatibility
Tested on:
* `FreeBSD backuppc 11.2-STABLE FreeBSD 11.2-STABLE #0 r325575+dac72894653(freenas/11-stable): Sun Sep  9 19:34:18 EDT 2018     root@nemesis.tn.ixsystems.com:/freenas-11.2-releng/freenas/_BE/objs/freenas-11.2-releng/freenas/_BE/os/sys/FreeNAS.amd64  amd64`
* `Linux backupm2 3.16.0-31-generic #41~14.04.1-Ubuntu SMP Wed Feb 11 19:30:13 UTC 2015 x86_64 x86_64 x86_64 GNU/Linux`
