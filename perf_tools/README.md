# Performance Profiling Tools for Code-Pathfinder

A collection of tools to measure and visualize memory usage and performance of code-pathfinder queries.

## Quick Start

### 1. Basic Usage (Easiest)

```bash
cd perf_tools
./benchmark.sh
```

This runs a default benchmark on the SaltStack codebase with a function definition query.

### 2. Custom Query

```bash
./benchmark.sh -q "FROM class_definition AS cd SELECT cd" -o class_results
```

### 3. Different Project

```bash
./benchmark.sh -p /path/to/your/project -q "FROM function_definition AS fd SELECT fd"
```

## Command Line Options

```
Usage: ./benchmark.sh [options]

Options:
  -p, --project DIR     Project directory to analyze (default: ~/src/shivasurya/salt)
  -q, --query QUERY     Query to run (default: 'FROM function_definition AS fd SELECT fd')
  -o, --output NAME     Output file prefix (default: 'benchmark')
  -b, --binary PATH     Path to pathfinder binary (default: ../sast-engine/build/go/pathfinder)
  -h, --help            Show this help message
```

## Examples

### Compare Class vs Function Queries

```bash
# Run class definition benchmark
./benchmark.sh -q "FROM class_definition AS cd SELECT cd" -o class_benchmark

# Run function definition benchmark
./benchmark.sh -q "FROM function_definition AS fd SELECT fd" -o function_benchmark

# Compare the PNG graphs!
open class_benchmark.png function_benchmark.png
```

### Test Different Codebases

```bash
# Test on your own project
./benchmark.sh -p ~/myproject -o myproject_benchmark

# Test on multiple projects
for proj in project1 project2 project3; do
    ./benchmark.sh -p ~/repos/$proj -o ${proj}_benchmark
done
```

## Output Files

Each benchmark run creates 3 files:

1. **`{name}.csv`** - Raw memory usage data (timestamp, RSS, VSZ)
2. **`{name}.png`** - Memory usage graph with timeline
3. **`{name}.log`** - Query execution log

Example:
```
benchmark.csv  - Memory data points
benchmark.png  - Visual graph
benchmark.log  - Execution log
```

## Understanding the Results

### Memory Metrics

- **RSS (Resident Set Size)**: Actual physical memory used (most important)
- **VSZ (Virtual Memory Size)**: Total virtual memory allocated

### Graph Interpretation

```
Memory Usage Over Time
│
│   Peak: 2943.6 MB
│   Avg: 2813.4 MB
│
│ 3000 MB ├─────────────────────── Flat line (good!)
│         │       ╱────────────────
│ 2000 MB │      ╱
│         │     ╱  Parsing phase
│ 1000 MB │    ╱
│         │   ╱
│    0 MB └──────────────────────►
          0s  20s  40s  60s  80s
```

**Good patterns:**
- ✅ Rapid rise then flat = efficient memory use
- ✅ Stable plateau = no memory leaks

**Bad patterns:**
- ❌ Continuous rise = possible memory leak
- ❌ Spikes during query = inefficient allocations

## Requirements

### Required
- Bash shell
- Built pathfinder binary (run `cd ../sast-engine && gradle buildGo`)

### Optional
- Python 3 with matplotlib and pandas for graph generation
  ```bash
  pip3 install matplotlib pandas
  ```

## Manual Mode (Advanced)

If you want more control, use the individual scripts:

### 1. Run Query with Monitoring

```bash
# Terminal 1: Start query
../sast-engine/build/go/pathfinder query --project ~/salt --query "..." &
PID=$!

# Terminal 2: Monitor memory
./fast_monitor.sh $PID memory_data.csv
```

### 2. Generate Graph

```bash
python3 plot_memory.py memory_data.csv
# Creates: memory_data.png
```

## Scripts Overview

| Script | Purpose |
|--------|---------|
| `benchmark.sh` | **Main tool** - Easy-to-use wrapper |
| `fast_monitor.sh` | Monitors process memory (100ms sampling) |
| `monitor_memory.sh` | Slower monitoring (500ms sampling) |
| `plot_memory.py` | Generates memory usage graphs |

## Comparing Optimizations

To measure the impact of performance optimizations:

```bash
# Before optimization
git checkout main
cd sast-engine && gradle clean buildGo && cd ../perf_tools
./benchmark.sh -o before_optimization

# After optimization
git checkout feature-branch
cd sast-engine && gradle clean buildGo && cd ../perf_tools
./benchmark.sh -o after_optimization

# Compare results
echo "Before: $(grep 'Peak RSS' before_optimization.csv | tail -1)"
echo "After:  $(grep 'Peak RSS' after_optimization.csv | tail -1)"
```

## Troubleshooting

### "Pathfinder binary not found"

Build the binary first:
```bash
cd ../sast-engine
gradle clean buildGo
cd ../perf_tools
```

### "Python3 not found"

The CSV data is still generated. You can:
1. Install Python: `brew install python3`
2. Use the CSV data with your own tools
3. Run without graphs (CSV has all the data)

### "Project directory not found"

Specify the correct path:
```bash
./benchmark.sh -p /absolute/path/to/your/project
```

## Tips

1. **Run multiple times**: Results can vary due to system load. Run 3 times and compare.

2. **Close other apps**: For accurate results, close memory-heavy applications.

3. **Use full paths**: When in doubt, use absolute paths for `-p` and `-b` options.

4. **Compare similar queries**: Compare "class vs class" or "function vs function" for fair comparisons.

## Contributing

Found a bug or want to improve these tools? The scripts are simple bash/Python:

- `benchmark.sh` - Main orchestration script
- `fast_monitor.sh` - Memory sampling loop
- `plot_memory.py` - matplotlib graphing

Feel free to modify and improve!

## License

Same as code-pathfinder project (AGPL-3.0).
