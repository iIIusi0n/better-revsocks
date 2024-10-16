use tokio::net::TcpStream;
use tokio_util::compat::{TokioAsyncReadCompatExt, FuturesAsyncReadCompatExt};
use yamux::{Config, Connection, Mode};

use client::SOCKClient;

type Result<T> = std::result::Result<T, Box<dyn std::error::Error>>;

#[tokio::main(flavor = "multi_thread")]
async fn main() -> Result<()> {
    let stream = TcpStream::connect("127.0.0.1:1080").await?;
    let mut conn = Connection::new(stream.compat(), Config::default(), Mode::Server);

    loop {
        let stream = match std::future::poll_fn(|cx| conn.poll_next_inbound(cx)).await {
            Some(Ok(stream)) => stream,
            Some(Err(e)) => {
                eprintln!("Error: {:?}", e);
                continue;
            }
            None => break,
        };

        tokio::spawn(async move {
            let mut client = SOCKClient::new_no_auth(stream.compat(), None);
            match client.init().await {
                Ok(_) => {
                    println!("Client connected");
                }
                Err(e) => {
                    eprintln!("Error: {:?}", e);
                }
            }
        });
    }

    Ok(())
}
