This is a small HTTP service which listens on a port and is capable of updating DNS A record on Google Cloud.


## Nginx Config

You want to put an Nginx, or any HTTP server with TLS enabled, in front of this service.
This service supports, and prioritises, `X-Forwarded-For` header. You need to set header when forwarding from Nginx to service.
Put this on your Nginx service:
```
        location ~* /dns_updater.* {
                proxy_set_header  X-Forwarded-For $remote_addr;
                proxy_pass http://127.0.0.1:8090;
        }
```
