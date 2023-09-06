# Signer

This is a small tool to sign threefold minting payout files. A minting
payout file is a line separated file, containing base64 XDR encoded
transactions. The output is a similar file, with a signature added on
each input transaction.

## Building

Building requires a recent version of the `go` compiler to be installed on
the system. It is assumed that `git` is used to get the code

```bash
git clone https://github.com/leesmet/signer
cd signer
go build
```

## Running

Once a local copy is available, you can run the code as follows

```bash
./signer -inputfile <FILE_WITH_TRANSACTIONS_TO_SIGN> -outputfile
<FILE_SIGNED_BY_ME_MYSELF_AND_I> -wallet-secret <YOURSTELLARSECRET>
```

The `-h` flag can be used to see the available options.

### Submitting transactions

If you are the last required signer, the `-submit` flag can be used to
also submit the transactions to the stellar network.

## Disclaimer

As always when providing secrets to an application, you should manually
review the code, and take other recommended precautions, to make sure no
funny business is happening.

## License

This project is licensed under the MIT license, available in ./LICENSE
