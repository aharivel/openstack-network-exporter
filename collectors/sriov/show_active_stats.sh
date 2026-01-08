#!/bin/bash
# SPDX-License-Identifier: Apache-2.0
# Show active (non-zero) ethtool statistics for SR-IOV interfaces

if [ $# -eq 0 ]; then
    echo "Usage: $0 <interface_name>"
    echo
    echo "Examples:"
    echo "  $0 eno12399      # Show stats for PF"
    echo "  $0 eno12399v0    # Show stats for VF 0"
    echo "  $0 eno12399v1    # Show stats for VF 1"
    exit 1
fi

IFACE=$1

if [ ! -d "/sys/class/net/$IFACE" ]; then
    echo "Error: Interface $IFACE not found"
    exit 1
fi

echo "=== Active Statistics for $IFACE ==="
echo

# Check if it's a PF or VF
if [ -f "/sys/class/net/$IFACE/device/sriov_numvfs" ]; then
    NUM_VFS=$(cat "/sys/class/net/$IFACE/device/sriov_numvfs")
    echo "Type: Physical Function (PF)"
    echo "Number of VFs: $NUM_VFS"
elif [ -L "/sys/class/net/$IFACE/device/physfn" ]; then
    PARENT=$(readlink -f "/sys/class/net/$IFACE/device/physfn" | xargs basename)
    echo "Type: Virtual Function (VF)"
    echo "Parent PF: $PARENT"
else
    echo "Type: Regular interface (not SR-IOV)"
fi

echo
echo "=== Non-Zero Statistics ==="
echo

# Get ethtool stats and filter out zeros
sudo ethtool -S "$IFACE" 2>/dev/null | grep -v "NIC statistics:" | grep -v ": 0$" | sort

echo
echo "=== Statistics by Category ==="
echo

# Traffic counters
echo "## Traffic Counters:"
sudo ethtool -S "$IFACE" 2>/dev/null | grep -E "^\s+(rx|tx)_(unicast|multicast|broadcast|bytes):" | grep -v ": 0$"
echo

# Error counters
echo "## Error/Drop Counters:"
sudo ethtool -S "$IFACE" 2>/dev/null | grep -E "^\s+(rx|tx).*(error|drop|fail):" | grep -v ": 0$"
echo

# Queue stats
echo "## Queue Statistics:"
sudo ethtool -S "$IFACE" 2>/dev/null | grep -E "^\s+(rx|tx)_queue_" | grep -v ": 0$" | head -20
echo "(showing first 20 queue stats...)"
echo

# Count stats
TOTAL_STATS=$(sudo ethtool -S "$IFACE" 2>/dev/null | grep -E "^\s+\w+:" | wc -l)
ACTIVE_STATS=$(sudo ethtool -S "$IFACE" 2>/dev/null | grep -E "^\s+\w+:" | grep -v ": 0$" | wc -l)

echo "=== Summary ==="
echo "Total statistics: $TOTAL_STATS"
echo "Active (non-zero): $ACTIVE_STATS"
echo

echo "=== Recommendation ==="
echo "Focus on these categories for metrics:"
echo "  1. Traffic counters (unicast/multicast/broadcast/bytes) - Already in metrics_intel.go ✓"
echo "  2. Error/drop counters - Already in metrics_intel.go ✓"
echo "  3. Queue stats (if needed for your use case) - Handled dynamically in collector_enhanced.go ✓"
echo
echo "To add more stats, edit: collectors/sriov/metrics_intel.go"
