Web Properties API
==================

This API is used to expose web configuration properties for the Frontend application.

Features
--------

* Individual web properties can be created, updated, and deleted.
* Support for multiple web layouts.

API Endpoints
-------------

* `GET /api/v1/properties` - Get all web properties

Headers:

```json
{
  "X-Client-Id": "71BBE7D8-FD04-459C-6D8B-08D9354E5B64",
  "X-Layout": "el4" // el1, el2, al etc
}
```

Response:

```json
{
  "apiKey": "8LKvrh6C2fe8mKXZ",
  "clientHeader": "71bbe7d8-fd04-459c-6d8b-08d9354e5b64",
  "clientId": "71BBE7D8-FD04-459C-6D8B-08D9354E5B64",
  "clientName": "QA Test Client EL4",
  "currencyCode": "INR",
  "isMaintenance": false,
  "languageCode": "EN",
  "showOddsType": false,
  "showThemeSwitch": false,
  "site": "10cric",
  "theme": "dusky-flycatcher",
  "themeMode": {},
  "viewType": "Euro2"
}
```

Backend
-------

All properties are stored in Azure Storage Accounts `song88webconfig` as JSON files.

Environment-specific configurations are stored in the following containers and directory structure:

* `staging`/`EL1` etc
* `production`/`EL4` etc

The API uses the `X-Layout` header to determine the environment-specific configuration to load.
