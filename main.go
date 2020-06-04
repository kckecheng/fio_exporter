package main

import (
	"errors"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"

	"github.com/kckecheng/fio_exporter/exporter"
	"github.com/kckecheng/fio_exporter/fiodriver"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"

	flag "github.com/spf13/pflag"
)

var (
	bpath    string
	jpath    string
	interval uint8
	port     uint64
)

func init() {
	flag.ErrHelp = errors.New("help requested")
	flag.StringVarP(&bpath, "path", "p", "fio", "fio binary path, if fio can be located within PATH, no need to specify the absolute path")
	flag.StringVarP(&jpath, "job", "j", "", "fio job file path")
	flag.Uint8VarP(&interval, "interval", "i", 30, "time interval fio to calculate average results (refer to option --status-interval)")
	flag.Uint64VarP(&port, "port", "l", 8080, "exporter listen port")
}

func optValidate() {
	if bpath == "" || jpath == "" {
		log.Fatalln(`Options "path" and "job" should both be specified`)
	}

	if interval == 0 {
		log.Fatalln("Refresh interval should be greater than 0")
	}

	if port == 0 {
		log.Fatalln("Exporter lisentening port should be larger than 0")
	}
}

func main() {
	flag.Parse()
	optValidate()

	mch := make(chan map[string]float64)
	pch := make(chan int)
	go fiodriver.FioRunner(bpath, jpath, interval, mch, pch)

	// Make sure fio process has been started
	pid := <-pch
	log.Printf("fio process %d has been started", pid)

	go func() {
		for metrics := range mch {
			log.Println("Update metric based on fio periodical stats")
			exporter.UpdateMetric(metrics)
		}
	}()

	mc := exporter.NewCollector()
	reg := prometheus.NewRegistry()
	reg.MustRegister(mc)
	http.Handle("/metrics", promhttp.HandlerFor(reg, promhttp.HandlerOpts{}))

	go func() {
		log.Printf("Start exporter, please access results from http://localhost:%d/metrics", port)
		err := http.ListenAndServe(fmt.Sprintf(":%d", port), nil)
		if err != nil {
			log.Fatalln("Fail to start exporter")
		}
	}()

	sigc := make(chan os.Signal, 1)
	signal.Notify(sigc, syscall.SIGHUP, syscall.SIGINT, syscall.SIGTERM, syscall.SIGQUIT)

	s := <-sigc
	log.Printf("Signal %s has been captured, stop fio", s.String())
	process, err := os.FindProcess(pid)
	if err != nil {
		log.Fatalf("Fail to find process with PID %d, please kill it manually", pid)
	}
	err = process.Kill()
	if err != nil {
		log.Fatalf("Fail to stop fio process %d, please kill it manually", pid)
	}
}
