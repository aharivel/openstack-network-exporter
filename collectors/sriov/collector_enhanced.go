// SPDX-License-Identifier: Apache-2.0
// Enhanced SR-IOV Collector with Queue Metrics Support
// This version handles both regular metrics and per-queue metrics

package sriov

import (
	"bufio"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/openstack-k8s-operators/openstack-network-exporter/log"
	"github.com/prometheus/client_golang/prometheus"
)

type Collector struct{}

func (Collector) Name() string {
	return "sriov"
}

func (Collector) Metrics() []lib.Metric {
	var res []lib.Metric
	for _, m := range metrics {
		res = append(res, m)
	}
	// Note: Queue metrics are dynamic, so we don't list them all here
	return res
}

func (c *Collector) Describe(ch chan<- *prometheus.Desc) {
	lib.DescribeEnabledMetrics(c, ch)
}

// InterfaceInfo contains metadata about an SR-IOV interface
type InterfaceInfo struct {
	Name     string
	IsPF     bool
	IsVF     bool
	ParentPF string
	VFNum    int
	NumVFs   int
}

// discoverSriovInterfaces discovers all SR-IOV interfaces using sysfs
func discoverSriovInterfaces() ([]InterfaceInfo, error) {
	var interfaces []InterfaceInfo

	netPath := "/sys/class/net"
	entries, err := os.ReadDir(netPath)
	if err != nil {
		return nil, err
	}

	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}

		name := entry.Name()
		info := InterfaceInfo{Name: name, VFNum: -1}

		// Check if it's a PF (has sriov_numvfs)
		numVFsPath := filepath.Join(netPath, name, "device/sriov_numvfs")
		if numVFs, err := readIntFromFile(numVFsPath); err == nil {
			info.IsPF = true
			info.NumVFs = numVFs
			interfaces = append(interfaces, info)
			continue
		}

		// Check if it's a VF (has physfn symlink)
		physfnPath := filepath.Join(netPath, name, "device/physfn")
		if _, err := os.Lstat(physfnPath); err == nil {
			info.IsVF = true

			devicePath := filepath.Join(netPath, name, "device")
			if vfNum, pfPath := getVFNumber(devicePath); vfNum >= 0 {
				info.VFNum = vfNum
				info.ParentPF = pfPath
			}

			interfaces = append(interfaces, info)
		}
	}

	return interfaces, nil
}

func readIntFromFile(path string) (int, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return 0, err
	}
	value, err := strconv.Atoi(strings.TrimSpace(string(data)))
	if err != nil {
		return 0, err
	}
	return value, nil
}

func getVFNumber(devicePath string) (int, string) {
	physfnPath := filepath.Join(devicePath, "physfn")
	pfDevice, err := os.Readlink(physfnPath)
	if err != nil {
		return -1, ""
	}

	pfDevicePath := filepath.Join(devicePath, pfDevice)
	entries, err := os.ReadDir(pfDevicePath)
	if err != nil {
		return -1, ""
	}

	myDevice, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return -1, ""
	}

	virtfnRe := regexp.MustCompile(`^virtfn(\d+)$`)
	for _, entry := range entries {
		match := virtfnRe.FindStringSubmatch(entry.Name())
		if match == nil {
			continue
		}

		virtfnPath := filepath.Join(pfDevicePath, entry.Name())
		target, err := os.Readlink(virtfnPath)
		if err != nil {
			continue
		}

		targetPath := filepath.Join(pfDevicePath, target)
		targetAbs, err := filepath.EvalSymlinks(targetPath)
		if err != nil {
			continue
		}

		if targetAbs == myDevice {
			vfNum, _ := strconv.Atoi(match[1])
			return vfNum, filepath.Base(pfDevicePath)
		}
	}

	return -1, ""
}

var ethtoolStatRe = regexp.MustCompile(`^\s+(\w+):\s+(\d+)$`)

// getEthtoolStats executes ethtool -S and parses the output
func getEthtoolStats(iface string) (map[string]float64, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	cmd := exec.CommandContext(ctx, "ethtool", "-S", iface)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	stats := make(map[string]float64)
	scanner := bufio.NewScanner(strings.NewReader(string(output)))

	for scanner.Scan() {
		line := scanner.Text()
		match := ethtoolStatRe.FindStringSubmatch(line)
		if match != nil {
			value, err := strconv.ParseFloat(match[2], 64)
			if err != nil {
				log.Debugf("failed to parse %s=%s: %s", match[1], match[2], err)
				continue
			}
			stats[match[1]] = value
		}
	}

	return stats, scanner.Err()
}

func buildLabels(info InterfaceInfo) []string {
	vfNum := ""
	if info.VFNum >= 0 {
		vfNum = strconv.Itoa(info.VFNum)
	}

	ifType := "unknown"
	if info.IsPF {
		ifType = "pf"
	} else if info.IsVF {
		ifType = "vf"
	}

	return []string{
		info.Name,
		ifType,
		info.ParentPF,
		vfNum,
	}
}

// Regular expression to extract queue number from stat names
// Matches: tx_queue_5_packets, rx_queue_12_bytes, etc.
var queueStatRe = regexp.MustCompile(`^(tx|rx)_queue_(\d+)_(packets|bytes)$`)

func (Collector) Collect(ch chan<- prometheus.Metric) {
	interfaces, err := discoverSriovInterfaces()
	if err != nil {
		log.Errf("failed to discover SR-IOV interfaces: %s", err)
		return
	}

	log.Debugf("discovered %d SR-IOV interfaces", len(interfaces))

	for _, iface := range interfaces {
		stats, err := getEthtoolStats(iface.Name)
		if err != nil {
			log.Debugf("ethtool -S %s: %s", iface.Name, err)
			continue
		}

		log.Debugf("collected %d stats for %s (PF=%v, VF=%v)",
			len(stats), iface.Name, iface.IsPF, iface.IsVF)

		labels := buildLabels(iface)

		// Process regular metrics (without queue dimension)
		for statName, value := range stats {
			// Check if it's a queue-specific stat
			if match := queueStatRe.FindStringSubmatch(statName); match != nil {
				// This is a queue stat, handle it separately
				direction := match[1]  // tx or rx
				queueNum := match[2]   // queue number
				statType := match[3]   // packets or bytes

				// Build the metric name and labels for queue stats
				metricName := "sriov_" + direction + "_queue_" + statType + "_total"
				queueLabels := append(labels, queueNum)

				// Create metric descriptor on the fly
				desc := prometheus.NewDesc(
					metricName,
					statType+" "+direction+" on queue",
					append(commonLabels, "queue"),
					nil,
				)

				// Only export if perf metrics are enabled
				if config.MetricSets().Has(config.METRICS_PERF) {
					ch <- prometheus.MustNewConstMetric(
						desc, prometheus.CounterValue, value, queueLabels...)
				}
				continue
			}

			// Regular metric (no queue dimension)
			if m, ok := metrics[statName]; ok {
				if config.MetricSets().Has(m.Set) {
					ch <- prometheus.MustNewConstMetric(
						m.Desc(), m.ValueType, value, labels...)
				}
			}
		}
	}
}
