# SR-IOV Collector - Integration Guide for Your System

## ✅ What We Know About Your Setup

Based on your test output:
- **NIC Type**: Intel (based on queue stat naming)
- **1 Physical Function**: `eno12399` (PCI: 0000:8a:00.0)
- **2 Virtual Functions**: `eno12399v0`, `eno12399v1`
- **181 ethtool statistics** available
- **Queue-specific stats**: Yes (tx_queue_N_*, likely rx_queue_N_* too)

## 📁 Files Created for Your Environment

I've created customized files for your Intel NIC:

1. **`metrics_intel.go`** - Metric definitions tailored to your NIC
   - Basic traffic counters (unicast/multicast/broadcast/bytes)
   - Error counters (dropped, alloc_fail, tx_errors, etc.)
   - Intel-specific stats (tx_linearize, tx_busy, tx_restart)

2. **`collector_enhanced.go`** - Enhanced collector with queue metrics support
   - Auto-discovers SR-IOV interfaces
   - Handles both regular metrics AND per-queue metrics
   - Dynamically creates queue metrics (tx_queue_5_packets, etc.)

3. **`show_active_stats.sh`** - Helper to see active stats on your interfaces

## 🚀 Quick Integration Steps

### Step 1: Choose Your Files

You have two options:

**Option A: Use Intel-specific files (Recommended for you)**
```bash
cd collectors/sriov/
mv collector_enhanced.go collector.go
mv metrics_intel.go metrics.go
rm POC_*.go  # Remove the generic POC files
```

**Option B: Start with POC and customize**
```bash
cd collectors/sriov/
mv POC_collector.go collector.go
mv POC_metrics.go metrics.go
# Then edit metrics.go to add Intel-specific stats
```

### Step 2: See What Stats Are Active

```bash
# Check what stats are actually non-zero on your interfaces
sudo ./collectors/sriov/show_active_stats.sh eno12399
sudo ./collectors/sriov/show_active_stats.sh eno12399v0
sudo ./collectors/sriov/show_active_stats.sh eno12399v1
```

This will show you which stats are worth monitoring.

### Step 3: Add to Main Collector Registry

Edit `collectors/collectors.go`:

```go
package collectors

import (
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/bridge"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/coverage"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/datapath"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/iface"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/lib"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/memory"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/ovn"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/ovnnorthd"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/ovsdbserver"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/pmd_perf"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/pmd_rxq"
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/sriov"  // ADD THIS
	"github.com/openstack-k8s-operators/openstack-network-exporter/collectors/vswitch"
)

// All supported collectors. Please keep alpha sorted.
var collectors = []lib.Collector{
	new(bridge.Collector),
	new(coverage.Collector),
	new(datapath.Collector),
	new(iface.Collector),
	new(memory.Collector),
	new(ovnnorthd.Collector),
	new(ovn.Collector),
	new(ovsdbserver.Collector),
	new(pmd_perf.Collector),
	new(pmd_rxq.Collector),
	new(sriov.Collector),   // ADD THIS (keep alphabetically sorted)
	new(vswitch.Collector),
}

func Collectors() []lib.Collector {
	return collectors
}
```

### Step 4: Build and Test

```bash
# Build
make

# Run with sudo (needed for ethtool)
sudo ./openstack-network-exporter

# In another terminal, check metrics
curl http://localhost:1981/metrics | grep sriov_
```

### Step 5: Verify Metrics

You should see metrics like:

```
# HELP sriov_rx_bytes_total Total number of received bytes on SR-IOV interface
# TYPE sriov_rx_bytes_total counter
sriov_rx_bytes_total{interface="eno12399",parent_pf="",type="pf",vf_num=""} 1954942
sriov_rx_bytes_total{interface="eno12399v0",parent_pf="0000:8a:00.0",type="vf",vf_num="0"} 0
sriov_rx_bytes_total{interface="eno12399v1",parent_pf="0000:8a:00.0",type="vf",vf_num="1"} 0

# HELP sriov_rx_multicast_packets_total Total number of received multicast packets
# TYPE sriov_rx_multicast_packets_total counter
sriov_rx_multicast_packets_total{interface="eno12399",parent_pf="",type="pf",vf_num=""} 3639
sriov_rx_multicast_packets_total{interface="eno12399v0",parent_pf="0000:8a:00.0",type="vf",vf_num="0"} 0
sriov_rx_multicast_packets_total{interface="eno12399v1",parent_pf="0000:8a:00.0",type="vf",vf_num="1"} 0

# Queue metrics (if collector_enhanced.go is used)
# HELP sriov_tx_queue_packets_total packets tx on queue
# TYPE sriov_tx_queue_packets_total counter
sriov_tx_queue_packets_total{interface="eno12399",parent_pf="",queue="5",type="pf",vf_num=""} 2
sriov_tx_queue_packets_total{interface="eno12399",parent_pf="",queue="0",type="pf",vf_num=""} 0
```

## 🎯 Example Prometheus Queries

Once integrated, you can query:

```promql
# Total RX bytes across all VFs on a PF
sum(rate(sriov_rx_bytes_total{parent_pf="0000:8a:00.0"}[5m])) by (parent_pf)

# Errors per VF
sum(rate(sriov_rx_dropped_total{type="vf"}[5m])) by (interface, vf_num)

# Compare PF vs all VFs traffic
sum(rate(sriov_tx_bytes_total{type="pf"}[5m])) by (interface)
sum(rate(sriov_tx_bytes_total{type="vf"}[5m])) by (parent_pf)

# Queue distribution on PF (with enhanced collector)
sum(rate(sriov_tx_queue_packets_total{interface="eno12399"}[5m])) by (queue)
```

## 📊 Grafana Dashboard Ideas

Panels you could create:

1. **VF Traffic Overview**
   - Graph showing RX/TX bytes for all VFs
   - Legend: VF number + interface name

2. **Error Rate by VF**
   - Heatmap or graph of rx_dropped, tx_errors per VF
   - Alert if errors > threshold

3. **PF vs VF Traffic Ratio**
   - Pie chart or stat panel comparing PF traffic to sum of VF traffic
   - Helps identify traffic not going through VFs

4. **Queue Utilization** (if using enhanced collector)
   - Stacked graph showing packets/bytes per queue
   - Identify queue imbalance

5. **Allocation Failures**
   - Graph of rx_alloc_fail, rx_pg_alloc_fail over time
   - Critical for performance debugging

## 🔧 Customization

### Adding More Metrics

1. Run this to see all active stats:
   ```bash
   sudo ./collectors/sriov/show_active_stats.sh eno12399
   ```

2. Find stats you want to monitor (non-zero ones)

3. Add them to `metrics_intel.go`:
   ```go
   "your_stat_name": {
       Name:        "sriov_your_stat_name_total",
       Description: "Description of what this measures",
       Labels:      commonLabels,
       ValueType:   prometheus.CounterValue,  // or GaugeValue
       Set:         config.METRICS_COUNTERS,  // or ERRORS, PERF, DEBUG
   },
   ```

### Filtering Interfaces

If you want to monitor only specific interfaces, modify `discoverSriovInterfaces()` in `collector.go` to add filtering.

## ⚠️ Known Considerations

### 1. Permissions
- Requires `ethtool` command
- Needs sudo/root to run ethtool on interfaces
- Deploy with appropriate capabilities

### 2. Performance
- With 2 VFs, performance is fine
- If you scale to 64+ VFs, consider:
  - Caching interface discovery
  - Parallel stat collection
  - Reducing scrape frequency

### 3. Queue Metrics Cardinality
- Queue metrics can create many time series
- Example: 8 queues × 3 interfaces × 2 directions = 48 series
- Consider if you need queue-level granularity
- If not, remove queue metric handling from collector

## 🐛 Troubleshooting

### "no stats available"
- Check: `sudo ethtool -S eno12399`
- Ensure driver supports ethtool stats
- Verify interface is UP

### "permission denied"
- Run exporter with sudo or appropriate capabilities
- Check: `sudo -u <user> ethtool -S eno12399`

### "interface not found"
- Verify interface exists: `ip link show eno12399`
- Check sysfs: `ls /sys/class/net/`

### Metrics not appearing
- Check collector is registered in `collectors/collectors.go`
- Verify metric sets are enabled in config
- Check logs: `sudo ./openstack-network-exporter` (look for sriov debug messages)

## 📝 Next Steps

1. ✅ Choose which file version to use (enhanced or POC)
2. ✅ Run `show_active_stats.sh` on all interfaces
3. ✅ Decide which additional stats to add
4. ✅ Integrate into main collector registry
5. ✅ Build and test
6. ✅ Create Grafana dashboards
7. ✅ Set up alerts for error conditions

## 📚 Documentation Updates

After integration, update:

1. **METRICS.md** - Add SR-IOV collector section with all metrics
2. **METRIC_SOURCES.md** - Document that SR-IOV uses sysfs + ethtool
3. **README.md** - Mention SR-IOV collector in features

Example entry for METRICS.md:

```markdown
### SR-IOV Collector

Metrics for SR-IOV network interfaces (Physical Functions and Virtual Functions).

**Data Sources**: sysfs for discovery, ethtool for statistics

| Metric | Type | Labels | Set | Description |
|--------|------|--------|-----|-------------|
| sriov_rx_bytes_total | counter | interface, type, parent_pf, vf_num | counters | Total received bytes |
| sriov_tx_bytes_total | counter | interface, type, parent_pf, vf_num | counters | Total transmitted bytes |
| sriov_rx_unicast_packets_total | counter | interface, type, parent_pf, vf_num | counters | Unicast packets received |
| ... | ... | ... | ... | ... |
```

Good luck with the integration! 🚀
