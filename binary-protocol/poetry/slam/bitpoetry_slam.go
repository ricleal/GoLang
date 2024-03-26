/*
poems.dat: Stream Poetry Protocol
| PoetryBook (variable size)       |
+-------+-------+-------+----------+
| Poem1 | Poem2 | Poem3 | PoemN ...|

| Poem (variable size)                                                       |
+-------------+-------------+-------------+-----------------+----------------+
| PoetryLine1 | PoetryLine2 | PoetryLine3 | PoetryLineN ... | EndPoem (0x00) |

| PoetryLine  (3 bytes)                                    |
+--------------------+-------------+-----------------------+
| PartOfSpeech uint8 | Count uint8 | DictionaryIndex uint8 |

Pseudo-BNF:
PoetryLine = PartOfSpeech uint8 | Count uint8 | DictionaryIndex uint8
Poem = PoetryLine* | endOfPoem
Slam = Poem*


dictionary.dat: Acceptable Vocabulary
| Dictionary (variable size)        |
+-------+-------+-------+-----------+
| Word1 | Word2 | Word3 | WordN ... |

| Word (variable size)                                                                      |
+------------------+------------------------------------------------------+-----------------+
| Header (2 bytes) | Body (variable size)                                 | Footer (1 byte) |
+------------------+----------------------+-------------------------------+-----------------+
| 0xBE 0xEF        | PartOfSpeech (uint8) | Content (variable byte array) | 0x00            |

Pseudo-BNF:
Word = 0xBE 0xEF | PartOfSpeech uint8 | Content string([]byte) | 0x00
Dictionary = Word*
*/

package slam

import (
	"fmt"
	"os"
)

type partOfSpeech int

const (
	endOfPoem partOfSpeech = iota
	verb
	noun
	adjective
)

type dictType map[partOfSpeech][]string

func parseDic(dic []byte) dictType {
	ret := make(dictType)
	idx := 0
	for {
		if dic[idx] == 0xBE && len(dic) > idx && dic[idx+1] == 0xEF {
			// parse word
			partOfSpeechParsed := partOfSpeech(dic[idx+2])
			idx += 3
			content := ""
			for {
				if dic[idx] == 0x00 {
					break
				}
				content += string(dic[idx])
				idx++
			}
			ret[partOfSpeechParsed] = append(ret[partOfSpeechParsed], content)
		}
		if idx+1 >= len(dic) {
			break
		}
		idx++
	}
	return ret
}

// Run will print the poem.
func Run() {
	// print all poems in poems.dat using definitions in dictionary.dat

	poetryBook, err := os.ReadFile("binary-protocol/poetry/slam/poems.dat")
	if err != nil {
		panic(err)
	}

	dicContent, err := os.ReadFile("binary-protocol/poetry/slam/dictionary.dat")
	if err != nil {
		panic(err)
	}

	dict := parseDic(dicContent)

	idx := 0
	for {
		if idx >= len(poetryBook) {
			break
		}
		if poetryBook[idx] == 0x00 {
			fmt.Println("----- End of Poem -----")
			idx++
			continue
		}
		poem := poetryBook[idx:]

		poetryLine := poem[0:3]

		partOfSpeechParsed := poetryLine[0]
		count := poetryLine[1]
		dicIndex := poetryLine[2]

		switch partOfSpeechParsed {
		case 1:
			word := dict[verb][dicIndex]
			for i := 0; i < int(count); i++ {
				fmt.Printf("%s ", word)
			}
		case 2:
			word := dict[noun][dicIndex]
			for i := 0; i < int(count); i++ {
				fmt.Printf("%s ", word)
			}
		case 3:
			word := dict[adjective][dicIndex]
			for i := 0; i < int(count); i++ {
				fmt.Printf("%s ", word)
			}
		}
		fmt.Println()

		idx += 3
	}
}
