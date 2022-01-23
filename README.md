### To run the server from the project root:
go run cmd/server/server.go

### To run the client:
go run cmd/client/client.go

### To run the unit test:
go test github.com/meddion/gorilla-ws/pkg/...

### The .env file should be located at root, with the following content (defaults included):
API_KEY    = <API_KEY>
API_SECRET = <API_SECRET>
URL        = wss://testnet.bitmex.com
VERB       = GET
ENDPOINT   = /realtime
