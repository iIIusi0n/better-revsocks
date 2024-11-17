use std::io;
use std::net::{Ipv4Addr, SocketAddr, SocketAddrV4};
use tokio::io::{AsyncRead, AsyncReadExt, AsyncWrite, AsyncWriteExt};
use tokio::net::{TcpStream, lookup_host};

const SOCKS_VERSION: u8 = 0x05;
const NO_AUTH: u8 = 0x00;
const CONNECT: u8 = 0x01;
const IPV4: u8 = 0x01;
const DOMAIN: u8 = 0x03;

pub struct SocksServer<T: AsyncRead + AsyncWrite + Unpin> {
    stream: T,
}

impl<T: AsyncRead + AsyncWrite + Unpin> SocksServer<T> {
    pub fn new(stream: T) -> Self {
        Self { stream }
    }

    pub async fn handle(&mut self) -> io::Result<()> {
        // Handle auth
        let mut header = [0u8; 2];
        self.stream.read_exact(&mut header).await?;
        
        if header[0] != SOCKS_VERSION {
            return Ok(());
        }
        
        let mut methods = vec![0u8; header[1] as usize];
        self.stream.read_exact(&mut methods).await?;
        
        // Send auth response (no auth)
        self.stream.write_all(&[SOCKS_VERSION, NO_AUTH]).await?;

        // Handle request
        let mut req_header = [0u8; 4];
        self.stream.read_exact(&mut req_header).await?;

        if req_header[1] != CONNECT {
            return Ok(());
        }

        // Parse address
        let addr = match req_header[3] {
            IPV4 => {
                let mut addr = [0u8; 4];
                self.stream.read_exact(&mut addr).await?;
                let addr = SocketAddr::V4(SocketAddrV4::new(
                    Ipv4Addr::new(addr[0], addr[1], addr[2], addr[3]),
                    0,
                ));
                vec![addr]
            }
            DOMAIN => {
                let mut len = [0u8; 1];
                self.stream.read_exact(&mut len).await?;
                let mut domain = vec![0u8; len[0] as usize];
                self.stream.read_exact(&mut domain).await?;
                let domain = String::from_utf8_lossy(&domain);
                lookup_host(format!("{}:0", domain)).await?.collect()
            }
            _ => return Ok(()),
        };

        // Read port
        let mut port = [0u8; 2];
        self.stream.read_exact(&mut port).await?;
        let port = ((port[0] as u16) << 8) | port[1] as u16;

        // Connect to target
        let mut target = TcpStream::connect(format!("{}:{}", addr[0].ip(), port)).await?;

        // Send success response
        let response = [SOCKS_VERSION, 0x00, 0x00, 0x01, 0, 0, 0, 0, 0, 0];
        self.stream.write_all(&response).await?;

        // Start proxying
        tokio::io::copy_bidirectional(&mut self.stream, &mut target).await?;

        Ok(())
    }
}
