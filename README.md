## Usage
```bash
NAME:
   pgold - ICBC paper gold trader assist system

USAGE:
   main.exe [global options] command [command options] [arguments...]

VERSION:
   0.1.0

COMMANDS:
     market     Crawl market data into database continuously
     export     Export market data from database into file
     import     Import market data from file into database
     autosave   Autosave market data into file daily
     server     Run http server showing market history data
     realtime   Email trade tips continuously based on policy
     test       Loopback test strategy using history data
     multitask  Run serveral tasks simultaneously
     dpgen      Generate deep training data
     help, h    Shows a list of commands or help for one command

GLOBAL OPTIONS:
   --config FILE, -c FILE  load configuration from FILE
   --help, -h              show help
   --version, -v           print the version
```