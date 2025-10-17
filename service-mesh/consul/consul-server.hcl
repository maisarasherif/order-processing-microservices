# Consul Server Configuration - With Vault as CA
# Consul requests certificates from Vault instead of using built-in CA

datacenter = "dc1"
node_name  = "consul-server-1"
server   = true
ui_config {
  enabled = true
}
bootstrap_expect = 1

# Networking - simplified for Docker
bind_addr = "0.0.0.0"
client_addr = "0.0.0.0"

ports = {
  dns      = 8600
  http     = 8500
  https    = 8501
  grpc     = 8502
  grpc_tls = 8503
  serf_lan = 8301
  serf_wan = 8302
  server   = 8300
}

# ============ mTLS & SECURITY ============
tls {
  defaults {
    ca_file   = "/etc/consul/tls/ca.crt"
    cert_file = "/etc/consul/tls/server.crt"
    key_file  = "/etc/consul/tls/server.key"

    verify_incoming = true
    verify_outgoing = true
  }

  internal_rpc {
    ca_file   = "/etc/consul/tls/ca.crt"
    cert_file = "/etc/consul/tls/server.crt"
    key_file  = "/etc/consul/tls/server.key"
  }
}

log_level = "info"
enable_debug = false

# ============ SERVICE MESH WITH VAULT CA ============
# This is the key change: Vault provides all service mesh certificates
connect {
  enabled = true
  
  # Use Vault as the certificate authority
  ca_provider = "vault"
  
  # Vault configuration for Consul to authenticate and request certs
  ca_config {
    # Vault server address (internal to Docker network)
    address      = "http://vault:8200"
    
    # Token for Consul to authenticate to Vault
    # This should be created/managed securely
    # We'll initialize this after Vault is unsealed
    token        = "s.ChangeMe-VaultToken"
    
    # Path to the PKI mount in Vault
    # We'll create this at: /pki/consul/issue/consul
    root_pki_path     = "pki/root/sign-self-issued"
    intermediate_pki_path = "pki/intermediate/issue/consul"
    
    # Certificate lifetime
    leaf_cert_ttl = "72h"
    
    # Certificate fields
  }
}

# ============ ACL SYSTEM (Zero-Trust) ============
acl {
  enabled                  = true
  default_policy           = "deny"
  enable_token_persistence = true
  enable_token_replication = true
  tokens {
    initial_management = "root-token-change-in-prod"
  }
}

data_dir = "/consul/data"

telemetry {
  prometheus_retention_time = "30s"
}