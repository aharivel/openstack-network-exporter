// SPDX-License-Identifier: Apache-2.0
// Proof of Concept - SR-IOV Metrics Collector
// This is a POC implementation showing the hybrid approach

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
	ParentPF string // PCI address or name of parent PF (for VFs)
	VFNum    int    // VF number (for VFs, -1 for PFs)
	NumVFs   int    // Number of VFs (for PFs, 0 for VFs)
}

// discoverSriovInterfaces discovers all SR-IOV interfaces using sysfs
func discoverSriovInterfaces() ([]InterfaceInfo, error) {
	var interfaces []InterfaceInfo

	// Read all network interfaces from /sys/class/net
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

			// Get VF number from virtfnN symlink in parent PF
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

// readIntFromFile reads an integer value from a file
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

// getVFNumber returns the VF number and parent PF path
func getVFNumber(devicePath string) (int, string) {
	// Read the physfn symlink to get parent PF
	physfnPath := filepath.Join(devicePath, "physfn")
	pfDevice, err := os.Readlink(physfnPath)
	if err != nil {
		return -1, ""
	}

	// Get absolute path to PF device
	pfDevicePath := filepath.Join(devicePath, pfDevice)

	// Look for virtfnN symlinks in the PF device directory
	entries, err := os.ReadDir(pfDevicePath)
	if err != nil {
		return -1, ""
	}

	// Get our device path
	myDevice, err := filepath.EvalSymlinks(devicePath)
	if err != nil {
		return -1, ""
	}

	// Find which virtfnN points to us
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

// ethtool -S output format:
// NIC statistics:
//      rx_packets: 2838909
//      rx_bytes: 3068300340
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

// buildLabels creates the label values for a metric
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
		info.Name,     // interface
		ifType,        // type
		info.ParentPF, // parent_pf (empty for PFs)
		vfNum,         // vf_num (empty for PFs)
	}
}

func (Collector) Collect(ch chan<- prometheus.Metric) {
	// Discover SR-IOV interfaces
	interfaces, err := discoverSriovInterfaces()
	if err != nil {
		log.Errf("failed to discover SR-IOV interfaces: %s", err)
		return
	}

	log.Debugf("discovered %d SR-IOV interfaces", len(interfaces))

	// Collect stats for each interface
	for _, iface := range interfaces {
		// Get ethtool statistics
		stats, err := getEthtoolStats(iface.Name)
		if err != nil {
			log.Debugf("ethtool -S %s: %s", iface.Name, err)
			continue
		}

		log.Debugf("collected %d stats for %s (PF=%v, VF=%v)",
			len(stats), iface.Name, iface.IsPF, iface.IsVF)

		// Build labels once for this interface
		labels := buildLabels(iface)

		// Export metrics
		for statName, value := range stats {
			// Look up metric definition
			if m, ok := metrics[statName]; ok {
				if config.MetricSets().Has(m.Set) {
					ch <- prometheus.MustNewConstMetric(
						m.Desc(), m.ValueType, value, labels...)
				}
			} else {
				// Optionally log unknown stats for discovery
				log.Debugf("unknown stat: %s=%f for %s", statName, value, iface.Name)
			}
		}
	}
}
