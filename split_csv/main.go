package main

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io"
	"strings"
)

var csvData = `id,firstname,lastname,frenchname,text,profession,loremipsum
1,Tybie,Ogren,"Ogren, Tybie","1:Ogren, Tybie
stbsq mCSkBIlNbcxiKVPRTQhW
",developer,stbsq mCSkBIlNbcxiKVPRTQhW
2,Berta,Means,"Means, Berta","2:Means, Berta
IYyYIQVWFJ lfOvAGjLXZ
",firefighter,IYyYIQVWFJ lfOvAGjLXZ
3,Janis,Fiester,"Fiester, Janis","3:Fiester, Janis
daYKpxAFkeHhmSiUCW HA
",police officer,daYKpxAFkeHhmSiUCW HA
4,Karolina,Meli,"Meli, Karolina","4:Meli, Karolina
ItnUPcQxUtFdBBaQyfuRSWxBSSMK
",worker,ItnUPcQxUtFdBBaQyfuRSWxBSSMK
5,Emma,Curren,"Curren, Emma","5:Curren, Emma
vyLTbOxPHFHXeidcPjgPUjsTFbKk
",worker,vyLTbOxPHFHXeidcPjgPUjsTFbKk
6,Sharlene,McNully,"McNully, Sharlene","6:McNully, Sharlene
OUnfdvtULQG
",firefighter,OUnfdvtULQG
7,Zondra,Schroth,"Schroth, Zondra","7:Schroth, Zondra
SnLwVdgXxuSzihhdMuNNYijGvXtIDLKOjdlZtNwlCTgEbneV
",developer,SnLwVdgXxuSzihhdMuNNYijGvXtIDLKOjdlZtNwlCTgEbneV
8,Alex,Amadas,"Amadas, Alex","8:Amadas, Alex
UTStVyoPowBc pxigqndYvJGilnnRMyt
",doctor,UTStVyoPowBc pxigqndYvJGilnnRMyt
9,Lucy,Federica,"Federica, Lucy","9:Federica, Lucy
XphuEoNWFWHfCSLMWYcjQvcaCvvIRzpdNlLzeIUyGeCJXdAb
",police officer,XphuEoNWFWHfCSLMWYcjQvcaCvvIRzpdNlLzeIUyGeCJXdAb
10,Sarette,Dearborn,"Dearborn, Sarette","10:Dearborn, Sarette
RZRqgPYAtZIOigapyKvwHMDZfpkNWravWOOhiUFg
",worker,RZRqgPYAtZIOigapyKvwHMDZfpkNWravWOOhiUFg
11,Julieta,Orpah,"Orpah, Julieta","11:Orpah, Julieta
nkbEDhfA bxdTwaD dYaHpxRMqyXINsHjznVVabUg
",firefighter,nkbEDhfA bxdTwaD dYaHpxRMqyXINsHjznVVabUg
12,Aryn,Sekofski,"Sekofski, Aryn","12:Sekofski, Aryn
FbRqEFSXQoPEHXDlv
",developer,FbRqEFSXQoPEHXDlv
13,Fred,Kylander,"Kylander, Fred","13:Kylander, Fred
YwlayZWGfCfXGhhMqqoaBrWIKSVaBgiai XSRlwGhcIzOiEs
",worker,YwlayZWGfCfXGhhMqqoaBrWIKSVaBgiai XSRlwGhcIzOiEs
14,Tiffie,Orpah,"Orpah, Tiffie","14:Orpah, Tiffie
sZjeLMEX zUEz vLRJeOxgWMIEphkfTcHdFrZWiTXEeQOkJUQ
",developer,sZjeLMEX zUEz vLRJeOxgWMIEphkfTcHdFrZWiTXEeQOkJUQ
15,Lusa,Alabaster,"Alabaster, Lusa","15:Alabaster, Lusa
eKueBGNXNfsVVbqrXlZLdeOqyHUeTPkCjIamqzXuoeBVzX
",doctor,eKueBGNXNfsVVbqrXlZLdeOqyHUeTPkCjIamqzXuoeBVzX
16,Myrtice,Abram,"Abram, Myrtice","16:Abram, Myrtice
nfYvrYMoZtQcdLrQzhKZsFYoXPImptGebKx
",developer,nfYvrYMoZtQcdLrQzhKZsFYoXPImptGebKx
17,Britte,Sekofski,"Sekofski, Britte","17:Sekofski, Britte
s VLrXQjKGZmBHNClxfmCauoEIUO yKEKeQdgyrXVo
",firefighter,s VLrXQjKGZmBHNClxfmCauoEIUO yKEKeQdgyrXVo
18,Nadine,Killigrew,"Killigrew, Nadine","18:Killigrew, Nadine
UopipR zohgRqDtvn
",worker,UopipR zohgRqDtvn
19,Judy,Malina,"Malina, Judy","19:Malina, Judy
MxpgNhLLUuFEfwjJyuGxmLeLi
",doctor,MxpgNhLLUuFEfwjJyuGxmLeLi
20,Stephanie,Daegal,"Daegal, Stephanie","20:Daegal, Stephanie
sxmYBBSunkdXGKtZrGFbE zEKHmIyZvnIAggw
",developer,sxmYBBSunkdXGKtZrGFbE zEKHmIyZvnIAggw`

func main() {

	buf := bytes.NewBufferString(csvData)

	// Read the CSV data using a CSV reader
	reader := csv.NewReader(buf)
	headers, err := reader.Read()
	if err != nil {
		fmt.Println("Error reading header:", err)
		return
	}

	maxChunkSize := 300 // Maximum number of characters per chunk
	header := strings.Join(headers, ",")
	chunk := bytes.NewBufferString(header)

	for {
		var csvBuilder strings.Builder
		writer := csv.NewWriter(&csvBuilder)

		record, err := reader.Read()
		if err == io.EOF {
			// Process the last chunk
			if chunk.Len() > len(header) {
				printChunk(chunk)
			}
			break
		} else if err != nil {
			fmt.Println("Error reading record:", err)
			break
		}

		// recordStr := strings.Join(record, ",")
		err = writer.Write(record)
		if err != nil {
			fmt.Println("Error writing record to buffer:", err)
			break
		}
		writer.Flush()
		if writer.Error() != nil {
			fmt.Println("Error flushing writer:", writer.Error())
			break
		}
		recordStr := csvBuilder.String()
		if recordStr[len(recordStr)-1] == '\n' {
			recordStr = recordStr[:len(recordStr)-1]
		}

		if chunk.Len()+len(recordStr) <= maxChunkSize {
			chunk.WriteString("\n")
			chunk.WriteString(recordStr)
		} else {
			printChunk(chunk)
			chunk.Reset()
			chunk.WriteString(header + "\n")
			chunk.WriteString(recordStr)
		}
	}
}

func printChunk(chunk *bytes.Buffer) {
	fmt.Println("-----------------------------")
	fmt.Println(chunk.String())
}
