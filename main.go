package main

import (
	"crypto/sha256"
	"encoding/binary"
	"encoding/hex"
	"encoding/json"
	"flag"
	"log"
	"math"
	"net"
	"strings"
	"sync"
	"time"

	"github.com/valyala/fasthttp"
)

var (
	processes = new(sync.Map)

	client = &fasthttp.Client{
		MaxConnDuration: time.Second * 30,
		ReadTimeout:     time.Second * 30,
		WriteTimeout:    time.Second * 30,
		Dial: func(addr string) (net.Conn, error) {
			return fasthttp.DialTimeout(addr, time.Second*5)
		},
	}

	ADDRESS = "Dvhg47J4J2ZgAujAZEJh4PihbWqbyR5BeUKJccNcs7QjC"
	WORKERS = 4

	NODE_URL = "https://api.upow.ai/"
)

func getTransactionsMerkleTree(transactions []string) string {

	var fullData []byte

	for _, transaction := range transactions {
		data, _ := hex.DecodeString(transaction)
		fullData = append(fullData, data...)
	}

	hash := sha256.New()
	hash.Write(fullData)

	return hex.EncodeToString(hash.Sum(nil))
}

func checkBlockIsValid(blockContent []byte, chunk string, idifficulty int, charset string, hasDecimal bool) bool {

	hash := sha256.New()
	hash.Write(blockContent)

	blockHash := hex.EncodeToString(hash.Sum(nil))

	if hasDecimal {
		return strings.HasPrefix(blockHash, chunk) && strings.Contains(charset, string(blockHash[idifficulty]))
	} else {
		return strings.HasPrefix(blockHash, chunk)
	}
}

func worker(start int, step int, res MiningInfoResult) {

	var difficulty float64 = res.Difficulty
	var idifficulty int = int(difficulty)

	_, decimal := math.Modf(difficulty)

	lastBlock := res.LastBlock
	if lastBlock.Hash == "" {
		var num uint32 = 30_06_2005

		data := make([]byte, 32)
		binary.LittleEndian.PutUint32(data, num)

		lastBlock.Hash = hex.EncodeToString(data)
	}

	chunk := lastBlock.Hash[len(lastBlock.Hash)-idifficulty:]

	charset := "0123456789abcdef"
	if decimal > 0 {
		count := math.Ceil(16 * (1 - decimal))
		charset = charset[:int(count)]
	}

	addressBytes := stringToBytes(ADDRESS)
	t := float64(time.Now().UnixMicro()) / 1000000.0
	i := start
	a := time.Now().Unix()
	txs := res.PendingTransactionsHashes
	merkleTree := getTransactionsMerkleTree(txs)

	if start == 0 {
		log.Printf("Difficulty: %f\n", difficulty)
		log.Printf("Block number: %d\n", lastBlock.Id)
		log.Printf("Confirming %d transactions\n", len(txs))
	}

	var prefix []byte
	dataHash, _ := hex.DecodeString(lastBlock.Hash)
	prefix = append(prefix, dataHash...)
	prefix = append(prefix, addressBytes...)
	dataMerkleTree, _ := hex.DecodeString(merkleTree)
	prefix = append(prefix, dataMerkleTree...)
	dataA := make([]byte, 4)
	binary.LittleEndian.PutUint32(dataA, uint32(a))
	prefix = append(prefix, dataA...)
	dataDifficulty := make([]byte, 2)
	binary.LittleEndian.PutUint16(dataDifficulty, uint16(difficulty*10))
	prefix = append(prefix, dataDifficulty...)

	if len(addressBytes) == 33 {
		data1 := make([]byte, 2)
		binary.LittleEndian.PutUint16(data1, uint16(2))

		oldPrefix := prefix
		prefix = data1[:1]
		prefix = append(prefix, oldPrefix...)
	}

	for {
		var _hex []byte

		found := true
		check := 5000000 * step

	checkLoop:
		for {
			if process, ok := processes.Load(start); !ok || !process.(Goroutine).Alive {
				return
			}

			_hex = _hex[:0]
			_hex = append(_hex, prefix...)
			dataI := make([]byte, 4)
			binary.LittleEndian.PutUint32(dataI, uint32(i))
			_hex = append(_hex, dataI...)

			if checkBlockIsValid(_hex, chunk, idifficulty, charset, decimal > 0) {
				break checkLoop
			}

			i = i + step
			if (i-start)%check == 0 {
				elapsedTime := float64(time.Now().UnixMicro())/1000000.0 - t
				log.Printf("Worker %d: %dk hash/s", start+1, i/step/int(elapsedTime)/1000)

				if elapsedTime > 90 {
					found = false
					break checkLoop
				}
			}
		}

		if found {
			var reqP PushBlock

			log.Println(hex.EncodeToString(_hex))

			req := POST(
				NODE_URL+"push_block",
				map[string]interface{}{
					"block_content": hex.EncodeToString(_hex),
					"txs":           txs,
					"block_no":            lastBlock.Id + 1,
				},
			)
			_ = json.Unmarshal(req.Body(), &reqP)

			if reqP.Ok {
				log.Println("BLOCK MINED")
			}

			stopWorkers()
			return
		}
	}
}

func main() {

	flag.StringVar(&ADDRESS, "address", ADDRESS, "address that'll receive mining rewards")
	flag.IntVar(&WORKERS, "workers", WORKERS, "number of concurrent workers to spawn")
	flag.StringVar(&NODE_URL, "node", NODE_URL, "node to which we'll retrieve mining info")

	flag.Parse()

	for {
		log.Printf("Starting %d workers", WORKERS)

		var reqP MiningInfo

		req := GET(NODE_URL+"get_mining_info", map[string]interface{}{})
		_ = json.Unmarshal(req.Body(), &reqP)

		for _, i := range createRange(1, WORKERS) {
			log.Printf("Starting worker n.%d", i)
			go worker(i-1, WORKERS, reqP.Result)

			processes.Store(i-1, Goroutine{Id: i - 1, Alive: true, StartedAt: time.Now().Unix()})
		}

		elapsedSeconds := 0

	waitLoop:
		for allAliveWorkers() {
			time.Sleep(1 * time.Second)
			elapsedSeconds += 1

			if elapsedSeconds > 180 {
				stopWorkers()
				break waitLoop
			}
		}
	}
}
