### To run the server from the project root:
```bash 
go run cmd/server/server.go
```

### To run the client:

```bash 
go run cmd/client/client.go
```

### To run the unit test:
```bash 
go test github.com/meddion/gorilla-ws/pkg/...
```

### The .env file should be located at root, with the following content (defaults included):
`API_KEY    = <API_KEY>`

`API_SECRET = <API_SECRET>`

`URL        = wss://testnet.bitmex.com`

`VERB       = GET`

`ENDPOINT   = /realtime`
