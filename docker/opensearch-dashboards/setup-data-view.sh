#!/bin/sh
echo "Waiting for OpenSearch Dashboards to be ready..."

until curl -s -o /dev/null -w "%{http_code}" http://opensearch-dashboards:5601/api/status | grep -q "200"; do
  sleep 2
done

echo "OpenSearch Dashboards is ready. Creating Data View..."

curl -X POST "http://opensearch-dashboards:5601/api/saved_objects/index-pattern" \
  -H "osd-xsrf: true" \
  -H "Content-Type: application/json" \
  -d '{
    "attributes": {
      "title": "gophprofile-logs*",
      "timeFieldName": "@timestamp"
    }
  }' || true

echo "Data View creation completed."