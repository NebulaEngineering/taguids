package main

import (
	"encoding/binary"
	"encoding/hex"
	"flag"
	"fmt"
	"log"
	"os"
	"strings"
	"time"

	"github.com/ebfe/scard"
)

var (
	pathtofile string
)

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
	flag.StringVar(&pathtofile, "f", "tags_uids.txt", "Path to file. data will be appended to this file [bytes; BigEndian; LittleEndian]")
}

func main() {

	flag.Parse()

	f, err := os.OpenFile(pathtofile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Printf("Error opening file: %v", err)
		time.Sleep(4 * time.Second)
		os.Exit(-1)
	}

	defer f.Close()

	ctx, err := scard.EstablishContext()
	if err != nil {
		log.Printf("Error establishing context: %v", err)
		time.Sleep(4 * time.Second)
		os.Exit(-1)
	}

	readers, err := ctx.ListReaders()
	if err != nil {
		log.Printf("Error listing readers: %v", err)
		time.Sleep(4 * time.Second)
		os.Exit(-1)
	}

	var readerName string
	for _, r := range readers {
		if strings.Contains(r, "PICC") {
			readerName = r
		}
	}

	reader, err := ctx.Connect(readerName, scard.ShareDirect, scard.ProtocolUndefined)
	if err != nil {
		log.Printf("Error connecting to reader: %v", err)
		time.Sleep(4 * time.Second)
		os.Exit(-1)
	}

	ctlCode := scard.CtlCode(2079)

	if _, err := reader.Control(ctlCode, []byte{0x23, 0x00}); err != nil {
		log.Printf("Error setting reader: %v", err)
		time.Sleep(4 * time.Second)
		os.Exit(-1)
	}
	if _, err := reader.Control(ctlCode, []byte{0x23, 0x01, 0x8F}); err != nil {
		log.Printf("Error setting reader: %v", err)
		time.Sleep(4 * time.Second)
		os.Exit(-1)
	}
	reader.Disconnect(scard.LeaveCard)

	tick := time.NewTicker(60 * time.Millisecond)
	defer tick.Stop()

	var lastUid string

	for range tick.C {

		func() {
			card, err := ctx.Connect(readerName, scard.ShareShared, scard.ProtocolT1)
			if err != nil {
				return
			}
			defer card.Disconnect(scard.LeaveCard)

			uid, err := card.Transmit([]byte{0xFF, 0xCA, 0x00, 0x00, 0x00})
			if err != nil {
				return
			}
			if len(uid) <= 2 || uid[len(uid)-2] != 0x90 || uid[len(uid)-1] != 0x00 {
				return
			}
			uids := hex.EncodeToString(uid[:len(uid)-2])
			if strings.EqualFold(uids, lastUid) {
				return
			}

			// idbytes := make([]byte, 8)
			idbytes_ := make([]byte, 8)

			if len(uid[:len(uid)-2]) > 8 {
				return
			}

			// copy(idbytes, uid[:len(uid)-2])
			copy(idbytes_[8-(len(uid)-2):], uid[:len(uid)-2])

			id := binary.LittleEndian.Uint64(idbytes_)
			id_ := binary.BigEndian.Uint64(idbytes_)

			if _, err := f.WriteString(fmt.Sprintf("%02X; %d; %d", uid[:len(uid)-2], id_, id) + "\n"); err != nil {
				return
			}

			fmt.Printf("%d\n", id_)

			lastUid = uids

		}()
	}

}
