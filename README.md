![](https://cdn.hashnode.com/res/hashnode/image/upload/v1527769914653/Hy7BiDakm.jpeg)

## mint

Mint is a Tendermint based blockchain protocol that lets anyone build social apps easily. Mint was created out of a need for efficient data storage on blockchain. It provides you with a simple boilerplate code for building social communities and gets out of your way quickly.

We have also released a front-end (client) of the blockchain which is known as [Uphack](http://uphack.co). Think of it as Hackernews on blockchain.

This is one of the many experiments we have been doing at Hashnode. Although it's super early, we have released the codebase to get initial feedback from the community and improve further.

## Alpha Software

This is an alpha software and shouldn't be used in production. You should use this software to experiment and learn more about blockchain.

## More Details

- This repo provides you with a boilerplate code for writing your own blockchain that stores JSON documents.
- It's based on [Tendermint BFT consensus](https://tendermint.com/).
- You will most likely add some sort of consensus like PoS (Proof of Stake) to this implementation for production deployment.
- Use this to build social apps that rely on data storage.
- Experimental. So, use with caution.

## Contribute

A blockchain network needs validators. We have deployed a [demo app](http://uphack.co) with 4 validators of our own and we encourage you to become one and test it out. At any time there will be 21 validators producing blocks. Remember there is no incentive to produce blocks yet. You should become a validator only if you want to experiment and learn how blockchain networks work.

### Become a validator

To become a validator please follow these steps.

- **Step 1**  
  Email us (sandeep@hashnode.com). Once we are ready, we'll add you to a Telegram channel where we'll collaborate.

- **Step 2**  
  Spin off a new machine on your favorite cloud (DO, AWS, Google Cloud etc). It must be running on Ubuntu and should have at least 4GB RAM and 30GB disk space.

- **Step 3**  
  Install Tendermint. [Check this guide](https://github.com/tendermint/tendermint/blob/master/docs/install.rst) to get started.

- **Step 4**  
  Install MongoDB. You will maintain the global state of the blockchain here.

- **Step 5**  
  Install Mint. Follow the steps below.
  - `cd` into your `$GOPATH/src`. Run `git clone https://github.com/Hashnode/mint`. This should clone the source code into `mint` directory.
  - Install [dep](https://github.com/golang/dep), a tool to manage dependencies in Go.
  - Run `cd mint && dep ensure`
  - Run `go install mint`

  Now `mint` should be available as a global binary. Note: You may have to set `GOBIN` variable for `go install` to work.

- **Step 6**  
  Now run `tendermint init`. This should create a few config files inside `~/.tendermint/config`. Now run `cat ~/.tendermint/config/genesis.json` and copy your generated public key. You need to give it to us in our Telegram channel. All other validators will update their `genesis.json` with your public key. Once everyone has updated their `genesis.json`, we'll give you a copy of it. You need to ssh into your machine and replace `~/.tendermint/config/genesis.json` with the new copy.

  Now open up `~/.tendermint/config/config.toml` and fill out the following details:

  - **moniker**: Enter a name for your validator node
  - **seed**: Paste `3f0d69a741e1cd399c5c2ca38d9f9711135e7a53@206.189.125.145:46656,0ee5713d18a6127dbeac10107860ef1c30edcfb9@192.241.232.63:46656,9446a039f0e2c1cd9a838bfb541f09e910a113ad@159.203.31.67:46656`. Tendermint will connect to these peers and start gossiping.
  - **persistent_peers**: Paste the content of `seeds`. Tendermint will maintain persistent connections with these peers.

  Save the file and exit.

  Now run `mint` and `tendermint` to finish set up. Make sure MongoDB is running at this point.

  - `cd ~`
  - `nohup mint >mint.log 2>&1 &`
  - `nohup tendermint node --consensus.create_empty_blocks=false >tendermint.log 2>&1 &`

  `nohup` will make sure that the processes are running even after you have logged out of SSH session and closed the terminal. The logs will be written to `mint.log` and `tendermint.log`.

  Now, you should be able to produce blocks and take part in consensus.


If you want to be a non-validating peer (which means you don't want to take part in consensus), you can do so by following the steps above (Step 2 onwards). However, the content of your `genesis.json` will be different. As it should contain the rest of the validators, you can paste the following into your `genesis.json`:

 ```
 {
  "genesis_time": "2018-06-01T06:56:45.810497687Z",
  "chain_id": "mint-test",
  "validators": [
    {
      "pub_key": {
        "type": "AC26791624DE60",
        "value": "4BLVMK+pB9ogowU2qxSH54H/eMdS2JLBmeGsUi3HsMg="
      },
      "power": 10,
      "name": ""
    },
    {
      "pub_key": {
        "type": "AC26791624DE60",
        "value": "kuaknLaXXOqPvUJa9O42HQ4dah3lpwdetRgud7Yb5jA="
      },
      "power": 10,
      "name": ""
    },
    {
      "pub_key": {
        "type": "AC26791624DE60",
        "value": "T/1Jn1K1vR7CWfNyU6P/t2D4pYLUr3FSyijuqmHjEkA="
      },
      "power": 10,
      "name": ""
    },
    {
      "pub_key": {
        "type": "AC26791624DE60",
        "value": "OSAe1dE/OYFxvMOK+NDraQ6EXOWxhYlup/IUPyjmoGA="
      },
      "power": 10,
      "name": ""
    }
  ],
  "app_hash": ""
}
 ```

As soon as you run `tendermint` and `mint`, you will start receiving blocks and the latest state will be saved in MongoDB. If you want to check the state, open up `mongo` shell and use `tendermintdb` to explore the collections.


### Contributing Code

If you want to improve the code and want to offer feedback, feel free to send a PR. The whole purpose of open sourcing the repo at such an early stage is to get feedback and improve the code.

The front-end for the blockchain is located at uphack.co and the corresponding code is available [here](https://github.com/Hashnode/Uphack).

---

### Do you think your blockchain product needs Mint?

Let's talk. Shoot an email to one of the following emails: 

- **Sandeep Panda** (sandeep@hashnode.com)
- **Mint team** (mint@hashnode.com)
