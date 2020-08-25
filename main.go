package main

import (
	"bufio"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

/* CLI flags */
var (
	addr     = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	interval = flag.Int("refresh-interval", 60, "Metric refresh interval in seconds")
	/* BackupPC-specific */
	dataDir   = flag.String("data-dir", "/var/lib/backuppc", "Path to directory with pc, cpool and pool directories")
	configDir = flag.String("config-dir", "/etc/backuppc", "Path to directory BackupPC configuration config.pl")
)

/* DiskUsage - https://gist.github.com/lunny/9828326 */

type DiskStatus struct {
	All  uint64 `json:"all"`
	Used uint64 `json:"used"`
	Free uint64 `json:"free"`
}

// disk usage of path/disk

func DiskUsage(path string) (disk DiskStatus) {
	fs := syscall.Statfs_t{}
	err := syscall.Statfs(path, &fs)
	if err != nil {
		return
	}
	disk.All = fs.Blocks * uint64(fs.Bsize)
	disk.Free = fs.Bfree * uint64(fs.Bsize)
	disk.Used = disk.All - disk.Free
	return
}

const (
	B  = 1
	KB = 1024 * B
	MB = 1024 * KB
	GB = 1024 * MB
)

/* Metrics */
var (
	disabledHostsMetric = prometheus.NewGauge(prometheus.GaugeOpts{Name: "backuppc_disabled_hosts_count", Help: "BackupPC disabled hosts"})
	poolUsageMetric     = prometheus.NewGauge(prometheus.GaugeOpts{Name: "backuppc_pool_usage", Help: "BackupPC pool usage (0 to 1)"})
	lastAgeMetric       = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backuppc_last_age",
			Help: "Age of most recent backup for every host, in seconds.",
		},
		[]string{
			"hostname",
			"disabled",
		},
	)
)

func poolUsageMetricFn() {
	disk := DiskUsage(fmt.Sprintf("%s/cpool", *dataDir))
	poolUsageFloat := float64(disk.Used) / float64(disk.All)
	poolUsageMetric.Set(poolUsageFloat)
}

func hosts() []string {
	var hostsFound []string

	hostsFile := fmt.Sprintf("%s/hosts", *configDir)
	file, err := os.Open(hostsFile)

	if err == nil {
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			s := scanner.Text()
			match, _ := regexp.MatchString("^ *(#).*$", s)
			fields := strings.Fields(s)
			if !match && len(fields) >= 2 {
				hostname := fields[0]
				if hostname != "host" {
					hostsFound = append(hostsFound, hostname)
				}
			}
		}
	}
	defer file.Close()
	return hostsFound
}

func lastAgeMetricFn() {
	for _, hostname := range hosts() {
		backupsPath := fmt.Sprintf("%s/pc/%s/backups", *dataDir, hostname)

		file, err := os.Open(backupsPath)
		if err == nil {
			scanner := bufio.NewScanner(file)
			minAge := int64(-1)
			for scanner.Scan() {
				s := scanner.Text()
				if strings.Contains(s, "full") || strings.Contains(s, "incr") {
					timestamp, _ := strconv.Atoi((strings.Fields(s)[2]))
					age := time.Now().Unix() - int64(timestamp)

					if minAge < 0 || minAge > age {
						minAge = age
					}
				}
			}
			var disabled int64
			if hostDisabled(hostname) {
				disabled = 1
			}
			lastAgeMetric.WithLabelValues(hostname, strconv.FormatInt(disabled, 10)).Set(float64(minAge))
		}
		defer file.Close()
	}
}

func hostDisabled(hostname string) (disabled bool) {
	hostConfigPath := fmt.Sprintf("%s/%s.pl", *configDir, hostname)

	file, err := os.Open(hostConfigPath)
	if err == nil {
		defer file.Close()
		scanner := bufio.NewScanner(file)
		for scanner.Scan() {
			s := scanner.Text()
			match, _ := regexp.MatchString(`^ *\$Conf{BackupsDisable} *= *1 *;`, s)
			if match {
				disabled = true
			}
		}
	}
	return
}

func disabledHostsMetricFn() {
	var disabled int
	for _, hostname := range hosts() {
		if hostDisabled(hostname) {
			disabled++
		}
	}
	disabledHostsMetric.Set(float64(disabled))
}

func main() {
	flag.Parse()
	disabledHostsMetricFn()

	prometheus.MustRegister(disabledHostsMetric)
	prometheus.MustRegister(poolUsageMetric)
	prometheus.MustRegister(lastAgeMetric)

	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	go func() {
		for range ticker.C {
			disabledHostsMetricFn()
			poolUsageMetricFn()
			lastAgeMetricFn()
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}
