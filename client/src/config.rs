use clap::Parser;

#[derive(Parser, Debug)]
#[command(about = "a reverse socks5 proxy agent", name = "client")]
pub struct Config {
    #[arg(help = "The host to connect to", value_name = "host")]
    pub host: String,

    #[arg(help = "The port to connect to", value_name = "port")]
    pub port: u16,

    #[arg(long, help = "Use TLS for connection", value_name = "tls", action = clap::ArgAction::SetTrue)]
    pub tls: bool,

    #[arg(long, help = "Use Tor for connection", value_name = "tor", action = clap::ArgAction::SetTrue)]
    pub tor: bool,
} 