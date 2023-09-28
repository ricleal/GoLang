package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"reflect"
	"strings"
	"testing"

	"golang.org/x/exp/slices"
)

const header = `id,firstname,lastname,french name,lorem,text,profession`

const csvTestData = header + `
1,Jinny,Lanita,"Lanita, Jinny",ODiQtDTcWemTfHvpTRIRCSWxYRApUZl,"Jinny
ODiQtDTcWemTfHvpTRIRCSWxYRApUZl",worker
2,Jany,Seagraves,"Seagraves, Jany",fAIvJ wFW kSmykgd,"Jany
fAIvJ wFW kSmykgd",worker
3,Tracey,Kaja,"Kaja, Tracey",raXZHRDJznBVNXYQqTBCpIxnLwUSOUbkzMLHGijjBpkOAzTKgkfuMoSuCHnqqHTEg,"Tracey
raXZHRDJznBVNXYQqTBCpIxnLwUSOUbkzMLHGijjBpkOAzTKgkfuMoSuCHnqqHTEg",police officer
4,Kalina,Erich,"Erich, Kalina",LlKizWQQKXuB,"Kalina
LlKizWQQKXuB",police officer
5,Chere,Brandice,"Brandice, Chere",NcosKZqfSmgONNlwVgyljZVUYLrsIvTGCcRy,"Chere
NcosKZqfSmgONNlwVgyljZVUYLrsIvTGCcRy",firefighter
6,Meghann,Oster,"Oster, Meghann",xDpXFJeAZdrthYiFDLvfPTWBDpUxcIXQrXjnTmndmTPiPkVWyEQtGgcLORjyJpXtOkCSbzvgjcyfOwhXXMzH,"Meghann
xDpXFJeAZdrthYiFDLvfPTWBDpUxcIXQrXjnTmndmTPiPkVWyEQtGgcLORjyJpXtOkCSbzvgjcyfOwhXXMzH",police officer
7,Dolli,Campball,"Campball, Dolli",fWsVvCmhPTNx q FKNFujGaicnoKaqoXfxgXOKhjqsVENxhDDWPIWfLOAAGFp,"Dolli
fWsVvCmhPTNx q FKNFujGaicnoKaqoXfxgXOKhjqsVENxhDDWPIWfLOAAGFp",police officer
8,Gianina,Ciapas,"Ciapas, Gianina",xoSdTZchiy,"Gianina
xoSdTZchiy",developer
9,Elka,Duwalt,"Duwalt, Elka",YsbpevxFyvbyGUvSzhwRtWWsthqbpxcJFIaVuLDpbOybyHadnJXHBR qV,"Elka
YsbpevxFyvbyGUvSzhwRtWWsthqbpxcJFIaVuLDpbOybyHadnJXHBR qV",worker
10,Annora,Gower,"Gower, Annora",lvIuFZqRfh uxHxZpnOZbPqviINcM bsInlqZKfeMpTEoVEVpViZzkXhkkAksiePSRmZLbC,"Annora
lvIuFZqRfh uxHxZpnOZbPqviINcM bsInlqZKfeMpTEoVEVpViZzkXhkkAksiePSRmZLbC",doctor
11,Lusa,Grayce,"Grayce, Lusa",iBGwYMsrNLoOmCGELcQrpwCLfAwrrdAdRYWMUewDmgVPFJEQGTkO hpJFTuqT,"Lusa
iBGwYMsrNLoOmCGELcQrpwCLfAwrrdAdRYWMUewDmgVPFJEQGTkO hpJFTuqT",firefighter
12,Trixi,Bettine,"Bettine, Trixi","GetX BvYiJVtORmKxtBHrpiuAOlZeYnwMOsznhmLCQjjOqunZSPlfYVMxsuV  ","Trixi
GetX BvYiJVtORmKxtBHrpiuAOlZeYnwMOsznhmLCQjjOqunZSPlfYVMxsuV  ",police officer
13,Ana,Pascia,"Pascia, Ana",zScRAwcCajBZzDhoBwKselcHHdDPyxXnbsrVitfBkTANifquRouCcuSSqgdHBIaXLEtwiNzTnWjOcbsxgw,"Ana
zScRAwcCajBZzDhoBwKselcHHdDPyxXnbsrVitfBkTANifquRouCcuSSqgdHBIaXLEtwiNzTnWjOcbsxgw",doctor
14,Ardenia,Dosia,"Dosia, Ardenia",EiWMnVCMAQeIdHZhHaYskSQiifGQUmhdQIzFKUGIeoXSErSF,"Ardenia
EiWMnVCMAQeIdHZhHaYskSQiifGQUmhdQIzFKUGIeoXSErSF",doctor
15,Deloria,Lissi,"Lissi, Deloria",mjgMzcEVnRNwwIHzRCTpQYaefAOnkMJniqlhjHOjaaYoeHrwtJmhKBLYYiUNdToUohv,"Deloria
mjgMzcEVnRNwwIHzRCTpQYaefAOnkMJniqlhjHOjaaYoeHrwtJmhKBLYYiUNdToUohv",firefighter
16,Korrie,Codding,"Codding, Korrie",uCopaKFLbZYSZiIAROOEmlAlb JBdeOXWkOzgNnmOrkgziAIDIC,"Korrie
uCopaKFLbZYSZiIAROOEmlAlb JBdeOXWkOzgNnmOrkgziAIDIC",worker
17,Laurene,Saint,"Saint, Laurene",LLMEgNMOzeJJjzctuoqcVaLDnZnVlhDKQJuuUofQjmeFfFljruIDAUFRnymWfUYdmM,"Laurene
LLMEgNMOzeJJjzctuoqcVaLDnZnVlhDKQJuuUofQjmeFfFljruIDAUFRnymWfUYdmM",worker
18,Marguerite,Craggie,"Craggie, Marguerite",wtLlNiBCWt FOAYHASjUBFBY,"Marguerite
wtLlNiBCWt FOAYHASjUBFBY",police officer
19,Robinia,Azeria,"Azeria, Robinia",TIlBafhrVUtIauukXCxSEDnxdXvTYSTZKACWEpLHujTE,"Robinia
TIlBafhrVUtIauukXCxSEDnxdXvTYSTZKACWEpLHujTE",developer
20,Priscilla,Kiyoshi,"Kiyoshi, Priscilla",SgaaIZIAguNfDMaakOWQKdhRN,"Priscilla
SgaaIZIAguNfDMaakOWQKdhRN",firefighter
21,Lolita,Federica,"Federica, Lolita",iyZM PUyyJlNvL umCLTvVfxmWSvKFlcChNaqHutiv nMucfrIPgpTsLhzyUWihLMxQFduKQEpNTCFjuMQuOUngufeRh,"Lolita
iyZM PUyyJlNvL umCLTvVfxmWSvKFlcChNaqHutiv nMucfrIPgpTsLhzyUWihLMxQFduKQEpNTCFjuMQuOUngufeRh",police officer`

func TestChunkedReader(t *testing.T) {
	reader := csv.NewReader(bytes.NewBufferString(csvTestData))
	expectedRecords, err := reader.ReadAll()
	if err != nil {
		t.Fatalf("error reading CSV: %v", err)
	}

	for _, chunkSize := range []int{1, 10, 100, 500, 1000, 2000, 10_000} {
		t.Run(fmt.Sprintf("chunk-size=%v", chunkSize), func(t *testing.T) {
			reader, err := NewChunkedCSVReader(bytes.NewBufferString(csvTestData), chunkSize)
			if err != nil {
				t.Fatalf("error creating chunked CSV reader: %v", err)
			}

			maxChunkSize := maxChunkSize(chunkSize, expectedRecords)
			records := make([][]string, 0, len(expectedRecords))
			totalDocs := 0
			for ; reader.NextChunk(); totalDocs++ {
				b, err := io.ReadAll(reader)
				if err != nil {
					t.Fatalf("error calling ReadAll on ChunkedReader: %v", err)
				}

				if len(b) == 0 {
					t.Errorf("empty document produced")
					break
				}
				if maxChunkSize < len(b) {
					t.Fatalf("chunk size is too large (max size %v): %s", maxChunkSize, b)
				}

				r := csv.NewReader(bytes.NewReader(b))
				chunkRecords, err := r.ReadAll()
				if err != nil {
					t.Fatalf("error reading output document: %v", err)
				}
				if len(chunkRecords) < 2 {
					t.Errorf("did not get at least 2 lines: %v", chunkRecords)
				}

				if !slices.Equal(chunkRecords[0], expectedRecords[0]) {
					t.Fatalf("header mismatch. expected=%v, got=%v", expectedRecords[0], records[0])
				}
				records = append(records, chunkRecords[1:]...)
			}

			if !reflect.DeepEqual(records, expectedRecords[1:]) {
				t.Fatalf("records not equal. expected=%v, got=%v", expectedRecords[1:], records)
			}
		})
	}
}

func expectedNumDocs(chunkSize int, expectedNumRecords int) int {
	i := (len(csvTestData) / chunkSize) + 1
	if i > (expectedNumRecords - 1) {
		return expectedNumRecords - 1
	}
	return i
}

func maxChunkSize(chunkSize int, expectedRecords [][]string) int {
	if s := strings.Join(expectedRecords[0], ","); len(s) > chunkSize {
		chunkSize = len(s)
	}
	lr := largestRecord(expectedRecords)
	chunkSize += len(lr)
	return chunkSize + 2 + (len(expectedRecords[0]) * 2) // two newlines plus quotes on every field.
}

func largestRecord(records [][]string) string {
	var largest string
	for _, recs := range records {
		s := strings.Join(recs, ",")
		if len(s) > len(largest) {
			largest = s
		}
	}
	return largest
}
