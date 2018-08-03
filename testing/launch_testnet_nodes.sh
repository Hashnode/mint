# Launch multiple bash terminal tab windows by running the command `bash launch.sh`

#!/bin/bash
# File: ~/launch.sh

# Original Code Reference: http://dan.doezema.com/2013/04/programmatically-create-title-tabs-within-the-mac-os-x-terminal-app/
# New-BSD License by Original Author Daniel Doezema http://dan.doezema.com/licenses/new-bsd/

# Modified by Luke Schoen in 2017 to include loading new tabs such as client and server and automatically open webpage in browser.
# Modified by Luke Schoen in 2018 to include launching multiple tabs for Tendermint Testnet Nodes

function new_tab() {
  TAB_NAME=$1
  DELAY=$2
  COMMAND=$3
  osascript \
    -e "tell application \"Terminal\"" \
    -e "tell application \"System Events\" to keystroke \"t\" using {command down}" \
    -e "do script \"$DELAY; printf '\\\e]1;$TAB_NAME\\\a'; $COMMAND\" in front window" \
    -e "end tell" > /dev/null
}

IP=0.0.0.0
AA=tcp://$IP
# SEEDS=""
SEEDS="72f768fd9e76271c43d05ce7fb32c367ac645f68@$IP:46656,5afaaa9876b8dc0628e244cb59163472ea8e0123@$IP:46666,1e467deae3d559bfc747dc9acd14a40dc00ef933@$IP:46676,47cf808a11961ac3797958042d81874d3ede9892@$IP:46686"
TESTNET_ROOT_DIR="~/go-workspace/src/mint/testing/"
TESTNET_FOLDER="mytestnet"
NODE_0_NAME="node0"
NODE_1_NAME="node1"
NODE_2_NAME="node2"
NODE_3_NAME="node3"
echo "Removing Previous Tendermint Testnet Files: $TESTNET_ROOT_DIR/$TESTNET_FOLDER"
rm -rf "$TESTNET_ROOT_DIR/$TESTNET_FOLDER"
echo "Tendermint Testnet Location: $TESTNET_ROOT_DIR/$TESTNET_FOLDER"
echo "Loading Nodes: $NODE_0_NAME, $NODE_1_NAME, $NODE_2_NAME, $NODE_3_NAME"
echo "Loading Seeds: $SEEDS"

new_tab "node_0" \
        "echo 'Loading node_0...'" \
        "bash -c 'echo node_0; tendermint node --home "$TESTNET_ROOT_DIR/$TESTNET_FOLDER/$NODE_0_NAME" --rpc.laddr="$AA:46657" --p2p.laddr="$AA:46656" --p2p.seeds=$SEEDS --proxy_app="tcp://127.0.0.1:46658" --p2p.persistent_peers=""; exec $SHELL'"

new_tab "mint_0" \
        "echo 'Loading mint_0...'" \
        "bash -c 'echo mint_0; cd ~/go-workspace/src/mint; ./mint 46658 'localhost:27013'; exec $SHELL'"

new_tab "node_1" \
        "echo 'Loading node_1...'" \
        "bash -c 'echo node_1; tendermint node --home "$TESTNET_ROOT_DIR/$TESTNET_FOLDER/$NODE_1_NAME" --rpc.laddr="$AA:46667" --p2p.laddr="$AA:46666" --p2p.seeds=$SEEDS --proxy_app="tcp://127.0.0.1:46668" --p2p.persistent_peers=""; exec $SHELL'"

new_tab "mint_1" \
        "echo 'Loading mint_1...'" \
        "bash -c 'echo mint_1; cd ~/go-workspace/src/mint; ./mint 46668 'localhost:27014'; exec $SHELL'"

new_tab "node_2" \
        "echo 'Loading node_2...'" \
        "bash -c 'echo node_2; tendermint node --home "$TESTNET_ROOT_DIR/$TESTNET_FOLDER/$NODE_2_NAME" --rpc.laddr="$AA:46677" --p2p.laddr="$AA:46676" --p2p.seeds=$SEEDS --proxy_app="tcp://127.0.0.1:46678" --p2p.persistent_peers=""; exec $SHELL'"

new_tab "mint_2" \
        "echo 'Loading mint_2...'" \
        "bash -c 'echo mint_2; cd ~/go-workspace/src/mint; ./mint 46678 'localhost:27015'; exec $SHELL'"

new_tab "node_3" \
        "echo 'Loading node_3...'" \
        "bash -c 'echo node_3; tendermint node --home "$TESTNET_ROOT_DIR/$TESTNET_FOLDER/$NODE_3_NAME" --rpc.laddr="$AA:46687" --p2p.laddr="$AA:46686" --p2p.seeds=$SEEDS --proxy_app="tcp://127.0.0.1:46688" --p2p.persistent_peers=""; exec $SHELL'"

new_tab "mint_3" \
        "echo 'Loading mint_3...'" \
        "bash -c 'echo mint_3; cd ~/go-workspace/src/mint; ./mint 46688 'localhost:27016'; exec $SHELL'"
