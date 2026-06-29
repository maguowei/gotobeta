# SMTP Email Sender

This package provides a small SMTP sender for infrastructure adapters. It sends plain text messages and keeps provider-specific wiring outside domain and application packages.

Configuration is loaded from the generated `smtp` config block:

```yaml
smtp:
  enabled: false
  host: "127.0.0.1"
  port: 1025
  username: ""
  password: ""
  from: "gotobeta <no-reply@example.com>"
  tls_mode: "none"
  timeout: "5s"
```

Use `tls_mode: "starttls"` or `"tls"` for shared and production environments. Production startup rejects `smtp.enabled=true` with `tls_mode: "none"`.
