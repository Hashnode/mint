## mint

Mint is a Tendermint based blockchain protocol that lets anyone build social apps easily. Mint was created out of a need for efficient data storage on blockchain. It provides you with a simple boilerplate code for building social communities and gets out of your way quickly.

We have also released a front-end (client) of the blockchain which is known as [UpHack](https://uphack.co). Think of it as Hackernews on blockchain.

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

A blockchain network needs validators. We have deployed a testnet with 4 validators of our own and we encourage you to become one and test it out. At any time there will be 21 validators producing blocks. Remember there is no incentive to produce blocks yet. Everything is an experiment here. You should become a validator only if you want to experiment and learn how blockchain networks work.

### Become a validator

To become a validator please follow these steps:

- Email us (sandeep@hashnode.com)
- Spin off a new machine on your favorite cloud (DO, AWS, Google Cloud etc)
- Install MongoDB
- Install Tendermint. See [this](https://github.com/tendermint/tendermint/blob/master/docs/install.rst).
- Install and start Mint. See installation section below.

You should be able to produce blocks now.
