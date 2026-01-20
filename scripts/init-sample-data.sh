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

# Products
curl -s -X POST "$ES_URL/products/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{}}
{"name":"Laptop","price":999.99}
{"index":{}}
{"name":"Keyboard","price":79.99}
{"index":{}}
{"name":"Mouse","price":29.99}
'

# Orders
curl -s -X POST "$ES_URL/orders/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{}}
{"product_id":"laptop-1","quantity":2}
{"index":{}}
{"product_id":"keyboard-1","quantity":5}
'

# Users
curl -s -X POST "$ES_URL/users/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{}}
{"email":"alice@example.com","name":"Alice Smith"}
{"index":{}}
{"email":"bob@example.com","name":"Bob Jones"}
'

# Logs
curl -s -X POST "$ES_URL/logs-2026.03/_bulk" -H 'Content-Type: application/x-ndjson' -d '
{"index":{}}
{"level":"INFO","message":"Application started","timestamp":"2026-03-15T10:00:00Z"}
{"index":{}}
{"level":"ERROR","message":"Connection timeout","timestamp":"2026-03-15T10:05:00Z"}
{"index":{}}
{"level":"WARN","message":"High memory usage","timestamp":"2026-03-15T10:10:00Z"}
'

echo ""
echo "Sample data initialized:"
curl -s "$ES_URL/_cat/indices?v"
echo ""
echo "Aliases:"
curl -s "$ES_URL/_cat/aliases?v"
