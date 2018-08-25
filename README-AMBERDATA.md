# Collect Internal Messages

Modify geth in order to collect internal messages.

## Deployment

[1] install golang 1.9
[2] git clone https://github.com/amberdata/go-ethereum.git
[3] cd go-ethereum; git checkout --track origin/<branch-name>
[4] run "make geth"
[5] set up env variables according to variables.env.exmaple
[6] start /home/ec2-user/go-ethereum/build/bin/geth like normal (the syncmode full flag is NOT needed because we want to fast sync until GETH_FULL_SYNC_START_BLOCK that we control). For example: /home/ec2-user/go-ethereum/build/bin/geth --cache=2048 --rpcaddr 0.0.0.0 --rpc --rpcapi eth,net,web3,admin --maxpeers......
