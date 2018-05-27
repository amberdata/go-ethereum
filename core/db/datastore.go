package db

import (
	"crypto/tls"
	"crypto/x509"
	"database/sql"
	"encoding/json"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"math/big"
	"os"
	"strconv"
	"time"

	"github.com/Shopify/sarama"
	"github.com/ethereum/go-ethereum/common"
	"github.com/ethereum/go-ethereum/core/types"

	_ "github.com/lib/pq"
)

var enableSaveInternalTxKafka = getEnableSaveInternalTxKafka()

func getEnableSaveInternalTxKafka() bool {
	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
	if flag.Lookup("test.v") != nil {
		return true
	}
	enableSaveInternalTxKafkaString := os.Getenv("GETH_ENABLE_SAVE_INTERNAL_MESSAGE_KAFKA")
	if len(enableSaveInternalTxKafkaString) > 0 {
		enableSaveInternalTxKafka, err := strconv.ParseBool(enableSaveInternalTxKafkaString)
		if err != nil {
			panic(fmt.Sprintf("Cannot parse enableSaveInternalTxKafkaString: %s", enableSaveInternalTxKafkaString))
		}
		fmt.Printf("enableSaveInternalTxKafka = %t\n", enableSaveInternalTxKafka)
		return enableSaveInternalTxKafka
	} else {
		return false
	}
}

var connStr = "host=" + os.Getenv("DATABASE_HOSTNAME") + " port=" + os.Getenv("DATABASE_PORT") + " dbname=" + os.Getenv("DATABASE_NAME") + " user=" + os.Getenv("DATABASE_USERNAME") + " password=" + os.Getenv("DATABASE_PASSWORD") + " sslmode=disable"
var DBO = connectDB(connStr)

func connectDB(connStr string) *sql.DB {
	dbo, err := sql.Open("postgres", connStr)
	common.CheckErr(err, nil)
	return dbo
}

var FullSyncStartBlock = getFullSyncStartBlock()

func getFullSyncStartBlock() (FullSyncStartBlock uint64) {
	fmt.Printf("flag.Lookup(\"test.v\") = %s\n", flag.Lookup("test.v")) // too strange: if this line is removed, one test will fail!
	if flag.Lookup("test.v") != nil {
		FullSyncStartBlock = 0
		return
	}
	err1 := DBO.QueryRow(`SELECT MAX("blockNumber") FROM internal_message`).Scan(&FullSyncStartBlock)
	common.CheckErr(err1, nil)
	FullSyncStartBlockString := os.Getenv("GETH_FULL_SYNC_START_BLOCK")
	if len(FullSyncStartBlockString) > 0 {
		var err error
		FullSyncStartBlock, err = strconv.ParseUint(FullSyncStartBlockString, 10, 64)
		if err != nil {
			panic(fmt.Sprintf("Cannot parse FullSyncStartBlockString: %s", FullSyncStartBlockString))
		}
	}
	fmt.Printf("FullSyncStartBlock = %d\n", FullSyncStartBlock)
	return
}

type Datastore interface {
	SaveInternalTxFromSingleBlock(blockNumber *big.Int, internalTxStore []*types.InternalTx) uint64
}

var (
	// addr      = flag.String("addr", ":8080", "The address to bind to")
	// brokers   = flag.String("brokers", os.Getenv("KAFKA_PEERS"), "The Kafka brokers to connect to, as a comma separated list")
	// verbose   = flag.Bool("verbose", false, "Turn on Sarama logging")
	certFile  = flag.String("certificate", "", "The optional certificate file for client authentication")
	keyFile   = flag.String("key", "", "The optional key file for client authentication")
	caFile    = flag.String("ca", "", "The optional certificate authority file for TLS client authentication")
	verifySsl = flag.Bool("verify", false, "Optional verify ssl certificates chain")
)

type KafkaDatastore struct {
	InternalTxProducer sarama.SyncProducer
}

func NewKafkaDatastore(brokerList []string) *KafkaDatastore {
	return &KafkaDatastore{
		InternalTxProducer: newInternalTxProducer(brokerList),
	}
}
func newInternalTxProducer(brokerList []string) sarama.SyncProducer {

	config := sarama.NewConfig()
	config.Producer.RequiredAcks = sarama.WaitForAll // Wait for all in-sync replicas to ack the message
	config.Producer.Retry.Max = 10                   // Retry up to 10 times to produce the message
	config.Producer.Return.Successes = true
	tlsConfig := createTlsConfiguration()
	if tlsConfig != nil {
		config.Net.TLS.Config = tlsConfig
		config.Net.TLS.Enable = true
	}

	// On the broker side, you may want to change the following settings to get
	// stronger consistency guarantees:
	// - For your broker, set `unclean.leader.election.enable` to false
	// - For the topic, you could increase `min.insync.replicas`.

	producer, err := sarama.NewSyncProducer(brokerList, config)
	if err != nil {
		log.Fatalln("Failed to start Sarama producer:", err)
	}

	return producer
}
func (kafkaDatastore *KafkaDatastore) SaveInternalTxFromSingleBlock(blockNumber *big.Int, internalTxStore []*types.InternalTx) uint64 {
	if !enableSaveInternalTxKafka {
		return 0
	}
	if len(internalTxStore) == 0 {
		return 0
	}

	// We are not setting a message key, which means that all messages will
	// be distributed randomly over the different partitions.
	startTimestamp := time.Now().UTC()
	msgs := []*sarama.ProducerMessage{}
	for _, internalTx := range internalTxStore {
		internalTxMarshalled, err := json.Marshal(internalTx)
		common.CheckErr(err, nil)
		msgs = append(msgs, &sarama.ProducerMessage{
			Topic: "internal-message",
			Value: sarama.StringEncoder(string(internalTxMarshalled)),
		})
	}

	err := kafkaDatastore.InternalTxProducer.SendMessages(msgs)

	common.CheckErr(err, nil)
	endTimestamp := time.Now().UTC()
	elapsed := endTimestamp.Sub(startTimestamp)
	fmt.Printf("%s: execution took %s, produced internal tx: blockNumber = %d\n", endTimestamp.Format("2006-01-02 15:04:05"), elapsed.Round(time.Millisecond).String(), blockNumber.Uint64())

	fmt.Printf("len(msgs) = %d\n", len(msgs))

	return uint64(len(msgs))
}
func createTlsConfiguration() (t *tls.Config) {
	if *certFile != "" && *keyFile != "" && *caFile != "" {
		cert, err := tls.LoadX509KeyPair(*certFile, *keyFile)
		if err != nil {
			log.Fatal(err)
		}

		caCert, err := ioutil.ReadFile(*caFile)
		if err != nil {
			log.Fatal(err)
		}

		caCertPool := x509.NewCertPool()
		caCertPool.AppendCertsFromPEM(caCert)

		t = &tls.Config{
			Certificates:       []tls.Certificate{cert},
			RootCAs:            caCertPool,
			InsecureSkipVerify: *verifySsl,
		}
	}
	// will be nil by default if nothing is provided
	return t
}
