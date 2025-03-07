# Traefik JSON Body2Header
This is a small [Traefik](https://traefik.io/) middleware plugin.  
The middleware enables the user to extracts top level fields values from a json body and set them as HTTP header.

## Config
```yaml
mappings:
  - match: .*     # which URL to apply this mapping to (default: match all)
    property: foo # json property to extract
    header: Bar   # header to set
```