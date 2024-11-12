mod socks;
mod config;
mod client;
mod error;

use crate::config::Config;
use crate::client::ReverseProxyClient;
use crate::error::Result;

use log::info;
use clap::Parser;

#[tokio::main(flavor = "multi_thread")]
async fn main() -> Result<()> {
    let config = Config::parse();
    
    let client = ReverseProxyClient::new(config);
    client.run().await
}
