package main

import (
	"bytes"
	"log"
	"net/http"
	"time"
	"warson-blockchain/core"
	"warson-blockchain/crypto"
	"warson-blockchain/network"
)

func main() {
	privKey := crypto.GeneratePrivateKey()
	localNode := makeServer("LOCAL_NODE", &privKey, ":3000", []string{":4000"}, ":9000")
	go localNode.Start()

	remoteNode := makeServer("REMOTE_NODE", nil, ":4000", []string{":4001"}, "")
	go remoteNode.Start()

	remoteNodeB := makeServer("REMOTE_NODE_B", nil, ":4001", nil, "")
	go remoteNodeB.Start()

	go func() {
		time.Sleep(11 * time.Second)

		lateNode := makeServer("LATE_NODE", nil, ":6000", []string{":4000"}, "")
		go lateNode.Start()
	}()
	time.Sleep(1 * time.Second)

	go func() {
		txSendTicker := time.NewTicker(2 * time.Second)
		for {
			txSender()
			<-txSendTicker.C
		}
	}()

	select {}
}

func makeServer(
	id string,
	pk *crypto.PrivateKey,
	addr string,
	seedNodes []string,
	apiListenAddr string,
) *network.Server {
	opts := network.ServerOpts{
		SeedNodes:     seedNodes,
		ListenAddr:    addr,
		PrivateKey:    pk,
		ID:            id,
		APIListenAddr: apiListenAddr,
	}

	s, err := network.NewServer(opts)
	if err != nil {
		log.Fatal(err)
	}

	return s
}

func txSender() {
	privKey := crypto.GeneratePrivateKey()
	data := []byte{0x03, 0x0a, 0x46, 0x0c, 0x4f, 0x0c, 0x4f, 0x0c, 0x0d, 0x05, 0x0a, 0x0f}
	tx := core.NewTransaction(data)
	tx.Sign(privKey)

	buf := &bytes.Buffer{}
	if err := tx.Encode(core.NewGobTxEncoder(buf)); err != nil {
		panic(err)
	}

	req, err := http.NewRequest("POST", "http://localhost:9000/tx", buf)
	if err != nil {
		panic(err)
	}

	client := http.Client{}
	_, err = client.Do(req)
	if err != nil {
		panic(err)
	}
}
