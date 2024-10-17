use tokio::net::TcpStream;
use tokio::io::{AsyncRead, AsyncWrite, AsyncWriteExt};
use tokio_util::compat::{TokioAsyncReadCompatExt, FuturesAsyncReadCompatExt};
use yamux::{Config, Connection, Mode};
use clap::Parser;
use log::{info, error};

use client::SOCKClient;

type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;

const MAGIC_BYTES: [u8; 4] = [0x1b, 0xc3, 0xbd, 0x0f];

#[derive(Parser, Debug)]
struct Args {
    #[arg(short, long)]
    addr: String,

    #[arg(short, long)]
    port: u16,

    #[arg(short, long)]
    websocket: bool,

    #[arg(short, long)]
    tls: bool,
}

#[tokio::main(flavor = "multi_thread")]
async fn main() -> Result<()> {
    let args = Args::parse();

    info!("connecting to {}:{}", args.addr, args.port);

    let stream = TcpStream::connect(format!("{}:{}", args.addr, args.port)).await?;

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
