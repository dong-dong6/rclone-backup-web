# Performance Optimization Guide

## Overview

The Rclone Backup Agent is designed to be lightweight and efficient. This guide provides recommendations for optimizing performance in various deployment scenarios.

## Resource Requirements

### Minimum Requirements
- **CPU**: 1 core (any x86_64 or ARM processor)
- **RAM**: 256 MB
- **Disk**: 100 MB for agent + space for temporary files
- **Network**: 1 Mbps (varies based on backup size)

### Recommended Requirements
- **CPU**: 2+ cores
- **RAM**: 1 GB
- **Disk**: 1 GB for agent + cache
- **Network**: 10+ Mbps

## Performance Tuning

### 1. Concurrent Task Execution

Adjust the number of concurrent tasks based on your system resources:

```json
{
  "max_concurrent": 3,  // Increase for more parallelism
  "worker_threads": 4    // Worker threads per task
}
```

**Guidelines**:
- **Single-core systems**: `max_concurrent: 1`
- **Multi-core systems**: `max_concurrent: CPU_cores - 1`
- **Memory-constrained**: Reduce concurrent tasks

### 2. Rclone Transfer Optimization

Configure rclone parameters for optimal transfer speeds:

```json
{
  "rclone_args": [
    "--transfers", "4",        // Parallel file transfers
    "--checkers", "8",         // Parallel file checkers
    "--buffer-size", "32M",    // In-memory buffer per transfer
    "--drive-chunk-size", "64M", // Upload chunk size for Google Drive
    "--s3-chunk-size", "64M",  // Upload chunk size for S3
    "--s3-upload-concurrency", "4", // S3 multipart upload threads
    "--multi-thread-cutoff", "256M", // Use multi-thread for large files
    "--multi-thread-streams", "4"    // Number of streams for multi-thread
  ]
}
```

### 3. Network Optimization

#### Bandwidth Limiting
Prevent the agent from saturating your connection:

```json
{
  "rclone_args": [
    "--bwlimit", "10M",        // Limit to 10 MB/s
    "--bwlimit-file", "1M",    // Per-file limit
    "--tpslimit", "10",        // Transactions per second
    "--tpslimit-burst", "20"   // Burst transactions
  ]
}
```

#### Connection Pooling
```json
{
  "rclone_args": [
    "--contimeout", "60s",     // Connection timeout
    "--timeout", "300s",       // IO timeout
    "--retries", "3",          // Retry failed operations
    "--low-level-retries", "10" // Low-level retries
  ]
}
```

### 4. Cache Configuration

Enable caching for better performance with cloud storage:

```json
{
  "cache_dir": "/opt/rclone-agent/cache",
  "cache_size": "1G",
  "rclone_args": [
    "--cache-dir", "/opt/rclone-agent/cache",
    "--vfs-cache-mode", "writes",
    "--vfs-cache-max-size", "1G",
    "--vfs-cache-max-age", "24h"
  ]
}
```

### 5. Memory Management

Control memory usage:

```json
{
  "rclone_args": [
    "--use-mmap",              // Use memory-mapped files
    "--buffer-size", "16M",    // Reduce for low memory
    "--checkers", "4",         // Reduce parallel operations
    "--transfers", "2"         // Reduce concurrent transfers
  ]
}
```

## Monitoring Performance

### 1. Built-in Metrics

Enable Prometheus metrics:

```json
{
  "enable_metrics": true,
  "metrics_port": 9091
}
```

Access metrics at: `http://agent-host:9091/metrics`

### 2. Key Metrics to Monitor

- **Task Execution Time**: `agent_task_duration_seconds`
- **Transfer Speed**: `agent_transfer_bytes_per_second`
- **Memory Usage**: `agent_memory_usage_bytes`
- **CPU Usage**: `agent_cpu_usage_percent`
- **Active Tasks**: `agent_active_tasks`
- **Failed Tasks**: `agent_failed_tasks_total`

### 3. Performance Logging

Enable detailed performance logging:

```json
{
  "log_level": "debug",
  "performance_logging": true,
  "rclone_args": [
    "--stats", "10s",
    "--stats-log-level", "INFO",
    "--log-level", "INFO"
  ]
}
```

## Optimization Strategies

### For Large Files (>1GB)

```json
{
  "rclone_args": [
    "--s3-chunk-size", "128M",
    "--drive-chunk-size", "128M",
    "--multi-thread-cutoff", "256M",
    "--multi-thread-streams", "8",
    "--buffer-size", "64M"
  ]
}
```

### For Many Small Files (<10MB)

```json
{
  "rclone_args": [
    "--transfers", "8",
    "--checkers", "16",
    "--fast-list",
    "--no-check-dest",
    "--ignore-checksum"
  ]
}
```

### For Limited Bandwidth

```json
{
  "rclone_args": [
    "--bwlimit", "1M:2M",      // 1MB/s normal, 2MB/s burst
    "--transfers", "1",
    "--checkers", "2",
    "--contimeout", "120s",
    "--timeout", "600s"
  ]
}
```

### For High Latency Networks

```json
{
  "rclone_args": [
    "--contimeout", "180s",
    "--timeout", "600s",
    "--retries", "5",
    "--low-level-retries", "20",
    "--buffer-size", "128M",
    "--checkers", "2"
  ]
}
```

## Storage Backend Optimization

### Amazon S3

```json
{
  "rclone_args": [
    "--s3-chunk-size", "64M",
    "--s3-upload-concurrency", "4",
    "--s3-storage-class", "STANDARD_IA",
    "--s3-acl", "private"
  ]
}
```

### Google Drive

```json
{
  "rclone_args": [
    "--drive-chunk-size", "64M",
    "--drive-acknowledge-abuse",
    "--drive-skip-gdocs",
    "--drive-server-side-across-configs"
  ]
}
```

### Azure Blob Storage

```json
{
  "rclone_args": [
    "--azureblob-chunk-size", "64M",
    "--azureblob-upload-concurrency", "4",
    "--azureblob-list-chunk", "5000"
  ]
}
```

### Local/NFS Storage

```json
{
  "rclone_args": [
    "--local-no-check-updated",
    "--local-no-unicode-normalization",
    "--one-file-system",
    "--copy-links"
  ]
}
```

## System-Level Optimization

### Linux Kernel Tuning

Add to `/etc/sysctl.conf`:

```bash
# Network optimization
net.core.rmem_max = 134217728
net.core.wmem_max = 134217728
net.ipv4.tcp_rmem = 4096 87380 134217728
net.ipv4.tcp_wmem = 4096 65536 134217728
net.ipv4.tcp_congestion_control = bbr
net.core.netdev_max_backlog = 5000

# File system optimization
fs.file-max = 2097152
fs.inotify.max_user_watches = 524288

# Memory optimization
vm.swappiness = 10
vm.dirty_ratio = 15
vm.dirty_background_ratio = 5
```

### Filesystem Mount Options

For backup sources, use optimal mount options:

```bash
# For ext4
mount -o noatime,nodiratime,data=writeback /dev/sdX /backup-source

# For XFS
mount -o noatime,nodiratime,logbufs=8,logbsize=256k /dev/sdX /backup-source
```

### CPU Governor

Set CPU to performance mode:

```bash
cpupower frequency-set -g performance
```

## Troubleshooting Performance Issues

### 1. Slow Transfer Speeds

**Diagnosis**:
```bash
# Check network speed
speedtest-cli

# Monitor rclone
rclone rc core/stats

# Check disk I/O
iostat -x 1
```

**Solutions**:
- Increase `--buffer-size`
- Adjust `--transfers` and `--checkers`
- Check network bandwidth limits

### 2. High Memory Usage

**Diagnosis**:
```bash
# Check memory usage
free -h
ps aux | grep rclone-agent
```

**Solutions**:
- Reduce `--buffer-size`
- Decrease `--transfers`
- Limit `max_concurrent` tasks

### 3. CPU Bottleneck

**Diagnosis**:
```bash
# Check CPU usage
top -p $(pgrep rclone-agent)
mpstat -P ALL 1
```

**Solutions**:
- Reduce `--checkers`
- Enable `--use-mmap`
- Decrease compression level

## Benchmarking

Run performance tests:

```bash
# Test transfer speed
rclone test speed remote:

# Benchmark configuration
./rclone-agent benchmark --config agent.json

# Load testing
./rclone-agent stress-test --concurrent 10 --duration 60m
```

## Best Practices

1. **Start Conservative**: Begin with default settings and optimize based on monitoring
2. **Test Changes**: Always test configuration changes in a non-production environment
3. **Monitor Continuously**: Use metrics to identify bottlenecks
4. **Document Changes**: Keep a log of performance tuning changes
5. **Regular Reviews**: Periodically review and adjust settings

## Performance FAQ

**Q: What's the optimal chunk size?**
A: Depends on your connection. Start with 64M and adjust based on throughput.

**Q: How many concurrent transfers?**
A: Usually 4-8 for broadband, 1-2 for limited connections.

**Q: Should I use compression?**
A: Only for uncompressed data over slow links. Most cloud storage already compresses.

**Q: How to handle rate limiting?**
A: Use `--tpslimit` and implement exponential backoff.