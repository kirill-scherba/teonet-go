FROM ubuntu

WORKDIR /root
COPY udptunnel .
COPY example/ example
RUN apt update && apt install -y iproute2 net-tools iputils-ping

# Build it:
#
#
#   go get github.com/dsnet/udptunnel 
#   cp -r ~/go/src/github.com/dsnet/udptunnel/* ./
#   go get
#   go build
#   docker build --tag udptunnel -f example/Dockerfile .
#
# Run in host:
#
#   sudo ./udptunnel example/server.conf
#
# Run in docker:
#
#   docker run --name udptunnel --cap-add=NET_ADMIN --device /dev/net/tun:/dev/net/tun -it --rm udptunnel bash
#   ./udptunnel example/client.conf
#
# Run in another host terminal
#   
#   ping 10.0.0.2   
#   docker exec -it udptunnel bash:
#   ping 10.0.0.1