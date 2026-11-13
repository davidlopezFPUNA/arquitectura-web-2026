#!/bin/bash
echo "=== PRUEBAS AUTOMATIZADAS DE MICROSERVICIOS ==="

echo -n "1. Search Service Health Check: "
curl -s http://localhost:8080/healthz | grep -q "alive" && echo "PASS" || echo "FAIL"

echo -n "2. Search Service Readiness Check: "
curl -s http://localhost:8080/readyz | grep -q "ready" && echo "PASS" || echo "FAIL"

echo -n "3. Search Service Metrics Prometheus: "
curl -s http://localhost:8080/metrics | grep -q "http_requests_total" && echo "PASS" || echo "FAIL"

echo -n "4. Export Service 202 Accepted Response: "
curl -s -X POST http://localhost:8081/v2/export | grep -q "queued" && echo "PASS" || echo "FAIL"

echo -n "5. Audit Service Event Processing: "
curl -s http://localhost:8082/healthz | grep -q "alive" && echo "PASS" || echo "FAIL"

echo -n "6. Media Service Processing Check: "
curl -s http://localhost:8083/healthz | grep -q "alive" && echo "PASS" || echo "FAIL"
