#!/bin/bash

# Example CI/CD pipeline script using claimctl with built-in wait
# This demonstrates the simplified approach using --wait flag

set -e

# Configuration
RESOURCE_TYPE="test-environment"
RESOURCE_LABEL="gpu"
TEST_DURATION="30m"
WAIT_TIMEOUT=600  # 10 minutes

echo "=== CI/CD Pipeline with claimctl ==="
echo ""

# Step 1: Reserve a resource and wait for activation
echo "Step 1: Reserving resource and waiting for activation..."
echo "  Type: ${RESOURCE_TYPE}"
echo "  Label: ${RESOURCE_LABEL}"
echo "  Duration: ${TEST_DURATION}"
echo "  Timeout: ${WAIT_TIMEOUT}s"
echo ""

# Reserve with built-in wait functionality
reservation_json=$(claimctl reserve \
    --type "${RESOURCE_TYPE}" \
    --label "${RESOURCE_LABEL}" \
    --duration "${TEST_DURATION}" \
    --wait \
    --timeout "${WAIT_TIMEOUT}" \
    --json)

# Check if reservation was successful
if [ $? -ne 0 ]; then
    echo "Failed to acquire resource"
    exit 1
fi

# Extract reservation ID and resource ID
reservation_id=$(echo "$reservation_json" | jq -r '.id')
resource_id=$(echo "$reservation_json" | jq -r '.resourceId')

echo "✓ Resource acquired: ID=${resource_id}, Reservation=${reservation_id}"
echo ""

# Cleanup function to ensure resource is released
cleanup() {
    exit_code=$?
    echo ""
    echo "=== Cleanup ==="
    echo "Releasing reservation #${reservation_id}..."

    if claimctl release "$reservation_id"; then
        echo "✓ Resource released successfully"
    else
        echo "✗ Failed to release resource"
    fi

    exit $exit_code
}

# Register cleanup to run on exit
trap cleanup EXIT INT TERM

# Step 2: Run your tests/workload
echo "Step 2: Running tests on resource #${resource_id}..."
echo ""

# Your actual test commands go here
echo "Running test suite..."
# Example: pytest tests/
# Example: ./run_integration_tests.sh
# Example: make test
sleep 5  # Simulated test execution
echo "✓ Tests completed successfully"
echo ""

# The cleanup function will automatically release the resource
echo "Pipeline completed successfully!"
