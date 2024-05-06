package main

import (
	"bufio"
	"encoding/hex"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/pkg/errors"
	"github.com/stellar/go/clients/horizonclient"
	"github.com/stellar/go/keypair"
	"github.com/stellar/go/network"
	"github.com/stellar/go/txnbuild"
)

var (
	inputFile    string
	outputFile   string
	preview      bool
	submit       bool
	walletsecret string
)

func init() {
	flag.StringVar(&inputFile, "inputfile", "payouts_to_sign.txt", "File with transactions to sign")
	flag.StringVar(&outputFile, "outputfile", "payouts_signed.txt", "File to place signed transactions")
	flag.BoolVar(&preview, "preview", false, "Print transactions before signing")
	flag.BoolVar(&submit, "submit", false, "Submit transactions to the network after signing")
	flag.StringVar(&walletsecret, "wallet-secret", "", "Secret key of the wallet to sign the transactions with")
}

func main() {
	flag.Parse()

	if walletsecret == "" {
		panic("wallet secret is required")
	}

	wallet, err := newWallet(walletsecret)
	if err != nil {
		panic("wallet secret is not valid")
	}

	horizon := horizonclient.DefaultPublicNetClient

	infile, err := os.Open(inputFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to open payouts input file %s", err))
	}
	reader := bufio.NewReader(infile)
	defer infile.Close()

	outFile, err := os.Create(outputFile)
	if err != nil {
		panic(fmt.Sprintf("Failed to open payouts output file %s", err))
	}
	defer outFile.Close()
	defer outFile.Sync()

	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			if errors.Is(err, io.EOF) {
				break
			}
			panic(err)
		}
		if line == "" {
			break
		}

		xdr := strings.TrimSpace(line)

		txn := &txnbuild.Transaction{}
		if err = txn.UnmarshalText([]byte(xdr)); err != nil {
			panic("Cant unmarshal transaction")
		}

		if err = basicValidation(txn); err != nil {
			panic("transaction validation failed")
		}

		if preview {
			previewTxn(txn)
			continue
		}

		txn, err = wallet.Sign(txn)
		if err != nil {
			panic("could not sign transaction")
		}

		if submit {
			for {
				previewTxn(txn)
				_, err := horizon.SubmitTransaction(txn)
				if err != nil {
					if err, ok := err.(*horizonclient.Error); ok {
						// Horizon timeout
						if err.Response.StatusCode == 504 {
							fmt.Println("Horizon timeout, retry request in 15 seconds")
							time.Sleep(time.Second * 15)
							continue
						} else if err.Response.StatusCode == 404 {
							fmt.Println("ERROR: Account does not exist", err)
							break
						}
						if resCodes, ok := err.Problem.Extras["result_codes"]; ok {
							if resMap, ok := resCodes.(map[string]string); ok {
								if v, exists := resMap["operations"]; exists {
									if strings.Contains(v, "tx_insufficient_fee") {
										fmt.Println("Tx insufficient fee, retry in 30 seconds")
										time.Sleep(time.Second * 30)
										continue
									} else if strings.Contains(v, "op_no_destination") {
										fmt.Println("ERROR: Account does not exsit")
										break
									}
								}

							}
						}
						fmt.Println(err)
						time.Sleep(time.Second * 60)
						continue

					}
					fmt.Println("ERROR", err)
					time.Sleep(time.Second * 60)
					continue
				}
				break
			}
		} else {
			xdr, err := txn.MarshalText()
			if err != nil {
				panic(err)
			}
			outFile.WriteString(string(xdr))
			outFile.WriteString("\n")
		}
	}

}

func basicValidation(txn *txnbuild.Transaction) error {
	if _, ok := txn.Memo().(txnbuild.MemoHash); !ok {
		if _, ok = txn.Memo().(txnbuild.MemoReturn); !ok {
			return errors.New("memo must be of type memo_hash or memo_return")
		}
	}
	if len(txn.Operations()) != 1 {
		return errors.New("transaction must contain a single operation")
	}
	if _, ok := txn.Operations()[0].(*txnbuild.Payment); !ok {
		return errors.New("transaction operation must be of type Payment")
	}
	return nil
}

func previewTxn(txn *txnbuild.Transaction) {
	rawMemo := txn.Memo().(txnbuild.MemoHash)
	memo := hex.EncodeToString(rawMemo[:])
	sequence := txn.SequenceNumber()
	paymentOp := txn.Operations()[0].(*txnbuild.Payment)
	asset := paymentOp.Asset
	amount := paymentOp.Amount
	dest := paymentOp.Destination
	sigs := len(txn.Signatures())
	fmt.Printf("Sending %s %s to %s with memo %s (%d signatures, %d seqno)\n", amount, asset, dest, memo, sigs, sequence)
}

type Wallet struct {
	keypair *keypair.Full
}

func newWallet(privateKey string) (*Wallet, error) {
	kp, err := keypair.ParseFull(privateKey)
	if err != nil {
		return nil, errors.Wrap(err, "could not parse private key")
	}
	return &Wallet{keypair: kp}, nil
}

func (w *Wallet) Sign(txn *txnbuild.Transaction) (*txnbuild.Transaction, error) {
	return txn.Sign(network.PublicNetworkPassphrase, w.keypair)
}
