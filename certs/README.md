Generating keys to encrypt data
```bash
$ openssl genpkey -algorithm EC -pkeyopt ec_paramgen_curve:P-256 -out private_key.pem

$ openssl ec -in private_key.pem -pubout -out public_key.pem
```

Generating the certs
```bash
$ openssl req -x509 -newkey rsa:4096 -keyout certs/server.key -out certs/server.crt -days 365 -nodes -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1"
```