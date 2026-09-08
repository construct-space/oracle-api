# Oracle API

Fresh Oracle backend API.

The old mixed frontend/backend implementation was moved to `../oracle_old` so this service can stay API-only.

Currently implemented:

- administrator auth (`/api/auth/*`)
- administrators list/create (`/api/administrators`)
- accounts read models used by the fresh `../../oracle/` frontend
- account admin proxy mutations (`/api/accounts/admin/*`)
- health endpoint
