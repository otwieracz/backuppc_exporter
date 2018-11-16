package main

import (
	"bytes"
	"flag"
	"fmt"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
	"net/http"
	"os/exec"
	"strconv"
	"time"
)

/* Perl middleware to BackupPC */

var backuppcMiddleware = `no  utf8;
use lib "/usr/local/lib";
use lib "/usr/share/backuppc/lib";
use BackupPC::Lib;
use List::Util qw( min max );

die("BackupPC::Lib->new failed\n") if ( !(my $bpc = BackupPC::Lib->new) );
my %Conf   = $bpc->Conf();

$bpc->ChildInit();

my $err = $bpc->ServerConnect($Conf{ServerHost}, $Conf{ServerPort});
if ( $err ) {
    print("Can't connect to server ($err)\n");
    exit(1);
}

sub pool_usage {
	eval $bpc->ServerMesg("status info");
	print "$Info{DUlastValue}";
}


sub max_last_age {
	my @ages;
	eval $bpc->ServerMesg("status hosts");
	foreach $host ( keys %Status ) {
		my @Backups = $bpc->BackupInfoRead($host);
		my $fullCnt = $incrCnt = 0;
		my $fullAge = $incrAge = $lastAge = -1;

		$bpc->ConfigRead($host);
		%Conf = $bpc->Conf();

		next if ( $Conf{XferMethod} eq "archive" );
		for ( my $i = 0 ; $i < @Backups ; $i++ ) {
			if ( $Backups[$i]{type} eq "full" ) {
				$fullCnt++;
				if ( $fullAge < 0 || $Backups[$i]{startTime} > $fullAge ) {
					$fullAge  = $Backups[$i]{startTime};
				}
			} else {
				$incrCnt++;
				if ( $incrAge < 0 || $Backups[$i]{startTime} > $incrAge ) {
					$incrAge = $Backups[$i]{startTime};
				}
			}
		}
		if ( $fullAge > $incrAge && $fullAge >= 0 )  {
			$lastAge = $fullAge;
		} else {
			$lastAge = $incrAge;
		}
		if ( $lastAge < 0 ) {
			$lastAge = 0;
		} else {
			$lastAge = (time - $lastAge) / (24 * 3600);
		}
		push @ages, $lastAge;
	}
	my $max = max @ages;
	print "$max";
}`

/* CLI flags */
var (
	addr     = flag.String("listen-address", ":8080", "The address to listen on for HTTP requests.")
	interval = flag.Int("refresh-interval", 60, "Metric refresh interval in seconds")
)

/* Metrics */
var (
	poolUsageMetric  = prometheus.NewGauge(prometheus.GaugeOpts{Name: "backuppc_pool_usage", Help: "BackupPC pool usage (0 to 1)"})
	maxLastAgeMetric = prometheus.NewGauge(prometheus.GaugeOpts{Name: "backuppc_max_last_age", Help: "Maximum age of last backup for every host. There is no host without backup for more days than this."})
)

func callBackuppcMiddleware(middleware_func string) string {
	var out bytes.Buffer
	var err bytes.Buffer
	args := []string{"perl", "-e", fmt.Sprintf("%s %s()", backuppcMiddleware, middleware_func)}
	cmd := exec.Command("/usr/bin/env", args...)
	cmd.Stderr = &err
	cmd.Stdout = &out
	errcode := cmd.Run()
	if errcode != nil {
		log.Fatal(err.String())
	}
	return out.String()
}

func poolUsageMetricFn() {
	poolUsagePercentage, _ := strconv.Atoi(callBackuppcMiddleware("pool_usage"))
	poolUsageFloat := float64(poolUsagePercentage) / 100
	poolUsageMetric.Set(poolUsageFloat)
}

func maxLastAgeMetricFn() {
	maxLastAge, _ := strconv.ParseFloat(callBackuppcMiddleware("max_last_age"), 64)
	maxLastAgeMetric.Set(maxLastAge)
}

func main() {
	flag.Parse()

	prometheus.MustRegister(poolUsageMetric)
	prometheus.MustRegister(maxLastAgeMetric)

	ticker := time.NewTicker(time.Duration(*interval) * time.Second)
	go func() {
		for range ticker.C {
			poolUsageMetricFn()
			maxLastAgeMetricFn()
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(*addr, nil))
}
