use tokio::net::TcpStream;
use tokio::io::{AsyncRead, AsyncWrite, AsyncWriteExt};
use tokio_util::compat::{TokioAsyncReadCompatExt, FuturesAsyncReadCompatExt};
use yamux::{Config as YamuxConfig, Connection, Mode};
use tokio_native_tls::TlsConnector;
use log::{info, error};

use crate::config::Config;
use crate::error::Result;
use crate::socks::SOCKClient;

const MAGIC_BYTES: [u8; 4] = [0x1b, 0xc3, 0xbd, 0x0f];

trait AsyncStream: AsyncRead + AsyncWrite {}
impl<T: AsyncRead + AsyncWrite> AsyncStream for T {}

pub struct ReverseProxyClient {
    config: Config,
}

impl ReverseProxyClient {
    pub fn new(config: Config) -> Self {
        Self { config }
    }

    pub async fn run(&self) -> Result<()> {
        info!("connecting to {}:{}", self.config.host, self.config.port);
        
        let stream = self.establish_connection().await?;
        self.handle_connection(stream).await
    }

    async fn establish_connection(&self) -> Result<Box<dyn AsyncStream + Send + Unpin + 'static>> {
        let stream = TcpStream::connect(format!("{}:{}", self.config.host, self.config.port)).await?;

        if self.config.tls {
            info!("using TLS connection");
            let mut builder = native_tls::TlsConnector::builder();
            builder.danger_accept_invalid_certs(true);
            let tls = TlsConnector::from(builder.build()?);
            Ok(Box::new(tls.connect(&self.config.host, stream).await?))
        } else {
            Ok(Box::new(stream))
        }
    }

    async fn handle_connection<T: AsyncRead + AsyncWrite + Send + Unpin + 'static>(&self, mut stream: T) -> Result<()> {
        stream.write_all(&MAGIC_BYTES).await?;

        let mut conn = Connection::new(stream.compat(), YamuxConfig::default(), Mode::Server);

        loop {
            let stream = match std::future::poll_fn(|cx| conn.poll_next_inbound(cx)).await {
                Some(Ok(stream)) => stream,
                Some(Err(e)) => {
                    error!("Connection error: {:?}", e);
                    continue;
                }
                None => return Err("connection closed".into()),
            };

            tokio::spawn(async move {
                let mut client = SOCKClient::new_no_auth(stream.compat(), None);
                match client.init().await {
                    Ok(_) => info!("client connected"),
                    Err(e) => error!("client error: {:?}", e),
                }
            });
        }
    }
} 