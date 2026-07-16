Web Properties API
==================

This API is used to expose web configuration properties for the Frontend application.

Features
--------

* Individual web properties can be created, updated, and deleted.
* Support for multiple web layouts.

API Endpoints
-------------

* `GET /api/v1/config/{clientId}` - Get configuration for a client
* `POST /api/v1/config/{clientId}` - Update configuration for a client (requires `X-Api-Key`; optional `X-Layout` header targets the layout-specific file; body is the full config JSON)

Response:

```json
{
  "ClientCode": "G01",
  "Features": {
    "BettingRules": true,
    "DarkMode": true,
    "GameTrackerProductNames": [
      "song88-light",
      "song88-dark"
    ],
    "LanguageOptions": true,
    "OGAuth": false,
    "PartialCashout": true
  },
  "Layout": {
    "AL": {
      "Themes": [
        "red",
        "gray",
        "blue",
        "orange-light",
        "orange-dark"
      ]
    },
    "EL2": {
      "Themes": [
        "red",
        "white"
      ]
    }
  }
}
```

Backend
-------

All properties are stored in AWS S3 Buckets: `song88-{env}-webprops` as JSON files.
