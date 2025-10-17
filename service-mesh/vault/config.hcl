# Vault Server Configuration
# Provides secrets management and PKI for Consul

# ============ STORAGE BACKEND ============
# Where Vault stores encrypted data
storage "file" {
  path = "/vault/data"
}

# ============ LISTENER ============
# HTTP listener for this dev setup
# In production: use HTTPS with TLS
listener "tcp" {
  address       = "0.0.0.0:8200"
  tls_disable   = true  # Disable for dev; enable HTTPS in production
}
api_addr = "http://vault:8200"
# ============ TELEMETRY ============
telemetry {
  prometheus_retention_time = "30s"
}

# ============ GENERAL ============
ui = true