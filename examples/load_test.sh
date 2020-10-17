set -e
set -x

vegeta attack -rate=400 -duration=10s -targets=targets.txt | vegeta report
