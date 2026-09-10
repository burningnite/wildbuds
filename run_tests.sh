#!/bin/bash
set -e

echo "=== Building project ==="
cargo build

echo "=== Running cargo check ==="
cargo check

echo "=== Running cargo clippy ==="
cargo clippy -- -D warnings

echo "=== Running unit tests ==="
cargo test

echo "=== Tests completed successfully! ==="
