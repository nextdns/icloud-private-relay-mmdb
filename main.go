package main

import (
	"bytes"
	"encoding/csv"
	"log"
	"net"
	"net/http"
	"os"

	"github.com/maxmind/mmdbwriter"
	"github.com/maxmind/mmdbwriter/mmdbtype"
)

const egressRanges = "https://mask-api.icloud.com/egress-ip-ranges.csv"
const etagFilename = "egress-ip-ranges.etag"
const outputFilename = "egress-ip-ranges.mmdb"

func main() {
	resp, err := http.Get(egressRanges)
	if err != nil {
		log.Fatalf("Failed to download file: %v", err)
	}
	defer resp.Body.Close()

	previousETag, _ := os.ReadFile(etagFilename)
	currentEtag := []byte(resp.Header.Get("ETag"))
	if bytes.Equal(currentEtag, previousETag) {
		log.Println("No changes in the egress IP ranges")
		return
	}

	reader := csv.NewReader(resp.Body)
	records, err := reader.ReadAll()
	if err != nil {
		log.Fatalf("Failed to read CSV file: %v", err)
	}

	writer, err := mmdbwriter.New(mmdbwriter.Options{
		DatabaseType: "GeoIP2-Country",
		RecordSize:   24,
	})
	if err != nil {
		log.Fatalf("Failed to create MMDB writer: %v", err)
	}

	// Insert records into the MMDB writer
	for _, record := range records {
		_, ipNet, err := net.ParseCIDR(record[0])
		if err != nil {
			log.Fatalf("Failed to parse CIDR: %v", err)
		}

		data := mmdbtype.Map{
			"country": mmdbtype.Map{
				"iso_code": mmdbtype.String(record[1]),
			},
		}

		err = writer.Insert(ipNet, data)
		if err != nil {
			log.Fatalf("Failed to insert record into MMDB: %v", err)
		}
	}

	// Write the MMDB file
	mmdbFile, err := os.Create(outputFilename)
	if err != nil {
		log.Fatalf("Failed to create MMDB file: %v", err)
	}
	defer mmdbFile.Close()

	if _, err = writer.WriteTo(mmdbFile); err != nil {
		log.Fatalf("Failed to write MMDB file: %v", err)
	}

	if err = os.WriteFile(etagFilename, currentEtag, 0644); err != nil {
		log.Fatalf("Failed to write etag file: %v", err)
	}
}
