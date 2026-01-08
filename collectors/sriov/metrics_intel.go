// SPDX-License-Identifier: Apache-2.0
// SR-IOV Metrics for Intel NICs
// Based on ethtool stats from your eno12399 interface

package sriov

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/config"
	"github.com/prometheus/client_golang/prometheus"
)

// Labels for SR-IOV metrics
var commonLabels = []string{
	"interface",  // Interface name (e.g., "eno12399" or "eno12399v0")
	"type",       // "pf" or "vf"
	"parent_pf",  // Parent PF PCI address (for VFs) or empty (for PFs)
	"vf_num",     // VF number as string (for VFs) or empty (for PFs)
}

// Metrics based on your Intel NIC's ethtool output
var metrics = map[string]lib.Metric{
	// ===== BASIC TRAFFIC COUNTERS =====
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

	// ===== UNICAST TRAFFIC =====
	"rx_unicast": {
		Name:        "sriov_rx_unicast_packets_total",
		Description: "Total number of received unicast packets",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_unicast": {
		Name:        "sriov_tx_unicast_packets_total",
		Description: "Total number of transmitted unicast packets",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},

	// ===== MULTICAST TRAFFIC =====
	"rx_multicast": {
		Name:        "sriov_rx_multicast_packets_total",
		Description: "Total number of received multicast packets",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_multicast": {
		Name:        "sriov_tx_multicast_packets_total",
		Description: "Total number of transmitted multicast packets",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},

	// ===== BROADCAST TRAFFIC =====
	"rx_broadcast": {
		Name:        "sriov_rx_broadcast_packets_total",
		Description: "Total number of received broadcast packets",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},
	"tx_broadcast": {
		Name:        "sriov_tx_broadcast_packets_total",
		Description: "Total number of transmitted broadcast packets",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_COUNTERS,
	},

	// ===== ERROR COUNTERS =====
	"rx_dropped": {
		Name:        "sriov_rx_dropped_total",
		Description: "Total number of received packets dropped",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"tx_errors": {
		Name:        "sriov_tx_errors_total",
		Description: "Total number of transmit errors",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rx_alloc_fail": {
		Name:        "sriov_rx_alloc_fail_total",
		Description: "Total number of RX buffer allocation failures",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rx_pg_alloc_fail": {
		Name:        "sriov_rx_pg_alloc_fail_total",
		Description: "Total number of RX page allocation failures",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},
	"rx_unknown_protocol": {
		Name:        "sriov_rx_unknown_protocol_total",
		Description: "Total number of packets with unknown protocol",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_ERRORS,
	},

	// ===== TX SPECIFIC ERRORS =====
	"tx_linearize": {
		Name:        "sriov_tx_linearize_total",
		Description: "Number of times TX linearization was needed",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"tx_busy": {
		Name:        "sriov_tx_busy_total",
		Description: "Number of times TX queue was busy",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},
	"tx_restart": {
		Name:        "sriov_tx_restart_total",
		Description: "Number of TX queue restarts",
		Labels:      commonLabels,
		ValueType:   prometheus.CounterValue,
		Set:         config.METRICS_PERF,
	},

	// Note: Queue-specific metrics (tx_queue_N_*) are handled dynamically
	// in the collector code below since they have a queue number dimension
}

// queueMetrics defines metrics that have a queue dimension
// These will be exported with an additional "queue" label
var queueMetrics = []struct {
	prefix      string
	metricName  string
	description string
	valueType   prometheus.ValueType
	set         config.MetricSet
}{
	{
		prefix:      "tx_queue_",
		metricName:  "sriov_tx_queue_packets_total",
		description: "Packets transmitted on TX queue",
		valueType:   prometheus.CounterValue,
		set:         config.METRICS_PERF,
	},
	{
		prefix:      "tx_queue_",
		metricName:  "sriov_tx_queue_bytes_total",
		description: "Bytes transmitted on TX queue",
		valueType:   prometheus.CounterValue,
		set:         config.METRICS_PERF,
	},
	{
		prefix:      "rx_queue_",
		metricName:  "sriov_rx_queue_packets_total",
		description: "Packets received on RX queue",
		valueType:   prometheus.CounterValue,
		set:         config.METRICS_PERF,
	},
	{
		prefix:      "rx_queue_",
		metricName:  "sriov_rx_queue_bytes_total",
		description: "Bytes received on RX queue",
		valueType:   prometheus.CounterValue,
		set:         config.METRICS_PERF,
	},
}

// TODO: Add more metrics based on the full ethtool -S output
// Run: sudo ethtool -S eno12399 | grep -v ": 0$" | head -50
// to see which stats are actually non-zero and might be useful
//
// Other stats you might want to add:
// - rx_csum_offload_errors
// - rx_length_errors
// - rx_crc_errors
// - rx_missed_errors
// - And any other vendor-specific counters you find useful
