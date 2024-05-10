package main

import (
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
	flag.StringVar(&pathtofile, "f", "tags_uids.csv", "Path to file")
}

func main() {

	flag.Parse()

	f, err := os.OpenFile(pathtofile, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatal(err)
	}

	defer f.Close()

	fi, err := f.Stat()
	if err != nil {
		log.Fatal(err)
	}

	if fi.Size() == 0 {
		// Añadir la primera línea de encabezados
		_, err = f.WriteString("UID\n") // Asegúrate de ajustar esto a tus encabezados necesarios
		if err != nil {
			log.Fatal(err)
		}
	}

	ctx, err := scard.EstablishContext()
	if err != nil {
		log.Fatal(err)
	}

	readers, err := ctx.ListReaders()
	if err != nil {
		log.Fatal(err)
	}

	var readerName string
	for _, r := range readers {
		if strings.Contains(r, "PICC") {
			readerName = r
		}
	}

	reader, err := ctx.Connect(readerName, scard.ShareDirect, scard.ProtocolUndefined)
	if err != nil {
		log.Fatal(err)
	}

	ctlCode := scard.CtlCode(2079)

	if _, err := reader.Control(ctlCode, []byte{0x23, 0x00}); err != nil {
		log.Fatal(err)
	}
	if _, err := reader.Control(ctlCode, []byte{0x23, 0x01, 0x8F}); err != nil {
		log.Fatal(err)
	}

	reader.Disconnect(scard.LeaveCard)

	tick := time.NewTicker(300 * time.Millisecond)
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
			uids := hex.EncodeToString(uid)
			if strings.EqualFold(uids, lastUid) {
				return
			}

			// uidfinal := make([]byte, 8)

			// copy(uidfinal[8-len(uid)-2:], uid[:len(uid)-2])

			// id := binary.BigEndian.Uint64(uidfinal)

			// if _, err := f.WriteString(uids + "," + fmt.Sprintf("%d", id) + "\n"); err != nil {
			// 	log.Fatal(err)
			// }

			if _, err := f.WriteString(uids + "\n"); err != nil {
				return
			}

			fmt.Printf("%02X\n", uid[:len(uid)-2])

			lastUid = uids

		}()
	}

}
