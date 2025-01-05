## Portsicle Server

- Portsicle is a free and open-source Ngrok alternative to expose local servers online.

- [Portsicle client](https://github.com/portsicle/portsicle-client) allows you to use the Portsicle Server via CLI.

- When a client connects to the portsicle server, it gets a unique public url in response.

- The client can use this URL to access the local server on public network!

## Usage

#### Refer the guide provided on [Portsicle client](https://github.com/portsicle/portsicle-client?tab=readme-ov-file#guide).

## How Portsicle Works

### Request Cycle:

1. The portsicle server converts the incoming HTTP request into a WebSocket message and forwards this message to the client via the WebSocket tunnel.

2. Upon receiving the message, the client converts it back into an HTTP request and sends it to the local server.

![request cycle](https://github.com/user-attachments/assets/96581c84-7583-42a7-b965-3f7bcae9edaa)

### Response Cycle:

1. When the local server sends back an HTTP response to the client, the client converts this response into a WebSocket message and forwards it to the server via the WebSocket tunnel.

2. Upon receiving the message, the server converts it back into an HTTP response and forwards it to the public endpoint.

![response cycle](https://github.com/user-attachments/assets/8c77844e-144a-472c-af29-051e5f302e06)
