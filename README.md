# BlueX Auth0 Server

A lightweight mock authentication server that simulates Auth0 integration for BlueX applications.

## Prerequisites

- Go (latest version)
- [ngrok](https://ngrok.com/downloads/mac-os) for exposing local services
- Make

## Installation

1. Clone the repository
2. Create `.env` file from `.env.example`:
```bash
cp .env.example .env
```

3. Configure the following environment variables in `.env`:
```config
AUTH0_CLIENT_ID=     # Your Auth0 application client ID
AUTH0_CLIENT_SECRET= # Your Auth0 application client secret
AUTH0_CLIENT_EMAIL=  # Your Auth0 application email
```

## Build and Run

Build and start the server:
```bash
make
./bin/server -c config/server.yml server
```

The server will start on port 4080 by default.

## Exposing the Service

To expose your local service to the internet using ngrok:

1. Install ngrok:
```bash
brew install ngrok
```

2. Configure ngrok with your auth token:
```bash
ngrok config add-authtoken <your-auth-token>
```

3. Start ngrok tunnel:
```bash
ngrok http 4080
```

4. Copy the generated HTTPS URL (e.g., `https://xxxx-127-0-0-1.ngrok-free.app`) and update your `.env`:
```config
ENDPOINT=https://xxxx-127-0-0-1.ngrok-free.app
```

5. Restart the server for changes to take effect

## Testing

Verify the server is running correctly by accessing the JWKS endpoint:

```bash
# Test local endpoint
curl http://localhost:4080/.well-known/jwks.json

# Test public endpoint (replace with your ngrok URL)
curl https://xxxx-127-0-0-1.ngrok-free.app/.well-known/jwks.json
```

## API Documentation

The server exposes the following endpoints:
- `/.well-known/jwks.json` - JWKS (JSON Web Key Set) endpoint
- Additional endpoints documentation coming soon

## Contributing

Please read our contributing guidelines before submitting pull requests.

## License

This project is proprietary software of BlueX.
