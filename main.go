package main

import (
	"flag"
	"fmt"
	"os"
)

/*
	Notes

	[Initial blob]: none(72610): ?

	Then into cycles of
	RSYN
	- always seems to be 3 bytes?
	- usually [12, 0, 0]

	SYNC
	- 7 bytes? (1 leading byte = "ordering tick") == SYNC is RSYN[0] - 6 + 1

	RTOK
	- 19 bytes (1 leading - same ordering tick?)

	----

	examples
	RSYN [12, 0, 0]

	---

	RSYN [68, 0, 0]
	SYNC(63)

	---

	Is RSYN saying
	[RSYN][B][B characters until RTOK]?

	-- I THINK

	RSYN[12 0 0] = nothing has happened this tick?

	RSYN[X 0 0] = there is something to process
	SYNC[T 0 0 0 X ?]
*/

func main() {
	fmt.Printf("IC Replay Parser\n")

	var filename = flag.String("file", "", "The SGM file to process.")
	var printcmds = flag.Bool("print-cmd", false, "Whether to print out each parsed command")
	var stallcmds = flag.Bool("stall-cmd", false, "Whether to stall after printing each parsed command")
	var summary = flag.Bool("summary", true, "Printout summary at the end")
	var ignorenonactions = flag.Bool("ignore-empty-ticks", true, "Do not print out RSYN cycles from empty ticks")

	flag.Parse()

	if *filename == "" {
		panic("no filename")
	}

	bytes, err := os.ReadFile(*filename)
	if err != nil {
		panic(err)
	}

	var counts = make(map[string]int)

	var firstmove = 0

	var last = 0
	var lastcmd = "none"
	var emptytick = false
	for i := 0; i < len(bytes)-4; i++ {
		var thiscmd = ""
		if test(bytes, i, []byte{0x52, 0x53, 0x59, 0x4e}) { //RSYN
			thiscmd = "RSYN"
		} else if test(bytes, i, []byte{0x53, 0x59, 0x4e, 0x43}) { //SYNC
			thiscmd = "SYNC"
		} else if test(bytes, i, []byte{0x52, 0x54, 0x4f, 0x4b}) { //RTOK
			thiscmd = "RTOK"
		}

		if thiscmd != "" {
			if _, exists := counts[thiscmd]; !exists {
				counts[thiscmd] = 0
			}
			counts[thiscmd] += 1

			slice := bytes[last : i-1]

			if lastcmd == "RSYN" {
				emptytick = slice[0] == 0xc
				if emptytick && firstmove == 0 {
					firstmove = counts["RSYN"]
				}
			}

			if *printcmds && (!*ignorenonactions || !emptytick) {
				fmt.Printf("\n%s (T%d):\nlen(%d) %v\n", lastcmd, counts["RSYN"], len(slice), slice)
				fmt.Printf("\n%s\n", string(slice))
				if *stallcmds {
					fmt.Scanln()
				}
			}

			last = i + len(thiscmd)
			lastcmd = thiscmd
		}
	}

	if *summary {
		seconds := counts["RSYN"] / 8
		time := fmt.Sprintf("%d:%02d", seconds/60, seconds%60)

		fmt.Printf("Game duration: %d ticks / %s time (Fast)\n", counts["RSYN"], time)
		fmt.Printf("First action : %d ticks\n", firstmove)
	}

}

func test(bytes []byte, index int, test []byte) bool {
	for i := 0; i < len(test); i++ {
		if bytes[index+i] != test[i] {
			return false
		}
	}

	return true
}
