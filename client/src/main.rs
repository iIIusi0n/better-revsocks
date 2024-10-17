mod socks;

use tokio::net::TcpStream;
use tokio::io::{AsyncRead, AsyncWrite, AsyncWriteExt};
use tokio_util::compat::{TokioAsyncReadCompatExt, FuturesAsyncReadCompatExt};
use yamux::{Config, Connection, Mode};
use clap::Parser;
use log::{info, error};

use socks::SOCKClient;

type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;

const MAGIC_BYTES: [u8; 4] = [0x1b, 0xc3, 0xbd, 0x0f];

#[derive(Parser, Debug)]
#[command(about = "A reverse SOCKS5 proxy agent")]
struct Args {
    #[arg(short = 'o', long, value_name = "HOST", help = "The host to connect to")]
    host: String,

    #[arg(short, long, value_name = "PORT", help = "The port to connect to")]
    port: u16,

    #[arg(short, help = "Use WebSocket for connection")]
    websocket: bool,

    #[arg(short, help = "Use TLS for connection")]
    tls: bool,
}

#[tokio::main(flavor = "multi_thread")]
async fn main() -> Result<()> {
    let args = Args::parse();

    info!("connecting to {}:{}", args.host, args.port);

    let stream = TcpStream::connect(format!("{}:{}", args.host, args.port)).await?;

    connect_to_agent_server(stream).await
}

async fn connect_to_agent_server<T: AsyncRead + AsyncWrite + Send + Unpin + 'static>(mut stream: T) -> Result<()> {
    stream.write_all(&MAGIC_BYTES).await?;

    let mut conn = Connection::new(stream.compat(), Config::default(), Mode::Server);

    loop {
        let stream = match std::future::poll_fn(|cx| conn.poll_next_inbound(cx)).await {
            Some(Ok(stream)) => stream,
            Some(Err(e)) => {
                eprintln!("error: {:?}", e);
                continue;
            }
            None => return Err("connection closed".into()),
        };

        tokio::spawn(async move {
            let mut client = SOCKClient::new_no_auth(stream.compat(), None);
            match client.init().await {
                Ok(_) => {
                    println!("client connected");
                }
                Err(e) => {
                    eprintln!("error: {:?}", e);
                }
            }
        });
    }
}
