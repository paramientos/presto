#!/bin/bash

# Presto vs Composer Performance Benchmark
# This script compares installation times between Presto and Composer

set -e

echo "🎵 Presto vs Composer Benchmark"
echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
echo ""

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Test projects
PROJECTS=(
    "laravel/laravel"
    "symfony/skeleton"
    "slim/slim-skeleton"
)

# Create benchmark directory
BENCH_DIR="benchmark_results_$(date +%Y%m%d_%H%M%S)"
mkdir -p "$BENCH_DIR"

echo "📊 Benchmark Results Directory: $BENCH_DIR"
echo ""

for project in "${PROJECTS[@]}"; do
    echo "${BLUE}Testing: $project${NC}"
    echo "━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━━"
    
    # Test Composer
    echo "${YELLOW}[Composer]${NC}"
    TEST_DIR="$BENCH_DIR/composer_$(basename $project)"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # First run (cold cache)
    composer create-project "$project" . --no-interaction --quiet 2>&1 > /dev/null
    COMPOSER_TIME_1=$({ time composer install --no-interaction 2>&1; } 2>&1 | grep real | awk '{print $2}')
    
    # Clear vendor
    rm -rf vendor
    
    # Second run (warm cache)
    COMPOSER_TIME_2=$({ time composer install --no-interaction 2>&1; } 2>&1 | grep real | awk '{print $2}')
    
    cd - > /dev/null
    
    # Test Presto
    echo "${YELLOW}[Presto]${NC}"
    TEST_DIR="$BENCH_DIR/presto_$(basename $project)"
    mkdir -p "$TEST_DIR"
    cd "$TEST_DIR"
    
    # Copy composer.json from composer test
    cp "$BENCH_DIR/composer_$(basename $project)/composer.json" .
    
    # First run (cold cache)
    PRESTO_TIME_1=$({ time ../../../bin/presto install 2>&1; } 2>&1 | grep real | awk '{print $2}')
    
    # Clear vendor
    rm -rf vendor
    
    # Second run (warm cache)
    PRESTO_TIME_2=$({ time ../../../bin/presto install 2>&1; } 2>&1 | grep real | awk '{print $2}')
    
    cd - > /dev/null
    
    # Calculate speedup
    echo ""
    echo "${GREEN}Results for $project:${NC}"
    echo "  Composer (cold): $COMPOSER_TIME_1"
    echo "  Presto (cold):   $PRESTO_TIME_1"
    echo "  Composer (warm): $COMPOSER_TIME_2"
    echo "  Presto (warm):   $PRESTO_TIME_2"
    echo ""
    
    # Save to CSV
    echo "$project,$COMPOSER_TIME_1,$PRESTO_TIME_1,$COMPOSER_TIME_2,$PRESTO_TIME_2" >> "$BENCH_DIR/results.csv"
done

echo ""
echo "${GREEN}✅ Benchmark Complete!${NC}"
echo "Results saved to: $BENCH_DIR/results.csv"
echo ""
echo "To view results:"
echo "  cat $BENCH_DIR/results.csv"
