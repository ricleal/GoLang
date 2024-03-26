package sample

import "fmt"

/*
Finite Poetry Protocol

| Poem (variable size)                                                       |
+-------------+-------------+-------------+-----------------+----------------+
| PoetryLine1 | PoetryLine2 | PoetryLine3 | PoetryLineN ... | EndPoem (0x00) |

| PoetryLine  (3 bytes)                                    |
+--------------------+-------------+-----------------------+
| PartOfSpeech uint8 | Count uint8 | DictionaryIndex uint8 |

| PartOfSpeech (1 byte)  |
+------------------------+
| Part      |   uint8    |
+-----------+------------+
| Verb      |    0x01    |
| Noun      |    0x02    |
| Adjective |    0x03    |

Pseudo-BNF:
PoetryLine = PartOfSpeech uint8 | Count uint8 | DictionaryIndex uint8
Poem = PoetryLine* | endOfPoem
*/

type partOfSpeech int

const (
	endOfPoem partOfSpeech = iota
	verb
	noun
	adjective
)

var (
	verbs      = []string{"jump", "dance", "scream"}
	nouns      = []string{"fish", "bear", "taco"}
	adjectives = []string{"blue", "tasty", "smelly"}
)

var poem = []byte{0x01, 0xA0, 0x00, 0x03, 0x02, 0x02, 0x02, 0x01, 0x00, 0x00}

// Run will print the poem.
func Run() {
	// print our poem!
	i := 0
	for {
		if i+3 > len(poem) {
			break
		}
		speech := poem[i : i+3]

		typ := speech[0]
		count := speech[1]
		dic := speech[2]

		switch typ {
		case 1:
			word := verbs[dic]
			for i := 0; i < int(count); i++ {
				fmt.Printf("%s ", word)
			}
		case 2:
			word := nouns[dic]
			for i := 0; i < int(count); i++ {
				fmt.Printf("%s ", word)
			}
		case 3:
			word := adjectives[dic]
			for i := 0; i < int(count); i++ {
				fmt.Printf("%s ", word)
			}
		default:
			return
		}
		i += 3
		fmt.Println()
	}
}
