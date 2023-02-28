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


	----

	When parsing SYNC blocks, commonly seeing:

	Header:

	[0:3]: SYNC
	[4]  : TICK
	[5:7]: ???
	[8]  : SIZE TOTAL
	[9]  : ??
	[10] : GC commands? (game command?)

	GCEE?XX?
	GCEN?XX?
	QHB

	GCE? ???X ??P? --> "group command? "

	P = player byte
	X seems related to player, maybe the position?

	Command lengths:

	GCEE: 40, 52
	GCEN: 28
	GCEP: 52

	GCPE: ??

	???

	GCEE -- unit command?

	Command block @


*/

func main() {
	fmt.Printf("IC Replay Parser\n")

	var filename = flag.String("file", "", "The SGM file to process.")
	var printcmds = flag.Bool("print-cmd", false, "Whether to print out each parsed command")
	var printgc = flag.String("print-gc", "", "The tick-commands to print")
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
	counts["RSYN"] = 0
	counts["SYNC"] = 0
	counts["RTOK"] = 0
	counts["GCE"] = 0

	var players = make(map[byte]map[string]int)

	var firstmove = 0

	//var last = 0
	//var lastcmd = "none"
	var emptytick = false
	for i := 0; i < len(bytes)-4; i++ {

		if test(bytes, i, []byte{0x52, 0x53, 0x59, 0x4e}) { //Found an RSYN cycle block
			counts["RSYN"] += 1
			counts["SYNC"] += 1
			counts["RTOK"] += 1

			bytestosync := int(bytes[i+4])

			rsynsl := bytes[i : i+8]
			syncsl := bytes[i+8 : i+8+bytestosync]
			rtoksl := bytes[i+8+bytestosync : i+8+bytestosync+18]

			if !test(rsynsl, 0, []byte{0x52, 0x53, 0x59, 0x4e}) ||
				!test(syncsl, 0, []byte{0x53, 0x59, 0x4e, 0x43}) ||
				!test(rtoksl, 0, []byte{0x52, 0x54, 0x4f, 0x4b}) {
				fmt.Printf("RSYN (T%d): len(%d) %v\n", counts["RSYN"], len(rsynsl), rsynsl)
				fmt.Printf("SYNC (T%06d): len(%d) %v\n", counts["RSYN"], len(syncsl), syncsl)
				fmt.Printf("RTOK (T%d): len(%d) %v\n", counts["RSYN"], len(rtoksl), rtoksl)
				panic("RSYN/SYNC/RTOK did not follow expected byte pattern")
			}

			emptytick = rsynsl[4] == 0xc
			if firstmove == 0 && !emptytick {
				firstmove = counts["RSYN"]
			}

			synchead := syncsl[0:12]
			syncbody := syncsl[12:]

			commands := int(syncsl[10])
			counts["GCE"] += commands
			cmdsize := 0
			if commands != 0 {
				cmdsize = len(syncbody) / commands

				for c := 0; c < int(commands); c++ {
					cmd := syncbody[cmdsize*0 : cmdsize*1]
					lead := cmd[0:4]
					buff := cmd[4:8]
					rest := cmd[8:]

					if _, exists := players[rest[2]]; !exists {
						players[rest[2]] = make(map[string]int)
					}

					if _, exists := players[rest[2]][string(lead)]; !exists {
						players[rest[2]][string(lead)] = 0
					}

					players[rest[2]][string(lead)] += 1

					if !test(cmd, 0, []byte{0x47, 0x43}) { // command starts GC
						fmt.Printf("%q\n", syncsl)
						panic("SYNC command did not follow expected byte pattern with [0:2]=0x47 0x43 0x45")
					}

					if *printgc != "" {

						if *printgc == "all" || *printgc == string(lead) {
							fmt.Printf("T%08d: %s : %q : %v\n", counts["RSYN"], string(lead), buff, rest)
						}
					}
				}
			}

			if *printcmds && (!*ignorenonactions || !emptytick) {
				if *printcmds {
					fmt.Printf("RSYN (T%06d): len(%d) %v\n", counts["RSYN"], len(rsynsl), rsynsl)
					fmt.Printf("SYNC (T%06d): len(%d) %v\n", counts["SYNC"], len(syncsl), syncsl)

					fmt.Printf("SYNC header   : %v (total sync len: %d; cmd in sync body: %d, cmd len: %d)\n", synchead, bytestosync, commands, cmdsize)

					for c := 0; c < int(commands); c++ {
						cmd := syncbody[cmdsize*0 : cmdsize*1]
						fmt.Printf("CMD%02d    %s : %v\n", c+1, string(cmd[0:4]), cmd[4:])
						fmt.Printf("`%v`\n", string(cmd[4:]))
					}

					fmt.Printf("RTOK (T%06d): len(%d) %v\n", counts["RTOK"], len(rtoksl), rtoksl)
					if *stallcmds {
						fmt.Scanln()
					}
				}
			}

			i += 8 + bytestosync + 18
		}

	}

	if *summary {
		seconds := counts["RSYN"] / 12
		time := fmt.Sprintf("%d:%02d", seconds/60, seconds%60)

		fmt.Printf("Game duration: %d ticks / %s time (Fast)\n", counts["RSYN"], time)
		fmt.Printf("Total GC issued : %d\n", counts["GCE"])
		fmt.Printf("First action : %d ticks\n", firstmove)

		fmt.Printf("Player summary:\n")
		for k := range players {
			fmt.Printf("Player ID %b:\n", k)
			for j := range players[k] {
				fmt.Printf("%s: %d\n", j, players[k][j])
			}
		}
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
