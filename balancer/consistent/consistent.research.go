package consistent

import (
	"fmt"
	"sort"
)

func Test() {
	// vnode matrix
	// x: bm servers (bmnodes)
	// y: cloud servers (nodes)
	var nodes, bmnodes = 18, 9
	var vnodes = nodes * bmnodes

	// create hash-ring
	//
	// 2 type of hash ring we can use
	// - 1 type: all cloud nodes take media from all bm nodes
	// - 2 type: (nodes / bmnodes) take media from 1 bm node

	// for the first type we need assign all vnodes horizontally
	// for the second type - vertically

	// the cons of the first method - ??
	// the cons of the second method
	// - full resharding after new new added;
	// - only (nodes/bmnodes) cloud can reach 1 bm server; hard mapping of cloud servers

	ringvirts := make(map[int]int, vnodes)
	ringnodes := make(map[int][]int, nodes)
	bmqueue := make(chan int, bmnodes)

	for node := bmnodes; node != 0; node-- {
		bmqueue <- node
	}

	// second type
	// for i := vnodes; i != 0; {
	// 	for k := nodes; k != 0; k-- {
	// 		ringnodes[k] = append(ringnodes[k], i)
	// 		ringvirts[i] = k
	// 		i--
	// 	}
	// }

	// first type
	for i := nodes; i != 0; i-- {
		for k := bmnodes; k != 0; k-- {
			ringvirts[i*bmnodes-k+1] = i
			ringnodes[i] = append(ringnodes[i], i*bmnodes-k+1)
		}
	}

	// failovering
	var failednode = 1
	var failedvirts = make(chan int, bmnodes)

	// search for failed virts
	for virt, node := range ringvirts {
		if node == failednode {
			failedvirts <- virt
		}
	}

	fmt.Printf("length of failed virts %d\n", len(failedvirts))

	// reassign failed virts
	fmt.Printf("reassigned virts - \t")
	for node := nodes; node != 0; node-- {
		if len(failedvirts) == 0 {
			break
		}

		virt := <-failedvirts
		fmt.Printf("%d ", virt)
		ringvirts[virt] = node
		ringnodes[node] = append(ringnodes[node], virt)
	}
	fmt.Println("")

	fmt.Println("")

	// debug
	for node := 1; node <= nodes; node++ {
		sort.Slice(ringnodes[node], func(i, j int) bool {
			return ringnodes[node][i] < ringnodes[node][j]
		})

		fmt.Printf("NODE %d -\t", node)

		for _, virt := range ringnodes[node] {
			fmt.Printf("%d ", virt)
		}

		fmt.Println("")
	}

	fmt.Println("")

	debugnode := make(map[int][]int)
	for virt := range ringvirts {
		bm := virt % bmnodes
		debugnode[bm+1] = append(debugnode[bm+1], virt)
	}

	for bm := 1; bm <= bmnodes; bm++ {
		sort.Slice(debugnode[bm], func(i, j int) bool {
			return debugnode[bm][i] < debugnode[bm][j]
		})

		fmt.Printf("BMNODE %d -\t", bm)

		for _, node := range debugnode[bm] {
			fmt.Printf("%d ", node)
		}

		fmt.Println("")

	}

	fmt.Println("")

	debugnode = make(map[int][]int)
	for virt, node := range ringvirts {
		bm := virt % bmnodes
		debugnode[bm+1] = append(debugnode[bm+1], node)
	}

	for bm := 1; bm <= bmnodes; bm++ {
		sort.Slice(debugnode[bm], func(i, j int) bool {
			return debugnode[bm][i] < debugnode[bm][j]
		})

		fmt.Printf("BMNODE %d -\t", bm)

		for _, node := range debugnode[bm] {
			fmt.Printf("%d ", node)
		}

		fmt.Println("")

	}
}

/*
SECOND TYPE ALLOCATION:
NODE 1 -	1 13 25 37 49 61 73 85 97
NODE 2 -	2 14 26 38 50 62 74 86 98
NODE 3 -	3 15 27 39 51 63 75 87 99
NODE 4 -	4 16 28 40 52 64 76 88 100
NODE 5 -	5 17 29 41 53 65 77 89 101
NODE 6 -	6 18 30 42 54 66 78 90 102
NODE 7 -	7 19 31 43 55 67 79 91 103
NODE 8 -	8 20 32 44 56 68 80 92 104
NODE 9 -	9 21 33 45 57 69 81 93 105
NODE 10 -	10 22 34 46 58 70 82 94 106
NODE 11 -	11 23 35 47 59 71 83 95 107
NODE 12 -	12 24 36 48 60 72 84 96 108

BMNODE 1 -	9 18 27 36 45 54 63 72 81 90 99 108
BMNODE 2 -	1 10 19 28 37 46 55 64 73 82 91 100
BMNODE 3 -	2 11 20 29 38 47 56 65 74 83 92 101
BMNODE 4 -	3 12 21 30 39 48 57 66 75 84 93 102
BMNODE 5 -	4 13 22 31 40 49 58 67 76 85 94 103
BMNODE 6 -	5 14 23 32 41 50 59 68 77 86 95 104
BMNODE 7 -	6 15 24 33 42 51 60 69 78 87 96 105
BMNODE 8 -	7 16 25 34 43 52 61 70 79 88 97 106
BMNODE 9 -	8 17 26 35 44 53 62 71 80 89 98 107

BMNODE 1 -	3 3 3 6 6 6 9 9 9 12 12 12
BMNODE 2 -	1 1 1 4 4 4 7 7 7 10 10 10
BMNODE 3 -	2 2 2 5 5 5 8 8 8 11 11 11
BMNODE 4 -	3 3 3 6 6 6 9 9 9 12 12 12
BMNODE 5 -	1 1 1 4 4 4 7 7 7 10 10 10
BMNODE 6 -	2 2 2 5 5 5 8 8 8 11 11 11
BMNODE 7 -	3 3 3 6 6 6 9 9 9 12 12 12
BMNODE 8 -	1 1 1 4 4 4 7 7 7 10 10 10
BMNODE 9 -	2 2 2 5 5 5 8 8 8 11 11 11


FIRST TYPE ALLOCATION:
NODE 1 -	1 2 3 4 5 6 7 8 9
NODE 2 -	10 11 12 13 14 15 16 17 18
NODE 3 -	19 20 21 22 23 24 25 26 27
NODE 4 -	28 29 30 31 32 33 34 35 36
NODE 5 -	37 38 39 40 41 42 43 44 45
NODE 6 -	46 47 48 49 50 51 52 53 54
NODE 7 -	55 56 57 58 59 60 61 62 63
NODE 8 -	64 65 66 67 68 69 70 71 72
NODE 9 -	73 74 75 76 77 78 79 80 81
NODE 10 -	82 83 84 85 86 87 88 89 90
NODE 11 -	91 92 93 94 95 96 97 98 99
NODE 12 -	100 101 102 103 104 105 106 107 108

BMNODE 1 -	9 18 27 36 45 54 63 72 81 90 99 108
BMNODE 2 -	1 10 19 28 37 46 55 64 73 82 91 100
BMNODE 3 -	2 11 20 29 38 47 56 65 74 83 92 101
BMNODE 4 -	3 12 21 30 39 48 57 66 75 84 93 102
BMNODE 5 -	4 13 22 31 40 49 58 67 76 85 94 103
BMNODE 6 -	5 14 23 32 41 50 59 68 77 86 95 104
BMNODE 7 -	6 15 24 33 42 51 60 69 78 87 96 105
BMNODE 8 -	7 16 25 34 43 52 61 70 79 88 97 106
BMNODE 9 -	8 17 26 35 44 53 62 71 80 89 98 107

BMNODE 1 -	1 2 3 4 5 6 7 8 9 10 11 12
BMNODE 2 -	1 2 3 4 5 6 7 8 9 10 11 12
BMNODE 3 -	1 2 3 4 5 6 7 8 9 10 11 12
BMNODE 4 -	1 2 3 4 5 6 7 8 9 10 11 12
BMNODE 5 -	1 2 3 4 5 6 7 8 9 10 11 12
BMNODE 6 -	1 2 3 4 5 6 7 8 9 10 11 12
BMNODE 7 -	1 2 3 4 5 6 7 8 9 10 11 12
BMNODE 8 -	1 2 3 4 5 6 7 8 9 10 11 12
BMNODE 9 -	1 2 3 4 5 6 7 8 9 10 11 12


FAILOVER ASSIGNMENT: TYPE 1
length of failed virts 9
reassigned virts - 	3 2 7 4 8 5 6 1 9

NODE 1 -	1 2 3 4 5 6 7 8 9
NODE 2 -	10 11 12 13 14 15 16 17 18
NODE 3 -	19 20 21 22 23 24 25 26 27
NODE 4 -	28 29 30 31 32 33 34 35 36
NODE 5 -	37 38 39 40 41 42 43 44 45
NODE 6 -	46 47 48 49 50 51 52 53 54
NODE 7 -	55 56 57 58 59 60 61 62 63
NODE 8 -	64 65 66 67 68 69 70 71 72
NODE 9 -	73 74 75 76 77 78 79 80 81
NODE 10 -	9 82 83 84 85 86 87 88 89 90
NODE 11 -	1 91 92 93 94 95 96 97 98 99
NODE 12 -	6 100 101 102 103 104 105 106 107 108
NODE 13 -	5 109 110 111 112 113 114 115 116 117
NODE 14 -	8 118 119 120 121 122 123 124 125 126
NODE 15 -	4 127 128 129 130 131 132 133 134 135
NODE 16 -	7 136 137 138 139 140 141 142 143 144
NODE 17 -	2 145 146 147 148 149 150 151 152 153
NODE 18 -	3 154 155 156 157 158 159 160 161 162

BMNODE 1 -	9 18 27 36 45 54 63 72 81 90 99 108 117 126 135 144 153 162
BMNODE 2 -	1 10 19 28 37 46 55 64 73 82 91 100 109 118 127 136 145 154
BMNODE 3 -	2 11 20 29 38 47 56 65 74 83 92 101 110 119 128 137 146 155
BMNODE 4 -	3 12 21 30 39 48 57 66 75 84 93 102 111 120 129 138 147 156
BMNODE 5 -	4 13 22 31 40 49 58 67 76 85 94 103 112 121 130 139 148 157
BMNODE 6 -	5 14 23 32 41 50 59 68 77 86 95 104 113 122 131 140 149 158
BMNODE 7 -	6 15 24 33 42 51 60 69 78 87 96 105 114 123 132 141 150 159
BMNODE 8 -	7 16 25 34 43 52 61 70 79 88 97 106 115 124 133 142 151 160
BMNODE 9 -	8 17 26 35 44 53 62 71 80 89 98 107 116 125 134 143 152 161

BMNODE 1 -	2 3 4 5 6 7 8 9 10 10 11 12 13 14 15 16 17 18
BMNODE 2 -	2 3 4 5 6 7 8 9 10 11 11 12 13 14 15 16 17 18
BMNODE 3 -	2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 17 18
BMNODE 4 -	2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 17 18 18
BMNODE 5 -	2 3 4 5 6 7 8 9 10 11 12 13 14 15 15 16 17 18
BMNODE 6 -	2 3 4 5 6 7 8 9 10 11 12 13 13 14 15 16 17 18
BMNODE 7 -	2 3 4 5 6 7 8 9 10 11 12 12 13 14 15 16 17 18
BMNODE 8 -	2 3 4 5 6 7 8 9 10 11 12 13 14 15 16 16 17 18
BMNODE 9 -	2 3 4 5 6 7 8 9 10 11 12 13 14 14 15 16 17 18

FAILOVER ASSIGNMENT: TYPE 2
length of failed virts 9
reassigned virts - 	145 55 91 73 19 37 127 109 1

NODE 1 -	1 19 37 55 73 91 109 127 145
NODE 2 -	2 20 38 56 74 92 110 128 146
NODE 3 -	3 21 39 57 75 93 111 129 147
NODE 4 -	4 22 40 58 76 94 112 130 148
NODE 5 -	5 23 41 59 77 95 113 131 149
NODE 6 -	6 24 42 60 78 96 114 132 150
NODE 7 -	7 25 43 61 79 97 115 133 151
NODE 8 -	8 26 44 62 80 98 116 134 152
NODE 9 -	9 27 45 63 81 99 117 135 153
NODE 10 -	1 10 28 46 64 82 100 118 136 154
NODE 11 -	11 29 47 65 83 101 109 119 137 155
NODE 12 -	12 30 48 66 84 102 120 127 138 156
NODE 13 -	13 31 37 49 67 85 103 121 139 157
NODE 14 -	14 19 32 50 68 86 104 122 140 158
NODE 15 -	15 33 51 69 73 87 105 123 141 159
NODE 16 -	16 34 52 70 88 91 106 124 142 160
NODE 17 -	17 35 53 55 71 89 107 125 143 161
NODE 18 -	18 36 54 72 90 108 126 144 145 162

BMNODE 1 -	9 18 27 36 45 54 63 72 81 90 99 108 117 126 135 144 153 162
BMNODE 2 -	1 10 19 28 37 46 55 64 73 82 91 100 109 118 127 136 145 154
BMNODE 3 -	2 11 20 29 38 47 56 65 74 83 92 101 110 119 128 137 146 155
BMNODE 4 -	3 12 21 30 39 48 57 66 75 84 93 102 111 120 129 138 147 156
BMNODE 5 -	4 13 22 31 40 49 58 67 76 85 94 103 112 121 130 139 148 157
BMNODE 6 -	5 14 23 32 41 50 59 68 77 86 95 104 113 122 131 140 149 158
BMNODE 7 -	6 15 24 33 42 51 60 69 78 87 96 105 114 123 132 141 150 159
BMNODE 8 -	7 16 25 34 43 52 61 70 79 88 97 106 115 124 133 142 151 160
BMNODE 9 -	8 17 26 35 44 53 62 71 80 89 98 107 116 125 134 143 152 161

BMNODE 1 -	9 9 9 9 9 9 9 9 9 18 18 18 18 18 18 18 18 18
BMNODE 2 -	10 10 10 10 10 10 10 10 10 10 11 12 13 14 15 16 17 18
BMNODE 3 -	2 2 2 2 2 2 2 2 2 11 11 11 11 11 11 11 11 11
BMNODE 4 -	3 3 3 3 3 3 3 3 3 12 12 12 12 12 12 12 12 12
BMNODE 5 -	4 4 4 4 4 4 4 4 4 13 13 13 13 13 13 13 13 13
BMNODE 6 -	5 5 5 5 5 5 5 5 5 14 14 14 14 14 14 14 14 14
BMNODE 7 -	6 6 6 6 6 6 6 6 6 15 15 15 15 15 15 15 15 15
BMNODE 8 -	7 7 7 7 7 7 7 7 7 16 16 16 16 16 16 16 16 16
BMNODE 9 -	8 8 8 8 8 8 8 8 8 17 17 17 17 17 17 17 17 17

*/
