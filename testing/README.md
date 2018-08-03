
Run:

docker run -d -p 27016:27017 -v ~/temp/mongo1:/data/db mongo
docker run -d -p 27015:27017 -v ~/temp/mongo2:/data/db mongo
docker run -d -p 27014:27017 -v ~/temp/mongo3:/data/db mongo
docker run -d -p 27013:27017 -v ~/temp/mongo4:/data/db mongo

Run:
- tendermint testnet --starting-ip-address 192.168.0.1
- get node_ids for each node:
    example: tendermint --home mytestnet/node3/ show_node_id
- update SEEDS variable in launch_testnet_nodes.sh with appropriate node_ids
- run ./launch_testnet_nodes.sh

Reset:
1. either wipe mytestnet folder and redo above steps
2. or do the following:
- go to each node config folder (example /mytestnet/node0)
- remove data folder, addressbook.json
- update priv_validator.json by replaing removing all fields starting with "last_*" and
adding below instead:
"last_height": 0,
"last_round": 0,
"last_step": 0,
