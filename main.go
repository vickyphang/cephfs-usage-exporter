package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os/exec"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

var (
	subvolumeUsage = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cephfs_subvolume_usage_bytes",
			Help: "Disk usage of CephFS subvolumes",
		},
		[]string{"subvolume"},
	)
	subvolumeQuota = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "cephfs_subvolume_quota_bytes",
			Help: "Disk quota of CephFS subvolumes",
		},
		[]string{"subvolume"},
	)
)

func init() {
	prometheus.MustRegister(subvolumeUsage, subvolumeQuota)
}

type SubvolumeInfo struct {
	Name       string `json:"name"`
	BytesUsed  int64  `json:"bytes_used"`
	BytesQuota int64  `json:"bytes_quota"`
}

func getSubvolumes(filesystem string, path string) ([]string, error) {
	cmd := exec.Command("ceph", "fs", "subvolume", "ls", filesystem, path)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	var subvolumes []map[string]string
	err = json.Unmarshal(output, &subvolumes)
	if err != nil {
		return nil, err
	}

	var names []string
	for _, subvolume := range subvolumes {
		names = append(names, path+"/"+subvolume["name"])
	}
	return names, nil
}

func getSubvolumeUsage(filesystem, subvolume string) (SubvolumeInfo, error) {
	cmd := exec.Command("ceph", "fs", "subvolume", "info", filesystem, subvolume)
	output, err := cmd.Output()
	if err != nil {
		return SubvolumeInfo{}, err
	}

	var info SubvolumeInfo
	err = json.Unmarshal(output, &info)
	if err != nil {
		return SubvolumeInfo{}, err
	}
	return info, nil
}

func collectMetrics(filesystem string, path string) {
	subvolumes, err := getSubvolumes(filesystem, path)
	if err != nil {
		log.Printf("Error fetching subvolumes: %v", err)
		return
	}

	for _, subvolume := range subvolumes {
		usage, err := getSubvolumeUsage(filesystem, subvolume)
		if err != nil {
			log.Printf("Error fetching usage for subvolume %s: %v", subvolume, err)
			continue
		}
		subvolumeUsage.WithLabelValues(subvolume).Set(float64(usage.BytesUsed))
		subvolumeQuota.WithLabelValues(subvolume).Set(float64(usage.BytesQuota))
	}
}

func main() {
	filesystem := "k8s-fs"
	path := "/volumes/csi"
	interval := 60 * time.Second

	go func() {
		for {
			collectMetrics(filesystem, path)
			time.Sleep(interval)
		}
	}()

	http.Handle("/metrics", promhttp.Handler())
	log.Fatal(http.ListenAndServe(":8000", nil))
}
