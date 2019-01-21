# TCP server
## To start TCP server:
##### Usage: 
    go run main.go -host=[localhost|IP] -port=[Port]
##### Example: 
    go run main.go -host="localhost" -port=9999
## To connet to TCP server:
##### Usage: 
    nc [localhost|IP] [Port]
##### Example: 
    nc localhost 9999
## To look TCP server statistics:
##### Usage: 
    http://[host]:8080/tcp
##### Example: 
    http://localhost:8080/tcp
