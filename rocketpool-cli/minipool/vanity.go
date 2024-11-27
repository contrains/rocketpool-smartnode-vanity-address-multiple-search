package minipool

import (
	"fmt"
	"math/big"
	"runtime"
	"strings"
	"sync"
	"time"

	"github.com/dustin/go-humanize"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/crypto"
	"github.com/rocket-pool/rocketpool-go/utils/eth"
	"github.com/urfave/cli"

	"github.com/rocket-pool/smartnode/shared/services/rocketpool"
	cliutils "github.com/rocket-pool/smartnode/shared/utils/cli"
)

type targetPrefixT struct {
    prefix *big.Int
    shiftAmount uint
}

type vanityAddressT struct {
    address common.Address
    prefix string
    targetPrefix targetPrefixT
    salt *big.Int
}

func findVanitySalt(c *cli.Context) error {

	// Get RP client
	rp, err := rocketpool.NewClientFromCtx(c).WithReady()
	if err != nil {
		return err
	}
	defer rp.Close()

	// Get the target prefix
	prefixes := c.StringSlice("prefix")
	targetPrefixes := []targetPrefixT{}
	foundAddresses := []vanityAddressT{}

    if len(prefixes) == 0 {
        prefix = cliutils.Prompt("Please specify the address prefix you would like to search for (must start with 0x):", "^0x[0-9a-fA-F]+$", "Invalid hex string")
        append(prefixes, prefix)
    }

	for _, prefix := range prefixes {
	    if prefix == "" {
	        continue
        }

        if !strings.HasPrefix(prefix, "0x") {
            return fmt.Errorf("Prefix must start with 0x: %s", prefix)
        }

        targetPrefix, success := big.NewInt(0).SetString(prefix, 0)
        if !success {
            return fmt.Errorf("Invalid prefix: %s", prefix)
        }
        append(targetPrefixes, targetPrefixT{ prefix: targetPrefix, shiftAmount: uint(42 - len(prefix))})
        append(foundAddresses, vanityAddressT{ prefix: prefix, targetPrefix: targetPrefixes[-1] })
    }

	// Get the starting salt
	saltString := c.String("salt")
	var salt *big.Int
	if saltString == "" {
		salt = big.NewInt(0)
	} else {
		salt, success = big.NewInt(0).SetString(saltString, 0)
		if !success {
			return fmt.Errorf("Invalid starting salt: %s", salt)
		}
	}

	// Get the core count
	threads := c.Int("threads")
	if threads == 0 {
		threads = runtime.GOMAXPROCS(0)
	} else if threads < 0 {
		threads = 1
	} else if threads > runtime.GOMAXPROCS(0) {
		threads = runtime.GOMAXPROCS(0)
	}

	// Get the node address
	nodeAddressStr := c.String("node-address")
	if nodeAddressStr == "" {
		nodeAddressStr = "0"
	}

	// Get deposit amount
	var amount float64
	if c.String("amount") != "" {
		// Parse amount
		if amount, err = cliutils.ValidatePositiveEthAmount("deposit", c.String("amount")); err != nil {
			return err
		}
	} else {
		// Get deposit amount options
		amountOptions := []string{
			"8 ETH",
			"16 ETH",
		}

		// Prompt for amount
		selected, _ := cliutils.Select("Please choose a deposit type to search for:", amountOptions)
		switch selected {
		case 0:
			amount = 8
		case 1:
			amount = 16
		}
	}
	amountWei := eth.EthToWei(amount)

	// Get the vanity generation artifacts
	vanityArtifacts, err := rp.GetVanityArtifacts(amountWei, nodeAddressStr)
	if err != nil {
		return err
	}

	// Set up some variables
	nodeAddress := vanityArtifacts.NodeAddress.Bytes()
	minipoolFactoryAddress := vanityArtifacts.MinipoolFactoryAddress
	initHash := vanityArtifacts.InitHash.Bytes()
	targetPrefixesPtr := &targetPrefixes

	// Run the search
	fmt.Printf("Running with %d threads.\n", threads)

    mu := new(sync.Mutex)
	wg := new(sync.WaitGroup)
	wg.Add(threads)
	stop := false
	stopPtr := &stop

	// Spawn worker threads
	start := time.Now()
	for i := 0; i < threads; i++ {
		saltOffset := big.NewInt(int64(i))
		workerSalt := big.NewInt(0).Add(salt, saltOffset)

		go func(i int) {
			foundSalt, foundAddress, matchingPrefix := runWorker(i == 0, stopPtr, targetPrefixes, nodeAddress, minipoolFactoryAddress, initHash, workerSalt, int64(threads))

			if foundSalt != nil {
				mu.Lock()

                var index big.Int

                for j := range foundAddresses {
                    if foundAddresses[j].targetPrefix.prefix == matchingPrefix.prefix {
                        foundAddresses[j].address := foundAddress
                        foundAddresses[j].salt := foundSalt
                        break
                    }
                }

                for j := range targetPrefixes {
                    if targetPrefixes[j].prefix == matchingPrefix.prefix {
                        *targetPrefixesPtr = append(targetPrefixes[:j], targetPrefixes[j+1:]...)
                        break
                    }
                }

				mu.Unlock()
			}

			if len(targetPrefixes) == 0 {
				*stopPtr = true
			    wg.Done()
            }
		}(i)
	}

	// Wait for the workers to finish and print the elapsed time
	wg.Wait()
	end := time.Now()
	elapsed := end.Sub(start)

	for _, foundAddress := range foundAddresses {
        fmt.Printf("Found for prefix %s: salt 0x%x = %s\n", foundAddress.prefix, foundAddress.salt, foundAddress.address.Hex())
	}
	fmt.Printf("Finished in %s\n", elapsed)

	// Return
	return nil

}

func runWorker(report bool, stop *bool, targetPrefixes []*big.Int, nodeAddress []byte, minipoolManagerAddress common.Address, initHash []byte, salt *big.Int, increment int64) (*big.Int, common.Address, targetPrefixT) {
	saltBytes := [32]byte{}
	hashInt := big.NewInt(0)
	incrementInt := big.NewInt(increment)
	hasher := crypto.NewKeccakState()
	nodeSalt := common.Hash{}
	addressResult := common.Hash{}

	// Set up the reporting ticker if requested
	var ticker *time.Ticker
	var tickerChan chan struct{}
	lastSalt := big.NewInt(0).Set(salt)
	if report {
		start := time.Now()
		reportInterval := 5 * time.Second
		ticker = time.NewTicker(reportInterval)
		tickerChan = make(chan struct{})
		go func() {
			for {
				select {
				case <-ticker.C:
					delta := big.NewInt(0).Sub(salt, lastSalt)
					deltaFloat, suffix := humanize.ComputeSI(float64(delta.Uint64()) / 5.0)
					deltaString := humanize.FtoaWithDigits(deltaFloat, 2) + suffix
					fmt.Printf("At salt 0x%x... %s (%s salts/sec)\n", salt, time.Since(start), deltaString)
					lastSalt.Set(salt)
				case <-tickerChan:
					ticker.Stop()
					return
				}
			}
		}()
	}

	// Run the main salt finder loop
	for {
		if *stop {
			return nil, common.Address{}
		}

		// Some speed optimizations -
		// This block is the fast way to do `nodeSalt := crypto.Keccak256Hash(nodeAddress, saltBytes)`
		salt.FillBytes(saltBytes[:])
		hasher.Write(nodeAddress)
		hasher.Write(saltBytes[:])
		hasher.Read(nodeSalt[:])
		hasher.Reset()

		// This block is the fast way to do `crypto.CreateAddress2(minipoolManagerAddress, nodeSalt, initHash)`
		// except instead of capturing the returned value as an address, we keep it as bytes. The first 12 bytes
		// are ignored, since they are not part of the resulting address.
		//
		// Because we didn't call CreateAddress2 here, we have to call common.BytesToAddress below, but we can
		// postpone that until we find the correct salt.
		hasher.Write([]byte{0xff})
		hasher.Write(minipoolManagerAddress.Bytes())
		hasher.Write(nodeSalt[:])
		hasher.Write(initHash)
		hasher.Read(addressResult[:])
		hasher.Reset()

		for _, targetPrefix := range targetPrefixes {
            hashInt.SetBytes(addressResult[12:])
            hashInt.Rsh(hashInt, targetPrefix.shiftAmount*4)

            if hashInt.Cmp(targetPrefix.prefix) == 0 {
                if report {
                    close(tickerChan)
                }
                address := common.BytesToAddress(addressResult[12:])
                return salt, address, targetPrefix
            }
        }
		salt.Add(salt, incrementInt)
	}
}
