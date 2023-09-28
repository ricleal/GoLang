package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
)

var csvData = `id,firstname,lastname,french name,lorem,text,profession,attrs_json
1,Fernande,Sandye,"Sandye, Fernande",XdFVrzBfhSNHKCCozhPNrraESNMwvssMUGEeuPojixoArpJJYKQlTxqgIOThUlVtqLnRhhxZJzlEuoYZGpq BjcgV,"Fernande
XdFVrzBfhSNHKCCozhPNrraESNMwvssMUGEeuPojixoArpJJYKQlTxqgIOThUlVtqLnRhhxZJzlEuoYZGpq BjcgV",doctor,"{""first_name"":""Fernande"",
""lorem"": ""XdFVrzBfhSNHKCCozhPNrraESNMwvssMUGEeuPojixoArpJJYKQlTxqgIOThUlVtqLnRhhxZJzlEuoYZGpq BjcgV""}"
2,Aili,Abernon,"Abernon, Aili",fHIfgXqcnTFmIuogEIEutexKORwERHmbQCJPpyEKEEakNRGMOapr AEKvHFnLyTnYSSTEeEqxNfElMa,"Aili
fHIfgXqcnTFmIuogEIEutexKORwERHmbQCJPpyEKEEakNRGMOapr AEKvHFnLyTnYSSTEeEqxNfElMa",worker,"{""first_name"":""Aili"",
""lorem"": ""fHIfgXqcnTFmIuogEIEutexKORwERHmbQCJPpyEKEEakNRGMOapr AEKvHFnLyTnYSSTEeEqxNfElMa""}"
3,Juliane,Jacinda,"Jacinda, Juliane",foYtZjPHEiq JG,"Juliane
foYtZjPHEiq JG",worker,"{""first_name"":""Juliane"",
""lorem"": ""foYtZjPHEiq JG""}"
4,Vevay,Hieronymus,"Hieronymus, Vevay",pzdYgCfHUNVpLtoXcjflFdureZJlnlrANHbSo H,"Vevay
pzdYgCfHUNVpLtoXcjflFdureZJlnlrANHbSo H",doctor,"{""first_name"":""Vevay"",
""lorem"": ""pzdYgCfHUNVpLtoXcjflFdureZJlnlrANHbSo H""}"
5,Fernande,Ackerley,"Ackerley, Fernande",GWVkDzUJkEUgVzeSViOsIuHkHJjXzcrVy,"Fernande
GWVkDzUJkEUgVzeSViOsIuHkHJjXzcrVy",developer,"{""first_name"":""Fernande"",
""lorem"": ""GWVkDzUJkEUgVzeSViOsIuHkHJjXzcrVy""}"
6,Margalo,Francyne,"Francyne, Margalo",sDzkIQZKQWIHJWM bATAZnerFbtYRkfsFGihTrU,"Margalo
sDzkIQZKQWIHJWM bATAZnerFbtYRkfsFGihTrU",doctor,"{""first_name"":""Margalo"",
""lorem"": ""sDzkIQZKQWIHJWM bATAZnerFbtYRkfsFGihTrU""}"
7,Kristina,Matthew,"Matthew, Kristina",BjqHEHDBdsoKKUeuXUGfkPUapYLdzHkJHPBcWBquxUZWLFDIUiKdWryQfinTNYoZDsNHCO VvlqOFTqTmmkXpYC,"Kristina
BjqHEHDBdsoKKUeuXUGfkPUapYLdzHkJHPBcWBquxUZWLFDIUiKdWryQfinTNYoZDsNHCO VvlqOFTqTmmkXpYC",doctor,"{""first_name"":""Kristina"",
""lorem"": ""BjqHEHDBdsoKKUeuXUGfkPUapYLdzHkJHPBcWBquxUZWLFDIUiKdWryQfinTNYoZDsNHCO VvlqOFTqTmmkXpYC""}"
8,Ira,Cynar,"Cynar, Ira",cLRWmtvrGAFpmQgDhup gUIcjHcuIcnQGwiHNGaSIMKgYsBKpnQgXrVdJGANzIqPeEmjxt xRCzxFdrHuIq,"Ira
cLRWmtvrGAFpmQgDhup gUIcjHcuIcnQGwiHNGaSIMKgYsBKpnQgXrVdJGANzIqPeEmjxt xRCzxFdrHuIq",worker,"{""first_name"":""Ira"",
""lorem"": ""cLRWmtvrGAFpmQgDhup gUIcjHcuIcnQGwiHNGaSIMKgYsBKpnQgXrVdJGANzIqPeEmjxt xRCzxFdrHuIq""}"
9,Leontine,Abbot,"Abbot, Leontine",bEweqcYwOHQysEaBZZlIELapiCuUNPnw,"Leontine
bEweqcYwOHQysEaBZZlIELapiCuUNPnw",doctor,"{""first_name"":""Leontine"",
""lorem"": ""bEweqcYwOHQysEaBZZlIELapiCuUNPnw""}"
10,Ingrid,Sekofski,"Sekofski, Ingrid",CYzRBYLD GKhrrwPnqeguAnF NgScMHoKG MaknwguSCwopWCMTxWnLS dookarUjRWx,"Ingrid
CYzRBYLD GKhrrwPnqeguAnF NgScMHoKG MaknwguSCwopWCMTxWnLS dookarUjRWx",doctor,"{""first_name"":""Ingrid"",
""lorem"": ""CYzRBYLD GKhrrwPnqeguAnF NgScMHoKG MaknwguSCwopWCMTxWnLS dookarUjRWx""}"
11,Dolli,Albertine,"Albertine, Dolli",qJhRNthNqOrbQs QUTfhEj rSYGoXVHevhvZYT,"Dolli
qJhRNthNqOrbQs QUTfhEj rSYGoXVHevhvZYT",developer,"{""first_name"":""Dolli"",
""lorem"": ""qJhRNthNqOrbQs QUTfhEj rSYGoXVHevhvZYT""}"
12,Consuela,Lalitta,"Lalitta, Consuela",gtBYjeJqKxaeAkfADZmrRUXtdEfYwqzgRlLuSfUYCm kmVHA,"Consuela
gtBYjeJqKxaeAkfADZmrRUXtdEfYwqzgRlLuSfUYCm kmVHA",doctor,"{""first_name"":""Consuela"",
""lorem"": ""gtBYjeJqKxaeAkfADZmrRUXtdEfYwqzgRlLuSfUYCm kmVHA""}"
13,Nerta,Chinua,"Chinua, Nerta",ebWQxpUrQFggTmgrOSZGIdN cFZTziLCcOLlbNgljBTDuUJcCgSWOQmjfZzZtOepNL,"Nerta
ebWQxpUrQFggTmgrOSZGIdN cFZTziLCcOLlbNgljBTDuUJcCgSWOQmjfZzZtOepNL",firefighter,"{""first_name"":""Nerta"",
""lorem"": ""ebWQxpUrQFggTmgrOSZGIdN cFZTziLCcOLlbNgljBTDuUJcCgSWOQmjfZzZtOepNL""}"
14,Ernesta,Llovera,"Llovera, Ernesta",bGQcKQd DDCcqKcCAWEXsHyYzRStmLsofDeENaWDbmHrfzhWtLoawOTNzVvDKM,"Ernesta
bGQcKQd DDCcqKcCAWEXsHyYzRStmLsofDeENaWDbmHrfzhWtLoawOTNzVvDKM",police officer,"{""first_name"":""Ernesta"",
""lorem"": ""bGQcKQd DDCcqKcCAWEXsHyYzRStmLsofDeENaWDbmHrfzhWtLoawOTNzVvDKM""}"
15,Julieta,O'Carroll,"O'Carroll, Julieta",mWYJnDZLMUVYIskbgOZTn KATGwmrcC,"Julieta
mWYJnDZLMUVYIskbgOZTn KATGwmrcC",police officer,"{""first_name"":""Julieta"",
""lorem"": ""mWYJnDZLMUVYIskbgOZTn KATGwmrcC""}"
16,Pamella,Newell,"Newell, Pamella",AHJANhRNPTAMhEbpDtJ,"Pamella
AHJANhRNPTAMhEbpDtJ",developer,"{""first_name"":""Pamella"",
""lorem"": ""AHJANhRNPTAMhEbpDtJ""}"
17,Mildrid,Ammann,"Ammann, Mildrid",xsleuYBjDN,"Mildrid
xsleuYBjDN",police officer,"{""first_name"":""Mildrid"",
""lorem"": ""xsleuYBjDN""}"
18,Lulita,Natica,"Natica, Lulita",jwNdKgONwppcf,"Lulita
jwNdKgONwppcf",doctor,"{""first_name"":""Lulita"",
""lorem"": ""jwNdKgONwppcf""}"
19,Kayla,Oneida,"Oneida, Kayla",VmtkCWNHbXwcCLuLtThD jtJbQPuvDwOBiwCGmtilXdSwMdzgsDezcfAYMoZMTdyZEdbQuIPe nHygnMSRgMIXNMDLSjsVg,"Kayla
VmtkCWNHbXwcCLuLtThD jtJbQPuvDwOBiwCGmtilXdSwMdzgsDezcfAYMoZMTdyZEdbQuIPe nHygnMSRgMIXNMDLSjsVg",worker,"{""first_name"":""Kayla"",
""lorem"": ""VmtkCWNHbXwcCLuLtThD jtJbQPuvDwOBiwCGmtilXdSwMdzgsDezcfAYMoZMTdyZEdbQuIPe nHygnMSRgMIXNMDLSjsVg""}"
20,Renae,Destinee,"Destinee, Renae",hEYZeQkuddMXQhYtByhR xbkKbMFByJlMTRyvChdpKyuxCnJ,"Renae
hEYZeQkuddMXQhYtByhR xbkKbMFByJlMTRyvChdpKyuxCnJ",developer,"{""first_name"":""Renae"",
""lorem"": ""hEYZeQkuddMXQhYtByhR xbkKbMFByJlMTRyvChdpKyuxCnJ""}"
21,Anallese,Hirsch,"Hirsch, Anallese",oMbRFzuejHTRldVlgYuphZnndX,"Anallese
oMbRFzuejHTRldVlgYuphZnndX",worker,"{""first_name"":""Anallese"",
""lorem"": ""oMbRFzuejHTRldVlgYuphZnndX""}"`

const chunkSize = 400

func main() {
	buf := bytes.NewBufferString(csvData)
	reader := csv.NewReader(buf)
	headers, err := reader.Read()
	if err != nil {
		fmt.Println("Error reading header:", err)
		return
	}

	chunk := bytes.NewBuffer(make([]byte, 0, chunkSize))
	csvWriter := csv.NewWriter(chunk)
	csvWriter.Write(headers) // Write header using CSV writer
	csvWriter.Flush()        // Flush the CSV writer to include the header in the chunk

	for {
		record, err := reader.Read()
		if err == io.EOF {
			// Process the last chunk
			if chunk.Len() > 0 {
				csvWriter.Flush()
				printChunk(chunk)
			}
			break
		} else if err != nil {
			fmt.Println("Error reading record:", err)
			break
		}

		if err := csvWriter.Write(record); err != nil {
			fmt.Println("Error writing record to buffer:", err)
			break
		}
		csvWriter.Flush()

		if chunk.Len()+len(record) > chunkSize {
			printChunk(chunk)
			chunk.Reset()
			csvWriter.Write(headers) // Write header again in the new chunk
		}
	}
}

func printChunk(chunk *bytes.Buffer) {
	fmt.Println("S-----------------------------")
	fmt.Println(chunk.String())
	fmt.Println("E----------------------------->", chunk.Len())
}
