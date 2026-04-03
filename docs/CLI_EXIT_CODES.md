# CLI Exit Codes

The claimctl CLI uses standard exit codes to indicate different failure
scenarios. This allows CI/CD pipelines to handle errors appropriately.

## Exit Code Reference

| Code | Name          | Description                                      |
| ---- | ------------- | ------------------------------------------------ |
| 0    | Success       | Operation completed successfully                 |
| 1    | General Error | Unspecified error occurred                       |
| 2    | Timeout       | Timeout waiting for reservation to become active |
| 3    | Cancelled     | Reservation was cancelled                        |
| 4    | Not Found     | Resource or reservation not found                |
| 5    | Unauthorized  | Authentication failed                            |
| 6    | Resource Busy | Resource is busy (with --no-queue flag)          |

## Usage in Scripts

### Basic Error Handling

```bash
if claimctl reserve --type "gpu" --wait; then
  echo "Resource acquired successfully"
  # Run your workload
else
  echo "Failed to acquire resource"
  exit 1
fi
```

### Advanced Error Handling

```bash
claimctl reserve --type "gpu" --wait --timeout 60
exit_code=$?

case $exit_code in
  0)
    echo "Success - resource acquired"
    ;;
  2)
    echo "Timeout - resource still busy after 60 seconds"
    # Maybe try a different resource or fail the build
    exit 1
    ;;
  3)
    echo "Reservation was cancelled"
    exit 1
    ;;
  4)
    echo "Resource not found"
    exit 1
    ;;
  5)
    echo "Authentication failed - check your token"
    exit 1
    ;;
  *)
    echo "Unknown error occurred"
    exit 1
    ;;
esac
```

### Retry on Timeout

```bash
max_retries=3
retry_count=0

while [ $retry_count -lt $max_retries ]; do
  if claimctl reserve --type "gpu" --wait --timeout 120; then
    echo "Resource acquired"
    break
  fi

  exit_code=$?
  if [ $exit_code -eq 2 ]; then
    retry_count=$((retry_count + 1))
    echo "Timeout (attempt $retry_count/$max_retries). Retrying..."
    sleep 10
  else
    echo "Non-timeout error occurred"
    exit $exit_code
  fi
done

if [ $retry_count -eq $max_retries ]; then
  echo "Failed to acquire resource after $max_retries attempts"
  exit 2
fi
```

### Pipeline with Fallback

```bash
# Try to get a GPU resource, fall back to CPU if timeout
if claimctl reserve --type "gpu" --wait --timeout 60; then
  echo "Using GPU resource"
  run_gpu_tests
else
  exit_code=$?
  if [ $exit_code -eq 2 ]; then
    echo "GPU timeout, falling back to CPU"
    claimctl reserve --type "cpu" --wait
    run_cpu_tests
  else
    echo "Error acquiring resource"
    exit $exit_code
  fi
fi
```

## CI/CD Integration Examples

### GitHub Actions

```yaml
- name: Reserve Test Environment
  id: reserve
  run: |
    reservation_id=$(claimctl reserve \
      --type "test-env" \
      --wait \
      --timeout 300 \
      --quiet)
    echo "reservation_id=$reservation_id" >> $GITHUB_OUTPUT
  continue-on-error: false

- name: Run Tests
  run: pytest tests/

- name: Release Resource
  if: always()
  run: claimctl release ${{ steps.reserve.outputs.reservation_id }}
```

### GitLab CI

```yaml
test:
  script:
    - |
      reservation_id=$(claimctl reserve \
        --type "test-env" \
        --wait \
        --timeout 300 \
        --quiet)
      trap "claimctl release $reservation_id" EXIT
      pytest tests/
  retry:
    max: 2
    when:
      - script_failure
```

### Jenkins

```groovy
pipeline {
  stages {
    stage('Reserve Resource') {
      steps {
        script {
          def result = sh(
            script: 'claimctl reserve --type "test-env" --wait --quiet',
            returnStdout: true
          ).trim()
          env.RESERVATION_ID = result
        }
      }
    }
    stage('Test') {
      steps {
        sh 'pytest tests/'
      }
    }
  }
  post {
    always {
      sh "claimctl release ${env.RESERVATION_ID}"
    }
  }
}
```

## Best Practices

1. **Always use cleanup handlers**: Use `trap` in bash or equivalent in other
   shells to ensure resources are released even if tests fail.

2. **Set appropriate timeouts**: Don't use infinite timeouts. Set reasonable
   limits based on your typical queue times.

3. **Handle specific exit codes**: Don't treat all errors the same. Timeouts
   might be retryable, but authentication errors are not.

4. **Use --quiet for scripting**: When capturing reservation IDs, use `--quiet`
   to get clean output without progress messages.

5. **Combine with --json**: For complex pipelines, use `--json` to get
   structured output that's easy to parse.

## Troubleshooting

### Exit Code 2 (Timeout)

**Cause**: Reservation didn't become active within the timeout period.

**Solutions**:

- Increase timeout value
- Check queue status: `claimctl reservations list`
- Try a different resource type
- Implement retry logic

### Exit Code 3 (Cancelled)

**Cause**: Reservation was cancelled while waiting.

**Solutions**:

- Check if admin cancelled the reservation
- Verify resource wasn't deleted
- Check maintenance schedules

### Exit Code 5 (Unauthorized)

**Cause**: Authentication failed.

**Solutions**:

- Verify `claimctl_TOKEN` is set correctly
- Check `.netrc` file permissions (should be 600)
- Ensure token hasn't expired
- Ensure token hasn't expired
- Verify server URL is correct

### Exit Code 6 (Resource Busy)

**Cause**: Resource was busy and `--no-queue` flag was used.

**Solutions**:

- Remove `--no-queue` to wait in queue
- Retry later
- Use a different resource type
