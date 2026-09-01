#!/bin/sh
echo "Waiting for OpenSearch Dashboards to be ready..."

until curl -s -o /dev/null -w "%{http_code}" http://opensearch-dashboards:5601/api/status | grep -q "200"; do
  sleep 2
done

echo "OpenSearch Dashboards is ready. Creating Data View..."

curl -X POST "http://opensearch-dashboards:5601/api/data_views/data_view" \
  -H "osd-xsrf: true" \
  -H "Content-Type: application/json" \
  -d '{
    "data_view": {
      "title": "gophprofile-logs*",
      "timeFieldName": "time"
    }
  }'

echo "Data View creation completed."
