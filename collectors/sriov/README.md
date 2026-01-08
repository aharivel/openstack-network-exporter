# SR-IOV Metrics Collector

## Overview

This collector gathers statistics from SR-IOV network interfaces (both Physical Functions and Virtual Functions) using ethtool.

## Status

**PROOF OF CONCEPT** - Not yet integrated into the main exporter.

## Files

- `POC_collector.go` - Main collector implementation (Hybrid approach)
- `POC_metrics.go` - Metric definitions
- `../../SRIOV_COLLECTOR_PROPOSAL.md` - Design proposals and rationale

## Approach

This POC implements the **Hybrid approach** (Approach 3 from the proposal):

1. **Discovery**: Uses sysfs (`/sys/class/net/`) to discover SR-IOV interfaces
   - Identifies PFs by presence of `/sys/class/net/<iface>/device/sriov_numvfs`
   - Identifies VFs by presence of `/sys/class/net/<iface>/device/physfn` symlink
   - Extracts VF number and parent PF relationship

2. **Collection**: Uses `ethtool -S` command to gather statistics
   - Executes once per interface
   - Parses text output with regex
   - Maps stat names to Prometheus metrics

## Metrics

All metrics include these labels:
- `interface` - Interface name (e.g., "enp1s0f0" or "enp1s0f0v0")
- `type` - "pf" or "vf"
- `parent_pf` - Parent PF PCI address (for VFs, empty for PFs)
- `vf_num` - VF number (for VFs, empty for PFs)

Example metrics:
```
sriov_rx_packets_total{interface="enp1s0f0",type="pf",parent_pf="",vf_num=""} 1234567
sriov_rx_packets_total{interface="enp1s0f0v0",type="vf",parent_pf="0000:01:00.0",vf_num="0"} 890123
sriov_tx_bytes_total{interface="enp1s0f0v0",type="vf",parent_pf="0000:01:00.0",vf_num="0"} 456789012
```

## Testing the POC

### 1. Check for SR-IOV interfaces on your system

```bash
# List all network interfaces
ls /sys/class/net/

# Check if an interface is a PF (has sriov_numvfs)
cat /sys/class/net/enp1s0f0/device/sriov_numvfs 2>/dev/null

# Check if an interface is a VF (has physfn symlink)
ls -l /sys/class/net/enp1s0f0v0/device/physfn 2>/dev/null

# Get ethtool stats for an interface
sudo ethtool -S enp1s0f0
```

### 2. Test ethtool parsing

```bash
# Run ethtool and check the output format
sudo ethtool -S <your_interface> | head -20
```

### 3. Discover what stats are available

```bash
# For PF
sudo ethtool -S enp1s0f0 | grep -E 'rx_|tx_' | head -20

# For VF
sudo ethtool -S enp1s0f0v0 | grep -E 'rx_|tx_' | head -20
```

## Integration Steps

To integrate this collector into the main exporter:

### 1. Move files to final location

```bash
# Remove POC_ prefix from filenames
mv POC_collector.go collector.go
mv POC_metrics.go metrics.go
```

### 2. Add to collectors registry

Edit `collectors/collectors.go`:
```go
import (
    // ... existing imports ...
    "github.com/openstack-k8s-operators/openstack-network-exporter/collectors/sriov"
)

var collectors = []lib.Collector{
    new(bridge.Collector),
    // ... existing collectors ...
    new(sriov.Collector),  // Add this line (keep alpha sorted)
    new(vswitch.Collector),
}
```

### 3. Add configuration support (optional)

Edit `config/config.go` to add SR-IOV specific configuration:

```go
type conf struct {
    // ... existing fields ...
    SriovAutoDiscover bool     `yaml:"sriov-auto-discover"`
    SriovPFFilter     []string `yaml:"sriov-pf-filter"`
    SriovIncludeVFs   bool     `yaml:"sriov-include-vfs"`
}

var c = conf{
    // ... existing defaults ...
    SriovAutoDiscover: true,
    SriovIncludeVFs:   true,
}

func SriovAutoDiscover() bool   { return c.SriovAutoDiscover }
func SriovPFFilter() []string   { return c.SriovPFFilter }
func SriovIncludeVFs() bool     { return c.SriovIncludeVFs }
```

### 4. Update configuration file

Edit `etc/openstack-network-exporter.yaml`:

```yaml
# SR-IOV interface monitoring
sriov-auto-discover: true
sriov-include-vfs: true
# Optional: filter which PFs to monitor
# sriov-pf-filter:
#   - "enp*"
#   - "eth*"
```

### 5. Build and test

```bash
# Build
make

# Test locally
sudo ./openstack-network-exporter

# Check metrics
curl http://localhost:1981/metrics | grep sriov_
```

### 6. Update documentation

Add to `METRICS.md`:

```markdown
### SR-IOV Collector

Metrics for SR-IOV network interfaces (Physical Functions and Virtual Functions).

| Metric | Type | Labels | Set | Description |
|--------|------|--------|-----|-------------|
| sriov_rx_packets_total | counter | interface, type, parent_pf, vf_num | counters | Total received packets |
| sriov_tx_packets_total | counter | interface, type, parent_pf, vf_num | counters | Total transmitted packets |
| sriov_rx_bytes_total | counter | interface, type, parent_pf, vf_num | counters | Total received bytes |
| sriov_tx_bytes_total | counter | interface, type, parent_pf, vf_num | counters | Total transmitted bytes |
| ... | ... | ... | ... | ... |
```

Add to `METRIC_SOURCES.md`:

```markdown
### SR-IOV Collector

**Data Sources:**

1. **sysfs** (`/sys/class/net/`)
   - Discovers SR-IOV interfaces (PFs and VFs)
   - Reads `/sys/class/net/<iface>/device/sriov_numvfs` to identify PFs
   - Reads `/sys/class/net/<iface>/device/physfn` to identify VFs
   - Extracts VF number from `virtfnN` symlinks

2. **ethtool command**
   - Command: `ethtool -S <interface>`
   - Requires ethtool binary and appropriate permissions
   - Gets detailed NIC statistics

**All metrics** come from parsing ethtool -S output.
```

## Customization for Your Environment

### Adding Vendor-Specific Metrics

1. Run `sudo ethtool -S <your_interface>` to see all available stats
2. Add entries to `metrics.go` for stats you want to export
3. Follow the naming pattern: `sriov_<stat_name>_total`

Example for Intel NICs with queue stats:

```go
"rx_queue_1_packets": {
    Name:        "sriov_rx_queue_packets_total",
    Description: "Packets received on queue",
    Labels:      append(commonLabels, "queue"),
    ValueType:   prometheus.CounterValue,
    Set:         config.METRICS_PERF,
},
```

### Filtering Interfaces

Modify `discoverSriovInterfaces()` to add filtering logic:

```go
func discoverSriovInterfaces() ([]InterfaceInfo, error) {
    // ... existing code ...

    // Filter based on configuration
    pfFilter := config.SriovPFFilter()
    includeVFs := config.SriovIncludeVFs()

    var filtered []InterfaceInfo
    for _, iface := range interfaces {
        // Skip VFs if not wanted
        if iface.IsVF && !includeVFs {
            continue
        }

        // Apply PF filter
        if iface.IsPF && len(pfFilter) > 0 {
            matched := false
            for _, pattern := range pfFilter {
                if matched, _ := filepath.Match(pattern, iface.Name); matched {
                    matched = true
                    break
                }
            }
            if !matched {
                continue
            }
        }

        filtered = append(filtered, iface)
    }

    return filtered, nil
}
```

## Performance Considerations

### With Many VFs

If you have many VFs (e.g., 64+), consider:

1. **Caching interface discovery**: Discovery doesn't need to run every scrape
   ```go
   var (
       interfaceCache     []InterfaceInfo
       interfaceCacheTime time.Time
       interfaceCacheTTL  = 5 * time.Minute
   )
   ```

2. **Parallel collection**: Collect stats from multiple interfaces concurrently
   ```go
   var wg sync.WaitGroup
   for _, iface := range interfaces {
       wg.Add(1)
       go func(i InterfaceInfo) {
           defer wg.Done()
           collectStats(i, ch)
       }(iface)
   }
   wg.Wait()
   ```

3. **Rate limiting**: If ethtool is slow, rate limit concurrent executions

## Alternative: Netlink Implementation

If subprocess overhead becomes an issue, consider the netlink approach:

```bash
# Add dependency
go get github.com/safchain/ethtool
```

Replace `getEthtoolStats()` with:

```go
import "github.com/safchain/ethtool"

func getEthtoolStats(iface string) (map[string]float64, error) {
    ethHandle, err := ethtool.NewEthtool()
    if err != nil {
        return nil, err
    }
    defer ethHandle.Close()

    stats, err := ethHandle.Stats(iface)
    if err != nil {
        return nil, err
    }

    result := make(map[string]float64)
    for k, v := range stats {
        result[k] = float64(v)
    }

    return result, nil
}
```

## Questions?

See `../../SRIOV_COLLECTOR_PROPOSAL.md` for detailed design discussion and alternative approaches.
