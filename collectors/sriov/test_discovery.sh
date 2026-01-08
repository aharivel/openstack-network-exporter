#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Test script to discover SR-IOV interfaces and check ethtool stats

set -e

echo "=== SR-IOV Interface Discovery Test ==="
echo

# Colors for output
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m' # No Color

# Function to check if interface is a PF
is_pf() {
    local iface=$1
    if [ -f "/sys/class/net/$iface/device/sriov_numvfs" ]; then
        return 0
    fi
    return 1
}

# Function to check if interface is a VF
is_vf() {
    local iface=$1
    if [ -L "/sys/class/net/$iface/device/physfn" ]; then
        return 0
    fi
    return 1
}

# Function to get number of VFs for a PF
get_num_vfs() {
    local iface=$1
    if [ -f "/sys/class/net/$iface/device/sriov_numvfs" ]; then
        cat "/sys/class/net/$iface/device/sriov_numvfs"
    else
        echo "0"
    fi
}

# Function to get VF number
get_vf_number() {
    local iface=$1
    local device_path="/sys/class/net/$iface/device"

    if [ ! -L "$device_path/physfn" ]; then
        echo "-1"
        return
    fi

    # Get parent PF device path
    local pf_path=$(readlink -f "$device_path/physfn")

    # Find which virtfnN points to this device
    local my_device=$(readlink -f "$device_path")

    for virtfn in "$pf_path"/virtfn*; do
        if [ -L "$virtfn" ]; then
            local target=$(readlink -f "$virtfn")
            if [ "$target" = "$my_device" ]; then
                basename "$virtfn" | sed 's/virtfn//'
                return
            fi
        fi
    done

    echo "-1"
}

# Discover all network interfaces
echo "Discovering network interfaces..."
echo

pf_count=0
vf_count=0

for iface in /sys/class/net/*; do
    iface=$(basename "$iface")

    # Skip loopback and common virtual interfaces
    if [[ "$iface" =~ ^(lo|docker|virbr|veth) ]]; then
        continue
    fi

    if is_pf "$iface"; then
        num_vfs=$(get_num_vfs "$iface")
        echo -e "${GREEN}[PF]${NC} $iface"
        echo "     Number of VFs: $num_vfs"

        # Try to get driver info
        if [ -f "/sys/class/net/$iface/device/driver/module" ]; then
            driver=$(readlink "/sys/class/net/$iface/device/driver/module" | xargs basename)
            echo "     Driver: $driver"
        fi

        # Try to get PCI address
        if [ -L "/sys/class/net/$iface/device" ]; then
            pci=$(readlink "/sys/class/net/$iface/device" | xargs basename)
            echo "     PCI: $pci"
        fi

        pf_count=$((pf_count + 1))
        echo

    elif is_vf "$iface"; then
        vf_num=$(get_vf_number "$iface")
        pf_device=$(readlink -f "/sys/class/net/$iface/device/physfn" | xargs basename)

        echo -e "${YELLOW}[VF]${NC} $iface"
        echo "     VF Number: $vf_num"
        echo "     Parent PF: $pf_device"

        vf_count=$((vf_count + 1))
        echo
    fi
done

echo "=== Summary ==="
echo "Physical Functions (PFs): $pf_count"
echo "Virtual Functions (VFs): $vf_count"
echo

# Test ethtool stats parsing
echo "=== Testing ethtool Statistics ==="
echo

# Find first SR-IOV interface to test
test_iface=""
for iface in /sys/class/net/*; do
    iface=$(basename "$iface")
    if is_pf "$iface" || is_vf "$iface"; then
        test_iface="$iface"
        break
    fi
done

if [ -z "$test_iface" ]; then
    echo -e "${RED}No SR-IOV interfaces found to test!${NC}"
    echo
    echo "To enable SR-IOV on a NIC:"
    echo "  1. Check if your NIC supports SR-IOV: lspci -vv | grep -i sriov"
    echo "  2. Enable VFs: echo 4 > /sys/class/net/<interface>/device/sriov_numvfs"
    echo "  3. Re-run this script"
    exit 0
fi

echo "Testing ethtool on interface: $test_iface"
echo

if ! command -v ethtool &> /dev/null; then
    echo -e "${RED}ethtool command not found!${NC}"
    echo "Please install: apt-get install ethtool  OR  yum install ethtool"
    exit 1
fi

# Check if we can run ethtool
if ! ethtool -S "$test_iface" &> /dev/null; then
    echo -e "${RED}Cannot run ethtool -S on $test_iface${NC}"
    echo "You may need to run this script with sudo"
    echo "Try: sudo $0"
    exit 1
fi

# Get and display stats
echo "Available statistics (first 30):"
ethtool -S "$test_iface" | head -30
echo
echo "..."
echo

# Count total stats
stat_count=$(ethtool -S "$test_iface" | grep -E '^\s+\w+:' | wc -l)
echo "Total statistics available: $stat_count"
echo

# Show some interesting stats if available
echo "=== Sample Statistics ==="
for stat in rx_packets tx_packets rx_bytes tx_bytes rx_errors tx_errors rx_dropped tx_dropped; do
    value=$(ethtool -S "$test_iface" | grep -E "^\s+${stat}:" | head -1 | awk '{print $2}')
    if [ -n "$value" ]; then
        printf "%-20s: %s\n" "$stat" "$value"
    fi
done
echo

# Check for vendor-specific stats
echo "=== Vendor-Specific Statistics Detected ==="
if ethtool -S "$test_iface" | grep -q "vport"; then
    echo -e "${GREEN}✓${NC} Mellanox/NVIDIA stats detected (vport_* counters)"
fi
if ethtool -S "$test_iface" | grep -q "queue_"; then
    echo -e "${GREEN}✓${NC} Queue-specific stats detected"
fi
if ethtool -S "$test_iface" | grep -q "prio[0-9]"; then
    echo -e "${GREEN}✓${NC} Priority queue stats detected"
fi
echo

echo "=== Integration Checklist ==="
echo
echo "To add these statistics to the exporter:"
echo "  1. Review the stat names above"
echo "  2. Add desired stats to collectors/sriov/POC_metrics.go"
echo "  3. Follow the pattern:"
echo '     "stat_name": {'
echo '         Name:        "sriov_stat_name_total",'
echo '         Description: "Description of the stat",'
echo '         Labels:      commonLabels,'
echo '         ValueType:   prometheus.CounterValue,'
echo '         Set:         config.METRICS_COUNTERS,'
echo '     },'
echo
echo "=== Recommended Next Steps ==="
echo "  1. Check which stats are most useful for your monitoring"
echo "  2. Add them to POC_metrics.go"
echo "  3. Follow integration steps in collectors/sriov/README.md"
echo "  4. Test with: make && sudo ./openstack-network-exporter"
echo
