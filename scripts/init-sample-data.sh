#!/bin/bash
set -e

ES_URL="${ES_URL:-http://elasticsearch:9200}"

echo "Waiting for Elasticsearch..."
until curl -s "$ES_URL/_cluster/health" | grep -q '"status":"green"\|"status":"yellow"'; do
  sleep 2
done
echo "Elasticsearch is ready"

echo "Creating sample indices..."

# Products index - 2 primary, 1 replica
curl -s -X PUT "$ES_URL/products" -H 'Content-Type: application/json' -d '{
  "settings": { "number_of_shards": 2, "number_of_replicas": 1 },
  "mappings": { "properties": { "name": { "type": "text" }, "price": { "type": "float" } } }
}'

# Orders index - 3 primary, 1 replica
curl -s -X PUT "$ES_URL/orders" -H 'Content-Type: application/json' -d '{
  "settings": { "number_of_shards": 3, "number_of_replicas": 1 },
  "mappings": { "properties": { "product_id": { "type": "keyword" }, "quantity": { "type": "integer" } } }
}'

# Users index - 1 primary, 1 replica
curl -s -X PUT "$ES_URL/users" -H 'Content-Type: application/json' -d '{
  "settings": { "number_of_shards": 1, "number_of_replicas": 1 },
  "mappings": { "properties": { "email": { "type": "keyword" }, "name": { "type": "text" } } }
}'

# Logs indices (time series pattern)
for month in 01 02 03; do
  curl -s -X PUT "$ES_URL/logs-2026.$month" -H 'Content-Type: application/json' -d '{
    "settings": { "number_of_shards": 2, "number_of_replicas": 1 },
    "mappings": { "properties": { "level": { "type": "keyword" }, "message": { "type": "text" }, "timestamp": { "type": "date" } } }
  }'
done

# Metrics indices
curl -s -X PUT "$ES_URL/metrics-cpu" -H 'Content-Type: application/json' -d '{
  "settings": { "number_of_shards": 1, "number_of_replicas": 1 }
}'
curl -s -X PUT "$ES_URL/metrics-memory" -H 'Content-Type: application/json' -d '{
  "settings": { "number_of_shards": 1, "number_of_replicas": 1 }
}'

echo ""
echo "Creating aliases..."

# Alias for current logs
curl -s -X POST "$ES_URL/_aliases" -H 'Content-Type: application/json' -d '{
  "actions": [
    { "add": { "index": "logs-2026.03", "alias": "logs-current" } },
    { "add": { "index": "logs-2026.*", "alias": "logs" } }
  ]
}'

# Alias for all metrics
curl -s -X POST "$ES_URL/_aliases" -H 'Content-Type: application/json' -d '{
  "actions": [
    { "add": { "index": "metrics-*", "alias": "metrics" } }
  ]
}'

# Alias for e-commerce data
curl -s -X POST "$ES_URL/_aliases" -H 'Content-Type: application/json' -d '{
  "actions": [
    { "add": { "index": "products", "alias": "ecommerce" } },
    { "add": { "index": "orders", "alias": "ecommerce" } },
    { "add": { "index": "users", "alias": "ecommerce" } }
  ]
}'

echo ""
echo "Inserting sample documents..."

# Products - electronics store inventory
curl -s -X POST "$ES_URL/products/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{"_id":"prod-001"}}
{"name":"MacBook Pro 16","category":"laptops","price":2499.99,"stock":45,"brand":"Apple","description":"16-inch Retina display, M3 Pro chip"}
{"index":{"_id":"prod-002"}}
{"name":"ThinkPad X1 Carbon","category":"laptops","price":1899.99,"stock":32,"brand":"Lenovo","description":"14-inch ultrabook, Intel Core i7"}
{"index":{"_id":"prod-003"}}
{"name":"Dell XPS 15","category":"laptops","price":1799.99,"stock":28,"brand":"Dell","description":"15.6-inch OLED display, Intel Core i9"}
{"index":{"_id":"prod-004"}}
{"name":"Mechanical Keyboard RGB","category":"keyboards","price":149.99,"stock":120,"brand":"Keychron","description":"Hot-swappable switches, wireless"}
{"index":{"_id":"prod-005"}}
{"name":"Magic Keyboard","category":"keyboards","price":99.99,"stock":85,"brand":"Apple","description":"Wireless, Touch ID"}
{"index":{"_id":"prod-006"}}
{"name":"MX Master 3S","category":"mice","price":99.99,"stock":67,"brand":"Logitech","description":"Wireless, ergonomic, 8K DPI"}
{"index":{"_id":"prod-007"}}
{"name":"Magic Mouse","category":"mice","price":79.99,"stock":54,"brand":"Apple","description":"Multi-touch surface, wireless"}
{"index":{"_id":"prod-008"}}
{"name":"UltraSharp 32 4K","category":"monitors","price":699.99,"stock":23,"brand":"Dell","description":"32-inch 4K USB-C hub monitor"}
{"index":{"_id":"prod-009"}}
{"name":"Studio Display","category":"monitors","price":1599.99,"stock":12,"brand":"Apple","description":"27-inch 5K Retina, built-in camera"}
{"index":{"_id":"prod-010"}}
{"name":"AirPods Pro","category":"audio","price":249.99,"stock":200,"brand":"Apple","description":"Active noise cancellation, spatial audio"}
{"index":{"_id":"prod-011"}}
{"name":"Sony WH-1000XM5","category":"audio","price":399.99,"stock":45,"brand":"Sony","description":"Wireless noise cancelling headphones"}
{"index":{"_id":"prod-012"}}
{"name":"USB-C Hub 7-in-1","category":"accessories","price":59.99,"stock":150,"brand":"Anker","description":"HDMI, USB-A, SD card reader"}
'

# Orders - recent customer orders
curl -s -X POST "$ES_URL/orders/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{"_id":"ord-001"}}
{"customer_id":"cust-101","product_id":"prod-001","quantity":1,"total":2499.99,"status":"delivered","created_at":"2026-01-10T14:30:00Z","shipping_address":"123 Main St, NYC"}
{"index":{"_id":"ord-002"}}
{"customer_id":"cust-102","product_id":"prod-004","quantity":2,"total":299.98,"status":"shipped","created_at":"2026-01-12T09:15:00Z","shipping_address":"456 Oak Ave, LA"}
{"index":{"_id":"ord-003"}}
{"customer_id":"cust-103","product_id":"prod-006","quantity":1,"total":99.99,"status":"processing","created_at":"2026-01-15T16:45:00Z","shipping_address":"789 Pine Rd, Chicago"}
{"index":{"_id":"ord-004"}}
{"customer_id":"cust-101","product_id":"prod-010","quantity":1,"total":249.99,"status":"delivered","created_at":"2026-01-08T11:00:00Z","shipping_address":"123 Main St, NYC"}
{"index":{"_id":"ord-005"}}
{"customer_id":"cust-104","product_id":"prod-002","quantity":1,"total":1899.99,"status":"pending","created_at":"2026-01-18T10:30:00Z","shipping_address":"321 Elm St, Seattle"}
{"index":{"_id":"ord-006"}}
{"customer_id":"cust-105","product_id":"prod-008","quantity":2,"total":1399.98,"status":"shipped","created_at":"2026-01-16T13:20:00Z","shipping_address":"654 Maple Dr, Austin"}
{"index":{"_id":"ord-007"}}
{"customer_id":"cust-102","product_id":"prod-011","quantity":1,"total":399.99,"status":"delivered","created_at":"2026-01-05T08:45:00Z","shipping_address":"456 Oak Ave, LA"}
{"index":{"_id":"ord-008"}}
{"customer_id":"cust-106","product_id":"prod-003","quantity":1,"total":1799.99,"status":"processing","created_at":"2026-01-17T15:10:00Z","shipping_address":"987 Cedar Ln, Denver"}
{"index":{"_id":"ord-009"}}
{"customer_id":"cust-103","product_id":"prod-012","quantity":3,"total":179.97,"status":"delivered","created_at":"2026-01-11T12:00:00Z","shipping_address":"789 Pine Rd, Chicago"}
{"index":{"_id":"ord-010"}}
{"customer_id":"cust-107","product_id":"prod-009","quantity":1,"total":1599.99,"status":"cancelled","created_at":"2026-01-14T17:30:00Z","shipping_address":"246 Birch Way, Miami"}
'

# Users - customer accounts
curl -s -X POST "$ES_URL/users/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{"_id":"cust-101"}}
{"email":"alice.smith@example.com","name":"Alice Smith","created_at":"2024-03-15T10:00:00Z","tier":"gold","total_orders":15,"total_spent":4299.97}
{"index":{"_id":"cust-102"}}
{"email":"bob.jones@example.com","name":"Bob Jones","created_at":"2024-06-20T14:30:00Z","tier":"silver","total_orders":8,"total_spent":1899.95}
{"index":{"_id":"cust-103"}}
{"email":"carol.white@example.com","name":"Carol White","created_at":"2025-01-10T09:15:00Z","tier":"bronze","total_orders":3,"total_spent":379.96}
{"index":{"_id":"cust-104"}}
{"email":"david.brown@example.com","name":"David Brown","created_at":"2025-08-05T16:45:00Z","tier":"bronze","total_orders":1,"total_spent":1899.99}
{"index":{"_id":"cust-105"}}
{"email":"eva.garcia@example.com","name":"Eva Garcia","created_at":"2024-11-22T11:00:00Z","tier":"gold","total_orders":22,"total_spent":8750.50}
{"index":{"_id":"cust-106"}}
{"email":"frank.miller@example.com","name":"Frank Miller","created_at":"2025-05-18T13:20:00Z","tier":"silver","total_orders":6,"total_spent":2499.94}
{"index":{"_id":"cust-107"}}
{"email":"grace.lee@example.com","name":"Grace Lee","created_at":"2025-09-30T08:45:00Z","tier":"bronze","total_orders":2,"total_spent":1699.98}
{"index":{"_id":"cust-108"}}
{"email":"henry.wilson@example.com","name":"Henry Wilson","created_at":"2024-07-12T15:10:00Z","tier":"platinum","total_orders":45,"total_spent":25999.55}
'

# Logs - January 2026
curl -s -X POST "$ES_URL/logs-2026.01/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{}}
{"level":"INFO","service":"api-gateway","message":"Server started on port 8080","timestamp":"2026-01-01T00:00:01Z","host":"prod-api-1"}
{"index":{}}
{"level":"INFO","service":"auth","message":"JWT token validated successfully","timestamp":"2026-01-01T00:05:23Z","host":"prod-auth-1","user_id":"cust-101"}
{"index":{}}
{"level":"WARN","service":"inventory","message":"Low stock alert: prod-009 has 12 units","timestamp":"2026-01-01T01:30:00Z","host":"prod-inv-1"}
{"index":{}}
{"level":"ERROR","service":"payment","message":"Payment gateway timeout after 30s","timestamp":"2026-01-01T02:15:45Z","host":"prod-pay-1","error_code":"TIMEOUT"}
{"index":{}}
{"level":"INFO","service":"api-gateway","message":"Health check passed","timestamp":"2026-01-01T03:00:00Z","host":"prod-api-1"}
{"index":{}}
{"level":"DEBUG","service":"search","message":"Query executed in 45ms","timestamp":"2026-01-01T04:22:10Z","host":"prod-search-1","query":"laptops"}
{"index":{}}
{"level":"ERROR","service":"email","message":"SMTP connection refused","timestamp":"2026-01-01T05:10:33Z","host":"prod-email-1","error_code":"CONN_REFUSED"}
{"index":{}}
{"level":"INFO","service":"orders","message":"Order ord-001 status changed to shipped","timestamp":"2026-01-01T06:45:00Z","host":"prod-orders-1"}
{"index":{}}
{"level":"WARN","service":"api-gateway","message":"Rate limit approaching for IP 192.168.1.100","timestamp":"2026-01-01T07:30:15Z","host":"prod-api-2"}
{"index":{}}
{"level":"INFO","service":"auth","message":"New user registration: cust-108","timestamp":"2026-01-01T08:00:00Z","host":"prod-auth-1"}
'

# Logs - February 2026
curl -s -X POST "$ES_URL/logs-2026.02/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{}}
{"level":"INFO","service":"api-gateway","message":"Deployed version 2.3.1","timestamp":"2026-02-01T00:00:00Z","host":"prod-api-1"}
{"index":{}}
{"level":"ERROR","service":"database","message":"Connection pool exhausted","timestamp":"2026-02-01T01:15:30Z","host":"prod-db-1","error_code":"POOL_EXHAUSTED"}
{"index":{}}
{"level":"WARN","service":"cache","message":"Cache miss rate above 40%","timestamp":"2026-02-01T02:30:00Z","host":"prod-cache-1"}
{"index":{}}
{"level":"INFO","service":"orders","message":"Daily order count: 1,247","timestamp":"2026-02-01T03:00:00Z","host":"prod-orders-1"}
{"index":{}}
{"level":"ERROR","service":"inventory","message":"Stock sync failed with warehouse API","timestamp":"2026-02-01T04:45:22Z","host":"prod-inv-1","error_code":"SYNC_FAILED"}
{"index":{}}
{"level":"INFO","service":"search","message":"Index rebuilt successfully","timestamp":"2026-02-01T05:30:00Z","host":"prod-search-1","duration_ms":12500}
{"index":{}}
{"level":"DEBUG","service":"payment","message":"Processing refund for ord-010","timestamp":"2026-02-01T06:15:45Z","host":"prod-pay-1"}
{"index":{}}
{"level":"WARN","service":"auth","message":"Multiple failed login attempts for user cust-105","timestamp":"2026-02-01T07:00:10Z","host":"prod-auth-2"}
{"index":{}}
{"level":"INFO","service":"api-gateway","message":"SSL certificate renewed","timestamp":"2026-02-01T08:30:00Z","host":"prod-api-1"}
{"index":{}}
{"level":"ERROR","service":"notification","message":"Push notification service unavailable","timestamp":"2026-02-01T09:45:33Z","host":"prod-notif-1","error_code":"SERVICE_DOWN"}
'

# Logs - March 2026 (current month, more entries)
curl -s -X POST "$ES_URL/logs-2026.03/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{}}
{"level":"INFO","service":"api-gateway","message":"Application started","timestamp":"2026-03-15T10:00:00Z","host":"prod-api-1"}
{"index":{}}
{"level":"ERROR","service":"database","message":"Connection timeout after 5000ms","timestamp":"2026-03-15T10:05:00Z","host":"prod-db-1","error_code":"CONN_TIMEOUT"}
{"index":{}}
{"level":"WARN","service":"api-gateway","message":"High memory usage: 85%","timestamp":"2026-03-15T10:10:00Z","host":"prod-api-2"}
{"index":{}}
{"level":"INFO","service":"orders","message":"New order created: ord-011","timestamp":"2026-03-15T10:15:30Z","host":"prod-orders-1"}
{"index":{}}
{"level":"DEBUG","service":"search","message":"Autocomplete query: mac","timestamp":"2026-03-15T10:20:15Z","host":"prod-search-1","results":3}
{"index":{}}
{"level":"ERROR","service":"payment","message":"Invalid card number","timestamp":"2026-03-15T10:25:45Z","host":"prod-pay-1","error_code":"INVALID_CARD"}
{"index":{}}
{"level":"INFO","service":"inventory","message":"Stock replenished: prod-009 +50 units","timestamp":"2026-03-15T10:30:00Z","host":"prod-inv-1"}
{"index":{}}
{"level":"WARN","service":"cache","message":"Evicting stale entries","timestamp":"2026-03-15T10:35:20Z","host":"prod-cache-1","evicted":1250}
{"index":{}}
{"level":"INFO","service":"auth","message":"Password reset requested","timestamp":"2026-03-15T10:40:00Z","host":"prod-auth-1","user_id":"cust-103"}
{"index":{}}
{"level":"ERROR","service":"email","message":"Template not found: order_confirmation_v2","timestamp":"2026-03-15T10:45:10Z","host":"prod-email-1","error_code":"TEMPLATE_NOT_FOUND"}
{"index":{}}
{"level":"INFO","service":"api-gateway","message":"Request rate: 450 req/s","timestamp":"2026-03-15T10:50:00Z","host":"prod-api-1"}
{"index":{}}
{"level":"DEBUG","service":"orders","message":"Calculating shipping for zone US-WEST","timestamp":"2026-03-15T10:55:30Z","host":"prod-orders-1"}
{"index":{}}
{"level":"WARN","service":"database","message":"Slow query detected: 2500ms","timestamp":"2026-03-15T11:00:00Z","host":"prod-db-1","query":"SELECT * FROM orders WHERE..."}
{"index":{}}
{"level":"INFO","service":"notification","message":"Sent 150 push notifications","timestamp":"2026-03-15T11:05:15Z","host":"prod-notif-1"}
{"index":{}}
{"level":"ERROR","service":"api-gateway","message":"Upstream service unavailable: inventory","timestamp":"2026-03-15T11:10:45Z","host":"prod-api-2","error_code":"SERVICE_UNAVAILABLE"}
'

# Metrics - CPU usage samples
curl -s -X POST "$ES_URL/metrics-cpu/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{}}
{"host":"prod-api-1","value":45.2,"timestamp":"2026-03-15T10:00:00Z"}
{"index":{}}
{"host":"prod-api-1","value":52.8,"timestamp":"2026-03-15T10:05:00Z"}
{"index":{}}
{"host":"prod-api-1","value":78.5,"timestamp":"2026-03-15T10:10:00Z"}
{"index":{}}
{"host":"prod-api-2","value":38.1,"timestamp":"2026-03-15T10:00:00Z"}
{"index":{}}
{"host":"prod-api-2","value":41.3,"timestamp":"2026-03-15T10:05:00Z"}
{"index":{}}
{"host":"prod-api-2","value":85.2,"timestamp":"2026-03-15T10:10:00Z"}
{"index":{}}
{"host":"prod-db-1","value":62.4,"timestamp":"2026-03-15T10:00:00Z"}
{"index":{}}
{"host":"prod-db-1","value":71.8,"timestamp":"2026-03-15T10:05:00Z"}
{"index":{}}
{"host":"prod-db-1","value":95.1,"timestamp":"2026-03-15T10:10:00Z"}
{"index":{}}
{"host":"prod-search-1","value":28.5,"timestamp":"2026-03-15T10:00:00Z"}
'

# Metrics - Memory usage samples
curl -s -X POST "$ES_URL/metrics-memory/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{}}
{"host":"prod-api-1","value":4096,"max":8192,"percent":50.0,"timestamp":"2026-03-15T10:00:00Z"}
{"index":{}}
{"host":"prod-api-1","value":5120,"max":8192,"percent":62.5,"timestamp":"2026-03-15T10:05:00Z"}
{"index":{}}
{"host":"prod-api-1","value":6963,"max":8192,"percent":85.0,"timestamp":"2026-03-15T10:10:00Z"}
{"index":{}}
{"host":"prod-api-2","value":3584,"max":8192,"percent":43.8,"timestamp":"2026-03-15T10:00:00Z"}
{"index":{}}
{"host":"prod-api-2","value":4608,"max":8192,"percent":56.3,"timestamp":"2026-03-15T10:05:00Z"}
{"index":{}}
{"host":"prod-api-2","value":7168,"max":8192,"percent":87.5,"timestamp":"2026-03-15T10:10:00Z"}
{"index":{}}
{"host":"prod-db-1","value":12288,"max":16384,"percent":75.0,"timestamp":"2026-03-15T10:00:00Z"}
{"index":{}}
{"host":"prod-db-1","value":13312,"max":16384,"percent":81.3,"timestamp":"2026-03-15T10:05:00Z"}
{"index":{}}
{"host":"prod-db-1","value":15360,"max":16384,"percent":93.8,"timestamp":"2026-03-15T10:10:00Z"}
{"index":{}}
{"host":"prod-search-1","value":2048,"max":4096,"percent":50.0,"timestamp":"2026-03-15T10:00:00Z"}
'

echo ""
echo "Sample data initialized:"
curl -s "$ES_URL/_cat/indices?v"
echo ""
echo "Aliases:"
curl -s "$ES_URL/_cat/aliases?v"
