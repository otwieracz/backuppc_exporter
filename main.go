package main

import (
	"bufio"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"syscall"
	"time"
)

/* CLI flags */
var (
	addr     = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	interval = flag.Int("refresh-interval", 60, "Metric refresh interval in seconds")
	/* BackupPC-specific */
	dataDir = flag.String("data-dir", "/var/lib/backuppc", "Path to directory with pc, cpool and pool directories")
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
	poolUsageMetric = prometheus.NewGauge(prometheus.GaugeOpts{Name: "backuppc_pool_usage", Help: "BackupPC pool usage (0 to 1)"})
	lastAgeMetric   = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backuppc_last_age",
			Help: "Age of most recent backup for every host, in seconds.",
		},
		[]string{
			"hostname",
		},
	)
	numberOfBackupsMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backuppc_number_of_backups",
			Help: "Number of backups for every host.",
		},
		[]string{
			"hostname",
		},
	)
	numberOfIncrementalBackupsMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backuppc_number_incremental_backups",
			Help: "Number of incremental backups for every host.",
		},
		[]string{
			"hostname",
		},
	)
	numberOfFullBackupsMetric = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "backuppc_number_full_backups",
			Help: "Number of full backups for every host.",
		},
		[]string{
			"hostname",
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

	hostsFolder := fmt.Sprintf("%s/pc/", *dataDir)
	hostFolderNames, errReadDir := ioutil.ReadDir(hostsFolder)

	if errReadDir != nil {
		log.Fatal(errReadDir)
	}

	for _, filename := range hostFolderNames {

		hostsFound = append(hostsFound, filename.Name())
	}

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
			lastAgeMetric.WithLabelValues(hostname).Set(float64(minAge))
		}
		defer file.Close()
	}
}

func numberOfBackupsMetricFn() {
	for _, hostname := range hosts() {
		backupsPath := fmt.Sprintf("%s/pc/%s/backups", *dataDir, hostname)

		var numberOfBackups float64 = 0
		file, err := os.Open(backupsPath)
		if err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {
				numberOfBackups++
			}
		}
		numberOfBackupsMetric.WithLabelValues(hostname).Set(numberOfBackups)
		defer file.Close()
	}
}

func numberOfIncrementalBackupsMetricFn() {
	for _, hostname := range hosts() {
		backupsPath := fmt.Sprintf("%s/pc/%s/backups", *dataDir, hostname)

		var numberOfIncrementalBackups float64 = 0
		file, err := os.Open(backupsPath)
		if err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {

				s := scanner.Text()
				if strings.Contains(s, "incr") {
					numberOfIncrementalBackups++
				}
			}
		}
		numberOfIncrementalBackupsMetric.WithLabelValues(hostname).Set(numberOfIncrementalBackups)
		defer file.Close()
	}
}

func numberOfFullBackupsMetricFn() {
	for _, hostname := range hosts() {
		backupsPath := fmt.Sprintf("%s/pc/%s/backups", *dataDir, hostname)

		var numberOfFullBackups float64 = 0
		file, err := os.Open(backupsPath)
		if err == nil {
			scanner := bufio.NewScanner(file)
			for scanner.Scan() {

				s := scanner.Text()
				if strings.Contains(s, "full") {
					numberOfFullBackups++
				}
			}
		}
		numberOfFullBackupsMetric.WithLabelValues(hostname).Set(numberOfFullBackups)
		defer file.Close()
	}
}

func main() {

	flag.Parse()

	prometheus.MustRegister(poolUsageMetric)
	prometheus.MustRegister(lastAgeMetric)
	prometheus.MustRegister(numberOfBackupsMetric)
	prometheus.MustRegister(numberOfIncrementalBackupsMetric)
	prometheus.MustRegister(numberOfFullBackupsMetric)
	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	go func() {
		for range ticker.C {
			poolUsageMetricFn()
			lastAgeMetricFn()
			numberOfBackupsMetricFn()
			numberOfIncrementalBackupsMetricFn()
			numberOfFullBackupsMetricFn()
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}
