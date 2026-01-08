// SPDX-License-Identifier: Apache-2.0
// Proof of Concept - SR-IOV Metrics Definitions

package sriov

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

// Labels for SR-IOV metrics
var commonLabels = []string{
	"interface",  // Interface name (e.g., "enp1s0f0" or "enp1s0f0v0")
	"type",       // "pf" or "vf"
	"parent_pf",  // Parent PF PCI address (for VFs) or empty (for PFs)
	"vf_num",     // VF number as string (for VFs) or empty (for PFs)
}

// Metrics map - maps ethtool stat names to Prometheus metrics
// This should be extended based on the actual stats available from your NICs
var metrics = map[string]lib.Metric{
	// Basic packet/byte counters
	"rx_packets": {
		Name:        "sriov_rx_packets_total",
		Description: "Total number of received packets on SR-IOV interface",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_packets": {
		Name:        "sriov_tx_packets_total",
		Description: "Total number of transmitted packets on SR-IOV interface",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"rx_bytes": {
		Name:        "sriov_rx_bytes_total",
		Description: "Total number of received bytes on SR-IOV interface",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_bytes": {
		Name:        "sriov_tx_bytes_total",
		Description: "Total number of transmitted bytes on SR-IOV interface",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},

	// Error counters
	"rx_errors": {
		Name:        "sriov_rx_errors_total",
		Description: "Total number of receive errors on SR-IOV interface",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"tx_errors": {
		Name:        "sriov_tx_errors_total",
		Description: "Total number of transmit errors on SR-IOV interface",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rx_dropped": {
		Name:        "sriov_rx_dropped_total",
		Description: "Total number of received packets dropped on SR-IOV interface",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"tx_dropped": {
		Name:        "sriov_tx_dropped_total",
		Description: "Total number of transmitted packets dropped on SR-IOV interface",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},

	// Mellanox-specific stats (common in OpenStack deployments)
	"rx_vport_unicast_packets": {
		Name:        "sriov_rx_vport_unicast_packets_total",
		Description: "Unicast packets received by vport (Mellanox)",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"tx_vport_unicast_packets": {
		Name:        "sriov_tx_vport_unicast_packets_total",
		Description: "Unicast packets transmitted by vport (Mellanox)",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"rx_vport_unicast_bytes": {
		Name:        "sriov_rx_vport_unicast_bytes_total",
		Description: "Unicast bytes received by vport (Mellanox)",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"tx_vport_unicast_bytes": {
		Name:        "sriov_tx_vport_unicast_bytes_total",
		Description: "Unicast bytes transmitted by vport (Mellanox)",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"rx_vport_multicast_packets": {
		Name:        "sriov_rx_vport_multicast_packets_total",
		Description: "Multicast packets received by vport (Mellanox)",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"tx_vport_multicast_packets": {
		Name:        "sriov_tx_vport_multicast_packets_total",
		Description: "Multicast packets transmitted by vport (Mellanox)",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"rx_vport_broadcast_packets": {
		Name:        "sriov_rx_vport_broadcast_packets_total",
		Description: "Broadcast packets received by vport (Mellanox)",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"tx_vport_broadcast_packets": {
		Name:        "sriov_tx_vport_broadcast_packets_total",
		Description: "Broadcast packets transmitted by vport (Mellanox)",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},

	// Additional error counters
	"rx_over_errors": {
		Name:        "sriov_rx_over_errors_total",
		Description: "Receiver FIFO overflow errors",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rx_crc_errors": {
		Name:        "sriov_rx_crc_errors_total",
		Description: "Received packets with CRC errors",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rx_frame_errors": {
		Name:        "sriov_rx_frame_errors_total",
		Description: "Received packets with frame errors",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"tx_carrier_errors": {
		Name:        "sriov_tx_carrier_errors_total",
		Description: "Transmit carrier errors",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},

	// Queue-specific stats (example - actual names vary by driver)
	"rx_queue_0_packets": {
		Name:        "sriov_rx_queue_packets_total",
		Description: "Packets received on queue 0",
		Labels:      append(commonLabels, "queue"),
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_DEBUG,
	},

	// TODO: Add more metrics based on your specific NIC vendor and requirements
	// Examples for Intel NICs:
	// - rx_queue_N_packets
	// - tx_queue_N_packets
	// - rx_queue_N_bytes
	// - tx_queue_N_bytes
	//
	// Examples for Mellanox NICs:
	// - rx_prio0_packets (per-priority counters)
	// - tx_prio0_packets
	// - rx_out_of_buffer (buffer exhaustion)
	//
	// You can discover these by running: ethtool -S <interface> on your hardware
}

// TODO: Implement vendor-specific metric normalization
// Different NIC vendors use different stat names for the same concepts
// We might want to normalize these to common names

// vendorStatMap could map vendor-specific names to common names
// var vendorStatMap = map[string]string{
//     // Mellanox -> common
//     "rx_vport_unicast_packets": "rx_unicast_packets",
//     "tx_vport_unicast_packets": "tx_unicast_packets",
//     // Intel -> common
//     "rx_queue_0_packets": "rx_queue_packets",
//     // etc.
// }
