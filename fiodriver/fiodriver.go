// Package fiodriver Run fio and parse its output
package fiodriver

import (
	"bufio"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"strconv"
	"strings"
)

// Fields fio terse output v3
// Refer to https://fio.readthedocs.io/en/latest/fio_doc.html#terse-output for details
var Fields = []string{
	// "terse_version_3", "fio_version", "jobname", "groupid",
	"error",
	"read_kb", "read_bandwidth", "read_iops", "read_runtime_ms",
	"read_slat_min", "read_slat_max", "read_slat_mean", "read_slat_dev",
	"read_clat_min", "read_clat_max", "read_clat_mean", "read_clat_dev",
	"read_clat_pct01", "read_clat_pct02", "read_clat_pct03", "read_clat_pct04", "read_clat_pct05",
	"read_clat_pct06", "read_clat_pct07", "read_clat_pct08", "read_clat_pct09", "read_clat_pct10",
	"read_clat_pct11", "read_clat_pct12", "read_clat_pct13", "read_clat_pct14", "read_clat_pct15",
	"read_clat_pct16", "read_clat_pct17", "read_clat_pct18", "read_clat_pct19", "read_clat_pct20",
	"read_tlat_min", "read_lat_max", "read_lat_mean", "read_lat_dev",
	"read_bw_min", "read_bw_max", "read_bw_agg_pct", "read_bw_mean", "read_bw_dev",
	"write_kb", "write_bandwidth", "write_iops", "write_runtime_ms",
	"write_slat_min", "write_slat_max", "write_slat_mean", "write_slat_dev",
	"write_clat_min", "write_clat_max", "write_clat_mean", "write_clat_dev",
	"write_clat_pct01", "write_clat_pct02", "write_clat_pct03", "write_clat_pct04", "write_clat_pct05",
	"write_clat_pct06", "write_clat_pct07", "write_clat_pct08", "write_clat_pct09", "write_clat_pct10",
	"write_clat_pct11", "write_clat_pct12", "write_clat_pct13", "write_clat_pct14", "write_clat_pct15",
	"write_clat_pct16", "write_clat_pct17", "write_clat_pct18", "write_clat_pct19", "write_clat_pct20",
	"write_tlat_min", "write_lat_max", "write_lat_mean", "write_lat_dev",
	"write_bw_min", "write_bw_max", "write_bw_agg_pct", "write_bw_mean", "write_bw_dev",
	"cpu_user", "cpu_sys", "cpu_csw", "cpu_mjf", "cpu_minf",
	"iodepth_1", "iodepth_2", "iodepth_4", "iodepth_8", "iodepth_16", "iodepth_32", "iodepth_64",
	"lat_2us", "lat_4us", "lat_10us", "lat_20us", "lat_50us", "lat_100us", "lat_250us", "lat_500us", "lat_750us", "lat_1000us",
	"lat_2ms", "lat_4ms", "lat_10ms", "lat_20ms", "lat_50ms", "lat_100ms", "lat_250ms", "lat_500ms", "lat_750ms", "lat_1000ms", "lat_2000ms", "lat_over_2000ms",
	// "disk_name", "disk_read_iops", "disk_write_iops", "disk_read_merges", "disk_write_merges", "disk_read_ticks", "write_ticks", "disk_queue_time", "disk_util",
}

// FioRunner  Run fio and capture its output
/*
	bpath: fio binary path. If it can be found from PATH, no need to specify the absolute path
	jpath: fio job profile path
	interval: interval in seconds to output result periodically
	ch: result will be delived to this channel
*/
func FioRunner(bpath, jpath string, interval uint8, mch chan map[string]float64, pch chan int) {
	cmd := exec.Command(bpath, jpath, "--group_reporting", fmt.Sprintf("--status-interval=%d", interval), "--output-format=terse", "--terse-version=3")
	stdout, err := cmd.StdoutPipe()
	if err != nil {
		log.Fatalf("Fail to create a pipe for fio output capture: %s", err)
	}

	if err := cmd.Start(); err != nil {
		log.Fatalf("Fail to start fio: %s", err)
	}

	go func() {
		pid := cmd.Process.Pid
		pch <- pid
		close(pch)

		err := cmd.Wait()
		if err != nil {
			log.Fatalf("fio run issue: %s", err)
		}
		log.Println("fio completed")
		os.Exit(0)
	}()

	var ecounter int
	scanner := bufio.NewScanner(stdout)
	for scanner.Scan() {
		line := scanner.Text()
		// log.Printf("Capture fio output: %s\n", line)
		metric, err := decode(line)
		// TBD - hard coded error toleration(3 x times), need to change the implementation
		if err != nil {
			if ecounter >= 3 {
				log.Println("Try to kill fio process")
				kerr := cmd.Process.Kill()
				if kerr != nil {
					log.Printf("Cannot kill fio process, please kill the process manually")
				}
				log.Fatalf("Hit error for %d times decoding fio output, exit", ecounter+1)
			}
			log.Printf("Hit error for %d times while decoding fio output", ecounter+1)
			ecounter++
		} else {
			if ecounter > 0 {
				ecounter--
			}
			mch <- metric
		}
	}

	if err := scanner.Err(); err != nil {
		log.Fatalf("Fail to parse fio output: %s", err)
	}
}

func decode(line string) (map[string]float64, error) {
	line = strings.TrimSuffix(line, "\n")

	rawv := strings.Split(line, ";")
	rawvl := len(rawv)
	if rawvl == 130 {
		rawv = rawv[4:121]
	} else {
		if rawvl <= 4 {
			msg := "Invalid result which contain less than expected fields"
			log.Printf(msg+": %v", rawv)
			return nil, errors.New(msg)
		}
		rawv = rawv[4:]
	}

	if len(rawv) != 117 {
		msg := "More fields are found for the result which cannot be understood"
		log.Printf(msg+": %v", rawv)
		return nil, errors.New(msg)
	}

	// log.Printf("Result: %v", rawv)
	for i, v := range rawv {
		if strings.Contains(v, "=") {
			rawv[i] = strings.Split(v, "=")[1]
		}
	}

	var metricv []float64
	for _, v := range rawv {
		if strings.Contains(v, "%") {
			v = strings.TrimSuffix(v, "%")
			m, err := strconv.ParseFloat(v, 64)
			if err != nil {
				m = 0
			}
			metricv = append(metricv, m/100.0)
		} else {
			m, err := strconv.ParseFloat(v, 64)
			if err != nil {
				m = 0
			}
			metricv = append(metricv, m)
		}
	}
	// log.Printf("Result after converting: %v", metricv)

	metrics := make(map[string]float64)
	for i, v := range metricv {
		k := Fields[i]
		metrics[k] = v
	}
	// log.Printf("Metric map: %+v", metrics)
	return metrics, nil
}
