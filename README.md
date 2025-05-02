## PubSub service

### What is it
Another publisher-subscriber protocol wrapped up into microservice
using grpc.

### Usage
```shell
git clone https://github.com/Kry0z1/subpub.git
cd subpub
CONFIG_PATH=./config/local.yaml go run .
```

And just like that server will be started and ran on `localhost:15000`.

You can change config by changing `CONFIG_PATH` variable or
config directly in [config](./config) folder. 

### Testing
```shell
go test ./tests -v -race -timeout 30s
```

### More on insides
This microservice is based on package inside [subpub](./pkg/subpub).
To understand how it works look inside, [README.md](./pkg/subpub/README.md)
is really extensive and code is commented heavily.

### Used patterns
 - DI: used all over the place in server part
 - Graceful shutdown: both microservice and subpub package
can die gracefully
 - Middlewares: as interceptors of grpc server
 - Publish-Subscribe: kinda obvious
 - Property-based testing: tested subpub according to properties
defined in [subpub](./pkg/subpub/README.md)